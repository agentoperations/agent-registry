package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/agentoperations/agent-registry/internal/model"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type apiResponse struct {
	Data       json.RawMessage  `json:"data"`
	Meta       *model.ResponseMeta `json:"_meta"`
	Pagination *model.Pagination   `json:"pagination"`
}

func (c *Client) do(method, path string, body interface{}) (json.RawMessage, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var problem model.ProblemDetail
		if err := json.Unmarshal(respBody, &problem); err == nil && problem.Detail != "" {
			return nil, fmt.Errorf("%s (HTTP %d)", problem.Detail, resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return respBody, nil
	}
	return apiResp.Data, nil
}

func (c *Client) CreateArtifact(kind string, artifact *model.RegistryArtifact) (*model.RegistryArtifact, error) {
	data, err := c.do("POST", fmt.Sprintf("/api/v1/%s", kind), artifact)
	if err != nil {
		return nil, err
	}
	var result model.RegistryArtifact
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetArtifact(kind, name, version string) (*model.RegistryArtifact, error) {
	parts := splitName(name)
	path := fmt.Sprintf("/api/v1/%s/%s/versions/%s", kind, parts, version)
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var result model.RegistryArtifact
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListArtifacts(kind string, status, category string) ([]*model.RegistryArtifact, error) {
	path := fmt.Sprintf("/api/v1/%s?limit=100", kind)
	if status != "" {
		path += "&status=" + url.QueryEscape(status)
	}
	if category != "" {
		path += "&category=" + url.QueryEscape(category)
	}
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var result []*model.RegistryArtifact
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DeleteArtifact(kind, name, version string) error {
	parts := splitName(name)
	_, err := c.do("DELETE", fmt.Sprintf("/api/v1/%s/%s/versions/%s", kind, parts, version), nil)
	return err
}

func (c *Client) Promote(kind, name, version string, req *model.PromotionRequest) (*model.RegistryArtifact, error) {
	parts := splitName(name)
	data, err := c.do("POST", fmt.Sprintf("/api/v1/%s/%s/versions/%s/promote", kind, parts, version), req)
	if err != nil {
		return nil, err
	}
	var result model.RegistryArtifact
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SubmitEval(kind, name, version string, eval *model.EvalRecord) (*model.EvalRecord, error) {
	parts := splitName(name)
	data, err := c.do("POST", fmt.Sprintf("/api/v1/%s/%s/versions/%s/evals", kind, parts, version), eval)
	if err != nil {
		return nil, err
	}
	var result model.EvalRecord
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListEvals(kind, name, version string, category string) ([]*model.EvalRecord, error) {
	parts := splitName(name)
	path := fmt.Sprintf("/api/v1/%s/%s/versions/%s/evals", kind, parts, version)
	if category != "" {
		path += "?category=" + url.QueryEscape(category)
	}
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var result []*model.EvalRecord
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Inspect(kind, name, version string) (*model.InspectResult, error) {
	parts := splitName(name)
	data, err := c.do("GET", fmt.Sprintf("/api/v1/%s/%s/versions/%s/inspect", kind, parts, version), nil)
	if err != nil {
		return nil, err
	}
	var result model.InspectResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetDependencies(kind, name, version string) (*model.DependencyGraph, error) {
	parts := splitName(name)
	data, err := c.do("GET", fmt.Sprintf("/api/v1/%s/%s/versions/%s/dependencies", kind, parts, version), nil)
	if err != nil {
		return nil, err
	}
	var result model.DependencyGraph
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Search(query string, kind string) ([]*model.RegistryArtifact, error) {
	path := fmt.Sprintf("/api/v1/search?q=%s", url.QueryEscape(query))
	if kind != "" {
		path += "&kind=" + url.QueryEscape(kind)
	}
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var result []*model.RegistryArtifact
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Ping() error {
	_, err := c.do("GET", "/api/v1/ping", nil)
	return err
}

func (c *Client) ExportStandardDoc(kind, name, version string) (json.RawMessage, error) {
	parts := splitName(name)
	path := fmt.Sprintf("/api/v1/%s/%s/versions/%s/export", kind, parts, version)

	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var problem model.ProblemDetail
		if err := json.Unmarshal(body, &problem); err == nil && problem.Detail != "" {
			return nil, fmt.Errorf("%s (HTTP %d)", problem.Detail, resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// splitName returns the name as a URL path segment: "acme/my-agent" -> "acme/my-agent"
func splitName(name string) string {
	return name
}

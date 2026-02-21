package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/agentregistry/agent-registry/internal/model"
	"github.com/agentregistry/agent-registry/internal/service"
	"github.com/agentregistry/agent-registry/internal/store"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc service.RegistryService
}

func New(svc service.RegistryService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

// --- Artifact CRUD ---

func (h *Handler) CreateArtifact(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	var artifact model.RegistryArtifact
	if err := json.Unmarshal(body, &artifact); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	created, err := h.svc.CreateArtifact(r.Context(), kind, &artifact)
	if err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "artifact already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created, nil)
}

func (h *Handler) GetArtifact(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)
	version := chi.URLParam(r, "version")

	artifact, err := h.svc.GetArtifact(r.Context(), kind, name, version)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, artifact, nil)
}

func (h *Handler) ListArtifacts(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	limit, offset := parsePagination(r)
	filter := &model.ArtifactFilter{
		Status:   model.Status(r.URL.Query().Get("status")),
		Category: r.URL.Query().Get("category"),
	}

	artifacts, total, err := h.svc.ListArtifacts(r.Context(), kind, filter, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if artifacts == nil {
		artifacts = []*model.RegistryArtifact{}
	}
	writeJSON(w, http.StatusOK, artifacts, &model.Pagination{
		Count: len(artifacts),
		Total: total,
	})
}

func (h *Handler) ListVersions(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)

	versions, err := h.svc.ListVersions(r.Context(), kind, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if versions == nil {
		versions = []*model.RegistryArtifact{}
	}
	writeJSON(w, http.StatusOK, versions, nil)
}

func (h *Handler) DeleteArtifact(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)
	version := chi.URLParam(r, "version")

	if err := h.svc.DeleteArtifact(r.Context(), kind, name, version); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found or not in draft status")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Promotion ---

func (h *Handler) PromoteArtifact(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)
	version := chi.URLParam(r, "version")

	var req model.PromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	artifact, err := h.svc.PromoteArtifact(r.Context(), kind, name, version, &req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		var gateErr *model.PromotionGateError
		if errors.As(err, &gateErr) {
			writeError(w, http.StatusConflict, gateErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, artifact, nil)
}

// --- Evals ---

func (h *Handler) SubmitEval(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)
	version := chi.URLParam(r, "version")

	var eval model.EvalRecord
	if err := json.NewDecoder(r.Body).Decode(&eval); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	created, err := h.svc.SubmitEval(r.Context(), kind, name, version, &eval)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created, nil)
}

func (h *Handler) ListEvals(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)
	version := chi.URLParam(r, "version")

	filter := &model.EvalFilter{
		Category: model.EvalCategory(r.URL.Query().Get("category")),
	}

	evals, err := h.svc.ListEvals(r.Context(), kind, name, version, filter)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if evals == nil {
		evals = []*model.EvalRecord{}
	}
	writeJSON(w, http.StatusOK, evals, nil)
}

// --- Inspect ---

func (h *Handler) Inspect(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)
	version := chi.URLParam(r, "version")

	result, err := h.svc.Inspect(r.Context(), kind, name, version)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result, nil)
}

// --- Search ---

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	limit, offset := parsePagination(r)

	var kinds []model.Kind
	if k := r.URL.Query().Get("kind"); k != "" {
		if kind, ok := model.ParseKind(k); ok {
			kinds = append(kinds, kind)
		}
	}

	artifacts, total, err := h.svc.Search(r.Context(), query, kinds, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if artifacts == nil {
		artifacts = []*model.RegistryArtifact{}
	}
	writeJSON(w, http.StatusOK, artifacts, &model.Pagination{
		Count: len(artifacts),
		Total: total,
	})
}

// --- Dependencies ---

func (h *Handler) GetDependencies(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)
	version := chi.URLParam(r, "version")

	graph, err := h.svc.GetDependencies(r.Context(), kind, name, version)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, graph, nil)
}

// --- Helpers ---

func parseKind(w http.ResponseWriter, r *http.Request) (model.Kind, bool) {
	kindStr := chi.URLParam(r, "kind")
	kind, ok := model.ParseKind(kindStr)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid artifact kind: "+kindStr)
		return "", false
	}
	return kind, true
}

func extractName(r *http.Request) string {
	ns := chi.URLParam(r, "ns")
	artifact := chi.URLParam(r, "artifact")
	if ns != "" && artifact != "" {
		return ns + "/" + artifact
	}
	// Fallback to wildcard
	wildcard := chi.URLParam(r, "*")
	decoded, err := url.PathUnescape(wildcard)
	if err != nil {
		return wildcard
	}
	return decoded
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 30
	offset = 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	return
}

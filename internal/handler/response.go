package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/google/uuid"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}, pagination *model.Pagination) {
	env := model.ResponseEnvelope{
		Data: data,
		Meta: &model.ResponseMeta{
			RequestID: uuid.New().String(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Registry:  "localhost",
		},
		Pagination: pagination,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(env); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, detail string) {
	titles := map[int]string{
		400: "Bad Request",
		404: "Not Found",
		409: "Conflict",
		422: "Unprocessable Entity",
		500: "Internal Server Error",
	}
	title := titles[status]
	if title == "" {
		title = "Error"
	}
	problem := model.ProblemDetail{
		Type:   "https://agentregistry.dev/errors/" + http.StatusText(status),
		Title:  title,
		Status: status,
		Detail: detail,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		log.Printf("failed to encode error response: %v", err)
	}
}

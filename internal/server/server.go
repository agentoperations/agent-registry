package server

import (
	"io/fs"
	"net/http"

	"github.com/agentregistry/agent-registry/internal/handler"
	"github.com/agentregistry/agent-registry/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(svc service.RegistryService, uiFS fs.FS) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	h := handler.New(svc)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", h.Ping)
		r.Get("/search", h.Search)
		r.Get("/{kind}", h.ListArtifacts)
		r.Post("/{kind}", h.CreateArtifact)
		r.Get("/{kind}/{ns}/{artifact}/versions", h.ListVersions)

		r.Route("/{kind}/{ns}/{artifact}/versions/{version}", func(r chi.Router) {
			r.Get("/", h.GetArtifact)
			r.Delete("/", h.DeleteArtifact)
			r.Post("/promote", h.PromoteArtifact)
			r.Post("/evals", h.SubmitEval)
			r.Get("/evals", h.ListEvals)
			r.Get("/inspect", h.Inspect)
			r.Get("/dependencies", h.GetDependencies)
		})
	})

	// UI — serve embedded index.html
	if uiFS != nil {
		fileServer := http.FileServer(http.FS(uiFS))
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			fileServer.ServeHTTP(w, r)
		})
	}

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

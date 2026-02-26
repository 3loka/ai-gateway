package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/trilok/ai-gateway/internal/config"
	"github.com/trilok/ai-gateway/internal/gateway"
)

// Server wraps the HTTP server and the Gateway.
type Server struct {
	gw     *gateway.Gateway
	cfg    config.ServerConfig
	mux    *http.ServeMux
}

// New creates the server and registers all routes.
func New(gw *gateway.Gateway, cfg config.ServerConfig) *Server {
	s := &Server{gw: gw, cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /v1/generate", s.chain(s.handleGenerate))
	s.mux.HandleFunc("GET /v1/health", s.chain(s.handleHealth))
	s.mux.HandleFunc("GET /v1/profiles", s.chain(s.handleProfiles))
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("ai-gateway listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// ---- Middleware chain ----

type handlerFunc func(w http.ResponseWriter, r *http.Request)

func (s *Server) chain(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth
		if s.cfg.APIKey != "" {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.Header.Get("Authorization") // accept "Bearer <key>" too
				if len(key) > 7 && key[:7] == "Bearer " {
					key = key[7:]
				}
			}
			if key != s.cfg.APIKey {
				writeError(w, http.StatusUnauthorized, "invalid or missing X-API-Key")
				return
			}
		}

		// Logging
		if s.cfg.LogRequests {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w, status: 200}
			h(lrw, r)
			log.Printf("%s %s → %d (%dms)", r.Method, r.URL.Path, lrw.status, time.Since(start).Milliseconds())
			return
		}

		h(w, r)
	}
}

// ---- Handlers ----

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req gateway.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	resp, err := s.gw.Generate(ctx, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	statuses := s.gw.Health(ctx)
	writeJSON(w, http.StatusOK, map[string]any{"providers": statuses})
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"profiles": s.gw.Profiles()})
}

// ---- Helpers ----

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// loggingResponseWriter captures the status code for logging.
type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}

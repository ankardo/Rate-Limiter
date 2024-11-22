package webserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/ankardo/Rate-Limiter/config/logger"
)

func NewRouter(rateLimiterMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(rateLimiterMiddleware)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Request received", zap.String("path", r.URL.Path))
		w.Write([]byte("Welcome to the Rate Limiter!"))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Health check endpoint hit")
		writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	r.Get("/ip", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		if ip == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing IP in query parameters"})
			return
		}

		logger.Info("IP request processed", zap.String("ip", ip))
		writeJSON(w, http.StatusOK, map[string]string{"message": "IP request handled successfully", "ip": ip})
	})

	r.Get("/token", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing token in query parameters"})
			return
		}

		logger.Info("Token request processed", zap.String("token", token))
		writeJSON(w, http.StatusOK, map[string]string{"message": "Token request handled successfully", "token": token})
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

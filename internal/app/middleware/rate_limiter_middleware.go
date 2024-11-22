package middleware

import (
	"net"
	"net/http"

	"github.com/ankardo/Rate-Limiter/config/logger"
	"github.com/ankardo/Rate-Limiter/internal/domain"
	"go.uber.org/zap"
)

func RateLimiterMiddleware(limiter domain.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var key string
			isToken := false

			if token := r.Header.Get("API_KEY"); token != "" {
				key = token
				isToken = true
			} else if r.URL.Path == "/token" {
				queryToken := r.URL.Query().Get("token")
				if queryToken == "" {
					http.Error(w, "Missing token in query parameters", http.StatusBadRequest)
					return
				}
				key = queryToken
				isToken = true
			} else if r.URL.Path == "/ip" {
				queryIP := r.URL.Query().Get("ip")
				if queryIP == "" {
					http.Error(w, "Missing IP in query parameters", http.StatusBadRequest)
					return
				}
				key = queryIP
			} else {
				ip, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					logger.Error("Failed to parse IP address", err, zap.String("RemoteAddr", r.RemoteAddr))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				key = ip
			}

			logger.Debug("Processing request", zap.String("key", key), zap.Bool("isToken", isToken))

			allowed, err := limiter.AllowRequest(key, isToken)
			if err != nil {
				logger.Error("Rate limiter error", err, zap.String("key", key))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ankardo/Rate-Limiter/internal/app/limiter"
	"github.com/ankardo/Rate-Limiter/internal/domain"
)

func TestRateLimiterMiddleware(t *testing.T) {
	config := domain.LimiterConfig{
		MaxRequests:   2,
		BlockDuration: 10,
	}
	rateLimiter := limiter.NewMemoryRateLimiter(config)

	handler := RateLimiterMiddleware(rateLimiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	server := httptest.NewServer(handler)
	defer server.Close()

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest("GET", server.URL, nil)
		res := httptest.NewRecorder()

		handler.ServeHTTP(res, req)

		if i <= 2 && res.Code != http.StatusOK {
			t.Fatalf("Request %d should have been allowed but got %d", i, res.Code)
		}
		if i > 2 && res.Code != http.StatusTooManyRequests {
			t.Fatalf("Request %d should have been blocked but got %d", i, res.Code)
		}
	}
}

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ankardo/Rate-Limiter/internal/app/limiter"
	"github.com/ankardo/Rate-Limiter/internal/app/middleware"
	"github.com/ankardo/Rate-Limiter/internal/domain"
)

func TestMemoryRateLimiterMiddlewareIntegration(t *testing.T) {
	memoryLimiter := limiter.NewMemoryRateLimiter(domain.LimiterConfig{
		MaxRequests:      5,
		TokenMaxRequests: 10,
		BlockDuration:    3,
	})

	handler := middleware.RateLimiterMiddleware(memoryLimiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{}

	tests := []struct {
		name         string
		key          string
		headerKey    string
		iterations   int
		expectStatus []int
	}{
		{
			name:         "Limit by IP",
			key:          "192.168.1.1",
			headerKey:    "",
			iterations:   6,
			expectStatus: []int{200, 200, 200, 200, 200, 429},
		},
		{
			name:         "Limit by Token",
			key:          "test-token",
			headerKey:    "API_KEY",
			iterations:   11,
			expectStatus: []int{200, 200, 200, 200, 200, 200, 200, 200, 200, 200, 429},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.iterations; i++ {
				req, _ := http.NewRequest("GET", server.URL, nil)
				if tt.headerKey != "" {
					req.Header.Set(tt.headerKey, tt.key)
				} else {
					req.RemoteAddr = tt.key
				}

				res, err := client.Do(req)
				if err != nil {
					t.Fatalf("Request %d failed: %v", i+1, err)
				}

				if res.StatusCode != tt.expectStatus[i] {
					t.Fatalf("Iteration %d failed: expected %d, got %d", i+1, tt.expectStatus[i], res.StatusCode)
				}
			}
		})
	}
}

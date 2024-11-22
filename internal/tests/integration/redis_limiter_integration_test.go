package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ankardo/Rate-Limiter/internal/app/limiter"
	"github.com/ankardo/Rate-Limiter/internal/app/middleware"
	"github.com/ankardo/Rate-Limiter/internal/domain"
	"github.com/ankardo/Rate-Limiter/internal/infrastructure/persistence"
)

func TestRedisLimiterMiddlewareIntegration(t *testing.T) {
	ctx := context.Background()
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	client, err := persistence.NewRedisClient(ctx, redisAddr, "")
	if err != nil {
		t.Fatalf("Failed to connect to Redis at %v: %v", redisAddr, err)
	}
	defer client.Close()
	store := persistence.NewRedisStore(client)

	tests := []struct {
		name             string
		maxRequests      int
		tokenMaxRequests int
		blockDuration    int
		key              string
		headerKey        string
		iterations       int
		expectCode       []int
		blockTest        bool
	}{
		{
			name:             "Limit by IP",
			maxRequests:      5,
			tokenMaxRequests: 10,
			blockDuration:    3,
			key:              "192.168.1.1",
			headerKey:        "",
			iterations:       6,
			expectCode:       []int{200, 200, 200, 200, 200, 429},
			blockTest:        true,
		},
		{
			name:             "Limit by Token",
			maxRequests:      5,
			tokenMaxRequests: 10,
			blockDuration:    3,
			key:              "abc123",
			headerKey:        "API_KEY",
			iterations:       11,
			expectCode:       []int{200, 200, 200, 200, 200, 200, 200, 200, 200, 200, 429},
			blockTest:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.FlushDB(ctx).Err(); err != nil {
				t.Fatalf("Failed to flush Redis: %v", err)
			}

			limiterConfig := domain.LimiterConfig{
				MaxRequests:      tt.maxRequests,
				TokenMaxRequests: tt.tokenMaxRequests,
				BlockDuration:    int64(tt.blockDuration),
			}
			rateLimiter := limiter.NewRedisRateLimiter(store, limiterConfig)

			handler := middleware.RateLimiterMiddleware(rateLimiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			server := httptest.NewServer(handler)
			defer server.Close()

			httpClient := &http.Client{}

			for i := 0; i < tt.iterations; i++ {
				req, _ := http.NewRequest("GET", server.URL, nil)
				if tt.headerKey != "" {
					req.Header.Set(tt.headerKey, tt.key)
				} else {
					req.RemoteAddr = tt.key
				}
				res, err := httpClient.Do(req)
				if err != nil {
					t.Fatalf("Request %d failed: %v", i+1, err)
				}
				defer res.Body.Close()

				if res.StatusCode != tt.expectCode[i] {
					t.Fatalf("Iteration %d failed: expected %d, got %d", i+1, tt.expectCode[i], res.StatusCode)
				}
			}

			if tt.blockTest {
				time.Sleep(time.Duration(tt.blockDuration) * time.Second)

				req, _ := http.NewRequest("GET", server.URL, nil)
				if tt.headerKey != "" {
					req.Header.Set(tt.headerKey, tt.key)
				} else {
					req.RemoteAddr = tt.key
				}
				res, err := httpClient.Do(req)
				if err != nil {
					t.Fatalf("Failed after block duration: %v", err)
				}
				defer res.Body.Close()

				if res.StatusCode != 200 {
					t.Fatalf("Failed to allow request after block duration: expected 200, got %d", res.StatusCode)
				}
			}
		})
	}
}

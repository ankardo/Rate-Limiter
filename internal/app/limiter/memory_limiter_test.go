package limiter

import (
	"testing"

	"github.com/ankardo/Rate-Limiter/internal/domain"
)

func TestMemoryRateLimiter(t *testing.T) {
	rateLimiter := NewMemoryRateLimiter(domain.LimiterConfig{
		MaxRequests:      5,
		TokenMaxRequests: 10,
		BlockDuration:    60,
	})

	tests := []struct {
		name         string
		key          string
		isToken      bool
		requests     int
		expectStatus []bool
	}{
		{
			name:         "Limit by IP",
			key:          "192.168.1.1",
			isToken:      false,
			requests:     6,
			expectStatus: []bool{true, true, true, true, true, false},
		},
		{
			name:     "Limit by Token",
			key:      "valid-token",
			isToken:  true,
			requests: 11,
			expectStatus: []bool{
				true, true, true, true, true, true, true, true,
				true, true, false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for i := 0; i < test.requests; i++ {
				allowed, _ := rateLimiter.AllowRequest(test.key, test.isToken)
				if allowed != test.expectStatus[i] {
					t.Fatalf("Test %s: Request %d: expected %v, got %v", test.name, i+1, test.expectStatus[i], allowed)
				}
			}
		})
	}
}

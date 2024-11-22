package limiter

import (
	"errors"
	"testing"
	"time"

	"github.com/ankardo/Rate-Limiter/internal/domain"
)

func TestRedisRateLimiter_BlockKey(t *testing.T) {
	mockStore := &MockRedisStore{
		SetExpirationFunc: func(key string, duration int64) error {
			if key == "test-block-key" {
				return nil
			}
			return errors.New("failed to set expiration")
		},
		GetTTLFunc: func(key string) (int64, error) {
			if key == "test-block-key" {
				return 5, nil
			}
			return 0, errors.New("key not found")
		},
	}

	redisLimiter := NewRedisRateLimiter(mockStore, domain.LimiterConfig{
		MaxRequests:      5,
		TokenMaxRequests: 10,
		BlockDuration:    5,
	})

	key := "test-block-key"

	err := redisLimiter.BlockKey(key, 5)
	if err != nil {
		t.Fatalf("Failed to block key: %v", err)
	}

	ttl, err := mockStore.GetTTL(key)
	if err != nil {
		t.Fatalf("Failed to get key TTL: %v", err)
	}

	if ttl <= 0 {
		t.Fatalf("Expected TTL > 0 for key %s, got %d", key, ttl)
	}

	mockStore.GetTTLFunc = func(key string) (int64, error) {
		return 0, errors.New("key expired")
	}

	time.Sleep(6 * time.Second)

	ttl, err = mockStore.GetTTL(key)
	if err == nil {
		t.Fatalf("Key %s should be expired, but still has TTL: %d", key, ttl)
	}
}

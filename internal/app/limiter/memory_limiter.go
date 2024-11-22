package limiter

import (
	"sync"
	"time"

	"github.com/ankardo/Rate-Limiter/internal/domain"
)

type MemoryRateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limits   map[string]time.Time
	config   domain.LimiterConfig
}

func NewMemoryRateLimiter(config domain.LimiterConfig) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		requests: make(map[string][]time.Time),
		limits:   make(map[string]time.Time),
		config:   config,
	}
}

func (m *MemoryRateLimiter) AllowRequest(key string, isToken bool) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prefixedKey := key
	if isToken {
		prefixedKey = "token:" + key
	} else {
		prefixedKey = "ip:" + key
	}

	if _, exists := m.requests[prefixedKey]; !exists {
		m.requests[prefixedKey] = []time.Time{}
	}

	now := time.Now()
	windowStart := now.Add(-time.Second)

	filtered := []time.Time{}
	for _, t := range m.requests[prefixedKey] {
		if t.After(windowStart) {
			filtered = append(filtered, t)
		}
	}
	m.requests[prefixedKey] = filtered

	limit := m.config.MaxRequests
	if isToken {
		limit = m.config.TokenMaxRequests
	}

	if len(filtered) >= limit {
		return false, nil
	}

	m.requests[prefixedKey] = append(m.requests[prefixedKey], now)
	return true, nil
}

func (m *MemoryRateLimiter) BlockKey(key string, duration int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limits[key] = time.Now().Add(time.Duration(duration) * time.Second)
	return nil
}

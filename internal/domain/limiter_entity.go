package domain

type LimiterConfig struct {
	TokenMaxRequests int
	MaxRequests      int
	BlockDuration    int64
	TTLExpiration    int64
}

type Limiter interface {
	AllowRequest(key string, isToken bool) (bool, error)
	BlockKey(key string, duration int64) error
}

type RateLimiterStore interface {
	Increment(key string) (int64, error)
	GetTTL(key string) (int64, error)
	SetExpiration(key string, duration int64) error
}

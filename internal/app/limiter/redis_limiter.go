package limiter

import (
	"context"

	"github.com/ankardo/Rate-Limiter/config/logger"
	"github.com/ankardo/Rate-Limiter/internal/domain"
	"go.uber.org/zap"
)

type RedisRateLimiter struct {
	store  domain.RateLimiterStore
	config domain.LimiterConfig
	ctx    context.Context
}

func NewRedisRateLimiter(store domain.RateLimiterStore, config domain.LimiterConfig) *RedisRateLimiter {
	return &RedisRateLimiter{
		store:  store,
		config: config,
		ctx:    context.Background(),
	}
}

func (r *RedisRateLimiter) AllowRequest(key string, isToken bool) (bool, error) {
	prefixedKey := key
	if isToken {
		prefixedKey = "token:" + key
	} else {
		prefixedKey = "ip:" + key
	}

	logger.Debug("AllowRequest called",
		zap.String("key", key),
		zap.Bool("isToken", isToken),
		zap.String("expectedPrefix", prefixedKey[:3]),
	)

	count, err := r.store.Increment(prefixedKey)
	if err != nil {
		logger.Error("Store Increment failed", err, zap.String("prefixedKey", prefixedKey))
		return false, err
	}
	logger.Debug("Store Increment result",
		zap.String("prefixedKey", prefixedKey),
		zap.Int64("count", count),
	)

	ttl, err := r.store.GetTTL(prefixedKey)
	if err != nil {
		logger.Error("Store GetTTL failed", err, zap.String("prefixedKey", prefixedKey))
		return false, err
	}
	logger.Debug("Store TTL result",
		zap.String("prefixedKey", prefixedKey),
		zap.Int64("ttl", ttl),
	)

	if ttl < 0 {
		err := r.store.SetExpiration(prefixedKey, r.config.TTLExpiration)
		if err != nil {
			logger.Error("Store SetExpiration failed", err, zap.String("prefixedKey", prefixedKey))
			return false, err
		}
		logger.Debug("Store Expiration set",
			zap.String("prefixedKey", prefixedKey),
			zap.Int64("expiry", r.config.TTLExpiration),
		)
	}

	limit := int64(r.config.MaxRequests)
	if isToken {
		limit = int64(r.config.TokenMaxRequests)
		logger.Debug("Token limit check",
			zap.String("prefixedKey", prefixedKey),
			zap.Int64("TokenMaxRequests", limit),
		)
	}

	if count > limit {
		logger.Debug("Request limit exceeded",
			zap.String("prefixedKey", prefixedKey),
			zap.Int64("count", count),
			zap.Int64("limit", limit),
		)
		_ = r.BlockKey(prefixedKey, r.config.BlockDuration)
		return false, nil
	}

	return true, nil
}

func (r *RedisRateLimiter) BlockKey(key string, duration int64) error {
	logger.Debug("Blocking key", zap.String("key", key), zap.Int64("duration", duration))
	err := r.store.SetExpiration(key, duration)
	if err != nil {
		logger.Error("Store BlockKey failed", err, zap.String("key", key))
		return err
	}
	return nil
}

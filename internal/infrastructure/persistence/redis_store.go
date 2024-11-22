package persistence

import (
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (r *RedisStore) Increment(key string) (int64, error) {
	return r.client.Incr(r.client.Context(), key).Result()
}

func (r *RedisStore) GetTTL(key string) (int64, error) {
	duration, err := r.client.TTL(r.client.Context(), key).Result()
	if err != nil {
		return 0, err
	}
	return int64(duration.Seconds()), nil
}

func (r *RedisStore) SetExpiration(key string, duration int64) error {
	return r.client.Expire(r.client.Context(), key, time.Duration(duration)*time.Second).Err()
}

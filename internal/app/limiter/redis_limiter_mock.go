package limiter

import "errors"

type MockRedisStore struct {
	SetExpirationFunc func(key string, duration int64) error
	GetTTLFunc        func(key string) (int64, error)
	IncrementFunc     func(key string) (int64, error)
}

func (m *MockRedisStore) SetExpiration(key string, duration int64) error {
	if m.SetExpirationFunc != nil {
		return m.SetExpirationFunc(key, duration)
	}
	return errors.New("SetExpirationFunc not implemented")
}

func (m *MockRedisStore) GetTTL(key string) (int64, error) {
	if m.GetTTLFunc != nil {
		return m.GetTTLFunc(key)
	}
	return 0, errors.New("GetTTLFunc not implemented")
}

func (m *MockRedisStore) Increment(key string) (int64, error) {
	if m.IncrementFunc != nil {
		return m.IncrementFunc(key)
	}
	return 0, errors.New("IncrementFunc not implemented")
}

package adapters

import (
	"context"
	"encoding/json"
	"time"

	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/pkg/cache"
)

type JSONCacheStore struct {
	inner cache.Cache
}

func NewJSONCacheStore(inner cache.Cache) iamservice.CacheStore {
	if inner == nil {
		return nil
	}
	return JSONCacheStore{inner: inner}
}

func (s JSONCacheStore) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	value, err := s.inner.Get(ctx, key)
	if err != nil {
		if cache.IsKeyNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(value), dest); err != nil {
		return false, err
	}
	return true, nil
}

func (s JSONCacheStore) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.inner.Set(ctx, key, string(raw), ttl)
}

func (s JSONCacheStore) Delete(ctx context.Context, keys ...string) error {
	return s.inner.Delete(ctx, keys...)
}

func (s JSONCacheStore) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	next, err := s.inner.Incr(ctx, key)
	if err != nil {
		return 0, err
	}
	if ttl > 0 {
		_ = s.inner.Expire(ctx, key, ttl)
	}
	return next, nil
}

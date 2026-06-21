package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type hybridCache struct {
	local Cache
	redis Cache
}

func NewHybrid(localConfig LocalConfig, redisConfig *Config, logger Logger) (Cache, error) {
	local, err := NewLocal(localConfig)
	if err != nil {
		return nil, err
	}
	redis, err := NewRedis(redisConfig, logger)
	if err != nil {
		if logger != nil {
			logger.Error("redis unavailable, hybrid cache degraded to local", "error", err)
		}
		return &hybridCache{local: local}, nil
	}
	return &hybridCache{local: local, redis: redis}, nil
}

func (h *hybridCache) Get(ctx context.Context, key string) (string, error) {
	if value, err := h.local.Get(ctx, key); err == nil {
		return value, nil
	}
	if h.redis == nil {
		return "", fmt.Errorf(ErrMsgKeyNotFound, key)
	}
	value, err := h.redis.Get(ctx, key)
	if err != nil {
		return "", err
	}
	_ = h.local.Set(ctx, key, value, 0)
	return value, nil
}

func (h *hybridCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := h.local.Set(ctx, key, value, expiration); err != nil {
		return err
	}
	if h.redis != nil {
		return h.redis.Set(ctx, key, value, expiration)
	}
	return nil
}

func (h *hybridCache) Delete(ctx context.Context, keys ...string) error {
	localErr := h.local.Delete(ctx, keys...)
	var redisErr error
	if h.redis != nil {
		redisErr = h.redis.Delete(ctx, keys...)
	}
	return errors.Join(localErr, redisErr)
}

func (h *hybridCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	seen := map[string]struct{}{}
	for _, key := range keys {
		if _, err := h.local.Get(ctx, key); err == nil {
			seen[key] = struct{}{}
		}
	}
	if h.redis != nil {
		values, err := h.redis.MGet(ctx, keys...)
		if err != nil {
			return int64(len(seen)), nil
		}
		for i, value := range values {
			if value != nil {
				seen[keys[i]] = struct{}{}
			}
		}
	}
	return int64(len(seen)), nil
}

func (h *hybridCache) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	values := make([]interface{}, len(keys))
	missing := make([]string, 0, len(keys))
	missingIndexes := make([]int, 0, len(keys))
	for i, key := range keys {
		value, err := h.local.Get(ctx, key)
		if err == nil {
			values[i] = value
			continue
		}
		missing = append(missing, key)
		missingIndexes = append(missingIndexes, i)
	}
	if h.redis != nil && len(missing) > 0 {
		redisValues, err := h.redis.MGet(ctx, missing...)
		if err == nil {
			for i, value := range redisValues {
				if value == nil {
					continue
				}
				text := fmt.Sprint(value)
				values[missingIndexes[i]] = text
				_ = h.local.Set(ctx, missing[i], text, 0)
			}
		}
	}
	return values, nil
}

func (h *hybridCache) MSet(ctx context.Context, pairs ...interface{}) error {
	if len(pairs)%2 != 0 {
		return fmt.Errorf("mset requires an even number of arguments")
	}
	if err := h.local.MSet(ctx, pairs...); err != nil {
		return err
	}
	if h.redis != nil {
		return h.redis.MSet(ctx, pairs...)
	}
	return nil
}

func (h *hybridCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	localErr := h.local.Expire(ctx, key, expiration)
	var redisErr error
	if h.redis != nil {
		redisErr = h.redis.Expire(ctx, key, expiration)
	}
	return errors.Join(localErr, redisErr)
}

func (h *hybridCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if h.redis != nil {
		return h.redis.TTL(ctx, key)
	}
	return h.local.TTL(ctx, key)
}

func (h *hybridCache) Incr(ctx context.Context, key string) (int64, error) {
	return h.IncrBy(ctx, key, 1)
}

func (h *hybridCache) Decr(ctx context.Context, key string) (int64, error) {
	return h.IncrBy(ctx, key, -1)
}

func (h *hybridCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	next, err := h.local.IncrBy(ctx, key, value)
	if err != nil {
		return 0, err
	}
	if h.redis != nil {
		if redisNext, redisErr := h.redis.IncrBy(ctx, key, value); redisErr == nil {
			next = redisNext
			_ = h.local.Set(ctx, key, next, 0)
		}
	}
	return next, nil
}

func (h *hybridCache) Ping(ctx context.Context) error {
	if err := h.local.Ping(ctx); err != nil {
		return err
	}
	if h.redis != nil {
		return h.redis.Ping(ctx)
	}
	return nil
}

func (h *hybridCache) Close() error {
	var errs []error
	if h.local != nil {
		errs = append(errs, h.local.Close())
	}
	if h.redis != nil {
		errs = append(errs, h.redis.Close())
	}
	return errors.Join(errs...)
}

func (h *hybridCache) Reload(ctx context.Context, config *Config) error {
	if h.redis == nil {
		return nil
	}
	return h.redis.Reload(ctx, config)
}

func IsKeyNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "cache key not found:")
}

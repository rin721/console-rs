package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/dgraph-io/ristretto"
)

type localCache struct {
	cache *ristretto.Cache
	cfg   LocalConfig
}

type localEntry struct {
	Value     string
	ExpiresAt time.Time
	HasExpiry bool
}

func NewLocal(config LocalConfig) (Cache, error) {
	config = config.normalized()
	client, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxCost,
		BufferItems: config.BufferItems,
	})
	if err != nil {
		return nil, err
	}
	return &localCache{cache: client, cfg: config}, nil
}

func (l *localCache) Get(_ context.Context, key string) (string, error) {
	raw, ok := l.cache.Get(key)
	if !ok {
		return "", fmt.Errorf(ErrMsgKeyNotFound, key)
	}
	entry, ok := raw.(localEntry)
	if !ok {
		return fmt.Sprint(raw), nil
	}
	if entry.HasExpiry && time.Now().After(entry.ExpiresAt) {
		l.cache.Del(key)
		l.cache.Wait()
		return "", fmt.Errorf(ErrMsgKeyNotFound, key)
	}
	return entry.Value, nil
}

func (l *localCache) Set(_ context.Context, key string, value interface{}, expiration time.Duration) error {
	valueText := fmt.Sprint(value)
	if expiration == 0 && l.cfg.DefaultTTL > 0 {
		expiration = l.cfg.DefaultTTL
	}
	entry := localEntry{Value: valueText}
	if expiration > 0 {
		entry.HasExpiry = true
		entry.ExpiresAt = time.Now().Add(expiration)
	}
	cost := int64(len(key) + len(valueText) + 1)
	if cost < 1 {
		cost = 1
	}
	if expiration > 0 {
		l.cache.SetWithTTL(key, entry, cost, expiration)
	} else {
		l.cache.Set(key, entry, cost)
	}
	l.cache.Wait()
	return nil
}

func (l *localCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		l.cache.Del(key)
	}
	l.cache.Wait()
	return nil
}

func (l *localCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		if _, err := l.Get(ctx, key); err == nil {
			count++
		}
	}
	return count, nil
}

func (l *localCache) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	values := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		value, err := l.Get(ctx, key)
		if err != nil {
			values = append(values, nil)
			continue
		}
		values = append(values, value)
	}
	return values, nil
}

func (l *localCache) MSet(ctx context.Context, pairs ...interface{}) error {
	if len(pairs)%2 != 0 {
		return fmt.Errorf("mset requires an even number of arguments")
	}
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprint(pairs[i])
		if err := l.Set(ctx, key, pairs[i+1], 0); err != nil {
			return err
		}
	}
	return nil
}

func (l *localCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	value, err := l.Get(ctx, key)
	if err != nil {
		return err
	}
	return l.Set(ctx, key, value, expiration)
}

func (l *localCache) TTL(_ context.Context, key string) (time.Duration, error) {
	raw, ok := l.cache.Get(key)
	if !ok {
		return -2, nil
	}
	entry, ok := raw.(localEntry)
	if !ok || !entry.HasExpiry {
		return -1, nil
	}
	ttl := time.Until(entry.ExpiresAt)
	if ttl <= 0 {
		l.cache.Del(key)
		l.cache.Wait()
		return -2, nil
	}
	return ttl, nil
}

func (l *localCache) Incr(ctx context.Context, key string) (int64, error) {
	return l.IncrBy(ctx, key, 1)
}

func (l *localCache) Decr(ctx context.Context, key string) (int64, error) {
	return l.IncrBy(ctx, key, -1)
}

func (l *localCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	currentText, err := l.Get(ctx, key)
	var current int64
	if err == nil {
		current, err = strconv.ParseInt(currentText, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("local incr failed: %w", err)
		}
	}
	current += value
	if err := l.Set(ctx, key, current, 0); err != nil {
		return 0, err
	}
	return current, nil
}

func (l *localCache) Ping(context.Context) error {
	return nil
}

func (l *localCache) Close() error {
	l.cache.Close()
	return nil
}

func (l *localCache) Reload(context.Context, *Config) error {
	return nil
}

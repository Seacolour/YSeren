package server

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cache 是一个按实例持有、带 TTL 与 singleflight 的通用缓存。
type Cache[T any] struct {
	data  sync.Map
	group singleflight.Group
	ttl   time.Duration
}

type cacheEntry[T any] struct {
	value T
	at    time.Time
}

func NewCache[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{ttl: ttl}
}

func (c *Cache[T]) Get(key string, loader func() (T, error)) (T, error) {
	if value, ok := c.data.Load(key); ok {
		entry := value.(cacheEntry[T])
		if time.Since(entry.at) <= c.ttl {
			return entry.value, nil
		}
	}

	result, err, _ := c.group.Do(key, func() (any, error) {
		if value, ok := c.data.Load(key); ok {
			entry := value.(cacheEntry[T])
			if time.Since(entry.at) <= c.ttl {
				return entry.value, nil
			}
		}
		value, err := loader()
		if err != nil {
			return value, err
		}
		c.data.Store(key, cacheEntry[T]{value: value, at: time.Now()})
		return value, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return result.(T), nil
}

func (c *Cache[T]) Delete(key string) {
	c.data.Delete(key)
}

func (c *Cache[T]) Clear() {
	c.data.Range(func(key, _ any) bool {
		c.data.Delete(key)
		return true
	})
}

package main

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cache 是一个带 TTL 和 singleflight 的通用缓存
// 防止缓存击穿：相同 key 的并发请求只会触发一次 loader
type Cache[T any] struct {
	data  sync.Map
	group singleflight.Group
	ttl   time.Duration
}

type cacheEntry[T any] struct {
	value T
	at    time.Time
}

// NewCache 创建一个新的缓存实例
func NewCache[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{ttl: ttl}
}

// Get 获取缓存值，如果不存在或已过期则调用 loader 加载
// loader 会被 singleflight 保护，相同 key 的并发调用只执行一次
func (c *Cache[T]) Get(key string, loader func() (T, error)) (T, error) {
	// 如果缓存有效则直接返回
	if v, ok := c.data.Load(key); ok {
		entry := v.(cacheEntry[T])
		if time.Since(entry.at) <= c.ttl {
			return entry.value, nil
		}
	}

	// 使用 singleflight 防止并发加载
	result, err, _ := c.group.Do(key, func() (interface{}, error) {
		// 双重检查：可能在等待期间已被其他 goroutine 加载
		if v, ok := c.data.Load(key); ok {
			entry := v.(cacheEntry[T])
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

// Delete 删除指定 key 的缓存
func (c *Cache[T]) Delete(key string) {
	c.data.Delete(key)
}

// Clear 清空所有缓存
func (c *Cache[T]) Clear() {
	c.data.Range(func(key, _ interface{}) bool {
		c.data.Delete(key)
		return true
	})
}

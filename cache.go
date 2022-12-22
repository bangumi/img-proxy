package main

import (
	"time"

	"github.com/dgraph-io/ristretto"
)

func NewCache() *Cache[string, Item] {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,       // number of keys to track frequency of (10M).
		MaxCost:     cacheSize, // maximum cost of cache (1GB).
		BufferItems: 64,        // number of keys per Get buffer.
		Metrics:     true,
	})
	if err != nil {
		panic(err)
	}

	return &Cache[string, Item]{cache: cache}
}

type Item struct {
	content     []byte
	contentType string
}

type Cache[K, V any] struct {
	cache *ristretto.Cache
}

func (c Cache[K, V]) Get(key K) (V, bool) {
	value, ok := c.cache.Get(key)

	if !ok {
		// can't convert nil to Item{}
		var v V
		return v, ok
	}

	return value.(V), true
}

func (c Cache[K, V]) Set(key K, value V, cost int64) bool {
	return c.cache.Set(key, value, cost)
}

func (c Cache[K, V]) SetWithTTL(key K, value V, cost int64, ttl time.Duration) bool {
	return c.cache.SetWithTTL(key, value, cost, ttl)
}

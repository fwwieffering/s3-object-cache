package main

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Add(key string, value interface{})
	Get(key string) (interface{}, bool)
}

// MapCache implements an LRU cache with per-item expiration
type MapCache struct {
	cache         *lru.TwoQueueCache
	expiryMap     map[string]time.Time
	expirySeconds int
}

// NewMapCache returns a pointer to a MapCache
func NewMapCache(size int, expirySeconds int) *MapCache {
	c, _ := lru.New2Q(size)
	return &MapCache{
		cache:         c,
		expiryMap:     make(map[string]time.Time),
		expirySeconds: expirySeconds,
	}
}

// Add adds an item to the MapCache and sets the expiration time of that item in the expiryMap
func (c *MapCache) Add(key string, value interface{}) {
	expiryTime := time.Now().Add(time.Second * time.Duration(c.expirySeconds))
	c.expiryMap[key] = expiryTime
	c.cache.Add(key, value)
}

// GetMap returns an item if it exists in the cache and has not expired. If the item has expired
// it is removed from the cache and nil is returned
func (c *MapCache) Get(key string) (interface{}, bool) {
	expiryTime, timeOk := c.expiryMap[key]
	val, cacheOk := c.cache.Get(key)
	if cacheOk && timeOk && time.Now().Before(expiryTime) {
		return val, true
	} else {
		c.cache.Remove(key)
		delete(c.expiryMap, key)
		return nil, false
	}
}

package main

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Add(key string, value interface{})
	Get(key string) (interface{}, bool)
}

// ObjectCache implements an LRU cache with per-item expiration
type ObjectCache struct {
	cache         *lru.TwoQueueCache
	expiryObject  map[string]time.Time
	expirySeconds int
}

// NewObjectCache returns a pointer to a ObjectCache
func NewObjectCache(size int, expirySeconds int) *ObjectCache {
	c, _ := lru.New2Q(size)
	return &ObjectCache{
		cache:         c,
		expiryObject:  make(map[string]time.Time),
		expirySeconds: expirySeconds,
	}
}

// Add adds an item to the ObjectCache and sets the expiration time of that item in the expiryObject
func (c *ObjectCache) Add(key string, value interface{}) {
	expiryTime := time.Now().Add(time.Second * time.Duration(c.expirySeconds))
	c.expiryObject[key] = expiryTime
	c.cache.Add(key, value)
}

// GetObject returns an item if it exists in the cache and has not expired. If the item has expired
// it is removed from the cache and nil is returned
func (c *ObjectCache) Get(key string) (interface{}, bool) {
	expiryTime, timeOk := c.expiryObject[key]
	val, cacheOk := c.cache.Get(key)
	if cacheOk && timeOk && time.Now().Before(expiryTime) {
		return val, true
	} else {
		c.cache.Remove(key)
		delete(c.expiryObject, key)
		return nil, false
	}
}

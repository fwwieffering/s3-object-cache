package main

import (
	"testing"
	"time"
)

func TestObjectCacheUnexpired(t *testing.T) {
	c := NewObjectCache(1000, 300)
	c.Add("unit-test", "foobar")
	if !time.Now().Before(c.expiryObject["unit-test"]) {
		t.Fatalf("ObjectCache.expiryObject['unit-test'] is already expired")
	}
	_, ok := c.cache.Get("unit-test")
	if !ok {
		t.Fatalf("ObjectCache.cache does not contain entry 'unit-test'")
	}

	val, ok := c.Get("unit-test")
	if !ok {
		t.Fatalf("ObjectCache.Get('unit-test') should be in the cache")
	}
	if val.(string) != "foobar" {
		t.Fatalf("ObjectCache.Get('unit-test') should return foobar")
	}
}

func TestObjectCacheAddExpiry(t *testing.T) {
	c := NewObjectCache(1000, 5)
	c.Add("unit-test", "foobar")
	// sleep until expiration
	time.Sleep(time.Second * 5)
	if time.Now().Before(c.expiryObject["unit-test"]) {
		t.Fatalf("ObjectCache.expiryObject['unit-test'] should be expired")
	}
	_, ok := c.Get("unit-test")
	if ok {
		t.Fatalf("ObjectCache.Get should return ok=false for expired entries")
	}
	_, ok = c.cache.Get("unit-test")
	if ok {
		t.Fatalf("ObjectCache.Get should remove expired entries from ObjectCache.cache")
	}
}

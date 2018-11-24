package main

import (
	"testing"
	"time"
)

func TestMapCacheUnexpired(t *testing.T) {
	c := NewMapCache(1000, 300)
	c.Add("unit-test", "foobar")
	if !time.Now().Before(c.expiryMap["unit-test"]) {
		t.Fatalf("MapCache.expiryMap['unit-test'] is already expired")
	}
	_, ok := c.cache.Get("unit-test")
	if !ok {
		t.Fatalf("MapCache.cache does not contain entry 'unit-test'")
	}

	val, ok := c.Get("unit-test")
	if !ok {
		t.Fatalf("MapCache.Get('unit-test') should be in the cache")
	}
	if val.(string) != "foobar" {
		t.Fatalf("MapCache.Get('unit-test') should return foobar")
	}
}

func TestMapCacheAddExpiry(t *testing.T) {
	c := NewMapCache(1000, 5)
	c.Add("unit-test", "foobar")
	// sleep until expiration
	time.Sleep(time.Second * 5)
	if time.Now().Before(c.expiryMap["unit-test"]) {
		t.Fatalf("MapCache.expiryMap['unit-test'] should be expired")
	}
	_, ok := c.Get("unit-test")
	if ok {
		t.Fatalf("MapCache.Get should return ok=false for expired entries")
	}
	_, ok = c.cache.Get("unit-test")
	if ok {
		t.Fatalf("MapCache.Get should remove expired entries from MapCache.cache")
	}
}

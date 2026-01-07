package api

import (
	"sync"
	"time"
)

type Cache struct {
	mu  sync.Mutex
	at  time.Time
	val []byte
	ttl time.Duration
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{ttl: ttl}
}

func (c *Cache) Get() ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.val == nil || time.Since(c.at) > c.ttl {
		return nil, false
	}
	return c.val, true
}

func (c *Cache) Set(b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.val = b
	c.at = time.Now()
}

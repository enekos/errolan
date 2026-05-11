// Package cache is a tiny TTL+LRU cache for hot lookups (sites, threads).
package cache

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key       string
	value     any
	expiresAt time.Time
}

type TTL struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	ll       *list.List
	items    map[string]*list.Element
}

func New(capacity int, ttl time.Duration) *TTL {
	if capacity <= 0 {
		capacity = 128
	}
	return &TTL{
		capacity: capacity,
		ttl:      ttl,
		ll:       list.New(),
		items:    make(map[string]*list.Element, capacity),
	}
}

func (c *TTL) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	e := el.Value.(*entry)
	if time.Now().After(e.expiresAt) {
		c.ll.Remove(el)
		delete(c.items, key)
		return nil, false
	}
	c.ll.MoveToFront(el)
	return e.value, true
}

func (c *TTL) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		el.Value.(*entry).value = value
		el.Value.(*entry).expiresAt = time.Now().Add(c.ttl)
		c.ll.MoveToFront(el)
		return
	}
	e := &entry{key: key, value: value, expiresAt: time.Now().Add(c.ttl)}
	el := c.ll.PushFront(e)
	c.items[key] = el
	if c.ll.Len() > c.capacity {
		back := c.ll.Back()
		if back != nil {
			c.ll.Remove(back)
			delete(c.items, back.Value.(*entry).key)
		}
	}
}

func (c *TTL) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.Remove(el)
		delete(c.items, key)
	}
}

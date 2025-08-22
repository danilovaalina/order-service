package cache

import (
	"container/list"
	"sync"
	"time"
)

type Item struct {
	key       string
	value     interface{}
	timestamp time.Time
}

type LRUCache struct {
	capacity uint64
	items    map[string]*list.Element
	queue    *list.List
	mu       sync.RWMutex
	ttl      time.Duration
}

func New(capacity uint64, ttl time.Duration) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		queue:    list.New(),
		ttl:      ttl,
	}
}

func (c *LRUCache) Set(key string, value interface{}) {
	if element, exists := c.items[key]; exists {
		c.queue.MoveToFront(element)
		element.Value.(*Item).value = value
		return
	}

	if c.queue.Len() == int(c.capacity) {
		element := c.queue.Back()
		if element != nil {
			item := c.queue.Remove(element).(*Item)
			delete(c.items, item.key)
		}
	}

	item := &Item{
		key:       key,
		value:     value,
		timestamp: time.Now(),
	}

	element := c.queue.PushFront(item)
	c.items[item.key] = element

	return
}

func (c *LRUCache) Get(key string) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.items[key]
	if !exists {
		return nil
	}

	item := element.Value.(*Item)

	if time.Since(item.timestamp) > c.ttl {
		c.queue.Remove(element)
		delete(c.items, item.key)
		return nil
	}

	c.queue.MoveToFront(element)
	return item.value
}

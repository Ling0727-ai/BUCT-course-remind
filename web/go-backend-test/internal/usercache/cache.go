package usercache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

var invalidators struct {
	sync.RWMutex
	callbacks []func(string)
}

// Invalidate is called by model writes; caches remain independent of model packages.
func Invalidate(userID string) {
	invalidators.RLock()
	defer invalidators.RUnlock()
	for _, invalidate := range invalidators.callbacks {
		invalidate(userID)
	}
}

type entry[T any] struct {
	key     string
	value   T
	size    int
	expires time.Time
}
type flight[T any] struct {
	done        chan struct{}
	value       T
	err         error
	invalidated bool
}

type Cache[T any] struct {
	mu                          sync.Mutex
	items                       map[string]*list.Element
	lru                         *list.List
	flights                     map[string]*flight[T]
	maxEntries, maxBytes, bytes int
	slots                       chan struct{}
}

func New[T any](maxEntries, maxBytes int) *Cache[T] {
	c := &Cache[T]{items: make(map[string]*list.Element), lru: list.New(), flights: make(map[string]*flight[T]),
		maxEntries: maxEntries, maxBytes: maxBytes, slots: make(chan struct{}, 8)}
	invalidators.Lock()
	invalidators.callbacks = append(invalidators.callbacks, c.Invalidate)
	invalidators.Unlock()
	return c
}

func (c *Cache[T]) remove(element *list.Element) {
	item := element.Value.(*entry[T])
	c.bytes -= item.size
	delete(c.items, item.key)
	c.lru.Remove(element)
}

func (c *Cache[T]) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.items[key]; e != nil {
		c.remove(e)
	}
	if f := c.flights[key]; f != nil {
		f.invalidated = true
	}
}

// Values are immutable after publication. In-flight invalidations prevent stale refill.
func (c *Cache[T]) Get(ctx context.Context, key string, load func() (T, int, time.Duration, error)) (T, error) {
	var zero T
	for {
		c.mu.Lock()
		if e := c.items[key]; e != nil {
			item := e.Value.(*entry[T])
			if time.Now().Before(item.expires) {
				c.lru.MoveToFront(e)
				value := item.value
				c.mu.Unlock()
				return value, nil
			}
			c.remove(e)
		}
		if f := c.flights[key]; f != nil {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-f.done:
				return f.value, f.err
			}
		}
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case c.slots <- struct{}{}:
		}
		c.mu.Lock()
		if c.items[key] != nil || c.flights[key] != nil {
			c.mu.Unlock()
			<-c.slots
			continue
		}
		f := &flight[T]{done: make(chan struct{})}
		c.flights[key] = f
		c.mu.Unlock()
		value, size, ttl, err := load()
		c.mu.Lock()
		f.value, f.err = value, err
		if err == nil && !f.invalidated && size <= c.maxBytes && ttl > 0 {
			for c.lru.Len() > 0 && (c.lru.Len() >= c.maxEntries || c.bytes+size > c.maxBytes) {
				c.remove(c.lru.Back())
			}
			c.items[key] = c.lru.PushFront(&entry[T]{key: key, value: value, size: size, expires: time.Now().Add(ttl)})
			c.bytes += size
		}
		delete(c.flights, key)
		close(f.done)
		c.mu.Unlock()
		<-c.slots
		return value, err
	}
}

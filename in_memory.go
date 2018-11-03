package cache

import (
	"sync"
	"time"
)

// InMem is a struct which implements Storage interface using
// memory as storage
type InMem struct {
	data   map[string]interface{}
	expire map[string]time.Time
	done   chan struct{}
	sync.RWMutex
}

// InMemory creates a new in memory storage which can be passed to
// cache.New(WithStorage(...))
func InMemory() *InMem {
	inMemory := &InMem{
		data:   make(map[string]interface{}),
		expire: make(map[string]time.Time),
	}
	return inMemory
}

// Write writes the given content for the given key in
// memory storage
func (i *InMem) Write(key string, v interface{}, d time.Duration) error {
	i.Lock()
	defer i.Unlock()

	i.data[key] = v
	if d != 0 {
		i.expire[key] = time.Now().Add(d)
	}
	return nil
}

// Read reads coontent for the given key from in memory storage
func (i *InMem) Read(key string) (interface{}, error) {
	i.RLock()
	defer i.RUnlock()

	v, ok := i.data[key]
	if !ok {
		return nil, ErrKeyNotExist
	}
	expire, ok := i.expire[key]
	if ok && expire.Before(time.Now()) {
		i.del(key)
		return nil, ErrKeyNotExist
	}
	return v, nil
}

// Delete deletes content of the given key from in memory storage
func (i *InMem) Delete(key string) error {
	i.Lock()
	defer i.Unlock()

	err := i.del(key)
	return err
}
func (i *InMem) del(key string) error {
	delete(i.data, key)
	delete(i.expire, key)
	return nil
}

// Flush flushes in momory storage
func (i *InMem) Flush() error {
	i.data = make(map[string]interface{})
	i.expire = make(map[string]time.Time)

	return nil
}

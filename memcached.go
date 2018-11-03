package cache

import (
	"encoding/json"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// Memcache is a structure implemening Storage interface using
// memcached as a storage provider
type Memcache struct {
	client *memcache.Client
}

// Memcached creates a new memcached storage which can be passed to
// cache.New(WithStorage(...))
func Memcached(servers ...string) Memcache {
	// if servers list is empty use localhost with the default port
	if len(servers) == 0 {
		servers = append(servers, "localhost:11211")
	}

	return Memcache{
		client: memcache.New(servers...),
	}
}

// Write writes the given content for the given key in
// memcached storage
func (m Memcache) Write(key string, v interface{}, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	item := &memcache.Item{Key: key, Value: b}
	if ttl > 0 {
		item.Expiration = int32(time.Now().Add(ttl).Unix())
	}
	return m.client.Set(item)
}

// Read reads coontent for the given key from in memcached storage
func (m Memcache) Read(key string) (interface{}, error) {
	item, err := m.client.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return nil, ErrKeyNotExist
		}
		return nil, err
	}
	return item.Value, nil
}

// Delete deletes content of the given key from in memcached storage
func (m Memcache) Delete(key string) error {
	return m.client.Delete(key)
}

// Flush flushes memcached storage
func (m Memcache) Flush() error {
	return m.client.DeleteAll()
}

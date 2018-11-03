package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

const (
	// PriorityMedium is a medium level storage priority
	PriorityMedium Priority = iota

	// PriorityHigh is a high level storage priority
	PriorityHigh
)

var (
	// ErrKeyNotExist indicates that the key does not exist in the storage
	ErrKeyNotExist = errors.New("key does not exist")

	// ErrNotJSONMarshalable indicates that the content is not json marshalable
	ErrNotJSONMarshalable = errors.New("value is not json marshalable")
)

// Cache manages to Set, Get, Delet and Tag keys
type Cache struct {
	storage map[Priority][]Storage
	logger  *logrus.Logger
	tagger  Tagger
	ns      string
}

// New constructs a new Cache instance which can store, read
// and remove items with tags
func New(options ...Option) *Cache {
	c := &Cache{
		logger:  logrus.New(),
		storage: make(map[Priority][]Storage),
	}
	options = append(
		[]Option{
			WithTagger(newStdTagger(c.logger, "go:cache:tagger")),
			WithNamespace("go:cache"),
		},
		options...,
	)
	for i := range options {
		options[i](c)
	}
	return c
}

// NsKey wraps the given key with the namespace prefix
func (c *Cache) NsKey(key string) string {
	return fmt.Sprintf("%s:%s", c.ns, key)
}

// Loop iterates through registered high and medium storage and pass them to the
// coressponding function to use
func (c *Cache) Loop(high func(s Storage) (bool, error), medium func(s Storage) (bool, error)) (e error) {
	if medium == nil {
		medium = high
	}

	highPriority, _ := c.storage[PriorityHigh]
	for _, storage := range highPriority {
		if terminate, err := high(storage); err != nil {
			e = multierror.Append(e, err)
		} else if terminate {
			return nil
		}
	}

	mediumPriority, _ := c.storage[PriorityMedium]
	for _, storage := range mediumPriority {
		terminate, _ := medium(storage)
		if terminate {
			return nil
		}
	}
	return
}

// Set stores a value into configured stores expiration and tags list
// Zero expiration means the key has no expiration time. Additionally,
// all string argument passed after expiration will be used to tag the value
// It ignores any error occurred for medium level storage
func (c *Cache) Set(key string, v interface{}, expiration time.Duration, tags ...string) error {
	return c.set(key, v, expiration, tags...)
}
func (c *Cache) set(key string, v interface{}, expiration time.Duration, tags ...string) (err error) {
	item := item{Key: key, Val: v, Created: time.Now(), Expires: expiration}
	return c.Loop(
		func(s Storage) (bool, error) {
			return false, c.write(s, key, item, expiration, tags...)
		},
		nil,
	)
}

func (c *Cache) write(s Storage, key string, v interface{}, expiration time.Duration, tags ...string) error {
	if err := s.Write(c.NsKey(key), v, expiration); err != nil {
		return err
	}
	if len(tags) > 0 {
		if err := c.tagger.Tag(s, key, tags...); err != nil {
			return err
		}
	}
	return nil
}

// Get reads for the given key from the registered storage unless
// a valid content is received. It will ignore any error occurred
// for medium level storage
func (c *Cache) Get(key string, out interface{}) error { return c.get(key, out) }
func (c *Cache) get(key string, out interface{}) error {
	var (
		p  []Storage
		it *item
		w  Storage
	)
	err := c.Loop(
		func(s Storage) (bool, error) {
			item, err := c.read(s, key)
			if err != nil {
				if ErrKeyNotExist == err {
					p = append(p, s)
				}
				return false, err
			}
			it = item
			w = s
			if err := mapstructure.Decode(item.Val, out); err != nil {
				return false, err
			}
			return true, nil
		},
		func(s Storage) (bool, error) {
			item, err := c.read(s, key)
			if err != nil {
				return false, err
			}
			it = item
			w = s
			if err := mapstructure.Decode(item.Val, out); err != nil {
				return false, err
			}
			return true, nil
		},
	)

	if it != nil {
		for _, s := range p {
			c.propagate(s, w, it, key)
		}
	}
	return err
}
func (c *Cache) read(s Storage, key string) (*item, error) {
	v, err := s.Read(c.NsKey(key))
	if err != nil {
		return nil, err
	}

	var cacheItem = &item{}
	switch x := v.(type) {
	case item:
		cacheItem = &x
	case string:
		json.Unmarshal([]byte(x), cacheItem)
	case []byte:
		json.Unmarshal(x, cacheItem)
	}
	if cacheItem.expired() {
		return nil, c.del(c.NsKey(key))
	}
	return cacheItem, nil
}

// Del deletes the given key from all registered storage
func (c *Cache) Del(keys ...string) error { return c.del(keys...) }
func (c *Cache) del(keys ...string) error {
	return c.Loop(
		func(s Storage) (bool, error) {
			for _, key := range keys {
				if err := s.Delete(c.NsKey(key)); err != nil {
					return false, err
				}

				if err := c.tagger.UnTag(s, key); err != nil {
					return false, err
				}
			}
			return false, nil
		},
		nil,
	)
}

// Extend sets the new expiration time for the given key
// If the expiration has not initially been set this method
// will add one
func (c *Cache) Extend(key string, expiration time.Duration) error {
	return c.Loop(
		func(s Storage) (bool, error) {
			v, err := s.Read(c.NsKey(key))
			if err != nil {
				return false, err
			}

			it := v.(item)
			it.Created = time.Now()
			it.Expires = expiration

			return false, s.Write(key, it, expiration)
		},
		nil,
	)
}

// Propagate propagates all the given keys from s1 data storage
// into s2 data storage
func (c *Cache) propagate(s1, s2 Storage, it *item, keys ...string) error {
	for _, key := range keys {
		tags, err := c.tagger.Tags(s2, key)
		if err != nil {
			return err
		}
		New(WithStorage(s1), WithNamespace(c.ns)).Set(key, it.Val, it.Expires-(time.Now().Sub(it.Created)), tags...)
	}
	return nil
}

// Flush flushes all the data in all registered storage
func (c *Cache) Flush() (err error) {
	return c.Loop(
		func(s Storage) (bool, error) {
			return false, s.Flush()
		},
		nil,
	)
}

// Close closes the storage resource if it implements io.Closer
func (c *Cache) Close() (err error) {
	return c.Loop(
		func(s Storage) (bool, error) {
			if closer, ok := s.(io.Closer); ok {
				return false, closer.Close()
			}
			return false, nil
		},
		nil,
	)
}

// ByTag reads tagged values into `out`
// `out` is always a slice of values
func (c *Cache) ByTag(tag string, out interface{}) error {
	if out == nil {
		out = make([]interface{}, 0, 0)
	}

	return c.Loop(func(s Storage) (bool, error) {
		keys, err := c.tagger.Keys(s, tag)

		if err != nil {
			return false, err
		}

		output := make([]interface{}, 0, len(keys))

		for _, key := range keys {
			v, errRead := s.Read(c.NsKey(key))
			if errRead != nil {
				err = multierror.Append(err, errRead)
			}
			it := c.item(v)
			if it.Val != nil {
				output = append(output, it.Val)
			}
		}
		return true, mapstructure.Decode(output, out)
	}, nil)
}

// DelByTag deletes tagged values
func (c *Cache) DelByTag(tags ...string) error {
	return c.Loop(
		func(s Storage) (bool, error) {
			for _, tag := range tags {
				keys, err := c.tagger.Keys(s, tag)
				if err != nil {
					return false, err
				}

				var slice = make([]string, len(keys))
				copy(slice, keys)

				if err := c.Del(slice...); err != nil {
					return false, err
				}
			}
			return false, nil
		},
		nil,
	)
}

func (c *Cache) item(v interface{}) item {
	var i item
	switch x := v.(type) {
	case item:
		return x
	case []byte:
		json.Unmarshal(x, &i)
	case string:
		json.Unmarshal([]byte(x), &i)
	default:
		mapstructure.Decode(v, &i)
	}
	return i
}

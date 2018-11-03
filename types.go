package cache

import (
	"time"
)

// Priority refers to the storage priority
// There are `high` and `medium` priorities for storage
// Any error occurred for high level storage will be reported
// Any error occurred for medium level storage will be ignored
type Priority int

type item struct {
	Key     string        `json:"key"`
	Val     interface{}   `json:"val"`
	Created time.Time     `json:"created"`
	Expires time.Duration `json:"expires"`
}

func (i item) expired() bool {
	if i.Expires == 0 {
		return false
	}
	return i.Created.Add(i.Expires).Before(time.Now())
}

// Storage is an interface to write, read, delete and empty
// a data storage. Any struct implementing Storage interface
// can be passed to cache.New(WithStorage(...)) to use as taggable
// cache data adapter
type Storage interface {
	// Write writes to the storage
	Write(key string, v interface{}, ttl time.Duration) error

	// Read reads from the storage for the key
	Read(key string) (interface{}, error)

	// Delete deletes from the storage
	Delete(key string) error

	// Flush flushes cache storage
	Flush() error
}

// Tagger ins an interface to tag and untag data with the given tags
// Any struct implemening tagger interface can be passed to
// cache.New(WithTagger(...)) to use a data tagger
type Tagger interface {
	// attach tags to the given key
	Tag(s Storage, key string, tags ...string) error

	// unattach tags from the given key
	UnTag(s Storage, key string, tags ...string) error

	// receive all key's tags
	Tags(s Storage, key string) ([]string, error)

	// receive all tag's keys
	Keys(s Storage, tag string) ([]string, error)
}

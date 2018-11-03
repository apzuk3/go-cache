package cache

import (
	"github.com/sirupsen/logrus"
)

// Option is the type of constructor options for New(...).
type Option func(c *Cache)

// WithStorage configures a cache client with a cache.Storage to store
// data
func WithStorage(storage ...Storage) Option {
	return func(c *Cache) {
		if len(storage) == 0 {
			return
		}
		c.storage[PriorityHigh] = storage
	}
}

// WithHighPriorityStorage configures a cache instance with a priority
// cache.Storage to store data
func WithHighPriorityStorage(storage ...Storage) Option {
	return withPriorityStorage(PriorityHigh, storage...)
}

// WithMediumPriorityStorage configures a cache instance with a medium
// cache.Storage to store data
func WithMediumPriorityStorage(storage ...Storage) Option {
	return withPriorityStorage(PriorityMedium, storage...)
}

func withPriorityStorage(priority Priority, storage ...Storage) Option {
	return func(c *Cache) {
		c.storage[priority] = append(c.storage[priority], storage...)
	}
}

// WithDebug configures a cache instance with debug flag on
func WithDebug() Option {
	return func(c *Cache) {
		c.logger.SetLevel(logrus.DebugLevel)
	}
}

// WithTagger configures a cache instance with custom cache.Tagger
func WithTagger(tagger Tagger) Option {
	return func(c *Cache) {
		c.tagger = tagger
	}
}

// WithNamespace configures a cache instance with namespace.
func WithNamespace(ns string) Option {
	return func(c *Cache) {
		c.ns = ns
	}
}

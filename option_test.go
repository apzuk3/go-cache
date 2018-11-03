package cache

import (
	"testing"

	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func Test_WithHighPriorityStorage(t *testing.T) {
	c := New(WithHighPriorityStorage(InMemory(), InMemory()))
	assert.Equal(t, 2, len(c.storage[PriorityHigh]))
	assert.Equal(t, 0, len(c.storage[PriorityMedium]))
}

func Test_WithMediumPriorityStorage(t *testing.T) {
	c := New(WithMediumPriorityStorage(InMemory(), InMemory()))
	assert.Equal(t, 2, len(c.storage[PriorityMedium]))
	assert.Equal(t, 0, len(c.storage[PriorityHigh]))
}

func Test_WithNamespace(t *testing.T) {
	c := New(WithNamespace("ns"))
	assert.Equal(t, "ns", c.ns)
}

func Test_WithDebug(t *testing.T) {
	c := New(WithDebug())
	assert.Equal(t, c.logger.Level, logrus.DebugLevel)
}

func Test_WithStorageEmptySet(t *testing.T) {
	c := New(WithStorage())
	assert.Equal(t, 0, len(c.storage))
}

func Test_WithStorage(t *testing.T) {
	c := New(
		WithStorage(InMemory(), InMemory(), InMemory()),
	)
	assert.Equal(t, 1, len(c.storage))

	assert.Equal(t, 3, len(c.storage[PriorityHigh]))
}

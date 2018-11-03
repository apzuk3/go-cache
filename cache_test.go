package cache

import (
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gotest.tools/assert"
)

type mock struct {
	err    error
	closed bool
	val    interface{}
}

// write t the storage
func (m *mock) Write(key string, v interface{}, ttl time.Duration) error {
	return m.err
}

// read from the storage for the key
func (m *mock) Read(key string) (interface{}, error) {
	return m.val, m.err
}

// delete from the storage
func (m *mock) Delete(key string) error {
	return m.err
}

// Empty cache storage
func (m *mock) Flush() error {
	return m.err
}

func (m *mock) Close() error {
	m.closed = true
	return m.err
}

type some struct {
	Field1 int
	Field2 string
	Field3 []interface{}
	Field4 map[string]interface{}
	Field5 *some
}

func TestCache_NsKey(t *testing.T) {
	c := New(
		WithStorage(InMemory()),
		WithNamespace("custom:ns"),
	)
	assert.Equal(t, "custom:ns:key1", c.NsKey("key1"))
}

func TestCache_LoopNoErrorNoTermination(t *testing.T) {
	c := New(
		WithHighPriorityStorage(InMemory(), InMemory()),
		WithMediumPriorityStorage(InMemory(), InMemory(), InMemory(), InMemory()),
	)
	var (
		high   int
		medium int
	)
	c.Loop(func(s Storage) (bool, error) {
		high++
		return false, nil
	}, func(s Storage) (bool, error) {
		medium++
		return false, nil
	})
	assert.Equal(t, 2, high)
	assert.Equal(t, 4, medium)
}

func TestCache_LoopNoErrorNoTerminationNilMediumHandler(t *testing.T) {
	c := New(
		WithHighPriorityStorage(InMemory(), InMemory()),
		WithMediumPriorityStorage(InMemory(), InMemory(), InMemory(), InMemory()),
	)
	var (
		called int
	)
	c.Loop(func(s Storage) (bool, error) {
		called++
		return false, nil
	}, nil)
	assert.Equal(t, 6, called)
}

func TestCache_LoopNoErrorHighPriorityStorageTermination(t *testing.T) {
	c := New(
		WithHighPriorityStorage(InMemory(), InMemory(), InMemory(), InMemory(), InMemory()),
		WithHighPriorityStorage(InMemory(), InMemory()),
	)

	var (
		high   int
		medium int
	)
	c.Loop(func(s Storage) (bool, error) {
		high++
		if high == 3 {
			return true, nil
		}
		return false, nil
	}, func(s Storage) (bool, error) {
		medium++
		return false, nil
	})
	assert.Equal(t, 3, high)
	assert.Equal(t, 0, medium)
}

func TestCache_LoopNoErrorMediumPriorityStorageTermination(t *testing.T) {
	c := New(
		WithHighPriorityStorage(InMemory(), InMemory(), InMemory(), InMemory(), InMemory()),
		WithMediumPriorityStorage(InMemory(), InMemory(), InMemory(), InMemory()),
	)

	var (
		high   int
		medium int
	)
	c.Loop(func(s Storage) (bool, error) {
		high++
		return false, nil
	}, func(s Storage) (bool, error) {
		medium++
		if medium == 3 {
			return true, nil
		}
		return false, nil
	})
	assert.Equal(t, 5, high)
	assert.Equal(t, 3, medium)
}

func TestCache_LoopHighPriorityStorageErrorNoTermination(t *testing.T) {
	c := New(
		WithHighPriorityStorage(InMemory(), InMemory(), InMemory(), InMemory(), InMemory()),
		WithMediumPriorityStorage(InMemory(), InMemory()),
	)

	var (
		high   int
		medium int
	)
	err := c.Loop(func(s Storage) (bool, error) {
		high++
		return false, errors.Errorf("high_%d", high)
	}, func(s Storage) (bool, error) {
		medium++
		return false, nil
	})
	assert.Equal(t, 5, high)
	assert.Equal(t, 2, medium)
	assert.Error(t, err, "5 errors occurred:\n\t* high_1\n\t* high_2\n\t* high_3\n\t* high_4\n\t* high_5\n\n")
}

func TestCache_Set(t *testing.T) {
	var (
		inMem = []Storage{InMemory(), InMemory(), InMemory(), InMemory(), InMemory(), InMemory()}
	)
	c := New(
		WithHighPriorityStorage(inMem[:2]...),
		WithMediumPriorityStorage(inMem[2:]...),
		WithNamespace("go:test"),
	)
	assert.NilError(t, c.Set("key1", 1234, 0))
	assert.NilError(t, c.Set("key2", "abc", 0))
	assert.NilError(t, c.Set("key3", true, time.Second))

	var assertItem = func(t *testing.T, s Storage, key string, expected *item) {
		val, ok := s.(*InMem).data[key]
		assert.Assert(t, ok)

		item := &item{}
		mapstructure.Decode(val, &item)
		assert.Equal(t, expected.Val, item.Val)
		assert.Equal(t, expected.Expires, item.Expires)
	}

	for i := range inMem {
		assertItem(t, inMem[i], "go:test:key1", &item{Val: 1234, Expires: 0})
		assertItem(t, inMem[i], "go:test:key2", &item{Val: "abc", Expires: 0})
		assertItem(t, inMem[i], "go:test:key3", &item{Val: true, Expires: time.Second})
	}
}

func TestCache_SetPriorityError(t *testing.T) {
	var (
		inMem = []Storage{
			InMemory(),
			&mock{err: errors.New("some error")},
			InMemory(),
			&mock{err: errors.New("some error")},
			InMemory(),
			InMemory(),
		}
	)

	c := New(WithHighPriorityStorage(inMem...), WithNamespace("go:test"))
	err := c.Set("key1", "val1", 0)
	assert.ErrorType(t, err, &multierror.Error{})
	assert.Equal(t, len(err.(*multierror.Error).Errors), 2)

	var assertItem = func(t *testing.T, s Storage, key string, expected *item) {
		val, ok := s.(*InMem).data[key]
		assert.Assert(t, ok)

		item := &item{}
		mapstructure.Decode(val, &item)
		assert.Equal(t, expected.Val, item.Val)
		assert.Equal(t, expected.Expires, item.Expires)
	}

	for i := range inMem {
		if i == 1 || i == 3 {

		} else {
			assertItem(t, inMem[i], "go:test:key1", &item{Val: "val1", Expires: 0})
		}
	}
}

func TestCache_SetMediumStorageError(t *testing.T) {
	var (
		inMem = []Storage{
			InMemory(),
			InMemory(),
			InMemory(),
			InMemory(),
		}
	)

	c := New(
		WithHighPriorityStorage(inMem...),
		WithMediumPriorityStorage(
			&mock{err: errors.New("some error")},
			&mock{err: errors.New("some error")},
		),
		WithNamespace("go:test"),
	)
	assert.NilError(t, c.Set("key1", "val1", 0))

	var assertItem = func(t *testing.T, s Storage, key string, expected *item) {
		val, ok := s.(*InMem).data[key]
		assert.Assert(t, ok)

		item := &item{}
		mapstructure.Decode(val, &item)
		assert.Equal(t, expected.Val, item.Val)
		assert.Equal(t, expected.Expires, item.Expires)
	}

	for i := range inMem {
		assertItem(t, inMem[i], "go:test:key1", &item{Val: "val1", Expires: 0})
	}
}

func TestCache_SetWithTags(t *testing.T) {
	var (
		inMem = []Storage{InMemory(), InMemory(), InMemory(), InMemory(), InMemory(), InMemory()}
	)
	c := New(
		WithHighPriorityStorage(inMem[:2]...),
		WithMediumPriorityStorage(inMem[2:]...),
		WithNamespace("go:test"),
	)
	assert.NilError(t, c.Set("key1", 1234, 0, "tag1", "tag2", "tag3", "tag4"))
	assert.NilError(t, c.Set("key2", "abc", 0, "tag1", "tag3", "tag5", "tag6"))
	assert.NilError(t, c.Set("key3", true, 0, "tag2", "tag3", "tag7", "tag8"))

	var assertTags = func(t *testing.T, s Storage, key string, expected []string) {
		val, ok := s.(*InMem).data[key]
		assert.Assert(t, ok)
		assert.DeepEqual(t, cast.ToStringSlice(val), expected)
	}

	for i := range inMem {
		assertTags(t, inMem[i], "go:cache:tagger:key:key1:tags", []string{"tag1", "tag2", "tag3", "tag4"})
		assertTags(t, inMem[i], "go:cache:tagger:key:key2:tags", []string{"tag1", "tag3", "tag5", "tag6"})
		assertTags(t, inMem[i], "go:cache:tagger:key:key3:tags", []string{"tag2", "tag3", "tag7", "tag8"})

		assertTags(t, inMem[i], "go:cache:tagger:tag:tag1:keys", []string{"key1", "key2"})
		assertTags(t, inMem[i], "go:cache:tagger:tag:tag2:keys", []string{"key1", "key3"})
		assertTags(t, inMem[i], "go:cache:tagger:tag:tag3:keys", []string{"key1", "key2", "key3"})
		assertTags(t, inMem[i], "go:cache:tagger:tag:tag4:keys", []string{"key1"})
		assertTags(t, inMem[i], "go:cache:tagger:tag:tag5:keys", []string{"key2"})
		assertTags(t, inMem[i], "go:cache:tagger:tag:tag6:keys", []string{"key2"})
		assertTags(t, inMem[i], "go:cache:tagger:tag:tag7:keys", []string{"key3"})
		assertTags(t, inMem[i], "go:cache:tagger:tag:tag8:keys", []string{"key3"})
	}
}

func TestCache_Get(t *testing.T) {
	c := New(WithStorage(InMemory()))
	c.Set("key1", 1234, 0)
	c.Set("key2", "abcd", 0)
	c.Set("key3", []string{"a", "b", "c"}, 0)
	c.Set("key4", map[interface{}]interface{}{"1234": 12345, "abc": "home"}, 0)
	c.Set("key5", some{
		Field1: 123,
		Field2: "abc",
		Field3: []interface{}{"abc", 123, true},
		Field4: map[string]interface{}{
			"blabla": 1234,
		},
		Field5: &some{
			Field1: 456,
			Field2: "edf",
		},
	}, 0)

	var v1 int
	assert.NilError(t, c.Get("key1", &v1))
	assert.Equal(t, 1234, v1)

	var v2 string
	assert.NilError(t, c.Get("key2", &v2))
	assert.Equal(t, "abcd", v2)

	var v3 []string
	assert.NilError(t, c.Get("key3", &v3))
	assert.DeepEqual(t, []string{"a", "b", "c"}, v3)

	var v4 map[interface{}]interface{}
	assert.NilError(t, c.Get("key4", &v4))
	assert.DeepEqual(t, map[interface{}]interface{}{"1234": 12345, "abc": "home"}, v4)
}

func TestCache_Del(t *testing.T) {
	var (
		inMem = []Storage{InMemory(), InMemory(), InMemory(), InMemory(), InMemory(), InMemory()}
	)

	c := New(
		WithHighPriorityStorage(inMem[:2]...),
		WithMediumPriorityStorage(inMem[2:]...),
		WithNamespace("go:test"),
	)

	c.Set("key1", "val1", 0, "tag1", "tag2", "tag3")
	c.Set("key2", "val2", 0, "tag4", "tag5", "tag3")
	c.Set("key3", "val3", 0, "tag2", "tag4", "tag9")
	c.Set("key4", "val4", 0, "tag1", "tag3", "tag2")

	assert.NilError(t, c.Del("key1", "key3"))

	for i := range inMem {
		{
			_, ok := inMem[i].(*InMem).data["go:test:key3"]
			assert.Assert(t, !ok)
		}

		{
			_, ok := inMem[i].(*InMem).data["go:test:key:key3:tags"]
			assert.Assert(t, !ok)
		}

		{
			data, _ := inMem[i].(*InMem).data["go:cache:tagger:tag:tag2:keys"]
			assert.DeepEqual(t, data, []string{"key4"})

			data1, _ := inMem[i].(*InMem).data["go:cache:tagger:tag:tag4:keys"]
			assert.DeepEqual(t, data1, []string{"key2"})

			_, ok := inMem[i].(*InMem).data["go:cache:tagger:tag:tag9:keys"]
			assert.Assert(t, !ok)
		}
	}

	err := c.Get("key3", "")

	assert.ErrorType(t, err, &multierror.Error{})
	errors := err.(*multierror.Error).Errors
	assert.Equal(t, len(errors), 2)
	assert.Error(t, errors[0], "key does not exist")
	assert.Error(t, errors[1], "key does not exist")
}

func TestCache_Flush(t *testing.T) {
	var (
		inMem = []Storage{InMemory(), InMemory(), InMemory(), InMemory(), InMemory(), InMemory()}
	)

	c := New(
		WithHighPriorityStorage(inMem[:2]...),
		WithMediumPriorityStorage(inMem[2:]...),
		WithNamespace("go:test"),
	)

	assert.NilError(t, c.Flush())
	for _, s := range inMem {
		assert.DeepEqual(t, s.(*InMem).data, map[string]interface{}{})
		assert.DeepEqual(t, s.(*InMem).expire, map[string]time.Time{})
	}
}

func TestCache_Extend(t *testing.T) {
	var (
		inMem = []Storage{InMemory(), InMemory(), InMemory(), InMemory(), InMemory(), InMemory()}
	)

	c := New(
		WithHighPriorityStorage(inMem[:2]...),
		WithMediumPriorityStorage(inMem[2:]...),
		WithNamespace("go:test"),
	)

	c.Set("key1", 1234, time.Minute)
	c.Set("key2", "acbd", time.Minute)
	c.Set("key3", true, time.Minute)

	assert.NilError(t, c.Extend("key2", 2*time.Second))
	for _, s := range inMem {
		it, _ := s.Read("key2")
		assert.DeepEqual(t, it.(item).Expires, 2*time.Second)
	}
}

func TestCache_Close(t *testing.T) {
	var (
		inMem = []Storage{&mock{}, InMemory(), InMemory(), InMemory(), &mock{}}
	)

	c := New(
		WithHighPriorityStorage(inMem[:2]...),
		WithMediumPriorityStorage(inMem[2:]...),
		WithNamespace("go:test"),
	)

	assert.NilError(t, c.Close())
	assert.Assert(t, inMem[4].(*mock).closed)
	assert.Assert(t, inMem[0].(*mock).closed)
}

func TestCache_ByTagIntegers(t *testing.T) {
	c := New(WithStorage(InMemory()))
	c.Set("key1", 1, 0, "tag1", "tag2", "tag3")
	c.Set("key2", 2, 0, "tag3", "tag1", "tag4")
	c.Set("key3", 3, 0, "tag6", "tag6", "tag1")
	c.Set("key4", 4, 0, "tag5", "tag1", "tag6")

	var intarr []int
	assert.NilError(t, c.ByTag("tag1", &intarr))
	assert.DeepEqual(t, intarr, []int{1, 2, 3, 4})
}

func TestCache_ByTagFloats(t *testing.T) {
	c := New(WithStorage(InMemory()))
	c.Set("key1", 1.1, 0, "tag1", "tag2", "tag3")
	c.Set("key2", 2.2, 0, "tag3", "tag1", "tag4")
	c.Set("key3", 3.3, 0, "tag6", "tag6", "tag1")
	c.Set("key4", 4.4, 0, "tag5", "tag1", "tag6")

	var floatarr []float32
	assert.NilError(t, c.ByTag("tag1", &floatarr))
	assert.DeepEqual(t, floatarr, []float32{1.1, 2.2, 3.3, 4.4})
}

func TestCache_ByTagStrings(t *testing.T) {
	c := New(WithStorage(InMemory()))
	c.Set("key1", "abc", 0, "tag1", "tag2", "tag3")
	c.Set("key2", "def", 0, "tag3", "tag1", "tag4")
	c.Set("key3", "ghi", 0, "tag6", "tag6", "tag1")
	c.Set("key4", "jkl", 0, "tag5", "tag1", "tag6")

	var stringarr []string
	assert.NilError(t, c.ByTag("tag1", &stringarr))
	assert.DeepEqual(t, stringarr, []string{"abc", "def", "ghi", "jkl"})
}

func TestCache_ByTagBools(t *testing.T) {
	c := New(WithStorage(InMemory()))
	c.Set("key1", true, 0, "tag1", "tag2", "tag3")
	c.Set("key2", false, 0, "tag3", "tag1", "tag4")
	c.Set("key3", true, 0, "tag6", "tag6", "tag1")
	c.Set("key4", false, 0, "tag5", "tag1", "tag6")

	var boolarr []bool
	assert.NilError(t, c.ByTag("tag1", &boolarr))
	assert.DeepEqual(t, boolarr, []bool{true, false, true, false})
}

func TestCache_ByTagMixed(t *testing.T) {
	c := New(WithStorage(InMemory()))
	c.Set("key1", 123, 0, "tag1", "tag2", "tag3")
	c.Set("key2", 123.456, 0, "tag3", "tag1", "tag4")
	c.Set("key3", "ghi", 0, "tag6", "tag6", "tag1")
	c.Set("key4", true, 0, "tag5", "tag1", "tag6")
	c.Set("key5", some{
		Field1: 123,
		Field2: "abc",
		Field3: []interface{}{"abc", 123, true},
		Field4: map[string]interface{}{
			"blabla": 1234,
		},
		Field5: &some{
			Field1: 456,
			Field2: "edf",
		},
	}, 0, "tag5", "tag1", "tag6")

	var interfacearr []interface{}
	assert.NilError(t, c.ByTag("tag1", &interfacearr))
	assert.DeepEqual(t, interfacearr, []interface{}{
		123,
		123.456,
		"ghi",
		true,
		some{
			Field1: 123,
			Field2: "abc",
			Field3: []interface{}{"abc", 123, true},
			Field4: map[string]interface{}{
				"blabla": 1234,
			},
			Field5: &some{
				Field1: 456,
				Field2: "edf",
			},
		}})
}

func Test_DelByTag(t *testing.T) {
	inMem := []Storage{InMemory(), InMemory(), InMemory()}
	c := New(WithStorage(inMem...), WithNamespace("go:test"))
	c.Set("key1", 123, 0, "tag1", "tag2", "tag3")
	c.Set("key2", 123.456, 0, "tag3", "tag1", "tag4")
	c.Set("key3", "ghi", 0, "tag6", "tag6", "tag1")
	c.Set("key4", true, 0, "tag5", "tag1", "tag6")
	c.Set("key5", true, 0, "tag10")
	c.Set("key6", true, 0)
	c.Set("key7", true, 0, "tag11", "tag3")

	assert.NilError(t, c.DelByTag("tag1"))

	for _, in := range inMem {
		var keys = []string{"key1", "key2", "key3", "key4"}
		for _, key := range keys {
			_, ok := in.(*InMem).data["go:test:"+key]
			assert.Assert(t, !ok, "`%s` stil exists", key)
		}
	}
}

func TestCache_Propagate(t *testing.T) {
	var (
		s1 = InMemory()
		s2 = InMemory()
	)
	c := New(WithStorage(s1), WithNamespace("go:test"))
	c.Set("key1", 12345, time.Second, "tag1", "tag2", "tag3")

	var i int
	c = New(WithStorage(s2, s1), WithNamespace("go:test"))
	c.Get("key1", i)

	it, ok := s2.data["go:test:key1"]
	assert.Assert(t, ok)
	assert.Equal(t, it.(item).Val, 12345)
}

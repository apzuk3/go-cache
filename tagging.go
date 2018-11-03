package cache

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

type std struct {
	ns     string
	logger *logrus.Logger
}

func newStdTagger(logger *logrus.Logger, ns string) Tagger {
	return std{
		logger: logger,
		ns:     ns,
	}
}

func (std std) nsKey(key string) string {
	return fmt.Sprintf("%s:%s", std.ns, key)
}

func (std std) Tag(s Storage, key string, tags ...string) (err error) {
	if err := std.addTagsToKey(s, "key:"+key+":tags", tags...); err != nil {
		return err
	}
	for i := range tags {
		std.addKeysToTag(s, "tag:"+tags[i]+":keys", key)
	}
	return nil
}

func (std std) UnTag(s Storage, key string, tags ...string) error {
	if len(tags) == 0 {
		tags, _ = std.Tags(s, key)
	}

	std.logger.Debugf("Un-tag `%s` tags from `%s`", strings.Join(tags, ", "), key)
	if err := std.removeTagsFromKey(s, "key:"+key+":tags", tags...); err != nil {
		return err
	}

	std.logger.Debugf("Remove `%s` tags from `%s` key", strings.Join(tags, ", "), key)
	for i := range tags {
		std.removeKeysFromTag(s, "tag:"+tags[i]+":keys", key)
	}

	return s.Delete(std.nsKey("key:" + key + ":tags"))
}

func (std std) Tags(s Storage, key string) ([]string, error) {
	// get tags for the current key
	v, err := s.Read(std.nsKey("key:" + key + ":tags"))
	if err != nil && err != ErrKeyNotExist {
		return nil, err
	}
	return cast.ToStringSlice(v), nil
}

func (std std) Keys(s Storage, tag string) ([]string, error) {
	// get tags for the current key
	v, err := s.Read(std.nsKey("tag:" + tag + ":keys"))
	if err != nil && err != ErrKeyNotExist {
		return nil, err
	}
	return slice(v), nil
}

func (std std) addTagsToKey(s Storage, key string, tags ...string) error {
	// get tags for the current key
	v, err := s.Read(std.nsKey(key))
	if err != nil && err != ErrKeyNotExist {
		return err
	}

	var slice = slice(v)

	// We suppose that each key will have limited amount of tags
	// if this operation of checking for duplicates should not
	// affect the performance.
loop:
	for i := range tags {
		// skip if there are duplicates
		for k := range slice {
			if slice[k] == tags[i] {
				continue loop
			}
		}
		// append the tag
		slice = append(slice, tags...)
	}

	// Save new list of tags for the key
	return s.Write(std.nsKey(key), slice, 0)
}

func (std std) removeTagsFromKey(s Storage, key string, tags ...string) error {
	// get tags for the current key
	v, err := s.Read(std.nsKey(key))
	if err != nil && err != ErrKeyNotExist {
		return err
	}

	slice := cast.ToStringSlice(v)

	var newslice []string

	for _, tag := range tags {
		var b bool
		for k := range slice {
			if slice[k] == tag {
				b = true
				break
			}
		}
		if b {
			newslice = append(newslice, tag)
		}
	}

	if len(newslice) == 0 {
		return s.Delete(std.nsKey(key))
	}

	// Save new list of tags for the key
	return s.Write(std.nsKey(key), newslice, 0)
}

func (std std) addKeysToTag(s Storage, tag string, keys ...string) error {
	// get tags for the current key
	v, err := s.Read(std.nsKey(tag))
	if err != nil && err != ErrKeyNotExist {
		return err
	}

	// sorted list of tag's keys
	tagKeys := slice(v)

	for _, k := range keys {
		tagKeys = insertIntoSorted(tagKeys, k)
	}

	return s.Write(std.nsKey(tag), tagKeys, 0)
}

func (std std) removeKeysFromTag(s Storage, tag string, keys ...string) error {
	// get tags for the current key
	v, err := s.Read(std.nsKey(tag))
	if err != nil && err != ErrKeyNotExist {
		return err
	}

	// sorted list of tag's keys
	tagKeys := cast.ToStringSlice(v)

	for _, key := range keys {
		index := sort.Search(len(tagKeys), func(i int) bool {
			return tagKeys[i] > key
		})

		if index > 0 {
			tagKeys = append(tagKeys[:index-1], tagKeys[index:]...)
		}
	}

	if len(tagKeys) == 0 {
		return s.Delete(std.nsKey(tag))
	}

	return s.Write(std.nsKey(tag), tagKeys, 0)
}

func insertIntoSorted(slice []string, val string) []string {
	if len(slice) == 0 {
		return []string{val}
	}

	index := sort.Search(len(slice), func(i int) bool { return slice[i] > val })

	if index != 0 && slice[index-1] == val {
		return slice
	}

	if index < len(slice)-1 && slice[index+1] == val {
		return slice
	}

	if index == len(slice) {
		return append(slice, val)
	}

	newslice := slice[:index]
	newslice = append(newslice, val)
	newslice = append(newslice, slice[index+1:]...)
	return newslice
}

func slice(v interface{}) []string {
	var slice []string
	switch x := v.(type) {
	case []string:
		return x
	case []interface{}:
		return cast.ToStringSlice(x)
	case []byte:
		json.Unmarshal(x, &slice)
	case string:
		json.Unmarshal([]byte(x), &slice)
	}
	return slice
}

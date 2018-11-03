package cache

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Fs is a struct implementing Storage interface using File System
// as data storage
type Fs struct {
	dir string
}

// Filesystem creates a new File System storage which can be used when
// creating a new cache instance cache.New(WithStorage(...))
func Filesystem(dir string) Fs {
	return Fs{dir}
}

// Write writes the given content for the given key in
// File System storage
func (f Fs) Write(key string, v interface{}, ttl time.Duration) error {
	path := f.path(key)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	b, _ := json.Marshal(map[string]interface{}{key: v})
	return ioutil.WriteFile(path, b, 0600)
}

// Read reads the cached content from the corresponding file
func (f Fs) Read(key string) (interface{}, error) {
	path := f.path(key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrKeyNotExist
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return string(b), nil
	}

	v, has := m[key]
	if !has {
		return nil, ErrKeyNotExist
	}
	return v, nil
}

// Delete deletes file with cached content
func (f Fs) Delete(key string) error {
	os.Remove(f.path(key))
	return nil
}

// Flush flushes File System storage
func (f Fs) Flush() error {
	d, err := os.Open(f.dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		if errDel := os.RemoveAll(filepath.Join(f.dir, name)); errDel != nil {
			return errDel
		}
	}
	return os.Remove(f.dir)
}

func (f Fs) path(key string) string {
	h := sha1.New()
	io.WriteString(h, key)
	s := fmt.Sprintf("%x", h.Sum(nil))

	p := s[:4] + "/" + s[4:7] + "/" + s[7:]
	return filepath.Join(f.dir, p)
}

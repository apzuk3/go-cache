package cache

import (
	"testing"

	"gotest.tools/assert"
)

func TestFilesystem(t *testing.T) {
	fs := Filesystem("/cache")
	assert.Equal(t, fs.dir, "/cache")
	assert.Equal(t, fs.path("11"), "/cache/17ba/079/1499db908433b80f37c5fbc89b870084b")
}

func TestFilesystem_Write(t *testing.T) {
	fs := Filesystem("./cache")
	assert.NilError(t, fs.Write("key1", "val1", 0))

	v, err := fs.Read("key1")
	assert.NilError(t, err)
	assert.Equal(t, v, "val1")

	assert.NilError(t, fs.Flush())
}

func TestFilesystem_Delete(t *testing.T) {
	fs := Filesystem("./cache")
	fs.Write("key", "val", 0)

	assert.NilError(t, fs.Delete("key"))
}

func TestFilesystem_NotFound(t *testing.T) {
	fs := Filesystem("./cache")
	v, err := fs.Read("key1")

	assert.ErrorType(t, err, ErrKeyNotExist)
	assert.Assert(t, v == nil)
}

func TestFilesystem_WriteErrorCreatingDir(t *testing.T) {
	fs := Filesystem("/root/cache")
	assert.Error(t, fs.Write("key1", "val1", 0), "mkdir /root: permission denied")
}

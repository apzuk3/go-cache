package cache

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

type User struct {
	City  string
	Email string
}

func TestInMemory_Write(t *testing.T) {
	inMemory := InMemory()

	err := inMemory.Write("key1", 1, 0)
	assert.NilError(t, err)

	err = inMemory.Write("key2", "mystr", 0)
	assert.NilError(t, err)

	err = inMemory.Write("key3", "blabla", time.Second)
	assert.NilError(t, err)

	err = inMemory.Write("key4", User{City: "Yerevan, Armenia", Email: "aram.petrosyan.88@gmail.com"}, time.Second)
	assert.NilError(t, err)

	now := time.Now()
	err = inMemory.Write("key5", now, time.Minute)
	assert.NilError(t, err)

	assert.Equal(t, 5, len(inMemory.data))
	assert.Equal(t, 3, len(inMemory.expire))

	assert.Equal(t, 1, inMemory.data["key1"])
	assert.Equal(t, "mystr", inMemory.data["key2"])
	assert.Equal(t, "blabla", inMemory.data["key3"])
	assert.Equal(t, User{City: "Yerevan, Armenia", Email: "aram.petrosyan.88@gmail.com"}, inMemory.data["key4"])
	assert.Equal(t, now, inMemory.data["key5"])
}

func TestInMemory_Read(t *testing.T) {
	inMemory := InMemory()

	inMemory.Write("key1", 1, 0)
	inMemory.Write("key2", "mystr", 0)
	inMemory.Write("key3", "blabla", time.Second)
	inMemory.Write("key4", User{City: "Yerevan, Armenia", Email: "aram.petrosyan.88@gmail.com"}, time.Second)

	v, err := inMemory.Read("key1")
	assert.NilError(t, err)
	assert.Equal(t, 1, v)

	v, err = inMemory.Read("key2")
	assert.NilError(t, err)
	assert.Equal(t, "mystr", v)

	v, err = inMemory.Read("key3")
	assert.NilError(t, err)
	assert.Equal(t, "blabla", v)

	v, err = inMemory.Read("key4")
	assert.NilError(t, err)
	assert.Equal(t, User{City: "Yerevan, Armenia", Email: "aram.petrosyan.88@gmail.com"}, v)
}

func TestInMemory_ReadKeyNotFound(t *testing.T) {
	inMemory := InMemory()

	inMemory.Write("key1", 1, 0)
	inMemory.Write("key2", "mystr", 0)
	inMemory.Write("key3", "blabla", time.Second)
	inMemory.Write("key4", User{City: "Yerevan, Armenia", Email: "aram.petrosyan.88@gmail.com"}, time.Second)

	v, err := inMemory.Read("notExistingKey")
	assert.ErrorType(t, err, ErrKeyNotExist)
	assert.Equal(t, v, nil)
}

func TestInMemory_ReadExpire(t *testing.T) {
	inMemory := InMemory()

	inMemory.Write("key1", 1, 0)
	inMemory.Write("key2", "mystr", 0)
	inMemory.Write("key3", "blabla", time.Second)
	inMemory.Write("key4", User{City: "Yerevan, Armenia", Email: "aram.petrosyan.88@gmail.com"}, time.Millisecond*150)

	time.Sleep(time.Second + time.Millisecond*15)

	v, err := inMemory.Read("key1")
	assert.NilError(t, err)
	assert.Equal(t, 1, v)

	v, err = inMemory.Read("key2")
	assert.NilError(t, err)
	assert.Equal(t, "mystr", v)

	v, err = inMemory.Read("key3")
	assert.Error(t, err, ErrKeyNotExist.Error())
	assert.Equal(t, v, nil)

	v, err = inMemory.Read("key4")
	assert.Error(t, err, ErrKeyNotExist.Error())
	assert.Equal(t, v, nil)
}

func TestInMemory_Del(t *testing.T) {
	inMemory := InMemory()

	inMemory.Write("key1", 1, 0)
	inMemory.Write("key2", "mystr", 0)
	inMemory.Write("key3", "blabla", time.Millisecond*200)
	inMemory.Write("key4", User{City: "Yerevan, Armenia", Email: "aram.petrosyan.88@gmail.com"}, time.Millisecond*150)
	inMemory.Write("key5", time.Now(), time.Minute)

	err := inMemory.Delete("key1")
	assert.NilError(t, err)

	assert.Equal(t, 4, len(inMemory.data))
	assert.Equal(t, 3, len(inMemory.expire))

	err = inMemory.Delete("key3")
	assert.NilError(t, err)
	assert.Equal(t, 3, len(inMemory.data))
	assert.Equal(t, 2, len(inMemory.expire))
}

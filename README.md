# Go cache

[![GoDoc](https://godoc.org/apzuk3/go-cache?status.svg)](https://godoc.org/github.com/apzuk3/go-cache)
[![Build Status](https://travis-ci.com/apzuk3/go-cache.svg?branch=master)](https://travis-ci.com/apzuk3/go-cache)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/apzuk3/go-cache)](https://goreportcard.com/report/github.com/apzuk3/go-cache)

## Installation

To install the Go Cache, please execute the following `go get` command.

```bash
go get github.com/apzuk3/go-cache
```

## Usage

```go
package main

import (
    "fmt"
    "time"

    "github.com/apzuk3/go-cache"
)

func main() {
    c := cache.New(
        cache.WithStorage(cache.InMemory())
        cache.WithStorage(cache.Filesystem("./cache"))
    )

    c.Set("key1", 123, 0, "tag1", "tag2")
    c.Set("key2", "abc", 0, "tag2")

    var i int
    c.Get("key1", &i)
    fmt.Println(i) // prints 123

    var v interface{}
    c.ByTag("tag2", &v)
    fmt.Printf("%v\n", v) // prints []interface{}{123, "abs"}
}
```


Contributing
------------

If you found bugs please [file an issue](https://github.com/apzuk3/go-cache/issues/new) or pull-request with the fix


License
-------

The library is available as open source under the terms of the [MIT License](http://opensource.org/licenses/MIT).

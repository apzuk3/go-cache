# Go cache

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Installation

To install the Go Cache, please execute the following `go get` command.

```bash
go get github.com/apzuk3/go-cache
```

## Usage

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


Contributing
------------

Feel free to report or pull-request any bug at https://github.com/apzuk3/go-cache


License
-------

The library is available as open source under the terms of the [MIT License](http://opensource.org/licenses/MIT).
# go-cache

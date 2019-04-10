# Koko
Koko is a cache manager. It implements strategies used in cache, e.g. `read through`.

## Example
```go
package main

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/gomodule/redigo/redis"

	"koko"
)

var syncMap sync.Map

var redisPool = &redis.Pool{
	Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", "localhost:6379")
		if err != nil {
			return nil, err
		}
		return c, nil
	},
}

type myStruct struct {
	X int
	S string
}

// We assume this is an expensive operation.
func expensive(ks []string) ([]*myStruct, error) {
	vs := make([]*myStruct, len(ks))
	for i, k := range ks {
		vs[i] = &myStruct{
			X: i,
			S: k,
		}
	}
	return vs, nil
}

func getMyStructs(keys []string) ([]*myStruct, error) {
	valueType := reflect.TypeOf((*myStruct)(nil))
	values, err := koko.BatchReadThrough(
		// Koko can manage caches in multi layers.
		koko.ChainBatchCache(
			koko.NewMapCache(&syncMap, valueType),
			koko.NewRedisCache(redisPool, valueType),
		),
		koko.VariantFunc(expensive),
		keys,
	)
	if err != nil {
		return nil, err
	}
	return values.([]*myStruct), nil
}

func main() {
	for i := 0; i < 2; i++ {
		values, err := getMyStructs([]string{"abc", "xxx"})
		fmt.Println(values[0], err)
	}
}
```

## Introduction
Koko works with **Cache drivers** and **Callbacks**.
- **Cache drivers** wraps caches for operations like read and write. It's easy to implement a driver. Koko now provides simple driver for redis and sync.Map.
- **Callbacks** is called in situations like cache missing. With cache missing, you need to provide data for cache.
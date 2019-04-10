package main

import (
	"testing"
)

func init() {
	syncMap.Store("abc", &myStruct{0, "abc"})
	syncMap.Store("xxx", &myStruct{1, "xxx"})
}

func batchLoad(ks []string) ([]*myStruct, error) {
	values := make([]*myStruct, len(ks))
	for i, k := range ks {
		value, ok := syncMap.Load(k)
		if !ok {
			values[i] = nil
			continue
		}
		values[i] = value.(*myStruct)
	}
	return values, nil
}

func Benchmark_getMyStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		batchLoad([]string{"abc", "xxx"})
		//syncMap.Load("abc")
		//syncMap.Load("xxx")
		getMyStructs([]string{"abc", "xxx"})
	}
}

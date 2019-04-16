package koko

import (
	"reflect"
	"sync"
)

type MapCache struct {
	m         *sync.Map
	valueType reflect.Type
}

func (m *MapCache) ValueType() reflect.Type {
	return m.valueType
}

func (m *MapCache) BatchRead(keys *Slice) (*Slice, error) {
	//fmt.Println("read map:", keys)
	ks := keys.Interfaces()
	values := make([]interface{}, len(ks))
	for i, k := range ks {
		value, ok := m.m.Load(k)
		if !ok {
			values[i] = CacheMissed
			continue
		}
		values[i] = value
	}
	return SliceFromInterfaces(values, m.valueType), nil
}

func (m *MapCache) BatchWrite(keys *Slice, values *Slice) error {
	//fmt.Println("write map:", keys)
	ks := keys.Interfaces()
	vs := values.Interfaces()
	for i := 0; i < len(ks); i++ {
		if vs[i] == CacheMissed {
			continue
		}

		m.m.Store(ks[i], vs[i])
	}

	return nil
}

func (m *MapCache) CanDelete() bool {
	return true
}

func (m *MapCache) Delete(keys *Slice) error {
	for _, k := range keys.Interfaces() {
		m.m.Delete(k)
	}
	return nil
}

func NewMapCache(m *sync.Map, valueType reflect.Type) *MapCache {
	return &MapCache{
		m:         m,
		valueType: valueType,
	}
}

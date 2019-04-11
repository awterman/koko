package koko

import (
	"reflect"
)

// --- for driver, users do not need this. ---

type cacheMissed struct{}

var CacheMissed cacheMissed

type BatchCache interface {
	ValueType() reflect.Type
	// BatchRead returns values, missed values should be returned as CacheMissed.
	BatchRead(keys *Slice) (*Slice, error)
	BatchWrite(keys *Slice, values *Slice) error
}

// --- for driver, users do not need this. ---

// --- callbacks ---

type (
	BatchRead  func(keys interface{}) (interface{}, error)
	BatchWrite func(keys interface{}, values interface{}) error
)

func VariantBatchRead(fn interface{}) BatchRead {
	fnValue := reflect.ValueOf(fn)

	return func(keys interface{}) (interface{}, error) {
		out := fnValue.Call([]reflect.Value{reflect.ValueOf(keys)})
		var err error
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
		}

		return out[0].Interface(), err
	}
}

func VariantBatchWrite(fn interface{}) BatchWrite {
	fnValue := reflect.ValueOf(fn)

	return func(keys interface{}, values interface{}) error {
		out := fnValue.Call([]reflect.Value{reflect.ValueOf(keys), reflect.ValueOf(values)})
		var err error
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		return err
	}
}

// --- callbacks ---

func BatchReadThrough(c BatchCache, miss BatchRead, keys interface{}) (interface{}, error) {
	ks := SliceFromSpecific(keys)
	vs, err := c.BatchRead(ks)
	if err == nil {
		if vs == nil {
			vs = SliceFromInterfaces(nil, c.ValueType())
		}
		ks.Filter(vs.Missed())

		var missedValues interface{}
		if cc, ok := c.(chainedBatchCache); ok && cc.lower != nil {
			missedValues, err = BatchReadThrough(cc.lower, miss, ks.Specific())
		} else {
			missedValues, err = miss(ks.Specific())
		}
		if err != nil {
			return vs.Specific(), err
		}
		// It's driver's responsibility to handle write error.
		_ = c.BatchWrite(ks, SliceFromSpecific(missedValues))
		vs.Fill(SliceFromSpecific(missedValues))
		return vs.Specific(), nil
	} else {
		// It's driver's responsibility to handle read error.
		values, err := miss(keys)
		if err != nil {
			return nil, err
		}
		return values, nil
	}
}

type chainedBatchCache struct {
	BatchCache
	lower *chainedBatchCache
}

func ChainBatchCache(upper BatchCache, lowers ...BatchCache) BatchCache {
	ret := chainedBatchCache{
		BatchCache: upper,
	}
	cc := &ret

	for _, c := range lowers {
		cc.lower = &chainedBatchCache{
			BatchCache: c,
		}
		cc = cc.lower
	}

	return ret
}

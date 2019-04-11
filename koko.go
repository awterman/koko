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

func available(s *Slice, err error) bool {
	return err == nil && s != nil && !s.IsEmpty()
}

func BatchReadThrough(c BatchCache, miss BatchRead, keys interface{}) (interface{}, error) {
	ks := SliceFromSpecific(keys)
	vs, err := c.BatchRead(ks)
	if !available(vs, err) {
		values, err := miss(keys)
		if err != nil {
			return nil, err
		}
		return values, nil
	}

	ks.Filter(vs.Missed())

	missedValues, err := miss(ks.Specific())
	if err != nil {
		return vs.Specific(), err
	}
	_ = c.BatchWrite(ks, SliceFromSpecific(missedValues))
	vs.FillMissed(SliceFromSpecific(missedValues))
	return vs.Specific(), nil
}

type chainedBatchCache struct {
	BatchCache
	lower *chainedBatchCache
}

func (cc chainedBatchCache) BatchRead(keys *Slice) (*Slice, error) {
	values, err := cc.BatchCache.BatchRead(keys)
	if !available(values, err) {
		return nil, err
	}

	if cc.lower != nil {
		missedKeys := keys.Copy().Filter(values.Missed())
		missValues, err := cc.lower.BatchRead(missedKeys)
		if available(missValues, err) {
			values.FillMissed(missValues)
			_ = cc.BatchWrite(missedKeys, missValues)
		}
	}
	return values, nil
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

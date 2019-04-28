package koko

import (
	"fmt"
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
	CanDelete() bool
	Delete(keys *Slice) error
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
		if !out[0].IsNil() {
			err = out[0].Interface().(error)
		}
		return err
	}
}

// --- callbacks ---

func available(s *Slice, err error) bool {
	return err == nil && s != nil && s.Initialized()
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
		return vs.Specific(), fmt.Errorf("koko.BatchReadThrough: %v", err)
	}
	_ = c.BatchWrite(ks, SliceFromSpecific(missedValues))
	vs.FillMissed(SliceFromSpecific(missedValues))
	return vs.Specific(), nil
}

type ChainedBatchCache struct {
	// upper -> lower
	caches    []BatchCache
	valueType reflect.Type
	canDelete bool
}

func (cc *ChainedBatchCache) ValueType() reflect.Type {
	return cc.valueType
}

func (cc *ChainedBatchCache) BatchRead(keys *Slice) (*Slice, error) {
	missedKeys := keys.Copy()
	values, err := cc.caches[0].BatchRead(missedKeys)
	missedIdxs := values.Missed()
	if available(values, err) {
		missedKeys = missedKeys.Filter(missedIdxs)
	}

	var stack []BatchCache
	for _, c := range cc.caches {
		missedValues, err := c.BatchRead(missedKeys)
		if !available(missedValues, err) {
			continue
		}

		values.FillMissed(missedValues)

		for i := len(stack) - 1; i >= 0; i-- {
			_ = stack[i].BatchWrite(missedKeys, missedValues)
		}

		stack = append(stack, c)
	}

	return values, nil
}

func (cc *ChainedBatchCache) BatchWrite(keys *Slice, values *Slice) error {
	for i := len(cc.caches) - 1; i >= 0; i-- {
		err := cc.caches[i].BatchWrite(keys, values)
		if err != nil {
			return fmt.Errorf("koko.ChainedBatchCache.BatchWrite: failed to write in (level %d) cache: %v", i, err)
		}
	}

	return nil
}

func (cc *ChainedBatchCache) CanDelete() bool {
	return cc.canDelete
}

func (cc *ChainedBatchCache) Delete(keys *Slice) error {
	if !cc.CanDelete() {
		return fmt.Errorf("koko.ChainedBatchCache.Delete: delete is not supported")
	}

	for i := len(cc.caches) - 1; i >= 0; i-- {
		err := cc.caches[i].Delete(keys)
		if err != nil {
			return fmt.Errorf("koko.ChainedBatchCache.Delete: failed to delete in (level %d) cache: %v", i, err)
		}
	}

	return nil
}

func ChainBatchCache(upper BatchCache, lowers ...BatchCache) (*ChainedBatchCache, error) {
	var cc ChainedBatchCache
	cc.valueType = upper.ValueType()
	cc.caches = []BatchCache{upper}
	cc.canDelete = upper.CanDelete()

	for _, c := range lowers {
		if cc.valueType != c.ValueType() {
			return nil, fmt.Errorf("koko.ChainBatchCache: valueType: %v != %v", cc.valueType, c.ValueType())
		}
		if !c.CanDelete() {
			cc.canDelete = false
		}

		cc.caches = append(cc.caches, c)
	}

	return &cc, nil
}

package koko

import (
	"reflect"
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedisCache struct {
	pool      *redis.Pool
	valueType reflect.Type
	ex        time.Duration
	nx        bool
}

func (r *RedisCache) ValueType() reflect.Type {
	return r.valueType
}

func (r *RedisCache) BatchRead(keys *Slice) (*Slice, error) {
	//fmt.Println("read redis:", keys)
	if keys.Len() == 0 {
		return nil, nil
	}

	conn := r.pool.Get()
	defer conn.Close()

	bs, err := redis.ByteSlices(conn.Do("MGET", keys.Interfaces()...))
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(bs))
	for i, b := range bs {
		if b == nil {
			values[i] = CacheMissed
			continue
		}

		obj, err := UnmarshalJSON(b, r.valueType)
		if err != nil {
			values[i] = CacheMissed
			continue
		}
		values[i] = obj
	}

	return SliceFromInterfaces(values, r.valueType), nil
}

func (r *RedisCache) BatchWrite(keys *Slice, values *Slice) error {
	//fmt.Println("write redis:", keys)
	if keys.Len() == 0 {
		return nil
	}

	conn := r.pool.Get()
	defer conn.Close()

	for i, key := range keys.Interfaces() {
		value := values.Interfaces()[i]
		if value == CacheMissed {
			continue
		}

		b, err := MarshalJSON(value)
		if err != nil {
			// log error: failed to marshal.
		}

		args := []interface{}{key, b}

		if r.ex > 0 {
			args = append(args, "EX", r.ex.Seconds())
		}

		if r.nx {
			args = append(args, "NX")
		}

		conn.Send("SET", args...)
	}
	conn.Flush()
	conn.Receive()

	return nil
}

func (r *RedisCache) CanDelete() bool {
	return true
}

func (r *RedisCache) Delete(keys *Slice) error {
	conn := r.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", keys.Interfaces()...)
	return err
}

func (r *RedisCache) WithValueType(valueType reflect.Type) *RedisCache {
	nr := *r
	r.valueType = valueType
	return &nr
}

func (r *RedisCache) WithEX(d time.Duration) *RedisCache {
	nr := *r
	nr.ex = d
	return &nr
}

func (r *RedisCache) WithNX() *RedisCache {
	nr := *r
	nr.nx = true
	return &nr
}

func NewRedisCache(pool *redis.Pool, valueType reflect.Type) *RedisCache {
	return &RedisCache{
		pool:      pool,
		valueType: valueType,
	}
}

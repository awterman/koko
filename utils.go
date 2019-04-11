package koko

import (
	"encoding/json"
	"reflect"
)

func filter(slice []interface{}, indexs []int) []interface{} {
	selects := make([]interface{}, len(indexs))
	for i, index := range indexs {
		selects[i] = slice[index]
	}
	return selects
}

func fill(values []interface{}, toFill []interface{}, indexs []int) {
	for i, index := range indexs {
		values[index] = toFill[i]
	}
}

func UnmarshalJSON(b []byte, typ reflect.Type) (interface{}, error) {
	ptr := reflect.New(typ)
	err := json.Unmarshal(b, ptr.Interface())
	if err != nil {
		return nil, err
	}
	return ptr.Elem().Interface(), nil
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

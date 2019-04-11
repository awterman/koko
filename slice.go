package koko

import (
	"fmt"
	"reflect"
)

type Slice struct {
	spec      interface{}
	specValue reflect.Value
	inte      []interface{}
	elemType  reflect.Type

	missedIdxs []int
}

func SliceFromSpecific(spec interface{}) *Slice {
	return &Slice{spec: spec}
}

func SliceFromInterfaces(inte []interface{}, elemType reflect.Type) *Slice {
	return &Slice{
		inte:     inte,
		elemType: elemType,
	}
}

func SliceFromReflectValue(value reflect.Value) *Slice {
	return &Slice{
		specValue: value,
	}
}

func (s *Slice) String() string {
	return fmt.Sprintf("%+v", s.Interfaces())
}

func (s *Slice) Copy() *Slice {
	ns := Slice{
		inte:     make([]interface{}, len(s.inte)),
		elemType: s.elemType,
	}
	copy(ns.inte, s.inte)
	return &ns
}

func (s *Slice) IsEmpty() bool {
	if s.spec == nil &&
		(s.inte == nil || s.elemType == nil) &&
		!s.specValue.IsValid() {

		return true
	}
	return false
}

func (s *Slice) Specific() interface{} {
	if s.spec == nil {
		s.specValue = reflect.MakeSlice(reflect.SliceOf(s.elemType), len(s.inte), len(s.inte))
		for i, v := range s.inte {
			if v == CacheMissed {
				continue
			}
			s.specValue.Index(i).Set(reflect.ValueOf(v))
		}
		s.spec = s.specValue.Interface()
	}
	return s.spec
}

func (s *Slice) SpecificValue() reflect.Value {
	if !s.specValue.IsValid() {
		s.specValue = reflect.ValueOf(s.Specific())
	}
	return s.specValue
}

func (s *Slice) Interfaces() []interface{} {
	if s.inte == nil {
		specValue := s.SpecificValue()
		s.inte = make([]interface{}, specValue.Len())
		for i := 0; i < specValue.Len(); i++ {
			s.inte[i] = specValue.Index(i).Interface()
		}
		s.elemType = specValue.Type().Elem()
	}
	return s.inte
}

func (s *Slice) Missed() []int {
	if s.missedIdxs == nil {
		for i, value := range s.Interfaces() {
			if value == CacheMissed {
				s.missedIdxs = append(s.missedIdxs, i)
			}
		}
	}
	return s.missedIdxs
}

func (s *Slice) Filter(indexs []int) *Slice {
	s.inte = filter(s.Interfaces(), indexs)
	*s = Slice{
		inte:     s.inte,
		elemType: s.elemType,
	}
	return s
}

func (s *Slice) FillMissed(data *Slice) {
	fill(s.Interfaces(), data.Interfaces(), s.Missed())
	*s = Slice{
		inte:     s.inte,
		elemType: s.elemType,
	}
}

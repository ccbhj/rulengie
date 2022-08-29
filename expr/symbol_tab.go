package expr

import (
	"reflect"

	"github.com/pkg/errors"
)

type (
	SymbolTab map[string]interface{}
	FnType    func(args ...interface{}) (interface{}, error)
)

var defaultSymbolTab = map[string]interface{}{
	"true":  true,
	"false": false,
}

func NewSymbolTab() SymbolTab {
	t := make(SymbolTab, len(defaultSymbolTab))
	for k, v := range defaultSymbolTab {
		t[k] = v
	}
	return t
}

func (t SymbolTab) WithInts(m map[string]int64) SymbolTab {
	for k, v := range m {
		t[k] = v
	}
	return t
}

func (t SymbolTab) WithStrings(m map[string]string) SymbolTab {
	for k, v := range m {
		t[k] = v
	}
	return t
}

func (t SymbolTab) WithInt(k string, v int64) SymbolTab {
	t[k] = v
	return t
}

func (t SymbolTab) WithString(k string, v string) SymbolTab {
	t[k] = v
	return t
}

func (t SymbolTab) WithStruct(key string, s interface{}) SymbolTab {
	typ := reflect.TypeOf(s)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		panic(errors.Errorf("WithStruct second argument is not a struct of pointer of struct"))
	}
	t[key] = s
	for _, field := range extractStructFieldName(typ) {
		t[field] = field
	}
	return t
}

func (t SymbolTab) WithFunction(key string, fn FnType) SymbolTab {
	t[key] = fn
	return t
}

func (t SymbolTab) Clone() SymbolTab {
	ret := make(SymbolTab, len(t))
	for k, v := range t {
		ret[k] = v
	}
	return ret
}

func (t SymbolTab) Append(s SymbolTab) {
	for k, v := range s {
		t[k] = v
	}
}

package expr

import (
	"go/token"
	"reflect"
)

func token2Sym(t token.Token) SymbolKind {
	sym, in := token2SymTab[t]
	if !in {
		return SymUnknown
	}
	return sym
}

// toInt convert a value into int64 and return an int64 ,
// and whether it can be converted
// i can be converted if its type is  integer
func toInt(i interface{}) (int64, bool) {
	switch v := i.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return int64(v), true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	}
	return 0, false
}

// toFloat convert a value into float64
// i must be float32/flaot64 or integer
func toFloat(i interface{}) (float64, bool) {
	switch v := i.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	}
	return 0, false
}

// extractStructFieldName extract the field names from a struct type
//
// NOTE: unexported and anonymous field will NOT be extracted
func extractStructFieldName(typ reflect.Type) []string {
	s := make([]string, 0, 2)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() || f.Anonymous {
			continue
		}
		s = append(s, f.Name)
	}
	return s
}

// getKindPrecedence is used for equal comparator
func getKindPrecedence(k reflect.Kind) int {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uintptr:
		return 1
	case reflect.Float32, reflect.Float64:
		return 2
	default:
		return 3
	}

}

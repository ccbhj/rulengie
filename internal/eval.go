package internal

import (
	"bytes"
	"reflect"

	"github.com/pkg/errors"
)

var ErrUnsupportSymbol = errors.Errorf("unsupport symbol")

var defaultOpFnTab = map[SymbolKind]EvalFn{
	SymAnd:   EvalLAnd,
	SymOr:    EvalLOr,
	SymEq:    EvalEq,
	SymNot:   EvalNot,
	SymDot:   EvalDot,
	SymMinus: EvalMinus,

	SymLess:      newCmpEvalFn(SymLess),
	SymLessEq:    newCmpEvalFn(SymLessEq),
	SymGreater:   newCmpEvalFn(SymGreater),
	SymGreaterEq: newCmpEvalFn(SymGreaterEq),
}

var defaultSymbolTab = map[string]interface{}{
	"true":  true,
	"false": false,
}

type FnType func(args ...interface{}) (interface{}, error)

type EvalContext struct {
	symbolTab map[string]interface{}
	opFnTab   map[SymbolKind]EvalFn
}

type EvalFn func(ctx *EvalContext, args ...interface{}) (interface{}, error)

func NewEvalContext(injected map[string]interface{}) *EvalContext {
	symTab := make(map[string]interface{}, len(injected)+len(defaultSymbolTab))
	for _, tab := range []map[string]interface{}{
		injected,
		defaultSymbolTab,
	} {
		for k, v := range tab {
			symTab[k] = v
		}
	}
	return &EvalContext{
		symbolTab: symTab,
		opFnTab:   defaultOpFnTab,
	}
}

func (c *EvalContext) Eval(sym SymbolKind, args ...interface{}) (interface{}, error) {
	fn := c.opFnTab[sym]
	if fn == nil {
		return nil, errors.Wrapf(ErrUnsupportSymbol, "%s", sym)
	}
	return fn(c, args...)
}

func EvalId(ctx *EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.New("invalid number of argument")
	}

	id, in := args[0].(string)
	if !in {
		return nil, errors.New("first argument is not a string")
	}

	val, in := ctx.symbolTab[id]
	if !in {
		return nil, errors.Errorf("symbol %s not found", id)
	}
	return val, nil
}

func EvalDot(ctx *EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.New("invalid number of argument")
	}
	x, y := args[0], args[1]

	id, ok := x.(string)
	if !ok {
		return nil, errors.New("first argument is not a string")
	}

	key, ok := y.(string)
	if !ok {
		return nil, errors.New("second argument is not a string")
	}

	val, ok := ctx.symbolTab[id]
	if !ok {
		return nil, errors.Errorf("symbol %s not found", id)
	}

	var v reflect.Value
	bean := reflect.Indirect(reflect.ValueOf(val))
	switch bean.Kind() {
	case reflect.Struct:
		v = bean.FieldByName(key)
	case reflect.Map:
		v = bean.MapIndex(reflect.ValueOf(key))
	default:
		return nil, errors.Errorf("unsupport type %s to inject", bean.Kind())
	}

	if !v.IsValid() {
		return nil, errors.Errorf("fail to lookup %s in %s for symbol %s", key, bean.Type(), id)
	}

	return v.Interface(), nil
}

func EvalMinus(ctx *EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.New("invalid number of argument")
	}
	rhs, ok := args[0].(int64)
	if !ok {
		return nil, errors.New("first argument is not a bool")
	}
	return -rhs, nil
}

func EvalNot(ctx *EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.New("invalid number of argument")
	}
	rhs, ok := args[0].(bool)
	if !ok {
		return nil, errors.New("first argument is not a bool")
	}
	return !rhs, nil
}

func EvalLAnd(ctx *EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.New("invalid number of argument")
	}
	x, y := args[0], args[1]

	lhs, ok := x.(bool)
	if !ok {
		return nil, errors.New("first argument is not a bool")
	}
	if lhs == false {
		return false, nil
	}

	rhs, ok := y.(bool)
	if !ok {
		return nil, errors.New("second argument is not a bool")
	}

	return lhs && rhs, nil
}

func EvalLOr(ctx *EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.New("invalid number of argument")
	}
	x, y := args[0], args[1]

	lhs, ok := x.(bool)
	if !ok {
		return nil, errors.New("first argument is not a bool")
	}

	rhs, ok := y.(bool)
	if !ok {
		return nil, errors.New("second argument is not a bool")
	}

	return lhs || rhs, nil
}

func EvalEq(ctx *EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.New("invalid number of argument")
	}
	return ObjectsAreEqualValues(args[0], args[1]), nil
}

// ObjectsAreEqual determines if two objects are considered equal.
//
// This function does no assertion of any kind.
func ObjectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

// ObjectsAreEqualValues gets whether two objects are equal, or if their
// values are equal.
func ObjectsAreEqualValues(expected, actual interface{}) bool {
	if ObjectsAreEqual(expected, actual) {
		return true
	}

	actualType := reflect.TypeOf(actual)
	if actualType == nil {
		return false
	}
	expectedValue := reflect.ValueOf(expected)
	if expectedValue.IsValid() && expectedValue.Type().ConvertibleTo(actualType) {
		// Attempt comparison after type conversion
		return reflect.DeepEqual(expectedValue.Convert(actualType).Interface(), actual)
	}

	return false
}

func evalIntCmp(sym SymbolKind, x, y int64) (bool, error) {
	switch sym {
	case SymLess:
		return x < y, nil
	case SymLessEq:
		return x <= y, nil
	case SymGreater:
		return x > y, nil
	case SymGreaterEq:
		return x >= y, nil
	}
	return false, errors.New("unsupport type to compare")
}

func evalFloat64Cmp(sym SymbolKind, x, y float64) (bool, error) {
	switch sym {
	case SymLess:
		return x-y < 1e-6, nil
	case SymLessEq:
		return x-y <= 1e-6, nil
	case SymGreater:
		return y-x < 1e-6, nil
	case SymGreaterEq:
		return y-x <= 1e-6, nil
	}
	return false, errors.New("unsupport type to compare")
}

func newCmpEvalFn(sym SymbolKind) EvalFn {
	return func(ctx *EvalContext, args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("invalid number of argument")
		}

		x, y := args[0], args[1]

		i1, ok1 := x.(int64)
		i2, ok2 := y.(int64)
		if ok1 && ok2 {
			return evalIntCmp(sym, i1, i2)
		}

		f1, ok1 := x.(float64)
		f2, ok2 := y.(float64)
		if ok1 && ok2 {
			return evalFloat64Cmp(sym, f1, f2)
		}

		return nil, errors.New("invalid type for comparing")
	}
}

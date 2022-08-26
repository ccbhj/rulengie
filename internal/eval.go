package internal

import (
	"bytes"
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
)

var _ ast.Visitor = (*EvalContext)(nil)

type EvalContext struct {
	Tokens    []Token
	symbolTab map[string]Sym
	opFn      map[SymbolKind]OpFn
}

type EvalFn func(ctx *EvalContext, args ...interface{}) interface{}

func (p *EvalContext) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch v := node.(type) {
	case *ast.UnaryExpr:
		op := token2Sym(v.Op)
		if op.kind == SymUnknown {
			panic(NewUnsupportOpErr(int(v.Pos())))
		}
		p.Tokens = append(p.Tokens, op)
	case *ast.BinaryExpr:
		op := token2Sym(v.Op)
		if op.kind == SymUnknown {
			panic(NewUnsupportOpErr(int(v.Pos())))
		}
		p.Tokens = append(p.Tokens, op)
	case *ast.SelectorExpr:
		p.Tokens = append(p.Tokens, Sym{2, SymDot})
	case *ast.BasicLit:
		p.Tokens = append(p.Tokens, basicLit2Operand(v))
	case *ast.Ident:
		p.Tokens = append(p.Tokens, Sym{2, SymId})
		p.Tokens = append(p.Tokens, Operand{
			val:  v.Name,
			txt:  v.Name,
			kind: reflect.String,
		})
	case *ast.ParenExpr:
		// do nothing
	default:
		panic(NewInvalidSyntax(int(node.Pos())))
	}
	return p
}

func (c *EvalContext) Eval() (interface{}, error) {
	if len(c.Tokens) == 0 {
		return nil, nil
	}
	var (
		i   = len(c.Tokens) - 1
		stk = NewStack(len(c.Tokens))
	)
	for i > 0 {
		var sym Sym
		t := c.Tokens[i]
		switch v := t.(type) {
		case Operand:
			stk.Push(v)
			i--
			continue
		case Sym:
			sym = v
			break
		default:
			panic("invalid value in tokens")
		}

		vals := stk.PopN(sym.noperand)
		if len(vals) != sym.noperand {
			return nil, errors.New("evaluate error, no enough operand")
		}

		fn := c.opFn[sym.kind]
		if fn == nil {
			return nil, errors.Errorf("apply function for %s is nil or not register", sym.kind)
		}
		val := fn(c, vals...)
		stk.Push(val)
		i--
	}

	return stk.Pop(), nil
}

func EvalId(ctx *EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.New("invalid number of argument")
	}

	id, ok := args[0].(string)
	if !ok {
		return nil, errors.New("first argument is not a string")
	}

	val, ok := ctx.symbolTab[id]
	if !ok {
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

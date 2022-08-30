package expr

import (
	"reflect"

	"github.com/pkg/errors"
)

type (
	// EvalFn evaluates values passed from the parser
	EvalFn func(ctx EvalContext, args ...interface{}) (interface{}, error)

	EvalContext interface {
		Eval(sym SymbolKind, args ...interface{}) (interface{}, error)
		LookupSymbol(key string) (interface{}, bool)
	}

	// exprEvalContext maintains states when we evaluate values.
	// It tells how to evaluate by an operator and provide a symbol table for
	// EvalFn to lookup
	//
	// By changing a EvalFn for a operator, we can 'overload' the operator.
	// By injecting value into symbal tab, we can visit those value outside the
	// expression
	exprEvalContext struct {
		symbolTab SymbolTab
		opFnTab   map[SymbolKind]EvalFn
	}
)

var goExprOpFnTab = map[SymbolKind]EvalFn{
	SymAnd:   evalLAnd,
	SymOr:    evalLOr,
	SymEq:    evalEq,
	SymNeq:   evalNeq,
	SymNot:   evalNot,
	SymDot:   evalDot,
	SymMinus: evalMinus,
	SymParen: evalParen,

	SymLess:      newCmpEvalFn(SymLess),
	SymLessEq:    newCmpEvalFn(SymLessEq),
	SymGreater:   newCmpEvalFn(SymGreater),
	SymGreaterEq: newCmpEvalFn(SymGreaterEq),
}

var esQueryOpFnTab = map[SymbolKind]EvalFn{
	SymAnd:   evalLAnd,
	SymOr:    evalLOr,
	SymEq:    evalEq,
	SymNeq:   evalNeq,
	SymDot:   evalDot,
	SymMinus: evalMinus,
}

func NewEvalContext(injected SymbolTab) *exprEvalContext {
	symTab := make(map[string]interface{}, len(injected)+len(defaultSymbolTab))
	for _, tab := range []map[string]interface{}{
		injected,
		defaultSymbolTab,
	} {
		for k, v := range tab {
			symTab[k] = v
		}
	}
	return &exprEvalContext{
		symbolTab: symTab,
		opFnTab:   goExprOpFnTab,
	}
}

func (c *exprEvalContext) Eval(sym SymbolKind, args ...interface{}) (interface{}, error) {
	fn := c.opFnTab[sym]
	if fn == nil {
		return nil, errors.Wrapf(ErrUnsupportSymbol, "%s", sym)
	}
	return fn(c, args...)
}

func (c *exprEvalContext) LookupSymbol(key string) (interface{}, bool) {
	v, in := c.symbolTab[key]
	return v, in
}

func evalId(ctx EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}

	id, in := args[0].(string)
	if !in {
		return nil, errors.Wrap(ErrEvalError, "first argument is not a string")
	}

	val, in := ctx.LookupSymbol(id)
	if !in {
		return nil, errors.Wrapf(ErrEvalError, "symbol %s not found", id)
	}
	return val, nil
}

func evalDot(ctx EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	val, x := args[0], args[1]
	if val == nil {
		return nil, errors.Wrap(ErrEvalError, "first arugment is nil")
	}

	key, ok := x.(string)
	if !ok {
		return nil, errors.Wrap(ErrEvalError, "second argument is not a string")
	}

	var v reflect.Value
	bean := reflect.Indirect(reflect.ValueOf(val))
	switch bean.Kind() {
	case reflect.Struct:
		v = bean.FieldByName(key)
	default:
		return nil, errors.Wrapf(ErrEvalError, "unsupport type %s to inject", bean.Kind())
	}

	if !v.IsValid() {
		return nil, errors.Wrapf(ErrEvalError, "fail to lookup %v in %s for symbol %s", val, bean.Type(), key)
	}

	return v.Interface(), nil
}

func evalMinus(ctx EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	rhs, ok := toInt(args[0])
	if !ok {
		return nil, errors.Wrap(ErrEvalError, "first argument is not a bool")
	}
	return -rhs, nil
}

func evalNot(ctx EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	rhs, ok := args[0].(bool)
	if !ok {
		return nil, errors.Wrapf(ErrEvalError, "first argument is not a bool")
	}
	return !rhs, nil
}

func evalLAnd(ctx EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	x, y := args[0], args[1]

	lhs, ok := x.(bool)
	if !ok {
		return nil, errors.Wrap(ErrEvalError, "first argument is not a bool")
	}
	if lhs == false {
		return false, nil
	}

	rhs, ok := y.(bool)
	if !ok {
		return nil, errors.Wrap(ErrEvalError, "second argument is not a bool")
	}

	return lhs && rhs, nil
}

func evalLOr(ctx EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	x, y := args[0], args[1]

	lhs, ok := x.(bool)
	if !ok {
		return nil, errors.Wrap(ErrEvalError, "first argument is not a bool")
	}

	rhs, ok := y.(bool)
	if !ok {
		return nil, errors.Wrap(ErrEvalError, "second argument is not a bool")
	}

	return lhs || rhs, nil
}

func evalNeq(ctx EvalContext, args ...interface{}) (interface{}, error) {
	res, err := evalEq(ctx, args...)
	if err != nil {
		return nil, err
	}
	return !res.(bool), nil
}

func evalEq(ctx EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	x, y := args[0], args[1]
	if x == y {
		return true, nil
	}

	tx := reflect.TypeOf(x)
	ty := reflect.TypeOf(y)
	if !tx.Comparable() {
		return nil, errors.Wrapf(ErrEvalError, "type %s is not comparable", tx.String())
	}
	if !ty.Comparable() {
		return nil, errors.Wrapf(ErrEvalError, "type %s is not comparable", ty.String())
	}

	if tx == ty {
		return reflect.DeepEqual(x, y), nil
	}

	if (tx.Kind() == reflect.String && ty.Kind() != reflect.String) ||
		(ty.Kind() == reflect.String && tx.Kind() != reflect.String) {
		return nil, errors.Wrapf(ErrEvalError, "type %s and %s is not comparable", tx.String(), ty.String())
	}
	px, py := getKindPrecedence(tx.Kind()), getKindPrecedence(ty.Kind())
	if px >= py {
		if ty.ConvertibleTo(tx) {
			v := reflect.ValueOf(y)
			if v.IsValid() {
				return reflect.DeepEqual(v.Convert(tx).Interface(), x), nil
			}
		}
	} else {
		if tx.ConvertibleTo(ty) {
			v := reflect.ValueOf(x)
			if v.IsValid() {
				return reflect.DeepEqual(v.Convert(ty).Interface(), y), nil
			}
		}
	}

	return nil, errors.Wrapf(ErrEvalError, "type %s and %s is not comparable", tx.String(), ty.String())
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
	return false, errors.Wrap(ErrEvalError, "unsupport type to compare")
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
	return false, errors.Wrap(ErrEvalError, "unsupport type to compare")
}

func newCmpEvalFn(sym SymbolKind) EvalFn {
	return func(ctx EvalContext, args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
		}

		x, y := args[0], args[1]

		i1, ok1 := toInt(x)
		i2, ok2 := toInt(y)
		if ok1 && ok2 {
			return evalIntCmp(sym, i1, i2)
		}

		f1, ok1 := toFloat(x)
		f2, ok2 := toFloat(y)
		if ok1 && ok2 {
			return evalFloat64Cmp(sym, f1, f2)
		}

		return nil, errors.Wrap(ErrEvalError, "invalid type for comparing")
	}
}

func evalParen(ctx EvalContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	return args[0], nil
}

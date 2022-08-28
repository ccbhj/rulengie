package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
)

var defaultVisitorTab = map[reflect.Type]VisitFn{
	reflect.TypeOf(&ast.BinaryExpr{}): BinaryExprVisitor,
	reflect.TypeOf(&ast.UnaryExpr{}):  UnaryExprVisitor,
	reflect.TypeOf(&ast.BasicLit{}):   BasicLitVisitor,
	reflect.TypeOf(&ast.Ident{}):      IndentVisitor,
	reflect.TypeOf(&ast.CallExpr{}):   CallVisitor,
}

type parseErr struct {
	pos          int
	end          int
	msg          string
	exprTypeName string
}

type VisitFn func(*EvalContext, *ParseContext, ast.Node) (interface{}, error)

type ParseContext struct {
	visitorTab map[reflect.Type]VisitFn
}

func (e parseErr) Error() string {
	msg := "parse error"
	if e.msg != "" {
		msg = e.msg
	}
	return fmt.Sprintf("%s at [%d, %d] for expr %s", msg, e.pos, e.end, e.exprTypeName)
}

func newParseErr(n ast.Node, msg string, args ...interface{}) parseErr {
	typeName := "unknown expression"
	if t := reflect.TypeOf(n); t != nil {
		typeName = t.String()
	}
	return parseErr{
		pos:          int(n.Pos()),
		end:          int(n.End()),
		msg:          fmt.Sprintf(msg, args...),
		exprTypeName: typeName,
	}
}

func NewParseContext() *ParseContext {
	return &ParseContext{
		visitorTab: defaultVisitorTab,
	}
}

func VisitNode(ectx *EvalContext, pctx *ParseContext, n ast.Node) (interface{}, error) {
	t := reflect.TypeOf(n)
	v, found := pctx.visitorTab[t]
	if !found {
		return nil, newParseErr(n, "unsupport expression")
	}
	return v(ectx, pctx, n)
}

func BinaryExprVisitor(ectx *EvalContext, pctx *ParseContext, node ast.Node) (interface{}, error) {
	bn := node.(*ast.BinaryExpr)
	sym := token2Sym(bn.Op)
	if sym == SymUnknown {
		return nil, newParseErr(node, "unsupport operator '%s'", bn.Op.String())
	}
	x, err := VisitNode(ectx, pctx, bn.X)
	if err != nil {
		return nil, err
	}
	y, err := VisitNode(ectx, pctx, bn.Y)
	if err != nil {
		return nil, err
	}

	return ectx.Eval(sym, x, y)
}

func UnaryExprVisitor(ectx *EvalContext, pctx *ParseContext, node ast.Node) (interface{}, error) {
	bn := node.(*ast.UnaryExpr)
	sym := token2Sym(bn.Op)
	if sym == SymUnknown {
		return nil, newParseErr(node, "unsupport operator '%s'", bn.Op.String())
	}
	x, err := VisitNode(ectx, pctx, bn.X)
	if err != nil {
		return nil, err
	}

	return ectx.Eval(sym, x)
}

func BasicLitVisitor(_ *EvalContext, _ *ParseContext, node ast.Node) (interface{}, error) {
	bl := node.(*ast.BasicLit)
	switch bl.Kind {
	case token.STRING, token.CHAR:
		return bl.Value, nil
	case token.INT:
		return strconv.ParseInt(bl.Value, 10, 64)
	case token.FLOAT:
		return strconv.ParseFloat(bl.Value, 64)
	}
	return nil, newParseErr(node, "unsupport constant %s of kind %s", bl.Value, bl.Kind)
}

func IndentVisitor(ectx *EvalContext, _ *ParseContext, node ast.Node) (interface{}, error) {
	id := node.(*ast.Ident)
	val, in := ectx.symbolTab[id.Name]
	if !in {
		return nil, newParseErr(node, "unknown symbol %s", id.Name)
	}
	return val, nil
}

func CallVisitor(ectx *EvalContext, pctx *ParseContext, node ast.Node) (interface{}, error) {
	call := node.(*ast.CallExpr)
	args := make([]interface{}, 0, 1)

	fnVal, err := VisitNode(ectx, pctx, call.Fun)
	if err != nil {
		return nil, err
	}

	fn, ok := fnVal.(FnType)
	if !ok {
		return nil, newParseErr(call.Fun, "injected value %s is not a type of func(args ...interface{})(interface{}, error)", fn)
	}

	for _, expr := range call.Args {
		v, err := VisitNode(ectx, pctx, expr)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}

	return fn(args...)
}

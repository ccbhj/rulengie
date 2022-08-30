package expr

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
)

// ParseFn is a function to parse for a specified ast node,
// each ast node we support should have a ParseFn
type ParseFn func(EvalContext, *ParseContext, ast.Node) (interface{}, error)

// ParseContext maintains states when the expression was parsed,
// it mainly tells the [ParseNode] how to parse an ast node.
//
// By modifying the parserTab, we parse an expression with a different syntax.
type ParseContext struct {
	astRoot   ast.Node
	parserTab map[reflect.Type]ParseFn
}

var defaultParserTab = map[reflect.Type]ParseFn{
	reflect.TypeOf(&ast.BinaryExpr{}):   binaryExprParser,
	reflect.TypeOf(&ast.UnaryExpr{}):    unaryExprParser,
	reflect.TypeOf(&ast.BasicLit{}):     basicLitParser,
	reflect.TypeOf(&ast.Ident{}):        indentParser,
	reflect.TypeOf(&ast.CallExpr{}):     callParser,
	reflect.TypeOf(&ast.ParenExpr{}):    parenParser,
	reflect.TypeOf(&ast.SelectorExpr{}): selectParser,
}

// NewParseContext a
func NewParseContext(exp string) (*ParseContext, error) {
	root, err := parser.ParseExpr(exp)
	if err != nil {
		return nil, errors.Wrapf(err, "expr: %s", exp)
	}
	return &ParseContext{
		parserTab: defaultParserTab,
		astRoot:   root,
	}, nil
}

func ParseExpr(exp string, sym SymbolTab) (interface{}, error) {
	pctx, err := NewParseContext(exp)
	if err != nil {
		return nil, err
	}
	ectx := NewEvalContext(sym)
	return parseNode(ectx, pctx, pctx.astRoot)
}

func (p *ParseContext) ParseAndEval(ectx EvalContext) (interface{}, error) {
	return parseNode(ectx, p, p.astRoot)
}

// parseNode parses an ast node and evaluate its value recursively
func parseNode(ectx EvalContext, pctx *ParseContext, n ast.Node) (interface{}, error) {
	t := reflect.TypeOf(n)
	v, found := pctx.parserTab[t]
	if !found {
		return nil, newParseErr(n, "unsupport expression")
	}
	return v(ectx, pctx, n)
}

// binaryExprParser parses left and right hand side values and evaluated by the
// operator like comparator and logic operator
func binaryExprParser(ectx EvalContext, pctx *ParseContext, node ast.Node) (interface{}, error) {
	bn := node.(*ast.BinaryExpr)
	sym := token2Sym(bn.Op)
	if sym == SymUnknown {
		return nil, newParseErr(node, "unsupport operator '%s'", bn.Op.String())
	}
	x, err := parseNode(ectx, pctx, bn.X)
	if err != nil {
		return nil, err
	}
	y, err := parseNode(ectx, pctx, bn.Y)
	if err != nil {
		return nil, err
	}

	return ectx.Eval(sym, x, y)
}

// unaryExprParser parses the left hand side value and evaluate by the operator
// like minus
func unaryExprParser(ectx EvalContext, pctx *ParseContext, node ast.Node) (interface{}, error) {
	bn := node.(*ast.UnaryExpr)
	sym := token2Sym(bn.Op)
	if sym == SymUnknown {
		return nil, newParseErr(node, "unsupport operator '%s'", bn.Op.String())
	}
	x, err := parseNode(ectx, pctx, bn.X)
	if err != nil {
		return nil, err
	}

	return ectx.Eval(sym, x)
}

// basicLitParser parses constant value, only string, char, integer and float
// are supported.
//
// Integer will be parsed into int64, string/char will be paresed into string
// and float will be parsed into float64
func basicLitParser(_ EvalContext, _ *ParseContext, node ast.Node) (interface{}, error) {
	bl := node.(*ast.BasicLit)
	switch bl.Kind {
	case token.STRING:
		return strconv.Unquote(bl.Value)
	case token.INT:
		return strconv.ParseInt(bl.Value, 10, 64)
	case token.FLOAT:
		return strconv.ParseFloat(bl.Value, 64)
	}
	return nil, newParseErr(node, "unsupport constant %s of kind %s", bl.Value, bl.Kind)
}

// indentParser parse [ast.Indent], then look up the symbol table and return
// the value, if no value is found in symbol table, an error will be thrown.
func indentParser(ectx EvalContext, _ *ParseContext, node ast.Node) (interface{}, error) {
	id := node.(*ast.Ident)
	val, in := ectx.LookupSymbol(id.Name)
	if !in {
		return nil, newParseErr(node, "unknown symbol %s", id.Name)
	}
	return val, nil
}

// callParser parse [ast.CallExpr] node, then lookup the symbol table to find
// a registered function whose type is [FnType]
func callParser(ectx EvalContext, pctx *ParseContext, node ast.Node) (interface{}, error) {
	call := node.(*ast.CallExpr)
	args := make([]interface{}, 0, 1)

	fnVal, err := parseNode(ectx, pctx, call.Fun)
	if err != nil {
		return nil, err
	}

	fn, ok := fnVal.(FnType)
	if !ok {
		return nil, newParseErr(call.Fun, "injected value %s is not a type of func(args ...interface{})(interface{}, error)", fnVal)
	}

	for _, expr := range call.Args {
		v, err := parseNode(ectx, pctx, expr)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}

	return fn(args...)
}

// parenParser parse a [ast.ParenExpr] node, which returns the value inside a pair of
// paren
func parenParser(ectx EvalContext, pctx *ParseContext, node ast.Node) (interface{}, error) {
	n := node.(*ast.ParenExpr)
	return parseNode(ectx, pctx, n.X)
}

// selectParser parse a [ast.SelectorExpr] node, which will lookup the symbol
// tab for both the selector and the field, and evaluate them by [SymDot]
func selectParser(ectx EvalContext, pctx *ParseContext, node ast.Node) (interface{}, error) {
	sel := node.(*ast.SelectorExpr)

	from, err := parseNode(ectx, pctx, sel.X)
	if err != nil {
		return nil, err
	}
	if from == nil {
		return nil, newParseErr(sel.X, "cannot select from nil value")
	}

	id := sel.Sel.Name

	ret, err := ectx.Eval(SymDot, from, id)
	if err != nil {
		return nil, newParseErr(node, err.Error())
	}
	return ret, nil
}

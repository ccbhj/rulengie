package internal

import (
	"go/token"
	"reflect"
)

type (
	TokenKind int

	Token interface {
		_token()
	}

	SymbolKind string
	Sym        struct {
		noperand int
		kind     SymbolKind
	}

	Operand struct {
		val  interface{}
		txt  string
		kind reflect.Kind
	}

	OpFn func(*EvalContext, ...interface{}) interface{}
)

const (
	SymUnknown            = "unknown"
	SymAnd     SymbolKind = "&&"
	SymOr      SymbolKind = "||"
	SymEq      SymbolKind = "=="
	SymNot     SymbolKind = "!"
	SymDot     SymbolKind = "."
	SymId      SymbolKind = "id"
	SymMinus   SymbolKind = "~"
)

var token2SymTab = map[token.Token]Sym{
	token.LAND: {
		noperand: 2,
		kind:     SymAnd,
	},
	token.LOR: {
		noperand: 1,
		kind:     SymOr,
	},
	token.NOT: {
		noperand: 1,
		kind:     SymOr,
	},
	token.EQL: {
		noperand: 1,
		kind:     SymEq,
	},
	token.SUB: {
		noperand: 1,
		kind:     SymMinus,
	},
}

func (Sym) _token() {}

func (Operand) _token() {}

func (s Sym) String() string {
	return string(s.kind)
}

func (o Operand) String() string {
	return o.txt
}

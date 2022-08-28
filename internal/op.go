package internal

import (
	"go/token"
)

type SymbolKind int

const (
	SymUnknown SymbolKind = iota
	SymAnd
	SymOr
	SymEq
	SymNot
	SymDot
	SymMinus
	SymLess
	SymLessEq
	SymGreater
	SymGreaterEq
)

var token2SymTab = map[token.Token]SymbolKind{
	token.LAND: SymAnd,
	token.LOR:  SymOr,
	token.NOT:  SymOr,

	token.EQL: SymEq,
	token.LSS: SymLess,
	token.LEQ: SymLessEq,
	token.GTR: SymGreater,
	token.GEQ: SymGreaterEq,

	token.SUB: SymMinus,
}

func (s SymbolKind) String() string {
	switch s {
	case SymAnd:
		return "&&"
	case SymOr:
		return "||"
	case SymNot:
		return "!"

	case SymEq:
		return "=="
	case SymLess:
		return "<"
	case SymLessEq:
		return "<="
	case SymGreater:
		return ">"
	case SymGreaterEq:
		return ">="
	case SymDot:
		return "."
	case SymMinus:
		return "-"
	}
	return "unknown"
}

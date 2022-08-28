package internal

import (
	"go/token"
)

func token2Sym(t token.Token) SymbolKind {
	sym, in := token2SymTab[t]
	if !in {
		return SymUnknown
	}
	return sym
}

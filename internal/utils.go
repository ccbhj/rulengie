package internal

import (
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
)

func token2Sym(t token.Token) Sym {
	sym, in := token2SymTab[t]
	if !in {
		return Sym{0, SymUnknown}
	}
	return sym
}

func basicLit2Operand(n *ast.BasicLit) Operand {
	switch n.Kind {
	case token.STRING, token.CHAR:
		return Operand{
			val:  n.Value,
			txt:  n.Value,
			kind: reflect.Int64,
		}
	case token.INT:
		v, err := strconv.ParseInt(n.Value, 10, 64)
		if err != nil {
			panic(err)
		}
		return Operand{
			txt:  n.Value,
			val:  v,
			kind: reflect.Int64,
		}
	case token.FLOAT:
		v, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			panic(err)
		}
		return Operand{
			val:  v,
			txt:  n.Value,
			kind: reflect.Int64,
		}
	}
	panic(NewUnsupportConstErr(int(n.Pos())))
}

type Stack struct {
	stk []interface{}
}

func NewStack(cp int) *Stack {
	return &Stack{
		stk: make([]interface{}, 0, cp),
	}
}

func (s *Stack) Empty() bool {
	return len(s.stk) == 0
}

func (s *Stack) Push(v interface{}) {
	s.stk = append(s.stk, v)
}

func (s *Stack) Pop() interface{} {
	if s.Empty() {
		panic("empty stack")
	}
	l := len(s.stk)
	v := s.stk[l-1]
	s.stk = s.stk[:l-1]
	return v
}

func (s *Stack) PopN(n int) []interface{} {
	vals := make([]interface{}, n)
	for i := n; !s.Empty() && n > 0; i-- {
		vals[i-1] = s.Pop()
	}
	return vals
}

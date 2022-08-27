package internal

import "fmt"

type EvalErr struct {
	Pos int
	Msg string
}

func (e EvalErr) Error() string {
	return fmt.Sprintf("eval error at %d: %s", e.Pos, e.Msg)
}

func NewUnsupportOpErr(pos int) EvalErr {
	return EvalErr{Pos: pos, Msg: "unsupported operator"}
}

func NewUnsupportConstErr(pos int) EvalErr {
	return EvalErr{Pos: pos, Msg: "unsupported constant value"}
}

func NewInvalidSyntax(pos int) EvalErr {
	return EvalErr{Pos: pos, Msg: "invalid syntax"}
}

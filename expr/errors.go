package expr

import (
	"fmt"
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
)

type EvalErr struct {
	Pos int
	Msg string
}

var ErrUnsupportSymbol = errors.Errorf("unsupport symbol")
var ErrEvalError = errors.Errorf("evaluation error")

type parseErr struct {
	pos          int
	end          int
	msg          string
	exprTypeName string
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

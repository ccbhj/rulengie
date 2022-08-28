package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/ccbhj/rulengine/internal"
)

func main() {
	var expr = `
		 add_one(2) == 3 
	`
	t, err := parser.ParseExpr(expr)
	if err != nil {
		panic(err)
	}
	if err := ast.Print(token.NewFileSet(), t); err != nil {
		panic(err)
	}

	fn := func(args ...interface{}) (interface{}, error) {
		return args[0].(int64) + 1, nil
	}
	ectx := internal.NewEvalContext(map[string]interface{}{
		"add_one": internal.FnType(fn),
	})
	pctx := internal.NewParseContext()

	val, err := internal.VisitNode(ectx, pctx, t)
	if err != nil {
		panic(err)
	}
	fmt.Printf("result: %+v", val)
}

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
		(false || true) && true
	`
	t, err := parser.ParseExpr(expr)
	if err != nil {
		panic(err)
	}
	if err := ast.Print(token.NewFileSet(), t); err != nil {
		panic(err)
	}
	ctx := &internal.EvalContext{}
	ast.Walk(ctx, t)
	for _, t := range ctx.Tokens {
		fmt.Println(t)
	}
}

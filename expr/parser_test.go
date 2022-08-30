package expr

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestEvalOperator(t *testing.T) {
	t.Skip()
	cases := map[string]interface{}{
		`2 == 1 && 1 == 1`:                     false,
		`1 != 1`:                               false,
		`true || false`:                        true,
		`(1 <= 3 || 4 >= 4) && !(-1 >= 0)`:     true,
		`(1 < 3 || 4 > 4) && !(-1 > 0)`:        true,
		`1.0001 == 1.0001 && 1.0001 >= 1.0001`: true,
		`1.001 > 1`:                            true,
		`"a" == "b" || "c" == "c"`:             true,
		`"aaa" != "bbb"`:                       true,

		// should convert to float64
		`1.0 == 1`: true,
		`1.1 != 1`: true,
	}

	errCase := []string{
		// unsupport operator
		"2 = 1",
		"*a",
		"&a",
		// invalid type to compare
		`"a" > 1`,
		`"a" >= 1`,
		`"a" <= 1`,
		`"a" < 1`,
		`"ab" != 1`,

		// unsupport type
		`'a' != 'a'`,

		// unknown symbol
		`len("abc") == len("123")`,

		// invalid syntax
		`2 == `,
		`var i := 2`,
		`(2 > 1 && 1 == 1`,
		`1 == 1 || 2`,
	}

	for exp, expect := range cases {
		result, err := ParseExpr(exp, nil)
		assert.NoError(t, err, exp)
		assert.EqualValues(t, expect, result, exp)
	}

	for _, exp := range errCase[7:7] {
		_, err := ParseExpr(exp, nil)
		assert.Error(t, err, exp)
	}
}

func TestInjectValue(t *testing.T) {
	t.Skip()
	lenFn := func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("function len need one argument")
		}
		v := reflect.ValueOf(args[0])
		switch v.Kind() {
		case reflect.String, reflect.Map, reflect.Chan, reflect.Array, reflect.Slice:
			return v.Len(), nil
		}
		return nil, errors.Errorf("function len not support for type %s", v.Type().String())
	}
	testStr := "test"
	exp := `len(val) == default_len`
	symbolTab := NewSymbolTab().WithFunction("len", lenFn).
		WithInt("default_len", int64(len(testStr))).
		WithStrings(map[string]string{"val": testStr})

	result, err := ParseExpr(exp, symbolTab)
	assert.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestInjectStruct(t *testing.T) {
	t.Skip()
	type Arg struct {
		S int
	}
	arg := Arg{S: 100}

	symbolTab := NewSymbolTab().WithStruct("arg", arg)
	exp := `arg.S == 100`

	result, err := ParseExpr(exp, symbolTab)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.(bool))

	type WrapStruct struct {
		Arg Arg
	}
	ws := WrapStruct{arg}

	symbolTab = NewSymbolTab().WithStruct("ws", ws).WithString("S", "S")
	result, err = ParseExpr("ws.Arg.S == 100", symbolTab)
	assert.NoError(t, err)
	assert.True(t, result.(bool))

	// unexport field
	type Arg1 struct {
		s int
	}
	arg1 := Arg1{s: 100}
	symbolTab = NewSymbolTab().WithStruct("arg", arg1)
	_, err = ParseExpr(exp, symbolTab)
	assert.Error(t, err)
}

func TestEsParser(t *testing.T) {
	expr := `query.ic_no != "123" && query.gender != 1 `

	ectx := NewEsQueryCtx(nil)
	parser, err := NewParseContext(expr)
	assert.NoError(t, err)

	query, err := parser.ParseAndEval(ectx)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	src, err := query.(*elastic.BoolQuery).Source()
	assert.NoError(t, err)

	s, err := json.MarshalIndent(src, "", "  ")
	assert.NoError(t, err)
	t.Logf("query =\n %s", s)
}

package expr

import (
	"github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
)

type (
	EsQueryContext struct {
		*exprEvalContext
	}

	queryType int

	queryKv struct {
		key string
		val interface{}
	}

	nestedQuery elastic.BoolQuery
)

const queryResultKey = "query"

func NewEsQueryCtx(inject SymbolTab) *EsQueryContext {
	if inject == nil {
		inject = make(SymbolTab, 1)
	}
	inject[queryResultKey] = queryResultKey
	return &EsQueryContext{NewEvalContext(inject)}
}

func (c *EsQueryContext) Eval(sym SymbolKind, args ...interface{}) (interface{}, error) {
	switch sym {
	case SymEq:
		return evalMatch(c, args...)
	case SymNeq:
		return evalNotMatch(c, args...)
	case SymAnd:
		return evalMust(c, args...)
	case SymOr:
		return evalShould(c, args...)
	case SymDot:
		return evalEsDot(c, args...)
	case SymParen:
		return evalEsNest(c, args...)
	}

	return nil, errors.Errorf("unsupport symbol %s", sym.String())
}

func (c *EsQueryContext) LookupSymbol(key string) (interface{}, bool) {
	v, in := c.exprEvalContext.symbolTab[key]
	return v, in
}

func (n *nestedQuery) Source() (interface{}, error) {
	return ((*elastic.BoolQuery)(n)).Source()
}

func evalEsDot(ctx *EsQueryContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	val, x := args[0], args[1]
	if val == nil {
		return nil, errors.Wrap(ErrEvalError, "first arugment is nil")
	}
	if v, ok := val.(string); ok && v == queryResultKey {
		if key, ok := x.(string); ok {
			return &queryKv{key: key}, nil
		}
	}

	return evalDot(ctx, args...)
}

func evalMatch(ctx *EsQueryContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	var query *queryKv
	x, y := args[0], args[1]
	if x == y {
		return true, nil
	}
	if q, ok := x.(*queryKv); ok {
		q.val = y
		query = q
	} else if q, ok := y.(*queryKv); ok {
		q.val = x
		query = q
	}
	if query == nil {
		return nil, errors.New("no query found")
	}

	return elastic.NewTermQuery(query.key, query.val), nil
}

func evalNotMatch(ctx *EsQueryContext, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	var query *queryKv
	x, y := args[0], args[1]
	if x == y {
		return true, nil
	}
	if q, ok := x.(*queryKv); ok {
		q.val = y
		query = q
	} else if q, ok := y.(*queryKv); ok {
		q.val = x
		query = q
	}
	if query == nil {
		return nil, errors.New("no query found")
	}
	result := elastic.NewBoolQuery().MustNot(elastic.NewTermQuery(query.key, query.val))
	return (*nestedQuery)(result), nil
}

func evalShouldOrMust(apply func(*elastic.BoolQuery, ...elastic.Query) *elastic.BoolQuery) func(*EsQueryContext, ...interface{}) (interface{}, error) {
	return func(ctx *EsQueryContext, args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
		}
		x, y := args[0], args[1]
		if !isElasticQuery(x) {
			return nil, errors.Errorf("left hand side is not a query: %v", x)
		}
		if !isElasticQuery(y) {
			return nil, errors.Errorf("right hand side is not a query: %v", y)
		}

		kv1, isKv1 := x.(*elastic.TermQuery)
		kv2, isKv2 := y.(*elastic.TermQuery)
		if isKv1 && isKv2 {
			q := elastic.NewBoolQuery()
			return apply(q, kv1, kv2), nil
		}

		bq1, isQry1 := x.(*elastic.BoolQuery)
		bq2, isQry2 := y.(*elastic.BoolQuery)
		if isQry1 && isQry2 {
			return apply(bq1, bq2), nil
		} else if isQry1 && isKv2 {
			return apply(bq1, kv2), nil
		} else if isQry2 && isKv1 {
			return apply(bq2, kv1), nil
		}

		nq1, isNq1 := x.(*nestedQuery)
		nq2, isNq2 := y.(*nestedQuery)
		if isNq1 && isNq2 {
			q := elastic.NewBoolQuery()
			return apply(q, nq1, nq2), nil
		} else if isNq1 && isKv2 {
			q := elastic.NewBoolQuery().Should(kv2)
			return apply(elastic.NewBoolQuery(), (*elastic.BoolQuery)(nq1), q), nil
		} else if isNq2 && isKv1 {
			q := elastic.NewBoolQuery().Should(kv1)
			return apply(elastic.NewBoolQuery(), (*elastic.BoolQuery)(nq2), q), nil
		} else if isNq1 && isQry2 {
			return apply(elastic.NewBoolQuery(), (*elastic.BoolQuery)(nq1), bq2), nil
		} else if isNq2 && isQry1 {
			return apply(elastic.NewBoolQuery(), (*elastic.BoolQuery)(nq2), bq1), nil
		}

		return nil, errors.New("both sides are not query")
	}
}

func evalShould(ctx *EsQueryContext, args ...interface{}) (interface{}, error) {
	return evalShouldOrMust(func(bq *elastic.BoolQuery, q ...elastic.Query) *elastic.BoolQuery {
		return bq.Should(q...)
	})(ctx, args...)
}

func evalMust(ctx *EsQueryContext, args ...interface{}) (interface{}, error) {
	return evalShouldOrMust(func(bq *elastic.BoolQuery, q ...elastic.Query) *elastic.BoolQuery {
		return bq.Must(q...)
	})(ctx, args...)
}

func evalEsNest(ctx *EsQueryContext, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Wrap(ErrEvalError, "invalid number of argument")
	}
	x := args[0]

	switch v := x.(type) {
	case *elastic.TermQuery:
		return (*nestedQuery)(elastic.NewBoolQuery().Must(v)), nil
	case *elastic.BoolQuery:
		return (*nestedQuery)(v), nil
	case *nestedQuery:
		return v, nil
	}
	return x, nil
}

func isElasticQuery(v interface{}) bool {
	_, ok := v.(elastic.Query)
	return ok
}

func isNestQuery(v interface{}) bool {
	_, ok := v.(*nestedQuery)
	return ok
}

package rulengie

import (
	"context"
	"log"

	"github.com/ccbhj/rulengine/expr"
	"github.com/pkg/errors"
)

type RuleEngine struct {
	workflows   map[string]Workflow
	parserCache map[string]*expr.ParseContext
	globalParam expr.SymbolTab
}

func NewRuleEngine(workflows []Workflow, globalParam expr.SymbolTab) *RuleEngine {
	wf := make(map[string]Workflow, len(workflows))
	for _, f := range workflows {
		wf[f.Name] = f
	}
	return &RuleEngine{
		workflows:   wf,
		parserCache: make(map[string]*expr.ParseContext, len(workflows)),
	}
}

func getParserKey(wfName, ruleName string) string {
	return wfName + ":" + ruleName
}

func (r *RuleEngine) ExecuteOneRule(ctx context.Context, workflow string, params expr.SymbolTab) (*RuleResult, error) {
	wf, in := r.workflows[workflow]
	if !in {
		return nil, errors.Errorf("workflow %s not found", workflow)
	}
	for _, rule := range wf.Rules {
		key := getParserKey(wf.Name, rule.Name)
		pctx, in := r.parserCache[key]
		if !in {
			p, err := expr.NewParseContext(rule.Expr)
			if err != nil {
				return nil, errors.Errorf("fail to compile rule expression, key=%s", key)
			}
			r.parserCache[key] = p
			pctx = p
		}

		p := make(expr.SymbolTab)
		p.Append(r.globalParam)
		p.Append(params)

		ectx := expr.NewEvalContext(p)
		result, err := pctx.ParseAndEval(ectx)
		if err != nil {
			log.Println("fail to ExecuteOneRule")
			continue
		}
		var matched bool
		switch v := result.(type) {
		case bool:
			matched = v
			break
		default:
			log.Println("rule not return a boolean")
			continue
		}

		if matched {
			return &RuleResult{
				WorkflowName: workflow,
				MatchedRule:  rule.Name,
				Event:        rule.SuccessEvent,
			}, nil
		}
	}
	return &RuleResult{
		WorkflowName: workflow,
		MatchedRule:  "",
		Event:        wf.DefaultEvent,
	}, nil
}

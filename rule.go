package rulengie

type (
	RuleType int

	Rule struct {
		Name         string   `json:"name"`
		SuccessEvent string   `json:"success_event"`
		ErrorMessage string   `json:"error_message"`
		RuleType     RuleType `json:"rule_type"`
		Expr         string   `json:"expr"`
	}

	Workflow struct {
		Name         string `json:"workflow"`
		DefaultEvent string `json:"default_event"`
		Rules        []Rule `json:"rules"`
	}

	RuleResult struct {
		WorkflowName string
		MatchedRule  string
		Event        string
	}
)

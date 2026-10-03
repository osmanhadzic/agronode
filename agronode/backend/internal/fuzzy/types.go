package fuzzy

type MembershipFunction struct {
    Name       string    `json:"name"`
    Sensor     string    `json:"sensor,omitempty"`
    Type       string    `json:"type"` // "triangle" | "trapezoid"
    Parameters []float64 `json:"parameters"`
}

type Condition struct {
    Sensor     string `json:"sensor"`
    Membership string `json:"membership"`
}

type Action struct {
    Type  string  `json:"type"`
    Value float64 `json:"value"`
}

type Rule struct {
    Name       string      `json:"name"`
    Conditions []Condition `json:"conditions"`
    Operator   string      `json:"operator,omitempty"` // AND | OR (default AND)
    Action     Action      `json:"action"`
}

type FuzzyTrigger struct {
    Name               string               `json:"name"`
    MembershipFunctions []MembershipFunction `json:"membershipFunctions"`
    Rules              []Rule               `json:"rules"`
}

// EvaluationResult contains detailed evaluation info for logging/debugging
type EvaluationResult struct {
    Input       map[string]float64            `json:"input"`
    Memberships map[string]map[string]float64 `json:"memberships"` // sensor -> membership name -> degree
    Rules       []RuleResult                  `json:"rules"`
}

type RuleResult struct {
    Name     string  `json:"name"`
    Strength float64 `json:"strength"`
    Action   Action  `json:"action"`
    Value    float64 `json:"value"`
}

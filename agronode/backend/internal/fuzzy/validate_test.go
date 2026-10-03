package fuzzy

import "testing"

func TestValidateFuzzyTrigger_ValidTriangle(t *testing.T) {
    trig := FuzzyTrigger{
        MembershipFunctions: []MembershipFunction{
            {Name: "low", Sensor: "temp", Type: "triangle", Parameters: []float64{0, 10, 20}},
        },
        Rules: []Rule{
            {Name: "r1", Conditions: []Condition{{Sensor: "temp", Membership: "low"}}, Operator: "AND", Action: Action{Type: "activate", Value: 1}},
        },
    }

    if err := ValidateFuzzyTrigger(trig); err != nil {
        t.Fatalf("expected valid trigger, got error: %v", err)
    }
}

func TestValidateFuzzyTrigger_InvalidMembershipParams(t *testing.T) {
    trig := FuzzyTrigger{
        MembershipFunctions: []MembershipFunction{
            {Name: "bad", Sensor: "temp", Type: "triangle", Parameters: []float64{10, 5, 0}},
        },
        Rules: []Rule{{Name: "r1", Conditions: []Condition{{Sensor: "temp", Membership: "bad"}}, Action: Action{Type: "x", Value: 1}}},
    }

    if err := ValidateFuzzyTrigger(trig); err == nil {
        t.Fatalf("expected error for invalid triangle parameters, got nil")
    }
}

func TestValidateFuzzyTrigger_UnknownMembershipInRule(t *testing.T) {
    trig := FuzzyTrigger{
        MembershipFunctions: []MembershipFunction{
            {Name: "low", Sensor: "temp", Type: "triangle", Parameters: []float64{0, 10, 20}},
        },
        Rules: []Rule{{Name: "r1", Conditions: []Condition{{Sensor: "temp", Membership: "unknown"}}, Action: Action{Type: "x", Value: 1}}},
    }

    if err := ValidateFuzzyTrigger(trig); err == nil {
        t.Fatalf("expected error for unknown membership in rule, got nil")
    }
}

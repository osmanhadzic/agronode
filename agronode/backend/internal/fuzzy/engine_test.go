package fuzzy

import (
    "testing"
)

func TestTriangleDegree(t *testing.T) {
    mf := MembershipFunction{Type: "triangle", Parameters: []float64{10, 20, 30}}

    cases := []struct{
        x float64
        want float64
    }{
        {5, 0},
        {10, 0},
        {15, 0.5},
        {20, 1},
        {25, 0.5},
        {30, 0},
        {35, 0},
    }

    for _, c := range cases {
        got, err := mf.Degree(c.x)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if (got - c.want) > 1e-6 || (c.want - got) > 1e-6 {
            t.Fatalf("triangle degree(%v) = %v, want %v", c.x, got, c.want)
        }
    }
}

func TestTrapezoidDegree(t *testing.T) {
    mf := MembershipFunction{Type: "trapezoid", Parameters: []float64{0, 10, 20, 30}}

    cases := []struct{
        x float64
        want float64
    }{
        {-5, 0},
        {0, 0},
        {5, 0.5},
        {10, 1},
        {15, 1},
        {25, 0.5},
        {30, 0},
        {35, 0},
    }

    for _, c := range cases {
        got, err := mf.Degree(c.x)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if (got - c.want) > 1e-6 || (c.want - got) > 1e-6 {
            t.Fatalf("trapezoid degree(%v) = %v, want %v", c.x, got, c.want)
        }
    }
}

func TestAndOrCombination(t *testing.T) {
    trig := FuzzyTrigger{
        MembershipFunctions: []MembershipFunction{
            {Sensor: "temp", Name: "hot", Type: "triangle", Parameters: []float64{20, 30, 40}},
            {Sensor: "hum", Name: "high", Type: "triangle", Parameters: []float64{60, 80, 90}},
        },
        Rules: []Rule{
            {Name: "both", Conditions: []Condition{{Sensor: "temp", Membership: "hot"}, {Sensor: "hum", Membership: "high"}}, Operator: "AND", Action: Action{Type: "set_fan", Value: 100}},
            {Name: "either", Conditions: []Condition{{Sensor: "temp", Membership: "hot"}, {Sensor: "hum", Membership: "high"}}, Operator: "OR", Action: Action{Type: "set_fan", Value: 100}},
        },
    }

    inputs := map[string]float64{"temp": 35, "hum": 70}
    res, err := EvaluateTrigger(trig, inputs)
    if err != nil {
        t.Fatalf("evaluate error: %v", err)
    }

    // temp: 35 -> hot degree = (40-35)/(40-30)=0.5
    // hum: 70 -> high degree = (70-60)/(80-60)=0.5

    var both, either float64
    for _, r := range res.Rules {
        if r.Name == "both" {
            both = r.Strength
        }
        if r.Name == "either" {
            either = r.Strength
        }
    }

    if both != 0.5 {
        t.Fatalf("AND strength = %v, want 0.5", both)
    }
    if either != 0.5 {
        t.Fatalf("OR strength = %v, want 0.5", either)
    }
}

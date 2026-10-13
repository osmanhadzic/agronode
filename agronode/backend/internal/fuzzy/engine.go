package fuzzy

import (
	"errors"
	"fmt"
	"math"
)

// Evaluate membership degree for a membership function given input x.
func (mf MembershipFunction) Degree(x float64) (float64, error) {
	switch mf.Type {
	case "triangle":
		return triangleDegree(x, mf.Parameters)
	case "trapezoid":
		return trapezoidDegree(x, mf.Parameters)
	case "left_shoulder":
		return leftShoulderDegree(x, mf.Parameters)
	case "right_shoulder":
		return rightShoulderDegree(x, mf.Parameters)
	case "gaussian":
		return gaussianDegree(x, mf.Parameters)
	case "bell":
		return bellDegree(x, mf.Parameters)
	case "sigmoid":
		return sigmoidDegree(x, mf.Parameters)
	default:
		return 0, errors.New("unsupported membership function type")
	}
}

func triangleDegree(x float64, p []float64) (float64, error) {
	if len(p) != 3 {
		return 0, errors.New("triangle requires 3 parameters")
	}
	a, b, c := p[0], p[1], p[2]
	if !(a <= b && b <= c) {
		return 0, errors.New("triangle parameters must satisfy a <= b <= c")
	}

	if x <= a || x >= c {
		if x == b && a == b && b == c {
			return 1.0, nil
		}
		return 0.0, nil
	}

	if x == b {
		return 1.0, nil
	}

	if x > a && x < b {
		return (x - a) / (b - a), nil
	}

	// x between b and c
	return (c - x) / (c - b), nil
}

func trapezoidDegree(x float64, p []float64) (float64, error) {
	if len(p) != 4 {
		return 0, errors.New("trapezoid requires 4 parameters")
	}
	a, b, c, d := p[0], p[1], p[2], p[3]
	if !(a <= b && b <= c && c <= d) {
		return 0, errors.New("trapezoid parameters must satisfy a <= b <= c <= d")
	}

	if x <= a || x >= d {
		return 0.0, nil
	}

	if x >= b && x <= c {
		return 1.0, nil
	}

	if x > a && x < b {
		return (x - a) / (b - a), nil
	}

	// x between c and d
	return (d - x) / (d - c), nil
}

// ValidateMembershipParams checks parameter count and constraints for a membership function type.
func ValidateMembershipParams(mfType string, p []float64) error {
	switch mfType {
	case "triangle":
		if len(p) != 3 {
			return errors.New("triangle requires 3 parameters")
		}
		if !(p[0] <= p[1] && p[1] <= p[2]) {
			return errors.New("triangle parameters must satisfy a<=b<=c")
		}
	case "trapezoid":
		if len(p) != 4 {
			return errors.New("trapezoid requires 4 parameters")
		}
		if !(p[0] <= p[1] && p[1] <= p[2] && p[2] <= p[3]) {
			return errors.New("trapezoid parameters must satisfy a<=b<=c<=d")
		}
	case "left_shoulder", "right_shoulder":
		if len(p) != 2 {
			return fmt.Errorf("%s requires 2 parameters", mfType)
		}
		if !(p[0] < p[1]) {
			return fmt.Errorf("%s parameters must satisfy a<b", mfType)
		}
	case "gaussian":
		if len(p) != 2 {
			return errors.New("gaussian requires 2 parameters (mean, sigma)")
		}
		if !(p[1] > 0) {
			return errors.New("gaussian sigma must be > 0")
		}
	case "bell":
		if len(p) != 3 {
			return errors.New("bell requires 3 parameters (width, slope, center)")
		}
		if !(p[0] > 0 && p[1] > 0) {
			return errors.New("bell width and slope must be > 0")
		}
	case "sigmoid":
		if len(p) != 2 {
			return errors.New("sigmoid requires 2 parameters (center, slope)")
		}
		if p[1] == 0 {
			return errors.New("sigmoid slope must not be 0")
		}
	default:
		return errors.New("unsupported membership function type")
	}
	return nil
}

func leftShoulderDegree(x float64, p []float64) (float64, error) {
	if err := ValidateMembershipParams("left_shoulder", p); err != nil {
		return 0, err
	}
	a, b := p[0], p[1]
	switch {
	case x <= a:
		return 1, nil
	case x >= b:
		return 0, nil
	}
	return (b - x) / (b - a), nil
}

func rightShoulderDegree(x float64, p []float64) (float64, error) {
	if err := ValidateMembershipParams("right_shoulder", p); err != nil {
		return 0, err
	}
	a, b := p[0], p[1]
	switch {
	case x <= a:
		return 0, nil
	case x >= b:
		return 1, nil
	}
	return (x - a) / (b - a), nil
}

func gaussianDegree(x float64, p []float64) (float64, error) {
	if err := ValidateMembershipParams("gaussian", p); err != nil {
		return 0, err
	}
	d := (x - p[0]) / p[1]
	return math.Exp(-0.5 * d * d), nil
}

func bellDegree(x float64, p []float64) (float64, error) {
	if err := ValidateMembershipParams("bell", p); err != nil {
		return 0, err
	}
	return 1 / (1 + math.Pow(math.Abs((x-p[2])/p[0]), 2*p[1])), nil
}

func sigmoidDegree(x float64, p []float64) (float64, error) {
	if err := ValidateMembershipParams("sigmoid", p); err != nil {
		return 0, err
	}
	return 1 / (1 + math.Exp(-p[1]*(x-p[0]))), nil
}

// EvaluateTrigger runs the fuzzy engine for a trigger and input values.
func EvaluateTrigger(trigger FuzzyTrigger, inputs map[string]float64) (EvaluationResult, error) {
	memberships := make(map[string]map[string]float64)

	// Evaluate all membership functions
	for _, mf := range trigger.MembershipFunctions {
		sensor := mf.Sensor
		if sensor == "" {
			// fallback: use trigger name as sensor — not ideal but keeps compatibility
			sensor = "default"
		}
		if _, ok := memberships[sensor]; !ok {
			memberships[sensor] = make(map[string]float64)
		}

		val, exists := inputs[sensor]
		if !exists {
			// no input for this sensor; membership degree is 0
			memberships[sensor][mf.Name] = 0.0
			continue
		}

		degree, err := mf.Degree(val)
		if err != nil {
			return EvaluationResult{}, err
		}
		// clamp
		if math.IsNaN(degree) || degree < 0 {
			degree = 0
		}
		if degree > 1 {
			degree = 1
		}

		memberships[sensor][mf.Name] = degree
	}

	// Evaluate rules
	var ruleResults []RuleResult
	for _, rule := range trigger.Rules {
		var degrees []float64
		for _, cond := range rule.Conditions {
			sensor := cond.Sensor
			if sensor == "" {
				sensor = "default"
			}
			deg := 0.0
			if sensorMemberships, ok := memberships[sensor]; ok {
				if v, ok2 := sensorMemberships[cond.Membership]; ok2 {
					deg = v
				}
			}
			degrees = append(degrees, deg)
		}

		operator := rule.Operator
		if operator == "" {
			operator = "AND"
		}

		strength := 0.0
		if len(degrees) == 0 {
			strength = 0.0
		} else if operator == "OR" {
			// OR = max
			strength = degrees[0]
			for _, d := range degrees[1:] {
				if d > strength {
					strength = d
				}
			}
		} else {
			// AND = min
			strength = degrees[0]
			for _, d := range degrees[1:] {
				if d < strength {
					strength = d
				}
			}
		}

		// compute action value: scale numeric action value by strength
		value := rule.Action.Value * strength

		ruleResults = append(ruleResults, RuleResult{
			Name:     rule.Name,
			Strength: strength,
			Action:   rule.Action,
			Value:    value,
		})
	}

	return EvaluationResult{
		Input:       inputs,
		Memberships: memberships,
		Rules:       ruleResults,
	}, nil
}

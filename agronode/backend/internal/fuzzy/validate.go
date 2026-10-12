package fuzzy

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var ErrInvalidFuzzy = errors.New("invalid fuzzy trigger")

// ValidateFuzzyTrigger performs structural validation on a FuzzyTrigger
func ValidateFuzzyTrigger(trigger FuzzyTrigger) error {
	if len(trigger.MembershipFunctions) == 0 {
		return fmt.Errorf("%w: no membership functions defined", ErrInvalidFuzzy)
	}

	// map membership name -> sensors where defined
	nameToSensors := make(map[string]struct{})

	for i, mf := range trigger.MembershipFunctions {
		if mf.Name == "" {
			return fmt.Errorf("%w: membership function #%d has empty name", ErrInvalidFuzzy, i)
		}
		if err := ValidateMembershipParams(mf.Type, mf.Parameters); err != nil {
			return fmt.Errorf("%w: membership %q: %v", ErrInvalidFuzzy, mf.Name, err)
		}

		nameToSensors[mf.Name] = struct{}{}
	}

	// validate rules
	for ri, rule := range trigger.Rules {
		if rule.Name == "" {
			return fmt.Errorf("%w: rule #%d has empty name", ErrInvalidFuzzy, ri)
		}
		op := rule.Operator
		if op == "" {
			op = "AND"
		}
		if op != "AND" && op != "OR" {
			return fmt.Errorf("%w: rule %q has invalid operator %q", ErrInvalidFuzzy, rule.Name, rule.Operator)
		}
		if len(rule.Conditions) == 0 {
			return fmt.Errorf("%w: rule %q has no conditions", ErrInvalidFuzzy, rule.Name)
		}
		for _, cond := range rule.Conditions {
			if cond.Membership == "" {
				return fmt.Errorf("%w: rule %q has condition with empty membership", ErrInvalidFuzzy, rule.Name)
			}
			// membership must exist
			if _, ok := nameToSensors[cond.Membership]; !ok {
				return fmt.Errorf("%w: rule %q references unknown membership %q", ErrInvalidFuzzy, rule.Name, cond.Membership)
			}
		}

		actionType := strings.TrimSpace(rule.Action.Type)
		if actionType == "http" {
			actionURL := strings.TrimSpace(rule.Action.URL)
			if actionURL == "" {
				return fmt.Errorf("%w: rule %q http action requires url", ErrInvalidFuzzy, rule.Name)
			}

			parsed, err := url.ParseRequestURI(actionURL)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				return fmt.Errorf("%w: rule %q http action has invalid url", ErrInvalidFuzzy, rule.Name)
			}

			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				return fmt.Errorf("%w: rule %q http action url must use http or https", ErrInvalidFuzzy, rule.Name)
			}
		}
	}

	return nil
}

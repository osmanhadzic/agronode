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
        if mf.Type != "triangle" && mf.Type != "trapezoid" {
            return fmt.Errorf("%w: membership %q has unsupported type %q", ErrInvalidFuzzy, mf.Name, mf.Type)
        }
        if mf.Type == "triangle" {
            if len(mf.Parameters) != 3 {
                return fmt.Errorf("%w: membership %q triangle requires 3 parameters", ErrInvalidFuzzy, mf.Name)
            }
            a, b, c := mf.Parameters[0], mf.Parameters[1], mf.Parameters[2]
            if !(a <= b && b <= c) {
                return fmt.Errorf("%w: membership %q triangle parameters must satisfy a<=b<=c", ErrInvalidFuzzy, mf.Name)
            }
        } else {
            if len(mf.Parameters) != 4 {
                return fmt.Errorf("%w: membership %q trapezoid requires 4 parameters", ErrInvalidFuzzy, mf.Name)
            }
            a, b, c, d := mf.Parameters[0], mf.Parameters[1], mf.Parameters[2], mf.Parameters[3]
            if !(a <= b && b <= c && c <= d) {
                return fmt.Errorf("%w: membership %q trapezoid parameters must satisfy a<=b<=c<=d", ErrInvalidFuzzy, mf.Name)
            }
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

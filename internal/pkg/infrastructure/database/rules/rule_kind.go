package rules

import "errors"

type kind int

const (
	UnknownRuleKind kind = iota
	kindV
	kindVS
	kindVB
)

var (
	ErrorNoKindSet       = errors.New("rule must have exactly one of V/VS/VB set (got none)")
	ErrorMultipleKindSet = errors.New("rule must have exactly one of V/VS/VB set (got multiple)")
)

func detectKind(r Rule) (kind, error) {
	hasV := r.RuleValues.V != nil &&
		(r.RuleValues.V.MinValue != nil || r.RuleValues.V.MaxValue != nil)

	hasVS := r.RuleValues.Vs != nil &&
		r.RuleValues.Vs.Value != nil

	hasVB := r.RuleValues.Vb != nil &&
		r.RuleValues.Vb.Value != nil

	counter := 0
	var k kind

	if hasV {
		counter++
		k = kindV
	}
	if hasVS {
		counter++
		k = kindVS
	}
	if hasVB {
		counter++
		k = kindVB
	}

	switch counter {
	case 0:
		return UnknownRuleKind, ErrorNoKindSet
	case 1:
		return k, nil
	default:
		return UnknownRuleKind, ErrorMultipleKindSet
	}
}

func NormalizedParams(r Rule) (vmin, vmax, vs, vb any, err error) {
	k, err := detectKind(r)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	switch k {
	case kindV:
		return returnVKind(r)

	case kindVS:
		return returnVSKind(r)

	case kindVB:
		return returnVBKind(r)

	default:
		return nil, nil, nil, nil, errors.New("unknown rule kind")
	}
}

func returnVBKind(r Rule) (any, any, any, any, error) {
	if r.RuleValues.Vb == nil || r.RuleValues.Vb.Value == nil {
		return nil, nil, nil, nil, errors.New("vb is set but value is nil")
	}
	return nil, nil, nil, *r.RuleValues.Vb.Value, nil
}

func returnVSKind(r Rule) (any, any, any, any, error) {
	if r.RuleValues.Vs == nil || r.RuleValues.Vs.Value == nil {
		return nil, nil, nil, nil, errors.New("vs is set but value is nil")
	}
	return nil, nil, *r.RuleValues.Vs.Value, nil, nil
}

func returnVKind(r Rule) (any, any, any, any, error) {
	if r.RuleValues.V == nil || (r.RuleValues.V.MinValue == nil && r.RuleValues.V.MaxValue == nil) {
		return nil, nil, nil, nil, errors.New("v is set but neither vmin nor vmax provided")
	}
	var minValue, maxValue any
	if r.RuleValues.V.MinValue != nil {
		minValue = *r.RuleValues.V.MinValue
	}
	if r.RuleValues.V.MaxValue != nil {
		maxValue = *r.RuleValues.V.MaxValue
	}
	return minValue, maxValue, nil, nil, nil
}

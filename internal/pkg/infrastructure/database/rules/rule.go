package rules

import (
	"errors"
)

type Rule struct {
	ID              string     `json:"id"`
	MeasurementID   string     `json:"measurement_id"`
	DeviceID        string     `json:"device_id"`
	MeasurementType int        `json:"measurement_type"`
	ShouldAbort     bool       `json:"should_abort"`
	RuleValues      RuleValues `json:"rule_values"`
}

type RuleValues struct {
	V  *RuleV  `json:"v"`
	Vs *RuleVs `json:"vs"`
	Vb *RuleVb `json:"vb"`
}

type RuleV struct {
	MinValue *float64 `json:"min_value"`
	MaxValue *float64 `json:"max_value"`
}

type RuleVs struct {
	Value *string `json:"value"`
}

type RuleVb struct {
	Value *bool `json:"value"`
}

var (
	ErrorNotFloatValue  = errors.New("v_min_value and v_min_value must be a valid float")
	ErrorVHasWrongOrder = errors.New("v_min_value must be lower than v_max_value")
)

func (rule *Rule) ValidatesRangedInteger() bool {
	return rule.RuleValues.V != nil && (rule.RuleValues.V.MinValue != nil || rule.RuleValues.V.MaxValue != nil)
}

func (rule *Rule) ValidatesStringValueExists() bool {
	return rule.RuleValues.Vs != nil && rule.RuleValues.Vs.Value != nil
}

func (rule *Rule) ValidatesBoolValueExists() bool {
	return rule.RuleValues.Vb != nil && rule.RuleValues.Vb.Value != nil
}

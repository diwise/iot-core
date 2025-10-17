package rules

import "errors"

var ErrRuleHasNoKind = errors.New("rule has no kind")

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

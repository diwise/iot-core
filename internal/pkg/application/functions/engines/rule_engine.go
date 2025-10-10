package engines

import (
	"context"
	"fmt"
	"math"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/repository"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
)

//go:generate moq -rm -out rule_engine_mock.go . RuleEngine

type RuleEngine interface {
	ValidateMessageReceived(ctx context.Context, msg events.MessageReceived) ([]RuleValidation, error)
	ValidateRecord(record senml.Record, rule rules.Rule) RuleValidation
}

type engine struct {
	repository repository.RuleRepository
}

func New(repository repository.RuleRepository) RuleEngine {
	if repository == nil {
		panic("New: repository is nil")
	}
	return &engine{repository: repository}
}

func (e *engine) ValidateRecord(record senml.Record, rule rules.Rule) RuleValidation {

	result := RuleValidation{
		MeasurementId:      rule.MeasurementID,
		DeviceId:           rule.DeviceID,
		ShouldAbort:        rule.ShouldAbort,
		IsValid:            true,
		ValidationMessages: []string{},
	}

	if validatesValueExists(record) == false {

		result.ValidationMessages = append(result.ValidationMessages, "No value found in record")
		result.IsValid = false

		return result
	}

	appliedRules := 0

	if validatesRangedInteger(rule) {
		appliedRules++

		if record.Value == nil {
			result.IsValid = false
			result.ValidationMessages = append(result.ValidationMessages, "V rule requires a numeric value but record.Value is nil")
		} else {
			val := deref(record.Value, math.NaN())

			hasMin := rule.RuleValues.V.MinValue != nil
			hasMax := rule.RuleValues.V.MaxValue != nil
			minValue := deref(rule.RuleValues.V.MinValue, math.Inf(-1))
			maxValue := deref(rule.RuleValues.V.MaxValue, math.Inf(+1))

			if hasMin && val < minValue {
				result.IsValid = false
				result.ValidationMessages = append(result.ValidationMessages, fmt.Sprintf("V value is too low. Min allowed: %g, got: %g", minValue, val))
			}
			if hasMax && val > maxValue {
				result.IsValid = false
				result.ValidationMessages = append(result.ValidationMessages, fmt.Sprintf("V value is too high. Max allowed: %g, got: %g", maxValue, val))
			}
		}
	}

	if validatesStringValueExists(rule) {
		appliedRules++

		expected := deref(rule.RuleValues.Vs.Value, "")
		actual := record.StringValue

		if actual == "" {
			result.IsValid = false
			result.ValidationMessages = append(result.ValidationMessages, "Vs rule requires a string value but record.StringValue is empty")
		} else if expected != actual {
			result.IsValid = false
			result.ValidationMessages = append(result.ValidationMessages, fmt.Sprintf("Vs value mismatch. Expected: '%q', got: '%q'", expected, actual))
		}
	}

	if validatesBoolValueExists(rule) {
		appliedRules++

		if record.BoolValue == nil {
			result.IsValid = false
			result.ValidationMessages = append(result.ValidationMessages, "Vb rule requires a boolean value but record.BoolValue is nil")
		} else {
			expected := deref(rule.RuleValues.Vb.Value, false)
			actual := deref(record.BoolValue, false)

			if expected != actual {
				result.IsValid = false
				result.ValidationMessages = append(result.ValidationMessages, fmt.Sprintf("Vb value mismatch. Expected: %t, got: %t", expected, actual))
			}
		}
	}

	if appliedRules != 1 {
		result.IsValid = false
		result.ValidationMessages = append(result.ValidationMessages, "Exactly one of (V, VS, Vb) must be set")
	}

	return result
}

func (e *engine) ValidateMessageReceived(ctx context.Context, msg events.MessageReceived) ([]RuleValidation, error) {

	pack := msg.Pack().Clone()

	err := pack.Validate()

	if err != nil {
		return nil, err
	}

	pack.Normalize()

	ruleList := e.repository.Get(ctx, msg.DeviceID())

	result := make([]RuleValidation, 0, len(ruleList))

	for _, rule := range ruleList {
		recordFinder := senml.FindByName(rule.MeasurementID)

		record, ok := pack.GetRecord(recordFinder)
		if ok {
			validatedRecord := e.ValidateRecord(record, rule)
			result = append(result, validatedRecord)
		}
	}

	return result, nil
}

func validatesRangedInteger(rule rules.Rule) bool {
	return rule.RuleValues.V != nil && (rule.RuleValues.V.MinValue != nil || rule.RuleValues.V.MaxValue != nil)
}

func validatesValueExists(record senml.Record) bool {
	return record.Value != nil || record.BoolValue != nil || record.StringValue != ""
}

func validatesStringValueExists(rule rules.Rule) bool {
	return rule.RuleValues.Vs != nil && rule.RuleValues.Vs.Value != nil
}

func validatesBoolValueExists(rule rules.Rule) bool {
	return rule.RuleValues.Vb != nil && rule.RuleValues.Vb.Value != nil
}

func deref[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

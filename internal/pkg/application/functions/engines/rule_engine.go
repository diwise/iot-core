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

	isValid := true
	validationMessageList := []string{}

	if record.Value == nil && record.BoolValue == nil && record.StringValue == "" {

		isValid = false
		validationMessageList = append(validationMessageList, "No value found in record")

		return RuleValidation{
			MeasurementId:      rule.MeasurementId,
			DeviceId:           rule.DeviceId,
			ShouldAbort:        rule.ShouldAbort,
			IsValid:            isValid,
			ValidationMessages: validationMessageList,
		}
	}

	appliedRules := 0

	if rule.RuleValues.V != nil && (rule.RuleValues.V.MinValue != nil || rule.RuleValues.V.MaxValue != nil) {
		appliedRules++

		if record.Value == nil {
			isValid = false
			validationMessageList = append(validationMessageList, "V rule requires a numeric value but record.Value is nil")
		} else {
			val := derefFloat64(record.Value, math.NaN())

			hasMin := rule.RuleValues.V.MinValue != nil
			hasMax := rule.RuleValues.V.MaxValue != nil
			min := derefFloat64(rule.RuleValues.V.MinValue, math.Inf(-1))
			max := derefFloat64(rule.RuleValues.V.MaxValue, math.Inf(+1))

			if hasMin && val < min {
				isValid = false
				validationMessageList = append(validationMessageList, fmt.Sprintf("V value is too low. Min allowed: %g, got: %g", min, val))
			}
			if hasMax && val > max {
				isValid = false
				validationMessageList = append(validationMessageList, fmt.Sprintf("V value is too high. Max allowed: %g, got: %g", max, val))
			}
		}
	}

	if rule.RuleValues.Vs != nil && rule.RuleValues.Vs.Value != nil {
		appliedRules++

		expected := derefString(rule.RuleValues.Vs.Value, "")
		actual := record.StringValue

		if expected != "" {
			if actual == "" {
				isValid = false
				validationMessageList = append(validationMessageList, "Vs rule requires a string value but record.StringValue is empty")
			} else if expected != actual {
				isValid = false
				validationMessageList = append(validationMessageList, fmt.Sprintf("Vs value mismatch. Expected: %q, got: %q", expected, actual))
			}
		}
	}

	if rule.RuleValues.Vb != nil && rule.RuleValues.Vb.Value != nil {
		appliedRules++

		if record.BoolValue == nil {
			isValid = false
			validationMessageList = append(validationMessageList, "Vb rule requires a boolean value but record.BoolValue is nil")
		} else {
			expected := derefBool(rule.RuleValues.Vb.Value, false)
			actual := derefBool(record.BoolValue, false)

			if expected != actual {
				isValid = false
				validationMessageList = append(validationMessageList, fmt.Sprintf("Vb value mismatch. Expected: %t, got: %t", expected, actual))
			}
		}
	}

	if appliedRules != 1 {
		isValid = false
		validationMessageList = append(validationMessageList, "Exactly one of (V, VS, Vb) must be set")
	}

	return RuleValidation{
		MeasurementId:      rule.MeasurementId,
		DeviceId:           rule.DeviceId,
		ShouldAbort:        rule.ShouldAbort,
		IsValid:            isValid,
		ValidationMessages: validationMessageList,
	}
}

func (e *engine) ValidateMessageReceived(ctx context.Context, msg events.MessageReceived) ([]RuleValidation, error) {

	result := []RuleValidation{}

	pack := msg.Pack().Clone()

	pack.Validate()
	pack.Normalize()

	ruleList, errList, err := e.repository.Get(ctx, msg.DeviceID())

	// Not found
	if ruleList == nil && errList == nil && err == nil {
		return []RuleValidation{}, nil
	}

	for _, rule := range ruleList {
		recordFinder := senml.FindByName(rule.MeasurementId)
		record, _ := pack.GetRecord(recordFinder)

		validateRecord := e.ValidateRecord(record, rule)

		result = append(result, validateRecord)
	}

	return result, nil
}

func derefBool(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func derefFloat64(p *float64, def float64) float64 {
	if p == nil {
		return def
	}
	return *p
}

func derefString(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

package engines

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/repository"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
)

//go:generate moq -rm -out rule_engine_mock.go . RuleEngine

type RuleEngine interface {
	ValidateMessageReceived(ctx context.Context, msg events.MessageReceived, logger *slog.Logger) ([]RuleValidation, error)
	ValidateRecord(record senml.Record, rule rules.Rule, logger *slog.Logger) (RuleValidation, error)
}

type engine struct {
	repository repository.RuleRepository
	logger     *slog.Logger
}

func NewEngine(repository repository.RuleRepository, logger *slog.Logger) RuleEngine {
	if repository == nil {
		panic("NewEngine: repository is nil")
	}
	if logger == nil {
		panic("NewEngine: logger is nil")
	}
	return &engine{repository: repository, logger: logger}
}

func (e *engine) ValidateRecord(record senml.Record, rule rules.Rule, logger *slog.Logger) (RuleValidation, error) {

	isValid := true
	errorList := []string{}

	if record.Value == nil && record.BoolValue == nil && record.StringValue == "" {
		isValid = false
		errorList = append(errorList, "No value found in record")
		return RuleValidation{
			MeasurementId: rule.MeasurementId,
			DeviceId:      rule.DeviceId,
			ShouldAbort:   rule.ShouldAbort,
			IsValid:       isValid,
			Errors:        errorList,
		}, nil
	}

	appliedRules := 0

	if rule.RuleValues.V != nil && (rule.RuleValues.V.MinValue != nil || rule.RuleValues.V.MaxValue != nil) {
		appliedRules++

		if record.Value == nil {
			isValid = false
			errorList = append(errorList, "V rule requires a numeric value but record.Value is nil")
		} else {
			val := derefFloat64(record.Value, math.NaN())

			hasMin := rule.RuleValues.V.MinValue != nil
			hasMax := rule.RuleValues.V.MaxValue != nil
			min := derefFloat64(rule.RuleValues.V.MinValue, math.Inf(-1))
			max := derefFloat64(rule.RuleValues.V.MaxValue, math.Inf(+1))

			if hasMin && val < min {
				isValid = false
				errorList = append(errorList, fmt.Sprintf("V value is too low. Min allowed: %g, got: %g", min, val))
			}
			if hasMax && val > max {
				isValid = false
				errorList = append(errorList, fmt.Sprintf("V value is too high. Max allowed: %g, got: %g", max, val))
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
				errorList = append(errorList, "Vs rule requires a string value but record.StringValue is empty")
			} else if expected != actual {
				isValid = false
				errorList = append(errorList, fmt.Sprintf("Vs value mismatch. Expected: %q, got: %q", expected, actual))
			}
		}
	}

	if rule.RuleValues.Vb != nil && rule.RuleValues.Vb.Value != nil {
		appliedRules++

		if record.BoolValue == nil {
			isValid = false
			errorList = append(errorList, "Vb rule requires a boolean value but record.BoolValue is nil")
		} else {
			expected := derefBool(rule.RuleValues.Vb.Value, false)
			actual := derefBool(record.BoolValue, false)

			if expected != actual {
				isValid = false
				errorList = append(errorList, fmt.Sprintf("Vb value mismatch. Expected: %t, got: %t", expected, actual))
			}
		}
	}

	if appliedRules != 1 {
		isValid = false
		errorList = append(errorList, "Exactly one of (V, VS, Vb) must be set")
	}

	return RuleValidation{
		MeasurementId: rule.MeasurementId,
		DeviceId:      rule.DeviceId,
		ShouldAbort:   rule.ShouldAbort,
		IsValid:       isValid,
		Errors:        errorList,
	}, nil
}

func (e *engine) ValidateMessageReceived(ctx context.Context, msg events.MessageReceived, logger *slog.Logger) ([]RuleValidation, error) {

	result := []RuleValidation{}

	pack := msg.Pack().Clone()

	pack.Validate()
	pack.Normalize()

	ruleList, errList, err := e.repository.Get(ctx, msg.DeviceID())

	// Not found
	if ruleList == nil && errList == nil && err == nil {
		return []RuleValidation{}, nil
	}

	for _, err := range errList {
		logger.Error("error getting rule", "device_id", msg.DeviceID(), "error", err)
	}

	for _, rule := range ruleList {
		recordFinder := senml.FindByName(rule.MeasurementId)
		record, _ := pack.GetRecord(recordFinder)

		validateRecord, err := e.ValidateRecord(record, rule, logger)
		if err != nil {

			return nil, err
		}

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

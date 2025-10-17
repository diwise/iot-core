package repository

import (
	"context"
	"errors"
	"math"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
)

//go:generate moq -rm -out rule_repository_mock.go . RuleRepository

type RuleRepository interface {
	Add(ctx context.Context, rule rules.Rule) error
	Get(ctx context.Context, id string) []rules.Rule
	Update(ctx context.Context, rule rules.Rule) error
	Delete(ctx context.Context, id string) error
}

type repository struct {
	storage rules.Storage
}

func NewRepository(storage rules.Storage) RuleRepository {
	return &repository{storage: storage}
}

func (e *repository) Add(ctx context.Context, rule rules.Rule) error {

	validateRuleErr := validateRule(rule)
	if validateRuleErr != nil {
		return validateRuleErr
	}

	return e.storage.Add(ctx, rule)
}

func (e *repository) Get(ctx context.Context, id string) []rules.Rule {
	result, _, _ := e.storage.Get(ctx, id)

	return result
}

func (e *repository) Update(ctx context.Context, rule rules.Rule) error {
	validateRuleErr := validateRule(rule)
	if validateRuleErr != nil {
		return validateRuleErr
	}

	return e.storage.Update(ctx, rule)
}

func (e *repository) Delete(ctx context.Context, id string) error {
	return e.storage.Delete(ctx, id)
}

func isValidFloat64(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

func validateRule(r rules.Rule) error {

	err := validateRuleValues(r)
	if err != nil {
		return err
	}

	if r.RuleValues.V == nil && r.RuleValues.Vs == nil && r.RuleValues.Vb == nil {
		return errors.New("No kinds. One of rule kind v, vs, vb must be set")
	}

	return nil
}

func validateRuleValues(r rules.Rule) error {
	if r.RuleValues.V != nil && r.RuleValues.V.MinValue != nil {
		if !isValidFloat64(*r.RuleValues.V.MinValue) {
			return errors.New("v_min_value must be a valid float")
		}
	}
	if r.RuleValues.V != nil && r.RuleValues.V.MaxValue != nil {
		if !isValidFloat64(*r.RuleValues.V.MaxValue) {
			return errors.New("v_max_value must be a valid float")
		}
	}
	if r.RuleValues.V != nil && r.RuleValues.V.MinValue != nil && r.RuleValues.V.MaxValue != nil {
		if *r.RuleValues.V.MinValue > *r.RuleValues.V.MaxValue {
			return errors.New("v_min_value must be <= v_max_value")
		}
	}

	return nil
}

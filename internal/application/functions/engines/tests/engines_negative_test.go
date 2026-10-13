package engine_test

import (
	"errors"
	"testing"

	"github.com/diwise/iot-core/internal/infrastructure/database/rules"
	prod "github.com/diwise/iot-core/internal/infrastructure/database/rules"
	rules_test "github.com/diwise/iot-core/internal/infrastructure/database/rules/tests"
	"github.com/matryer/is"
)

func TestAdd_Fails_WhenMultipleKindsSet(t *testing.T) {
	is := is.New(t)
	r, _ := resetDbAndStorage(t)

	in := rules_test.MakeRuleV(t, measurementId, deviceId, new(float64(3)), nil)
	in.RuleValues.Vs = &prod.RuleVs{Value: new("oops")}

	err := r.Add(t.Context(), in)

	is.True(err != nil) // expected error for multiple kinds, got nil
	is.True(errors.Is(err, rules.ErrorMultipleKindSet))
}

func TestAdd_Fails_WhenNoKindsSet(t *testing.T) {
	is := is.New(t)
	r, _ := resetDbAndStorage(t)

	in := rules_test.MakeRuleV(t, measurementId, deviceId, nil, nil)

	err := r.Add(t.Context(), in)

	is.True(errors.Is(err, rules.ErrorNoKindSet))
}

func TestInvalidRule_VMin_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, new(float64(30)), nil)

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

func TestInvalidRule_VMax_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, nil, new(float64(20)))

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

func TestInvalidRule_V_Min_Max_Mixed_Up_ReturnsError(t *testing.T) {
	is := is.New(t)
	r, _ := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, new(float64(30)), new(float64(20)))

	err := r.Add(t.Context(), in)
	is.True(err != nil) // expected error
	is.True(errors.Is(err, rules.ErrorVHasWrongOrder))
}

func TestInvalidRule_VS_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleVS(t, measurementId+msg.Pack()[2].Name, deviceId, new("wrong string"))

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

func TestInvalidRule_VB_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleVB(t, measurementId+msg.Pack()[3].Name, deviceId, new(false))

	r := newTestRepository()
	e := newTestEngine()

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

func TestValidationResults_ReturnsNonValid_WhenMeasurementIsMissingFromMessage(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleVS(t, measurementId+"missing-measurement", deviceId, new("w1e"))

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, err := e.ValidationResults(t.Context(), msg)
	is.NoErr(err)
	is.Equal(len(validations), 1)
	is.True(validations[0].IsValid == false)
	is.Equal(validations[0].MeasurementId, measurementId+"missing-measurement")
}

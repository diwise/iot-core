package engine_test

import (
	"testing"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	prod "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	rules_test "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules/tests"
	"github.com/matryer/is"
)

func TestAdd_Fails_WhenMultipleKindsSet(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	r := newTestRepository()
	in := rules_test.MakeRuleV(t, measurementId, deviceId, rules_test.F64(3), nil)
	in.RuleValues.Vs = &prod.RuleVs{Value: rules_test.S("oops")}

	err := r.Add(testCtx, in)

	is.True(err != nil) // expected error for multiple kinds, got nil
	is.Equal(err, rules.ErrorMultipleKindSet)
}

func TestAdd_Fails_WhenNoKindsSet(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	r := newTestRepository()
	in := rules_test.MakeRuleV(t, measurementId, deviceId, nil, nil)

	err := r.Add(testCtx, in)

	is.Equal(err, rules.ErrorNoKindSet)
}

func TestInvalidRule_VMin_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, rules_test.F64(30), nil)

	r := newTestRepository()
	e := newTestEngine()

	err := r.Add(testCtx, in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

func TestInvalidRule_VMax_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, nil, rules_test.F64(20))

	r := newTestRepository()
	e := newTestEngine()

	err := r.Add(testCtx, in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

func TestInvalidRule_V_Min_Max_Mixed_Up_ReturnsError(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, rules_test.F64(30), rules_test.F64(20))

	r := newTestRepository()

	err := r.Add(testCtx, in)
	is.True(err != nil) // expected error
	is.Equal(err, rules.ErrorVHasWrongOrder)
}

func TestInvalidRule_VS_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleVS(t, measurementId+msg.Pack()[2].Name, deviceId, rules_test.S("wrong string"))

	r := newTestRepository()
	e := newTestEngine()

	err := r.Add(testCtx, in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

func TestInvalidRule_VB_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules_test.MakeRuleVB(t, measurementId+msg.Pack()[3].Name, deviceId, rules_test.B(false))

	r := newTestRepository()
	e := newTestEngine()

	err := r.Add(testCtx, in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

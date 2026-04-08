package repository_test

import (
	"testing"

	rules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules/tests"
	"github.com/matryer/is"
)

func TestValidRule_V_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	in := rules.MakeRuleV(t, measurementId, deviceId, rules.F64(3), nil)

	r := newTestRepository()

	err := r.Add(t.Context(), in)

	is.NoErr(err)
}

func TestValidRule_VS_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	in := rules.MakeRuleVS(t, measurementId, deviceId, rules.S("test"))
	r := newTestRepository()
	err := r.Add(t.Context(), in)

	is.NoErr(err)
}

func TestValidRule_VB_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	in := rules.MakeRuleVB(t, measurementId, deviceId, rules.B(true))
	r := newTestRepository()
	err := r.Add(t.Context(), in)

	is.NoErr(err)
}

func TestGetRuleById_ReturnsRule(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	ruleID := "rule-vs-1"
	deviceID := "internal-id-for-device"

	in := rules.MakeRuleVS(t, ruleID, deviceID, rules.S("test"))
	r := newTestRepository()

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	got, err := r.GetRuleById(t.Context(), ruleID)
	is.NoErr(err)

	is.Equal(got.ID, in.ID)
	is.Equal(got.MeasurementID, in.MeasurementID)
	is.Equal(got.DeviceID, in.DeviceID)
	is.Equal(got.MeasurementType, in.MeasurementType)
	is.Equal(got.ShouldAbort, in.ShouldAbort)

	if got.RuleValues.Vs == nil || got.RuleValues.Vs.Value == nil {
		t.Fatalf("expected Vs rule value, got %#v", got.RuleValues)
	}
	AssertStringPtrEq(t, got.RuleValues.Vs.Value, in.RuleValues.Vs.Value)
}

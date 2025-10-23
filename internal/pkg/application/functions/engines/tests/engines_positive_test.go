package engine_test

import (
	"testing"

	rules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules/tests"
	"github.com/matryer/is"
)

func TestValidRule_VMin_ReturnsOk(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, rules.F64(3), nil)

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestValidRule_VMax_ReturnsOk(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, nil, rules.F64(30))

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestValidRule_V_ReturnsOk(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, rules.F64(20), rules.F64(30))

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestValidRule_VS_ReturnsOk(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleVS(t, measurementId+msg.Pack()[2].Name, deviceId, rules.S("w1e"))

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestValidRule_VB_ReturnsOk(t *testing.T) {
	is := is.New(t)
	r, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleVB(t, measurementId+msg.Pack()[3].Name, deviceId, rules.B(true))

	err := r.Add(t.Context(), in)
	is.NoErr(err)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestNoRule_ReturnsOk(t *testing.T) {
	is := is.New(t)
	_, e := resetDbAndStorage(t)

	msg := newMessageReceivedWithPacks(measurementId)

	validations, _ := e.ValidationResults(t.Context(), msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

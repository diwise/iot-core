package engine

import (
	"testing"

	rules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules/tests"
	"github.com/matryer/is"
)

func TestValidRule_VMin_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, rules.F64(3), nil)

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestValidRule_VMax_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, nil, rules.F64(30))

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestValidRule_V_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, rules.F64(20), rules.F64(30))

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestValidRule_VS_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleVS(t, measurementId+msg.Pack()[2].Name, deviceId, rules.S("w1e"))

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestValidRule_VB_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleVB(t, measurementId+msg.Pack()[3].Name, deviceId, rules.B(true))

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

func TestNoRule_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"

	msg := newMessageReceivedWithPacks(measurementId)

	e := newTestEngine()

	validations, _ := e.ValidateMessageReceived(testCtx, msg)

	for _, validation := range validations {
		is.True(validation.IsValid)
	}
}

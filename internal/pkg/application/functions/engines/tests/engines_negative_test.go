package engine

import (
	"strings"
	"testing"

	prod "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	rules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules/tests"
	"github.com/matryer/is"
)

func TestAdd_Fails_WhenMultipleKindsSet(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	r := newTestRepository()
	in := rules.MakeRuleV(t, measurementId, deviceId, rules.F64(3), nil)
	in.RuleValues.Vs = &prod.RuleVs{Value: rules.S("oops")}

	err := r.Add(testCtx, in)

	if err == nil {
		t.Fatalf("expected error for multiple kinds, got nil")
	}

	is.True(strings.Contains(err.Error(), "multiple"))
}

func TestAdd_Fails_WhenNoKindsSet(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	r := newTestRepository()
	in := rules.MakeRuleV(t, measurementId, deviceId, nil, nil)

	err := r.Add(testCtx, in)

	if err == nil {
		t.Fatalf("expected error for no kinds set, got nil")
	}

	is.True(strings.Contains(err.Error(), "No kinds"))
}

func TestInvalidRule_VMin_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, rules.F64(30), nil)

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg, log)

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
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, nil, rules.F64(20))

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg, log)

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
	in := rules.MakeRuleV(t, measurementId+msg.Pack()[1].Name, deviceId, rules.F64(30), rules.F64(20))

	r := newTestRepository()

	err := r.Add(testCtx, in)

	if err == nil {
		t.Fatalf("Test should fail due to mixed up min and max, got nil")
	}

	is.True(err != nil)
	is.True(err.Error() == "v_min_value must be <= v_max_value")
}

func TestInvalidRule_VS_ReturnsNonValid(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	msg := newMessageReceivedWithPacks(measurementId)
	in := rules.MakeRuleVS(t, measurementId+msg.Pack()[2].Name, deviceId, rules.S("wrong string"))

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg, log)

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
	in := rules.MakeRuleVB(t, measurementId+msg.Pack()[3].Name, deviceId, rules.B(false))

	r := newTestRepository()
	e := newTestEngine()

	if err := r.Add(testCtx, in); err != nil {
		t.Fatalf("Add(V): %v", err)
	}

	validations, _ := e.ValidateMessageReceived(testCtx, msg, log)

	for _, validation := range validations {
		is.True(validation.IsValid == false)
	}
}

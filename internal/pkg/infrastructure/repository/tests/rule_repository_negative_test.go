package repository

import (
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

	is.True(err != nil) // expected error for multiple kinds, got nil

	is.Equal(err.Error(), "rule must have exactly one of V/VS/VB set (got multiple)")
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

	is.True(err != nil) // expected error for no kinds set, got nil
	is.Equal(err.Error(), "No kinds. One of rule kind v, vs, vb must be set")
}

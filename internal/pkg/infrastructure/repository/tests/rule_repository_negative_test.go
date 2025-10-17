package repository_test

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

	is.NoErr(err) // expected error for multiple kinds, got nil

	is.Equal(err.Error(), "rule must have exactly one of V/VS/VB set (got multiple)")
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

	is.Equal(err, rules.ErrRuleHasNoKind)
}

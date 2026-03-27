package repository_test

import (
	"errors"
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

	err := r.Add(t.Context(), in)

	is.True(err != nil) // expected error for multiple kinds, got nil

	is.True(errors.Is(err, rules.ErrorMultipleKindSet))
}

func TestAdd_Fails_WhenNoKindsSet(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	r := newTestRepository()
	in := rules_test.MakeRuleV(t, measurementId, deviceId, nil, nil)

	err := r.Add(t.Context(), in)

	is.True(errors.Is(err, rules.ErrorNoKindSet))
}

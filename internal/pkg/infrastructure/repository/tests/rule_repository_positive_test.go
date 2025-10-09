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

	err := r.Add(testCtx, in)

	is.True(err == nil)
}

func TestValidRule_VS_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	in := rules.MakeRuleVS(t, measurementId, deviceId, rules.S("test"))
	r := newTestRepository()
	err := r.Add(testCtx, in)

	is.True(err == nil)
}

func TestValidRule_VB_ReturnsOk(t *testing.T) {
	is := is.New(t)
	requireDB(t)
	cleanDB(t)

	measurementId := "internal-id-for-device/3424/"
	deviceId := "internal-id-for-device"

	in := rules.MakeRuleVB(t, measurementId, deviceId, rules.B(true))
	r := newTestRepository()
	err := r.Add(testCtx, in)

	is.True(err == nil)
}

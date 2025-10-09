package rules_test

import (
	"testing"

	dbrules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	rules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules/tests"
	"github.com/matryer/is"
)

func Test_That_Rule_For_Vs_Verify(t *testing.T) {
	is := is.New(t)
	deviceId := "test"
	in := rules.MakeRuleVS(t, "r-vs", deviceId, rules.S("test"))
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	is.True(err == nil)                              // NormalizedParams
	is.True(vmin == nil && vmax == nil && vb == nil) // expected nil for vmin, vmax, vb

	s, ok := vs.(string)

	is.True(ok == true && s == "test") // expected vs=string("test")
}

func Test_That_Rule_For_V_Verify(t *testing.T) {
	is := is.New(t)
	deviceId := "test"
	in := rules.MakeRuleV(t, "r-vmax-vmin", deviceId, rules.F64(3), rules.F64(5))
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	is.True(err == nil)             // NormalizedParams
	is.True(vs == nil && vb == nil) // expected nil for vs, vb

	minValue, ok1 := vmin.(float64)
	maxValue, ok2 := vmax.(float64)

	is.True(ok1 == true) // expected vmin as float64
	is.True(ok2 == true) // expected vmax as float64

	is.True(minValue == 3 && maxValue == 5) // expected vmin=3 vmax=5
}

func Test_That_Rule_For_VMax_Verify(t *testing.T) {
	is := is.New(t)
	deviceId := "test"
	in := rules.MakeRuleV(t, "r-vmax", deviceId, nil, rules.F64(5))
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	is.True(err == nil)             // NormalizedParams
	is.True(vs == nil && vb == nil) // expected nil for vs, vb
	is.True(vmin == nil)            // expected nil for vmin

	maxValue, ok := vmax.(float64)

	is.True(ok == true && maxValue == 5) // expected vmax=5
}

func Test_That_Rule_For_VMin_Verify(t *testing.T) {
	is := is.New(t)
	deviceId := "test"
	in := rules.MakeRuleV(t, "r-vmin", deviceId, rules.F64(3), nil)
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	is.True(err == nil)             // NormalizedParams
	is.True(vs == nil && vb == nil) // expected nil for vs, vb
	is.True(vmax == nil)            // expected nil for vmax

	minValue, ok := vmin.(float64)

	is.True(ok == true && minValue == 3) // expected vmax=3
}

func Test_That_Rule_For_Vb_Verify(t *testing.T) {
	is := is.New(t)
	deviceId := "test"
	value := true
	in := rules.MakeRuleVB(t, "r-vb", deviceId, rules.B(value))
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	is.True(err == nil)                              // NormalizedParams
	is.True(vs == nil && vmin == nil && vmax == nil) // expected nil for vs,vmin,vmax

	b, ok := vb.(bool)
	is.True(ok == true && b == true) // expected vb=true (bool)
}

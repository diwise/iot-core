package rule_tests

import (
	"testing"

	dbrules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	"github.com/matryer/is"
)

func Test_That_Rule_For_V_Without_MinMax_Fails(t *testing.T) {
	is := is.New(t)
	in := MakeRuleV(t, "r-v-nil", "dev1", nil, nil)
	_, _, _, _, err := dbrules.NormalizedParamsAndValidate(in)

	is.True(err != nil) // // should return error for V rule with nil min/max
}

func Test_That_Rule_For_Vs_Without_Value_Fails(t *testing.T) {
	is := is.New(t)
	in := MakeRuleVS(t, "r-vs-nil", "dev1", nil)
	_, _, _, _, err := dbrules.NormalizedParamsAndValidate(in)

	is.True(err != nil) // should return error for VS rule with nil value
}

func Test_That_Rule_For_Vb_Without_Value_Fails(t *testing.T) {
	is := is.New(t)
	in := MakeRuleVB(t, "r-vb-nil", "dev1", nil)
	_, _, _, _, err := dbrules.NormalizedParamsAndValidate(in)

	is.True(err != nil) // should return error for VB rule with nil value
}

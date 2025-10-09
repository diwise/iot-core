package rules_test

import (
	"testing"

	dbrules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	rules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules/tests"
	"github.com/matryer/is"
)

func Test_That_Rule_For_V_Without_MinMax_Fails(t *testing.T) {
	is := is.New(t)
	in := rules.MakeRuleV(t, "r-v-nil", "dev1", nil, nil)
	_, _, _, _, err := dbrules.NormalizedParams(in)

	is.True(err != nil) // expected error for V rule with nil min/max, got nil on error
}

func Test_That_Rule_For_Vs_Without_Value_Fails(t *testing.T) {
	is := is.New(t)
	in := rules.MakeRuleVS(t, "r-vs-nil", "dev1", nil)
	_, _, _, _, err := dbrules.NormalizedParams(in)

	is.True(err != nil) // expected error for VS rule with nil value, got nil on error
}

func Test_That_Rule_For_Vb_Without_Value_Fails(t *testing.T) {
	is := is.New(t)
	in := rules.MakeRuleVB(t, "r-vb-nil", "dev1", nil)
	_, _, _, _, err := dbrules.NormalizedParams(in)

	is.True(err != nil) // expected error for VB rule with nil value, got nil on error
}

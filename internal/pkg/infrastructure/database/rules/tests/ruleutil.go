package rules

import (
	"testing"

	dbrules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
)

/** Helper functions to create rules of types **/
func MakeRuleBase(t *testing.T, measurementId, deviceId string) dbrules.Rule {
	t.Helper()
	return dbrules.Rule{
		Id:              measurementId,
		MeasurementId:   measurementId,
		DeviceId:        deviceId,
		MeasurementType: 1,
		ShouldAbort:     false,
		RuleValues:      dbrules.RuleValues{},
	}
}

func MakeRuleV(t *testing.T, measurementID, deviceID string, min, max *float64) dbrules.Rule {
	t.Helper()
	r := MakeRuleBase(t, measurementID, deviceID)

	if min != nil || max != nil {
		r.RuleValues.V = &dbrules.RuleV{
			MinValue: min,
			MaxValue: max,
		}
	}
	return r
}

func MakeRuleVS(t *testing.T, measurementID, deviceID string, s *string) dbrules.Rule {
	t.Helper()
	r := MakeRuleBase(t, measurementID, deviceID)

	if s != nil {
		r.RuleValues.Vs = &dbrules.RuleVs{Value: s}
	}
	return r
}

func MakeRuleVB(t *testing.T, measurementID, deviceID string, b *bool) dbrules.Rule {
	t.Helper()
	r := MakeRuleBase(t, measurementID, deviceID)

	if b != nil {
		r.RuleValues.Vb = &dbrules.RuleVb{Value: b}
	}
	return r
}

/** Creates basic types to pointer values **/
func F64(v float64) *float64 { return &v }
func S(v string) *string     { return &v }
func B(v bool) *bool         { return &v }

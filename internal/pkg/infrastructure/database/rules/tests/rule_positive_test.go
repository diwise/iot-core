package rules

import (
	"testing"

	dbrules "github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
)

func Test_That_Rule_For_Vs_Verify(t *testing.T) {
	deviceId := "test"
	in := MakeRuleVS(t, "r-vs", deviceId, S("test"))
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	if err != nil {
		t.Fatalf("NormalizedParams: %v", err)
	}

	if vmin != nil || vmax != nil || vb != nil {
		t.Fatalf("expected nil for vmin, vmax, vb; got %T=%v, %T=%v, %T=%v", vmin, vmin, vmax, vmax, vb, vb)
	}

	s, ok := vs.(string)
	if !ok || s != "test" {
		t.Fatalf("expected vs=string(\"test\"); got %T=%v", vs, vs)
	}
}

func Test_That_Rule_For_V_Verify(t *testing.T) {
	deviceId := "test"
	in := MakeRuleV(t, "r-vmax-vmin", deviceId, F64(3), F64(5))
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	if err != nil {
		t.Fatalf("NormalizedParams: %v", err)
	}

	if vs != nil || vb != nil {
		t.Fatalf("expected nil for vs, vb; got %T=%v, %T=%v", vs, vs, vb, vb)
	}

	min, ok1 := vmin.(float64)
	max, ok2 := vmax.(float64)
	if !ok1 || !ok2 {
		t.Fatalf("expected vmin,vmax as float64; got %T=%v, %T=%v", vmin, vmin, vmax, vmax)
	}
	if min != 3 || max != 5 {
		t.Fatalf("expected vmin=3 vmax=5; got vmin=%v vmax=%v", min, max)
	}
}

func Test_That_Rule_For_VMax_Verify(t *testing.T) {
	deviceId := "test"
	in := MakeRuleV(t, "r-vmax", deviceId, nil, F64(5))
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	if err != nil {
		t.Fatalf("NormalizedParams: %v", err)
	}

	if vs != nil || vb != nil {
		t.Fatalf("expected nil for vs, vb; got %T=%v, %T=%v", vs, vs, vb, vb)
	}

	if vmin != nil {
		t.Fatalf("expected nil for vmin; got %T=%v", vmin, vmin)
	}

	max, ok := vmax.(float64)
	if !ok || max != 5 {
		t.Fatalf("expected vmax=5 (float64); got %T=%v", vmax, vmax)
	}
}

func Test_That_Rule_For_VMin_Verify(t *testing.T) {
	deviceId := "test"
	in := MakeRuleV(t, "r-vmin", deviceId, F64(5), nil)
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	if err != nil {
		t.Fatalf("NormalizedParams: %v", err)
	}

	if vs != nil || vb != nil {
		t.Fatalf("expected nil for vs, vb; got %T=%v, %T=%v", vs, vs, vb, vb)
	}

	if vmax != nil {
		t.Fatalf("expected nil for vmax; got %T=%v", vmax, vmax)
	}

	min, ok := vmin.(float64)
	if !ok || min != 5 {
		t.Fatalf("expected vmin=5 (float64); got %T=%v", vmin, vmin)
	}
}

func Test_That_Rule_For_Vb_Verify(t *testing.T) {
	deviceId := "test"
	value := true
	in := MakeRuleVB(t, "r-vb", deviceId, B(value))
	vmin, vmax, vs, vb, err := dbrules.NormalizedParams(in)

	if err != nil {
		t.Fatalf("NormalizedParams: %v", err)
	}

	if vs != nil || vmin != nil || vmax != nil {
		t.Fatalf("expected nil for vs,vmin,vmax; got %T=%v, %T=%v, %T=%v", vs, vs, vmin, vmin, vmax, vmax)
	}

	b, ok := vb.(bool)
	if !ok || !b {
		t.Fatalf("expected vb=true (bool); got %T=%v", vb, vb)
	}
}

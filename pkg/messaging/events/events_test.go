package events

import (
	"testing"
	"time"

	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/senml"
	"github.com/matryer/is"
)

func TestGetValuesFromPack(t *testing.T) {
	is := testSetup(t)
	var v float64 = 1.0
	var b bool = true

	dt := time.Date(2022, time.January, 1, 12, 0, 0, 0, time.UTC)

	evt := NewMessageAccepted(senml.Pack{}, Rec("withValues", "str", &v, &b, float64(dt.Unix()), nil))

	b, ok := evt.Pack().GetBoolValue(senml.FindByName("withValues"))
	is.True(ok)
	v, ok = evt.Pack().GetValue(senml.FindByName("withValues"))
	is.True(ok)
	str, ok := evt.Pack().GetStringValue(senml.FindByName("withValues"))
	is.True(ok)
	date, ok := evt.Pack().GetTime(senml.FindByName("withValues"))
	is.True(ok)

	is.True(b)
	is.Equal(v, 1.0)
	is.Equal(str, "str")
	is.Equal(dt, date.UTC())
}

func TestMultipleTemperatures(t *testing.T) {
	is := testSetup(t)

	temp0 := lwm2m.NewTemperature("aaa-bbb-ccc/0", 10.0, time.Now())
	
	evt := NewMessageAccepted(lwm2m.ToPack(temp0))
	
	is.Equal("aaa-bbb-ccc", evt.DeviceID())
	is.Equal("3303", evt.ObjectID())
}

func testSetup(t *testing.T) *is.I {
	is := is.New(t)
	return is
}

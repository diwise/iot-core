package buildings

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestBuildingPower(t *testing.T) {
	is := is.New(t)

	b := New()

	b.Handle(context.Background(), newValue(lwm2m.Power, 22322.0), false, func(string, float64, time.Time) error { return nil })
	is.Equal(b.CurrentPower(), 22.322)
}

func TestBuildingEnergy(t *testing.T) {
	is := is.New(t)

	b := New()

	b.Handle(context.Background(), newValue(lwm2m.Energy, 3600.0), false, func(s string, f float64, ts time.Time) error { return nil })
	is.Equal(b.CurrentEnergy(), 0.001)
}

func newValue(baseName string, value float64) *events.MessageAccepted {
	e := &events.MessageAccepted{}
	json.Unmarshal([]byte(
		fmt.Sprintf(messageJSONFormat, baseName, value),
	), e)
	return e
}

const messageJSONFormat string = `{
	"sensorID":"sensorID",
	"pack":[
		{"bn":"%s","bt":1675801037,"n":"0","vs":"testId"},
		{"n":"5700","u":"m","v":%f}
	],
	"timestamp":"2023-02-07T20:17:17.312028Z"
}`

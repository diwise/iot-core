package buildings

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestBuildingPower(t *testing.T) {
	is := is.New(t)

	b := New()

	b.Handle(context.Background(), newValue(lwm2m.Power, 22322.0), func(s string, f float64) {})
	is.Equal(b.CurrentPower(), 22322.0)
}

func TestBuildingEnergy(t *testing.T) {
	is := is.New(t)

	b := New()

	b.Handle(context.Background(), newValue(lwm2m.Energy, 33333.0), func(s string, f float64) {})
	is.Equal(b.CurrentEnergy(), 33333.0)
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

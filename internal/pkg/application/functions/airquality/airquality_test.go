package airquality

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestAirQuality(t *testing.T) {
	is := is.New(t)

	aq := New()
	aq.Handle(context.Background(), newAirQuality(1.0, "2023-02-07T21:32:59.682607Z"), false, func(prop string, value float64, ts time.Time) error {
		return nil
	})

	is.Equal(aq.Temperature(), 1.0)
}

func newAirQuality(t float64, timestamp string) *events.MessageAccepted {
	e := &events.MessageAccepted{}
	json.Unmarshal([]byte(
		fmt.Sprintf(messageJSONFormat, t, t, t, t, t, t, t, timestamp),
	), e)
	return e
}

const messageJSONFormat string = `{
	"sensorID":"testId",
	"pack":[
		{"bn":"urn:oma:lwm2m:ext:3428","bt":1675805579,"n":"0","vs":"testId"},
		{"n":"5700","v":%f},
		{"n":"5","v":%f},
		{"n":"1","v":%f},
		{"n":"3","v":%f},
		{"n":"19","v":%f},
		{"n":"15","v":%f},
		{"n":"17","v":%f}
	],
	"timestamp":"%s"
}`

package timers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestTimer(t *testing.T) {
	is := is.New(t)

	tmr := New()
	tmr.Handle(context.Background(), newState(true, "2023-02-07T21:32:59.682607Z"), func(string, float64, time.Time) {})
	tmr.Handle(context.Background(), newState(false, "2023-02-07T23:32:59.682607Z"), func(string, float64, time.Time) {})

	is.True(tmr.State() == false)
}

func newState(on bool, timestamp string) *events.MessageAccepted {
	e := &events.MessageAccepted{}
	json.Unmarshal([]byte(
		fmt.Sprintf(messageJSONFormat, on, timestamp),
	), e)
	return e
}

const messageJSONFormat string = `{
	"sensorID":"testId",
	"pack":[
		{"bn":"urn:oma:lwm2m:ext:3200","bt":1675805579,"n":"0","vs":"testId"},
		{"n":"5500","vb":%t}
	],
	"timestamp":"%s"
}`

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
	tmr.Handle(context.Background(), newState(true, "2023-02-07T21:32:59.682607Z"), func(string, float64, time.Time) error { return nil })
	is.True(tmr.State() == true)
	tmr.Handle(context.Background(), newState(false, "2023-02-07T23:32:59.682607Z"), func(string, float64, time.Time) error { return nil })
	is.True(tmr.State() == false)

	is.Equal(tmr.(*timer).TotalDuration, 2*time.Hour)
}

func newState(on bool, timestamp string) *events.MessageAccepted {
	ts, _ := time.Parse(time.RFC3339Nano, timestamp)

	e := &events.MessageAccepted{}
	json.Unmarshal([]byte(
		fmt.Sprintf(messageJSONFormat, ts.Unix(), on, timestamp),
	), e)
	return e
}

const messageJSONFormat string = `{
	"sensorID":"testId",
	"pack":[
		{"bn":"testId","bt":%d,"n":"0","vs":"urn:oma:lwm2m:ext:3200"},
		{"n":"5500","vb":%t}
	],
	"timestamp":"%s"
}`

package stopwatch

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestStopwatch(t *testing.T) {
	is := is.New(t)

	sw := New()
	sw.Handle(context.Background(), newState(true, "2023-02-07T21:00:00.000000Z"), func(string, float64, time.Time) error { return nil })
	is.True(sw.State)
	is.True(sw.Count == 1)

	sw.Handle(context.Background(), newState(true, "2023-02-07T21:00:30.000000Z"), func(string, float64, time.Time) error { return nil })
	is.True(sw.State)
	is.True(sw.Count == 2)
	is.Equal(*sw.Duration, 30*time.Second)

	sw.Handle(context.Background(), newState(false, "2023-02-07T21:01:00.000000Z"), func(string, float64, time.Time) error { return nil })
	is.True(sw.State == false)
	is.True(sw.Count == 3)

	sw.Handle(context.Background(), newState(true, "2023-02-07T21:02:00.000000Z"), func(string, float64, time.Time) error { return nil })
	is.True(sw.State)
	is.True(sw.Count == 4)

	sw.Handle(context.Background(), newState(false, "2023-02-07T21:03:00.000000Z"), func(string, float64, time.Time) error { return nil })
	is.True(sw.State == false)
	is.True(sw.Count == 5)
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
	"pack":[
		{"bn":"testid/3200/","bt":%d,"n":"0","vs":"urn:oma:lwm2m:ext:3200"},
		{"n":"5500","vb":%t}
	],
	"timestamp":"%s"
}`

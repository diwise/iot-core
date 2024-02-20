package digitalinput

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestDigitalInputWithStateChange(t *testing.T) {
	is := is.New(t)

	di := New(0)
	_, err := di.Handle(context.Background(), newState(false, "2023-02-07T21:32:59.682607Z"), false, func(string, float64, time.Time) error { return nil })
	is.NoErr(err)

	b, err := di.Handle(context.Background(), newState(true, "2023-02-07T21:32:59.682607Z"), false, func(string, float64, time.Time) error { return nil })
	is.NoErr(err)
	is.True(b)

	is.True(di.State() == true)
}

func TestDigitalInputHandlesOnUpdate(t *testing.T) {
	is := is.New(t)

	di := New(0)

	b, err := di.Handle(context.Background(), newState(false, "2023-02-07T23:32:59.682607Z"), false, func(string, float64, time.Time) error { return nil })
	is.NoErr(err)
	is.True(!b) // should be true as both state and timestamp are set for the first time

	b, err = di.Handle(context.Background(), newState(false, "2023-02-14T23:32:59.682607Z"), true, func(string, float64, time.Time) error { return nil })
	is.NoErr(err)
	is.True(b) // should be true as timestamp has changed

	is.True(di.State() == false)
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

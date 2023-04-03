package counters

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestCounter(t *testing.T) {
	is := is.New(t)

	c := New()
	c.Handle(context.Background(), newState(true))

	is.Equal(c.Count(), 1) // Should have changed one time
	is.True(c.State())
}

func TestCounterDoubleOn(t *testing.T) {
	is := is.New(t)

	c := New()
	c.Handle(context.Background(), newState(true))
	c.Handle(context.Background(), newState(true))

	is.Equal(c.Count(), 1) // Should only have changed once
}

func TestCounterOnOffOn(t *testing.T) {
	is := is.New(t)

	c := New()
	c.Handle(context.Background(), newState(true))
	c.Handle(context.Background(), newState(false))
	c.Handle(context.Background(), newState(true))
	c.Handle(context.Background(), newState(false))

	is.Equal(c.Count(), 2) // Should have changed to on twice
	is.True(c.State() == false)
}

func newState(on bool) *events.MessageAccepted {
	e := &events.MessageAccepted{}
	json.Unmarshal([]byte(
		fmt.Sprintf(messageJSONFormat, on),
	), e)
	return e
}

const messageJSONFormat string = `{
	"sensorID":"testId",
	"pack":[
		{"bn":"urn:oma:lwm2m:ext:3200","bt":1675805579,"n":"0","vs":"testId"},
		{"n":"5500","vb":%t}
	],
	"timestamp":"2023-02-07T21:32:59.682607Z"
}`

package digitalinput

import (
	"context"
	"math"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "digitalinput"
)

type DigitalInput interface {
	Handle(context.Context, *events.MessageAccepted, bool, func(string, float64, time.Time) error) (bool, error)
	State() bool
}

func New(v float64) DigitalInput {
	return &digitalinput{
		State_: (math.Abs(v) >= 0.001),
	}
}

type digitalinput struct {
	Timestamp string `json:"timestamp"`
	State_    bool   `json:"state"`
}

func (t *digitalinput) Handle(ctx context.Context, e *events.MessageAccepted, onupdate bool, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {

	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	r, stateOk := e.GetRecord(DigitalInputState)

	ts := e.Timestamp

	stateValue := map[bool]float64{true: 1, false: 0}

	hasChanged := false

	if stateOk && r.BoolValue != nil {
		if t.State_ != *r.BoolValue || onupdate {
			hasChanged = true
		}

		timestamp, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return hasChanged, err
		}

		t.Timestamp = ts

		err = onchange("state", stateValue[t.State_], timestamp)
		if err != nil {
			return hasChanged, err
		}
	}

	return hasChanged, nil
}

func (t *digitalinput) State() bool {
	return t.State_
}

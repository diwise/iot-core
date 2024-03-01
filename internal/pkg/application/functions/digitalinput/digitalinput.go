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
	Handle(context.Context, *events.MessageAccepted, func(string, float64, time.Time) error) (bool, error)
	State() bool
}

func New(v float64) DigitalInput {
	return &digitalinput{
		State_: (math.Abs(v) >= 0.001),
	}
}

type digitalinput struct {
	Timestamp time.Time `json:"timestamp"`
	State_    bool      `json:"state"`
}

func (t *digitalinput) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {

	if !events.ObjectURNMatches(e, lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	r, stateOk := events.GetRecord(e, DigitalInputState)

	ts := e.Timestamp()

	stateValue := map[bool]float64{true: 1, false: 0}
	hasChanged := false

	if stateOk && r.BoolValue != nil {

		if t.State_ != *r.BoolValue {
			hasChanged = true
			t.State_ = *r.BoolValue

			err := onchange("state", stateValue[t.State_], e.Timestamp())
			if err != nil {
				return hasChanged, err
			}
		}

		if t.Timestamp != ts {
			hasChanged = true
			t.Timestamp = ts

			err := onchange("timestamp", 1, e.Timestamp())
			if err != nil {
				return hasChanged, err
			}
		}
	}

	return hasChanged, nil
}
func (t *digitalinput) State() bool {
	return t.State_
}

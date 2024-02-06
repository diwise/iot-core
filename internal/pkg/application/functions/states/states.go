package states

import (
	"context"
	"math"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "state"
)

type State interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64, time.Time) error) (bool, error)
	State() bool
}

func New(v float64) State {
	return &state{
		State_: (math.Abs(v) >= 0.001),
	}
}

type state struct {
	State_ bool `json:"state"`
}

func (t *state) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {

	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	r, stateOk := e.GetRecord(DigitalInputState)
	ts, timeOk := e.GetTimeForRec(DigitalInputState)

	if stateOk && timeOk && r.BoolValue != nil {
		state := *r.BoolValue

		if state != t.State_ {
			t.State_ = state
			stateValue := map[bool]float64{true: 1, false: 0}

			// Temporary fix to create square waves in the UI ...
			err := onchange("state", stateValue[!t.State_], ts)
			if err != nil {
				return true, err
			}

			err = onchange("state", stateValue[t.State_], ts)
			if err != nil {
				return true, err
			}

			return true, nil
		}
	}

	return false, nil
}

func (t *state) State() bool {
	return t.State_
}

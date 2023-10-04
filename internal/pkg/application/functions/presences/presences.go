package presences

import (
	"context"
	"math"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "presence"
)

type Presence interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64, time.Time) error) (bool, any, error)
	State() bool
}

func New(v float64) Presence {
	return &presence{
		State_: (math.Abs(v) >= 0.001),
	}
}

type presence struct {
	State_ bool `json:"state"`
}

func (t *presence) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, any, error) {

	if !e.BaseNameMatches(lwm2m.DigitalInput) && !e.BaseNameMatches(lwm2m.Presence) {
		return false, nil, nil
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
			presenceValue := map[bool]float64{true: 1, false: 0}

			// Temporary fix to create square waves in the UI ...
			err := onchange("presence", presenceValue[!t.State_], ts)
			if err != nil {
				return true, t, err
			}

			err = onchange("presence", presenceValue[t.State_], ts)
			if err != nil {
				return true, t, err
			}

			return true, t, nil
		}
	}

	return false, nil, nil
}

func (t *presence) State() bool {
	return t.State_
}

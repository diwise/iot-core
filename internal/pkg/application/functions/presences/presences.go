package presences

import (
	"context"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "presence"
)

type Presence interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64, time.Time) error) (bool, error)
	State() bool
}

func New() Presence {
	return &presence{}
}

type presence struct {
	State_ bool `json:"state"`
}

func (t *presence) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {

	if !e.BaseNameMatches(lwm2m.DigitalInput) && !e.BaseNameMatches(lwm2m.Presence) {
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
			presenceValue := map[bool]float64{true: 1, false: 0}

			// Temporary fix to create square waves in the UI ...
			err := onchange("presence", presenceValue[!t.State_], ts)
			if err != nil {
				return true, err
			}

			err = onchange("presence", presenceValue[t.State_], ts)
			if err != nil {
				return true, err
			}

			return true, nil
		}
	}

	return false, nil
}

func (t *presence) State() bool {
	return t.State_
}

package digitalinput

import (
	"context"
	"math"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
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
	if !events.Matches(*e, lwm2m.DigitalInput) {
		return false, events.ErrNoMatch
	}

	const (
		DigitalInputState string = "5500"
	)

	stateValue := map[bool]float64{true: 1, false: 0}
	hasChanged := false

	vb, ok := e.Pack.GetBoolValue(senml.FindByName(DigitalInputState))

	ts := e.Timestamp // TODO: time from pack not message?

	if ok {
		if t.State_ != vb {
			hasChanged = true
			t.State_ = vb

			err := onchange("state", stateValue[t.State_], ts)
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

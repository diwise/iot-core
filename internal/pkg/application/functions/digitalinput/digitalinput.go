package digitalinput

import (
	"context"
	"errors"
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
	Counter_  int       `json:"counter"`
}

func (t *digitalinput) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	if !events.Matches(e, lwm2m.DigitalInput) {
		return false, events.ErrNoMatch
	}

	const (
		DigitalInputState   string = "5500"
		DigitalInputCounter string = "5501"
	)

	stateValue := map[bool]float64{true: 1, false: 0}
	stateChanged := false
	counterChanged := false

	vb, vbOk := e.Pack().GetBoolValue(senml.FindByName(DigitalInputState))
	v, vOk := e.Pack().GetValue(senml.FindByName(DigitalInputCounter))
	ts, tsOk := e.Pack().GetTime(senml.FindByName(DigitalInputState))

	if !tsOk {
		ts = e.Timestamp
	}

	if vbOk {
		if t.State_ != vb {
			t.State_ = vb
			if vb {
				t.Counter_++
			}
			stateChanged = true
		}
	}

	if vOk {
		if t.Counter_ != int(v) {
			t.Counter_ = int(v)
			counterChanged = true
		}
	}

	var errs []error

	if stateChanged || counterChanged {
		if stateChanged {
			errs = append(errs, onchange("state", stateValue[t.State_], ts))
		}
		if counterChanged {
			errs = append(errs, onchange("counter", float64(t.Counter_), ts))
		}
		t.Timestamp = ts
	}

	return stateChanged || counterChanged, errors.Join(errs...)
}

func (t *digitalinput) State() bool {
	return t.State_
}

package features

import (
	"context"
	"math"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

type Registry interface {
	Find(ctx context.Context, sensorID string) ([]Feature, error)
}

type Feature interface {
	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
}

func NewRegistry() (Registry, error) {
	return &reg{}, nil
}

type reg struct{}

func (r *reg) Find(ctx context.Context, sensorID string) ([]Feature, error) {
	c := NewCounter("featureID", "overflow")
	return []Feature{c}, nil
}

type feat struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	SubType string `json:"subtype"`

	Counter *counter `json:"counter,omitempty"`
}

func (f *feat) Handle(ctx context.Context, e *events.MessageAccepted, msgctx messaging.MsgContext) error {
	if f.Counter != nil {
		changed, err := f.Counter.Handle(ctx, e)
		if err != nil {
			return err
		}

		if changed {
			msgctx.PublishOnTopic(ctx, f)
		}
	}

	return nil
}

func (f *feat) ContentType() string {
	return "application/json"
}

func (f *feat) TopicName() string {
	return "features.counters.updated"
}

func NewCounter(featureID, counterType string) Feature {

	f := &feat{
		ID:      featureID,
		Type:    "counter",
		SubType: counterType,
		Counter: &counter{
			Count: 0,
			State: false,
		},
	}

	return f
}

type counter struct {
	Count int  `json:"count"`
	State bool `json:"state"`
}

func (c *counter) Handle(ctx context.Context, e *events.MessageAccepted) (bool, error) {
	const (
		DigitalInputState   string = "5500"
		DigitalInputCounter string = "5501"
	)

	previousCount := c.Count

	count, countOk := e.GetFloat64(DigitalInputCounter)
	state, stateOk := e.GetBool(DigitalInputState)

	if countOk {
		c.Count = int(math.Ceil(count))
		if stateOk {
			c.State = state
		}
	} else if stateOk {
		if state != c.State {
			c.Count++
			c.State = state
		}
	}

	if previousCount != c.Count {
		return true, nil
	}

	return false, nil
}

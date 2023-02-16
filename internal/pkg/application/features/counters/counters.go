package counters

import (
	"context"
	"math"

	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FeatureTypeName string = "counter"
)

type Counter interface {
	Handle(ctx context.Context, e *events.MessageAccepted) (bool, error)
	Count() int
	State() bool
}

func New() Counter {

	c := &counter{
		Count_: 0,
		State_: false,
	}

	return c
}

type counter struct {
	Count_ int  `json:"count"`
	State_ bool `json:"state"`
}

func (c *counter) Handle(ctx context.Context, e *events.MessageAccepted) (bool, error) {
	const (
		DigitalInputState   string = "5500"
		DigitalInputCounter string = "5501"
	)

	previousCount := c.Count_
	previousState := c.State_

	count, countOk := e.GetFloat64(DigitalInputCounter)
	state, stateOk := e.GetBool(DigitalInputState)

	if countOk {
		c.Count_ = int(math.Ceil(count))
		if stateOk {
			c.State_ = state
		}
	} else if stateOk {
		if state != c.State_ {
			if state {
				c.Count_++
			}
			c.State_ = state
		}
	}

	return (previousCount != c.Count_ || previousState != c.State_), nil
}

func (c *counter) Count() int {
	return c.Count_
}

func (c *counter) State() bool {
	return c.State_
}

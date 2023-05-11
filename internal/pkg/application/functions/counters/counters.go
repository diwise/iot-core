package counters

import (
	"context"
	"math"

	"github.com/diwise/iot-core/internal/pkg/application/functions/metadata"
	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "counter"
)

type Counter interface {
	Handle(context.Context, *events.MessageAccepted, func(prop string, value float64)) (bool, error)
	Metadata() metadata.Metadata

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

func (c *counter) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64)) (bool, error) {
	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

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

	changed := false

	if previousCount != c.Count_ {
		onchange("count", float64(c.Count_))
		changed = true
	}

	if previousState != c.State_ {
		stateValue := map[bool]float64{true: 1.0, false: 0.0}
		onchange("state", stateValue[c.State_])
		changed = true
	}

	return changed, nil
}

func (c *counter) Metadata() metadata.Metadata {
	return metadata.Metadata{}
}

func (c *counter) Count() int {
	return c.Count_
}

func (c *counter) State() bool {
	return c.State_
}

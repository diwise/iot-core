package counters

import (
	"context"
	"math"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "counter"
)

type Counter interface {
	Handle(context.Context, *events.MessageAccepted, func(prop string, value float64, ts time.Time)) (bool, error)
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

func (c *counter) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time)) (bool, error) {
	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState   string = "5500"
		DigitalInputCounter string = "5501"
	)

	previousCount := c.Count_
	previousState := c.State_

	countRec, countOk := e.GetRecord(DigitalInputCounter)
	stateRec, stateOk := e.GetRecord(DigitalInputState)

	if countOk {
		count := *countRec.Value
		state := *stateRec.BoolValue

		c.Count_ = int(math.Ceil(count))
		if stateOk {
			c.State_ = state
		}
	} else if stateOk {
		state := *stateRec.BoolValue

		if state != c.State_ {
			if state {
				c.Count_++
			}
			c.State_ = state
		}
	}

	changed := false

	if previousCount != c.Count_ {
		onchange("count", float64(c.Count_), time.Unix(int64(countRec.Time), 0).UTC())
		changed = true
	}

	if previousState != c.State_ {
		stateValue := map[bool]float64{true: 1.0, false: 0.0}
		onchange("state", stateValue[c.State_], time.Unix(int64(stateRec.Time), 0).UTC())
		changed = true
	}

	return changed, nil
}

func (c *counter) Count() int {
	return c.Count_
}

func (c *counter) State() bool {
	return c.State_
}

package stopwatch

import (
	"context"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const FunctionTypeName = "stopwatch"

type Stopwatch interface {
	Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error)
	State() bool
	Count() int32
}

func New() Stopwatch {
	return &stopwatch{
		StartTime:      time.Time{},
		CumulativeTime: 0,
	}
}

type stopwatch struct {
	StartTime time.Time      `json:"startTime"`
	StopTime  *time.Time     `json:"stopTime,omitempty"`
	Duration  *time.Duration `json:"duration,omitempty"`

	State_ bool  `json:"state"`
	Count_ int32 `json:"count"`

	CumulativeTime time.Duration `json:"cumulativeTime"`
}

func (sw *stopwatch) State() bool {
	return sw.State_
}

func (sw *stopwatch) Count() int32 {
	return sw.Count_
}

func (sw *stopwatch) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState   string = "5500"
		DigitalInputCounter string = "5501"
	)

	currentState := sw.State_
	currentCount := sw.Count_

	r, stateOK := e.GetRecord(DigitalInputState)
	c, counterOK := e.GetFloat64(DigitalInputCounter)
	ts, timeOk := e.GetTimeForRec(DigitalInputState)

	if stateOK && timeOk && r.BoolValue != nil {
		state := *r.BoolValue

		if state != currentState {
			var err error

			if state {
				sw.StartTime = ts
				sw.State_ = state
				sw.StopTime = nil // setting end time and duration to nil values to ensure we don't send out the wrong ones later
				sw.Duration = nil

				err = onchange("state", 0, ts)
				if err != nil {
					return true, err
				}

				err = onchange("state", 1, ts)
				if err != nil {
					return true, err
				}
			} else {
				sw.StopTime = &ts
				sw.State_ = state
				duration := ts.Sub(sw.StartTime)
				sw.Duration = &duration
				sw.CumulativeTime = sw.CumulativeTime + duration

				err = onchange("state", 1, ts)
				if err != nil {
					return true, err
				}

				err = onchange("state", 0, ts)
				if err != nil {
					return true, err
				}

				dt := duration.Seconds()
				err = onchange("duration", dt, ts)
				if err != nil {
					return true, err
				}

				ct := sw.CumulativeTime.Seconds()
				err = onchange("cumulativeTime", ct, ts)
				if err != nil {
					return true, err
				}
			}
		} else if currentState {
			duration := ts.Sub(sw.StartTime)
			sw.Duration = &duration

			dt := duration.Seconds()
			err := onchange("duration", dt, ts)
			if err != nil {
				return true, err
			}
		}

		if counterOK {
			if int32(c) != currentCount {
				sw.Count_ = int32(c)
			}
		} else {
			sw.Count_++
		}

		err := onchange("count", float64(sw.Count_), ts)
		if err != nil {
			return true, err
		}
	}

	return currentState != sw.State_, nil
}

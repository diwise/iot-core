package stopwatch

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
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
	var err error
	var stateChanged bool = false

	if !events.ObjectURNMatches(e, lwm2m.DigitalInput) {
		return false, nil
	}

	log := logging.GetFromContext(ctx)

	const (
		DigitalInputState   string = "5500"
		DigitalInputCounter string = "5501"
	)

	r, stateOK := events.GetRecord(e, DigitalInputState)
	c, counterOK := events.GetFloat(e, DigitalInputCounter)
	ts, timeOk := events.GetTime(e, DigitalInputState)

	if !stateOK || !timeOk || r.BoolValue == nil {
		return false, fmt.Errorf("no state or time for stopwatch")
	}

	currentState := sw.State_
	currentCount := sw.Count_

	state := *r.BoolValue

	// On
	if state {
		// Off -> On = Start new stopwatch
		if !currentState {
			log.Debug("stopwatch: Off -> On, start new stopwatch")

			sw.StartTime = ts
			sw.State_ = state
			sw.StopTime = nil // setting end time and duration to nil values to ensure we don't send out the wrong ones later
			sw.Duration = nil

			err = onchange("state", 0, ts)
			if err != nil {
				return false, err
			}

			err = onchange("state", 1, ts)
			if err != nil {
				return false, err
			}

			stateChanged = true
		}

		// On -> On = Update duration
		if currentState {
			log.Debug("stopwatch: On -> On, update duration")

			duration := ts.Sub(sw.StartTime)
			sw.Duration = &duration

			dt := duration.Seconds()
			err := onchange("duration", dt, ts)
			if err != nil {
				return false, err
			}

			stateChanged = true
		}
	}

	// Off
	if !state {
		// On -> Off = Stop stopwatch
		if currentState {
			log.Debug("stopwatch: On -> Off, stop stopwatch")

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
				return false, err
			}

			dt := duration.Seconds()
			err = onchange("duration", dt, ts)
			if err != nil {
				return false, err
			}

			ct := sw.CumulativeTime.Seconds()
			err = onchange("cumulativeTime", ct, ts)
			if err != nil {
				return false, err
			}

			stateChanged = true
		}

		// Off -> Off = Do nothing
		if !currentState {
			log.Debug("Off -> Off, do nothing")
			return false, nil
		}
	}

	if counterOK {
		if int32(c) != currentCount {
			sw.Count_ = int32(c)
		}
	} else {
		sw.Count_++
	}

	err = onchange("count", float64(sw.Count_), ts)
	if err != nil {
		return false, err
	}

	return stateChanged, nil
}

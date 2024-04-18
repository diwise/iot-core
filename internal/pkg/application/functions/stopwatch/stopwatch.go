package stopwatch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const FunctionTypeName = "stopwatch"

type Stopwatch interface {
	Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error)
}

func New() *StopwatchImpl {
	return &StopwatchImpl{
		StartTime:      time.Time{},
		CumulativeTime: 0,
	}
}

type StopwatchImpl struct {
	StartTime time.Time      `json:"startTime"`
	StopTime  *time.Time     `json:"stopTime,omitempty"`
	Duration  *time.Duration `json:"duration,omitempty"`

	State bool  `json:"state"`
	Count int32 `json:"count"`

	CumulativeTime time.Duration `json:"cumulativeTime"`
}

func (sw *StopwatchImpl) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	var err error
	var stateChanged bool = false

	if !events.Matches(*e, lwm2m.DigitalInput) {
		return false, events.ErrNoMatch
	}

	log := logging.GetFromContext(ctx).With(slog.String("fnct", "stopwatch"))

	const (
		DigitalInputState   string = "5500"
		DigitalInputCounter string = "5501"
	)

	r, stateOK := e.Pack.GetRecord(senml.FindByName(DigitalInputState))
	ts, timeOk := r.GetTime()

	c, counterOK := e.Pack.GetValue(senml.FindByName(DigitalInputCounter))

	if !stateOK || !timeOk || r.BoolValue == nil {
		return false, fmt.Errorf("no state or time for stopwatch")
	}

	if ts.IsZero() {
		log.Warn("timestamp was Zero")
		ts = time.Now().UTC()
	}

	currentState := sw.State
	currentCount := sw.Count

	state := *r.BoolValue

	// On
	if state {
		// Off -> On = Start new stopwatch
		if !currentState {
			log.Debug("Off -> On, start new stopwatch")

			sw.StartTime = ts.UTC()
			sw.State = true
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
			log.Debug("On -> On, update duration")

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
			log.Debug("On -> Off, stop stopwatch")

			sw.StopTime = &ts
			sw.State = false
			duration := ts.Sub(sw.StartTime)
			sw.Duration = &duration
			sw.CumulativeTime = sw.CumulativeTime + duration

			err = onchange("state", 1, ts)
			if err != nil {
				return false, err
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
			sw.Count = int32(c)
		}
	} else {
		sw.Count++
	}

	err = onchange("count", float64(sw.Count), ts)
	if err != nil {
		return false, err
	}

	return stateChanged, nil
}

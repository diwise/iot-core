package stopwatch

import (
	"context"
	"encoding/json"
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

	State() bool
	Count() int32
	Duration() *time.Duration
	CumulativeTime() time.Duration
}

func New() Stopwatch {
	return &stopwatchImpl{
		StartTime:      time.Time{},
		CumulativeTime_: 0,
	}
}

type stopwatchImpl struct {
	StartTime time.Time      `json:"startTime"`
	StopTime  *time.Time     `json:"stopTime,omitempty"`
	Duration_  *time.Duration `json:"duration,omitempty"`

	State_ bool  `json:"state"`
	Count_ int32 `json:"count"`

	CumulativeTime_ time.Duration `json:"cumulativeTime"`
}

func (sw *stopwatchImpl) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	var err error
	var stateChanged bool = false

	if !events.Matches(e, lwm2m.DigitalInput) {
		return false, events.ErrNoMatch
	}

	log := logging.GetFromContext(ctx).With(slog.String("fnct", "stopwatch"))

	const (
		DigitalInputState   string = "5500"
		DigitalInputCounter string = "5501"
	)

	r, stateOK := e.Pack().GetRecord(senml.FindByName(DigitalInputState))
	ts, timeOk := r.GetTime()

	c, counterOK := e.Pack().GetValue(senml.FindByName(DigitalInputCounter))

	if !stateOK || !timeOk || r.BoolValue == nil {
		return false, fmt.Errorf("no state or time for stopwatch")
	}

	if ts.IsZero() {
		log.Warn("timestamp was Zero")
		ts = time.Now().UTC()
	}

	storedState, _ := json.Marshal(sw)
	log.Debug("handling stopwatch", slog.String("loaded_state", string(storedState)), slog.String("incoming_body", string(e.Body())))

	currentState := sw.State_
	currentCount := sw.Count_

	state := *r.BoolValue

	// On
	if state {
		// Off -> On = Start new stopwatch
		if !currentState {
			log.Debug("Off -> On, start new stopwatch")

			sw.StartTime = ts.UTC()
			sw.State_ = true
			sw.StopTime = nil // setting end time and duration to nil values to ensure we don't send out the wrong ones later
			sw.Duration_ = nil

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
			sw.Duration_ = &duration

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
			sw.State_ = false
			duration := ts.Sub(sw.StartTime)
			sw.Duration_ = &duration
			sw.CumulativeTime_ = sw.CumulativeTime_ + duration

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

			ct := sw.CumulativeTime_.Seconds()
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

	exitState, _ := json.Marshal(sw)
	log.Debug("handling stopwatch", slog.String("new_state", string(exitState)))

	return stateChanged, nil
}

func (sw *stopwatchImpl) CumulativeTime() time.Duration {
	return sw.CumulativeTime_
}

func (sw *stopwatchImpl) Count() int32  {
	return sw.Count_
}

func (sw *stopwatchImpl) State() bool {
	return sw.State_
}

func (sw *stopwatchImpl) Duration() *time.Duration {
	return sw.Duration_
}
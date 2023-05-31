package timers

import (
	"context"
	"errors"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const FunctionTypeName string = "timer"

type Timer interface {
	Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error)

	State() bool
}

func New() Timer {
	return &timer{
		StartTime: time.Time{},
	}
}

type timer struct {
	StartTime time.Time      `json:"startTime"`
	EndTime   *time.Time     `json:"endTime,omitempty"`
	Duration  *time.Duration `json:"duration,omitempty"`
	State_    bool           `json:"state"`

	totalDuration time.Duration
	valueUpdater  *time.Ticker
}

func (t *timer) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	previousState := t.State_

	r, stateOK := e.GetRecord(DigitalInputState)
	ts, timeOk := e.GetTimeForRec(DigitalInputState)

	errs := make([]error, 0)

	if stateOK && timeOk && r.BoolValue != nil {
		state := *r.BoolValue

		if state != previousState {
			if state {
				errs = append(errs, onchange("state", 0, ts))
				errs = append(errs, onchange("state", 1, ts))

				t.StartTime = ts
				t.State_ = state

				t.EndTime = nil // setting end time and duration to nil values to ensure we don't send out the wrong ones later
				t.Duration = nil

				errs = append(errs, onchange("time", t.totalDuration.Minutes(), ts))

				if t.valueUpdater == nil {
					t.valueUpdater = time.NewTicker(1 * time.Minute)
					go func() {
						for range t.valueUpdater.C {
							if t.State_ {
								now := time.Now().UTC()
								duration := t.totalDuration + now.Sub(t.StartTime)
								errs = append(errs, onchange("time", duration.Minutes(), now))
							}
						}
					}()
				}
			} else {
				errs = append(errs, onchange("state", 1, ts))
				errs = append(errs, onchange("state", 0, ts))

				t.EndTime = &ts
				t.State_ = state

				duration := t.EndTime.Sub(t.StartTime)
				t.Duration = &duration
				t.totalDuration = t.totalDuration + duration

				errs = append(errs, onchange("time", t.totalDuration.Minutes(), ts))
			}
		}
	}

	return previousState != t.State_, errors.Join(errs...)
}

func (t *timer) State() bool {
	return t.State_
}

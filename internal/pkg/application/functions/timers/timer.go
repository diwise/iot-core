package timers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const FunctionTypeName string = "timer"

type Timer interface {
	Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64) error) (bool, error)

	State() bool
}

func New() Timer {

	t := &timer{
		StartTime: time.Time{},
	}

	return t
}

type timer struct {
	StartTime time.Time      `json:"startTime"`
	EndTime   *time.Time     `json:"endTime,omitempty"`
	Duration  *time.Duration `json:"duration,omitempty"`
	State_    bool           `json:"state"`

	totalDuration time.Duration
	valueUpdater  *time.Ticker
}

func (t *timer) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64) error) (bool, error) {
	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	previousState := t.State_

	state, stateOK := e.GetBool(DigitalInputState)

	errs := make([]error, 0)

	if stateOK {
		if state != previousState {
			if state {
				errs = append(errs, onchange("state", 0))
				errs = append(errs, onchange("state", 1))

				start, err := time.Parse(time.RFC3339, e.Timestamp)
				if err != nil {
					return false, fmt.Errorf("failed to parse time from event timestamp: %s", err.Error())
				}

				t.StartTime = start
				t.State_ = state

				t.EndTime = nil // setting end time and duration to nil values to ensure we don't send out the wrong ones later
				t.Duration = nil

				errs = append(errs, onchange("time", t.totalDuration.Minutes()))

				if t.valueUpdater == nil {
					t.valueUpdater = time.NewTicker(1 * time.Minute)
					go func() {
						for range t.valueUpdater.C {
							if t.State_ {
								duration := t.totalDuration + time.Now().UTC().Sub(t.StartTime)
								errs = append(errs, onchange("time", duration.Minutes()))
							}
						}
					}()
				}
			} else {
				errs = append(errs, onchange("state", 1))
				errs = append(errs, onchange("state", 0))

				end, err := time.Parse(time.RFC3339, e.Timestamp)
				if err != nil {
					return false, fmt.Errorf("failed to parse time from event timestamp: %s", err.Error())
				}

				t.EndTime = &end
				t.State_ = state

				duration := t.EndTime.Sub(t.StartTime)
				t.Duration = &duration
				t.totalDuration = t.totalDuration + duration

				errs = append(errs, onchange("time", t.totalDuration.Minutes()))
			}
		}
	}

	return previousState != t.State_, errors.Join(errs...)
}

func (t *timer) State() bool {
	return t.State_
}

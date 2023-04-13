package timers

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const FeatureTypeName string = "timer"

type Timer interface {
	Handle(ctx context.Context, e *events.MessageAccepted) (bool, error)

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
}

func (t *timer) Handle(ctx context.Context, e *events.MessageAccepted) (bool, error) {
	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	previousState := t.State_

	state, stateOK := e.GetBool(DigitalInputState)

	if stateOK {
		if state != previousState && state {
			start, err := time.Parse(time.RFC3339, e.Timestamp)
			if err != nil {
				return false, fmt.Errorf("failed to parse time from event timestamp: %s", err.Error())
			}

			t.StartTime = start
			t.State_ = state

			t.EndTime = nil // setting end time and duration to nil values to ensure we don't send out the wrong ones later
			t.Duration = nil

		} else if state != previousState && !state {
			end, err := time.Parse(time.RFC3339, e.Timestamp)
			if err != nil {
				return false, fmt.Errorf("failed to parse time from event timestamp: %s", err.Error())
			}

			t.EndTime = &end
			t.State_ = state

			duration := t.EndTime.Sub(t.StartTime)
			t.Duration = &duration
		}

	}

	return previousState != t.State_, nil
}

func (t *timer) State() bool {
	return t.State_
}

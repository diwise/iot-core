package timers

import (
	"context"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const FunctionTypeName string = "timer"

type Timer interface {
	Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time)) (bool, error)

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

func (t *timer) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time)) (bool, error) {
	if !e.BaseNameMatches(lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	previousState := t.State_

	r, stateOK := e.GetRecord(DigitalInputState)

	if stateOK {
		state := *r.BoolValue
		ts, _ := e.GetTimeForRec(DigitalInputState)

		if state != previousState {
			if state {
				onchange("state", 0, ts)
				onchange("state", 1, ts)

				start := ts

				t.StartTime = start
				t.State_ = state

				t.EndTime = nil // setting end time and duration to nil values to ensure we don't send out the wrong ones later
				t.Duration = nil

				onchange("time", t.totalDuration.Minutes(), ts)

				if t.valueUpdater == nil {
					t.valueUpdater = time.NewTicker(1 * time.Minute)
					go func() {
						for range t.valueUpdater.C {
							if t.State_ {
								duration := t.totalDuration + time.Now().UTC().Sub(t.StartTime)
								onchange("time", duration.Minutes(), time.Now().UTC())
							}
						}
					}()
				}

			} else {
				onchange("state", 1, ts)
				onchange("state", 0, ts)

				end := ts

				t.EndTime = &end
				t.State_ = state

				duration := t.EndTime.Sub(t.StartTime)
				t.Duration = &duration
				t.totalDuration = t.totalDuration + duration

				onchange("time", t.totalDuration.Minutes(), ts)
			}
		}
	}

	return previousState != t.State_, nil
}

func (t *timer) State() bool {
	return t.State_
}

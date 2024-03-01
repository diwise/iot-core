package timers

import (
	"context"
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
	if !events.Matches(e, lwm2m.DigitalInput) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	previousState := t.State_

	r, stateOK := events.GetRecord(e, DigitalInputState)
	ts, timeOk := events.GetTime(e, DigitalInputState)

	if stateOK && timeOk && r.BoolValue != nil {
		state := *r.BoolValue

		if state != previousState {
			var err error

			if state {
				t.StartTime = ts
				t.State_ = state

				t.EndTime = nil // setting end time and duration to nil values to ensure we don't send out the wrong ones later
				t.Duration = nil

				err = onchange("state", 0, ts)
				if err != nil {
					return true, err
				}

				err = onchange("state", 1, ts)
				if err != nil {
					return true, err
				}

				err = onchange("time", t.totalDuration.Minutes(), ts)
				if err != nil {
					return true, err
				}

				if t.valueUpdater == nil {
					t.valueUpdater = time.NewTicker(1 * time.Minute)
					go func() error {
						for range t.valueUpdater.C {
							if t.State_ {
								now := time.Now().UTC()
								duration := t.totalDuration + now.Sub(t.StartTime)
								err = onchange("time", duration.Minutes(), now)
								if err != nil {
									t.valueUpdater.Stop()
									return err
								}
							}
						}
						return nil
					}()
				}
			} else {
				err = onchange("state", 1, ts)
				if err != nil {
					return true, err
				}

				err = onchange("state", 0, ts)
				if err != nil {
					return true, err
				}

				t.EndTime = &ts
				t.State_ = state

				duration := t.EndTime.Sub(t.StartTime)
				t.Duration = &duration
				t.totalDuration = t.totalDuration + duration

				err = onchange("time", t.totalDuration.Minutes(), ts)
				if err != nil {
					return true, err
				}
			}
		}
	}

	return previousState != t.State_, nil
}

func (t *timer) State() bool {
	return t.State_
}

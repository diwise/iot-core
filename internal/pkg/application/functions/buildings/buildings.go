package buildings

import (
	"context"
	"math"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
)

const (
	FunctionTypeName string = "building"
)

type Building interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64, time.Time) error) (bool, error)

	CurrentPower() float64
	CurrentEnergy() float64
}

func New() Building {
	return &building{}
}

type building struct {
	Energy float64 `json:"energy"`
	Power  float64 `json:"power"`
}

func (b *building) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	if !events.Matches(*e, lwm2m.Power) && !events.Matches(*e, lwm2m.Energy) {
		return false, events.ErrNoMatch
	}

	const SensorValue string = "5700"
	r, ok := e.Pack.GetRecord(senml.FindByName(SensorValue))
	ts, timeOk := e.Pack.GetTime(senml.FindByName(SensorValue))

	if ok && timeOk && r.Value != nil {
		value := *r.Value

		if events.Matches(*e, lwm2m.Power) {
			previousValue := b.Power
			value = value / 1000.0 // convert from Watt to kW
			b.Power = value

			if hasChanged(previousValue, value) {
				err := onchange("power", value, ts)
				return true, err
			}
		} else if events.Matches(*e, lwm2m.Energy) {
			previousValue := b.Energy
			value = value / 3600000.0 // convert from Joule to kWh
			b.Energy = value

			if hasChanged(previousValue, value) {
				err := onchange("energy", value, ts)
				return true, err
			}
		}
	}

	return false, nil
}

func hasChanged(previousLevel, newLevel float64) bool {
	return isNotZero(newLevel - previousLevel)
}

func isNotZero(value float64) bool {
	return (math.Abs(value) >= 0.001)
}

func (b *building) CurrentPower() float64 {
	return b.Power
}

func (b *building) CurrentEnergy() float64 {
	return b.Energy
}

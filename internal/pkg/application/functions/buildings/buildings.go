package buildings

import (
	"context"
	"math"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "building"
)

type Building interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64)) (bool, error)

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

func (b *building) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64)) (bool, error) {
	if !e.BaseNameMatches(lwm2m.Power) && !e.BaseNameMatches(lwm2m.Energy) {
		return false, nil
	}

	const SensorValue string = "5700"
	value, ok := e.GetFloat64(SensorValue)

	if ok {
		if e.BaseNameMatches(lwm2m.Power) {
			previousValue := b.Power
			b.Power = value

			if hasChanged(previousValue, value) {
				onchange("power", value)
				return true, nil
			}
		} else if e.BaseNameMatches(lwm2m.Energy) {
			previousValue := b.Energy
			b.Energy = value

			if hasChanged(previousValue, value) {
				onchange("energy", value)
				return true, nil
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

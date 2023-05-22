package waterqualities

import (
	"context"
	"math"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "waterquality"
)

type WaterQuality interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64) error) (bool, error)
}

func New() WaterQuality {
	return &waterquality{}
}

type waterquality struct {
	Temperature float64 `json:"temperature"`
}

func (wq *waterquality) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64) error) (bool, error) {

	if !e.BaseNameMatches(lwm2m.Temperature) {
		return false, nil
	}

	const SensorValue string = "5700"
	temp, tempOk := e.GetFloat64(SensorValue)

	if tempOk {
		temp = math.Round(temp*10) / 10

		oldTemp := wq.Temperature
		wq.Temperature = temp

		if hasChanged(oldTemp, temp) {
			err := onchange("temperature", temp)
			return true, err
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

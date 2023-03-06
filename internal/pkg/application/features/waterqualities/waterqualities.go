package waterqualities

import (
	"context"
	"math"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FeatureTypeName string = "waterquality"
)

type WaterQuality interface {
	Handle(ctx context.Context, e *events.MessageAccepted) (bool, error)
}

func New() WaterQuality {
	return &waterquality{}
}

type waterquality struct {
	Temperature float64 `json:"temperature"`
}

func (wq *waterquality) Handle(ctx context.Context, e *events.MessageAccepted) (bool, error) {

	if !e.BaseNameMatches(lwm2m.Temperature) {
		return false, nil
	}

	const SensorValue string = "5700"
	temp, tempOk := e.GetFloat64(SensorValue)

	if tempOk {
		temp = math.Round(temp*10) / 10

		oldTemp := wq.Temperature
		wq.Temperature = temp
		return hasChanged(oldTemp, temp), nil
	}

	return false, nil
}

func hasChanged(previousLevel, newLevel float64) bool {
	return isNotZero(newLevel - previousLevel)
}

func isNotZero(value float64) bool {
	return (math.Abs(value) >= 0.001)
}

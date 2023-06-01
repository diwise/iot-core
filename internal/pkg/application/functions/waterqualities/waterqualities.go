package waterqualities

import (
	"context"
	"math"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "waterquality"
)

type WaterQuality interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64, time.Time)) (bool, error)
}

func New() WaterQuality {
	return &waterquality{}
}

type waterquality struct {
	Temperature float64 `json:"temperature"`
}

func (wq *waterquality) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time)) (bool, error) {

	if !e.BaseNameMatches(lwm2m.Temperature) {
		return false, nil
	}

	const SensorValue string = "5700"
	r, tempOk := e.GetRecord(SensorValue)
	t, timeOk := e.GetTimeForRec(SensorValue)

	if tempOk && timeOk {
		temp := math.Round(*r.Value*10) / 10
		oldTemp := wq.Temperature
		wq.Temperature = temp
		if hasChanged(oldTemp, temp) {
			onchange("temperature", temp, t)
			return true, nil
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

package waterqualities

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
)

const (
	FunctionTypeName string = "waterquality"
)

type WaterQuality interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64, time.Time) error) (bool, error)
}

func New() WaterQuality {
	return &waterquality{}
}

type waterquality struct {
	Temperature float64   `json:"temperature"`
	Timestamp   time.Time `json:"timestamp"`
}

func (wq *waterquality) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	if !events.Matches(e, lwm2m.Temperature) {
		return false, events.ErrNoMatch
	}

	const SensorValue string = "5700"

	r, tempOk := e.Pack().GetRecord(senml.FindByName(SensorValue))
	ts, timeOk := e.Pack().GetTime(senml.FindByName(SensorValue))

	if ts.After(time.Now().Add(5 * time.Second)) {
		return false, fmt.Errorf("invalid timestamp %s in waterquality pack: %w", ts.Format(time.RFC3339), events.ErrBadTimestamp)
	}

	if tempOk && timeOk && r.Value != nil && ts.After(wq.Timestamp) {
		temp := math.Round(*r.Value*10) / 10
		oldTemp := wq.Temperature

		wq.Temperature = temp
		wq.Timestamp = ts

		if hasChanged(oldTemp, temp) {
			err := onchange("temperature", temp, ts)
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

package buildings

import (
	"context"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "building"
)

type Building interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64)) (bool, error)
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

	//const SensorValue string = "5700"

	return false, nil
}

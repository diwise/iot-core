package presences

import (
	"context"

	"github.com/diwise/iot-core/internal/pkg/application/functions/metadata"
	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "presence"
)

type Presence interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64)) (bool, error)
	Metadata() metadata.Metadata

	State() bool
}

func New() Presence {
	return &presence{}
}

type presence struct {
	State_ bool `json:"state"`
}

func (t *presence) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64)) (bool, error) {

	if !e.BaseNameMatches(lwm2m.DigitalInput) && !e.BaseNameMatches(lwm2m.Presence) {
		return false, nil
	}

	const (
		DigitalInputState string = "5500"
	)

	state, stateOk := e.GetBool(DigitalInputState)

	if stateOk && state != t.State_ {
		t.State_ = state
		presenceValue := map[bool]float64{true: 1, false: 0}
		// Temporary fix to create square waves in the UI ...
		onchange("presence", presenceValue[!t.State_])
		onchange("presence", presenceValue[t.State_])
		return true, nil
	}

	return false, nil
}

func (t *presence) State() bool {
	return t.State_
}

func (p *presence) Metadata() metadata.Metadata {
	return metadata.Metadata{}
}

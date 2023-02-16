package presences

import (
	"context"

	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FeatureTypeName string = "presence"
)

type Presence interface {
	Handle(ctx context.Context, e *events.MessageAccepted) (bool, error)
	State() bool
}

func New() Presence {
	return &presence{}
}

type presence struct {
	State_ bool `json:"state"`
}

func (t *presence) Handle(ctx context.Context, e *events.MessageAccepted) (bool, error) {
	const (
		DigitalInputState string = "5500"
	)

	state, stateOk := e.GetBool(DigitalInputState)

	if stateOk && state != t.State_ {
		t.State_ = state
		return true, nil
	}

	return false, nil
}

func (t *presence) State() bool {
	return t.State_
}

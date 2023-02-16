package features

import (
	"context"
	"encoding/json"

	"github.com/diwise/iot-core/internal/pkg/application/features/counters"
	"github.com/diwise/iot-core/internal/pkg/application/features/levels"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Feature interface {
	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
}

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type feat struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	SubType  string    `json:"subtype"`
	Location *location `json:"location,omitempty"`
	Tenant   string    `json:"tenant,omitempty"`

	Counter counters.Counter `json:"counter,omitempty"`
	Level   levels.Level     `json:"level,omitempty"`

	handle func(context.Context, *events.MessageAccepted) (bool, error)
}

func (f *feat) Handle(ctx context.Context, e *events.MessageAccepted, msgctx messaging.MsgContext) error {

	changed, err := f.handle(ctx, e)
	if err != nil {
		return err
	}

	logger := logging.GetFromContext(ctx)
	logger.Debug().Msgf("feature %s handled accepted message (changed = %v)", f.ID, changed)

	if e.HasLocation() {
		f.Location = &location{
			Latitude:  e.Latitude(),
			Longitude: e.Longitude(),
		}
	}

	// TODO: We need to be able to have tenant info before the first packet arrives,
	// 			so this lazy init version wont work in the long run ...
	tenant, ok := e.GetString("tenant")
	if ok {
		f.Tenant = tenant
	}

	if changed {
		body, _ := json.Marshal(f)
		logger.Debug().Str("body", string(body)).Msgf("publishing message to %s", f.TopicName())
		msgctx.PublishOnTopic(ctx, f)
	}

	return nil
}

func (f *feat) ContentType() string {
	return "application/json"
}

func (f *feat) TopicName() string {
	return "feature.updated"
}

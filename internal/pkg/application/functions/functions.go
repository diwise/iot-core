package functions

import (
	"context"
	"encoding/json"

	"github.com/diwise/iot-core/internal/pkg/application/functions/counters"
	"github.com/diwise/iot-core/internal/pkg/application/functions/levels"
	"github.com/diwise/iot-core/internal/pkg/application/functions/presences"
	"github.com/diwise/iot-core/internal/pkg/application/functions/timers"
	"github.com/diwise/iot-core/internal/pkg/application/functions/waterqualities"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Function interface {
	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
}

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type fnct struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	SubType  string    `json:"subtype"`
	Location *location `json:"location,omitempty"`
	Tenant   string    `json:"tenant,omitempty"`

	Counter      counters.Counter            `json:"counter,omitempty"`
	Level        levels.Level                `json:"level,omitempty"`
	Presence     presences.Presence          `json:"presence,omitempty"`
	Timer        timers.Timer                `json:"timer,omitempty"`
	WaterQuality waterqualities.WaterQuality `json:"waterquality,omitempty"`

	handle func(context.Context, *events.MessageAccepted) (bool, error)
}

func (f *fnct) Handle(ctx context.Context, e *events.MessageAccepted, msgctx messaging.MsgContext) error {

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
		// Temporary fix to force an update the first time a feature is called
		if f.Tenant == "" {
			changed = true
		}
		f.Tenant = tenant
	}

	if changed {
		body, _ := json.Marshal(f)
		logger.Debug().Str("body", string(body)).Msgf("publishing message to %s", f.TopicName())
		msgctx.PublishOnTopic(ctx, f)
	}

	return nil
}

func (f *fnct) ContentType() string {
	return "application/json"
}

func (f *fnct) TopicName() string {
	return "function.updated"
}

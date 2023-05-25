package functions

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions/buildings"
	"github.com/diwise/iot-core/internal/pkg/application/functions/counters"
	"github.com/diwise/iot-core/internal/pkg/application/functions/levels"
	"github.com/diwise/iot-core/internal/pkg/application/functions/presences"
	"github.com/diwise/iot-core/internal/pkg/application/functions/timers"
	"github.com/diwise/iot-core/internal/pkg/application/functions/waterqualities"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Function interface {
	ID() string
	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
	History(context.Context, string, int) ([]LogValue, error)
}

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type fnct struct {
	ID_      string    `json:"id"`
	Type     string    `json:"type"`
	SubType  string    `json:"subtype"`
	Location *location `json:"location,omitempty"`
	Tenant   string    `json:"tenant,omitempty"`
	Source   string    `json:"source,omitempty"`

	Counter      counters.Counter            `json:"counter,omitempty"`
	Level        levels.Level                `json:"level,omitempty"`
	Presence     presences.Presence          `json:"presence,omitempty"`
	Timer        timers.Timer                `json:"timer,omitempty"`
	WaterQuality waterqualities.WaterQuality `json:"waterquality,omitempty"`
	Building     buildings.Building          `json:"building,omitempty"`

	handle func(context.Context, *events.MessageAccepted, func(prop string, value float64) error) (bool, error)

	defaultHistoryLabel string
	storage             database.Storage
}

func (f *fnct) ID() string {
	return f.ID_
}

func (f *fnct) Handle(ctx context.Context, e *events.MessageAccepted, msgctx messaging.MsgContext) error {
	logger := logging.GetFromContext(ctx)

	timeWhenHandleWasCalled := time.Now().UTC()

	onchange := func(prop string, value float64) error {
		logger.Debug().Msgf("property %s changed to %f", prop, value)

		// onchange may be called repeatedly based on a timer, so we need to adjust
		// the event time if that happens
		timeWhenOnchangeWasCalled := time.Now().UTC()
		timeSinceHandleWasCalled := timeWhenOnchangeWasCalled.Sub(timeWhenHandleWasCalled)

		now, _ := time.Parse(time.RFC3339, e.Timestamp)
		if timeSinceHandleWasCalled >= time.Second {
			now = now.Add(timeSinceHandleWasCalled)
		}

		err := f.storage.Add(ctx, f.ID(), prop, value, now)
		if err != nil {
			logger.Error().Err(err).Msgf("failed to add values to database")
			return err
		}

		return nil
	}

	changed, err := f.handle(ctx, e, onchange)
	if err != nil {
		return err
	}

	logger.Debug().Msgf("function %s handled accepted message (changed = %v)", f.ID(), changed)

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
		// Temporary fix to force an update the first time a function is called
		if f.Tenant == "" {
			changed = true
		}
		f.Tenant = tenant
	}

	source, ok := e.GetString("source")
	if ok {
		if f.Source == "" {
			changed = true
		}
		f.Source = source
	}

	if changed {
		body, _ := json.Marshal(f)
		logger.Debug().Str("body", string(body)).Msgf("publishing message to %s", f.TopicName())
		msgctx.PublishOnTopic(ctx, f)
	}

	return nil
}

func (f *fnct) History(ctx context.Context, label string, lastN int) ([]LogValue, error) {
	if label == "" {
		label = f.defaultHistoryLabel
	}

	lv, err := f.storage.History(ctx, f.ID(), label, lastN)
	if err != nil {
		return nil, err
	}

	if len(lv) == 0 {
		return []LogValue{}, nil
	}

	loggedValues := make([]LogValue, len(lv))
	for i, v := range lv {
		loggedValues[i] = LogValue{Timestamp: v.Timestamp, Value: v.Value}
	}

	return loggedValues, nil
}

func (f *fnct) ContentType() string {
	return "application/json"
}

func (f *fnct) TopicName() string {
	return "function.updated"
}

type LogValue struct {
	Value     float64   `json:"v"`
	Timestamp time.Time `json:"ts"`
}

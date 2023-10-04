package functions

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions/airquality"
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
	Name() string

	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
	History(context.Context, string, int) ([]LogValue, error)
}

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type fnctMetadata struct {
	ID_      string    `json:"id"`
	Name_    string    `json:"name"`
	Type     string    `json:"type"`
	SubType  string    `json:"subtype"`
	Location *location `json:"location,omitempty"`
	Tenant   string    `json:"tenant,omitempty"`
	Source   string    `json:"source,omitempty"`
}

type fnct struct {
	fnctMetadata

	Counter      counters.Counter            `json:"counter,omitempty"`
	Level        levels.Level                `json:"level,omitempty"`
	Presence     presences.Presence          `json:"presence,omitempty"`
	Timer        timers.Timer                `json:"timer,omitempty"`
	WaterQuality waterqualities.WaterQuality `json:"waterquality,omitempty"`
	Building     buildings.Building          `json:"building,omitempty"`
	AirQuality   airquality.AirQuality       `json:"AirQuality,omitempty"`

	handle func(context.Context, *events.MessageAccepted, func(prop string, value float64, ts time.Time) error) (bool, any, error)

	defaultHistoryLabel string
	storage             database.Storage
}

func (f *fnct) ID() string {
	return f.ID_
}

func (f *fnct) Name() string {
	return f.Name_
}

func (f *fnct) Handle(ctx context.Context, e *events.MessageAccepted, msgctx messaging.MsgContext) error {
	logger := logging.GetFromContext(ctx).With().Str("function_id", f.ID()).Logger()
	ctx = logging.NewContextWithLogger(ctx, logger)

	onchange := func(prop string, value float64, ts time.Time) error {
		logger.Debug().Msgf("property %s changed to %f with time %s", prop, value, ts.Format(time.RFC3339Nano))

		err := f.storage.Add(ctx, f.ID(), prop, value, ts)
		if err != nil {
			logger.Error().Err(err).Msgf("failed to add values to database")
			return err
		}

		return nil
	}

	changed, diff, err := f.handle(ctx, e, onchange)
	if err != nil {
		return err
	}

	logger.Debug().Msgf("handled accepted message (changed = %v)", changed)

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
		fu := NewFnctUpdated(*f, diff)
		
		body, _ := json.Marshal(fu)
		logger.Debug().Str("body", string(body)).Msgf("publishing message to %s", fu.TopicName())

		msgctx.PublishOnTopic(ctx, fu)
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

type LogValue struct {
	Value     float64   `json:"v"`
	Timestamp time.Time `json:"ts"`
}

type fnctUpdated struct {
	fnctMetadata
	Data    any      `json:"data"`
}

func (f fnctUpdated) ContentType() string {
	return "application/json"
}

func (f fnctUpdated) TopicName() string {
	return "function.updated"
}

func NewFnctUpdated(f fnct, data any) fnctUpdated {
	return fnctUpdated{
		fnctMetadata: f.fnctMetadata,
		Data: data,
	}
}

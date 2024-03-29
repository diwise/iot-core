package functions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions/airquality"
	"github.com/diwise/iot-core/internal/pkg/application/functions/buildings"
	"github.com/diwise/iot-core/internal/pkg/application/functions/counters"
	"github.com/diwise/iot-core/internal/pkg/application/functions/digitalinput"
	"github.com/diwise/iot-core/internal/pkg/application/functions/levels"
	"github.com/diwise/iot-core/internal/pkg/application/functions/presences"
	"github.com/diwise/iot-core/internal/pkg/application/functions/stopwatch"
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

type fnct struct {
	ID_      string    `json:"id"`
	Name_    string    `json:"name"`
	Type     string    `json:"type"`
	SubType  string    `json:"subtype"`
	Location *location `json:"location,omitempty"`
	Tenant   string    `json:"tenant,omitempty"`
	Source   string    `json:"source,omitempty"`
	OnUpdate bool      `json:"onupdate"`

	Counter      counters.Counter            `json:"counter,omitempty"`
	Level        levels.Level                `json:"level,omitempty"`
	Presence     presences.Presence          `json:"presence,omitempty"`
	Timer        timers.Timer                `json:"timer,omitempty"`
	WaterQuality waterqualities.WaterQuality `json:"waterquality,omitempty"`
	Building     buildings.Building          `json:"building,omitempty"`
	AirQuality   airquality.AirQuality       `json:"AirQuality,omitempty"`
	Stopwatch    stopwatch.Stopwatch         `json:"Stopwatch,omitempty"`
	DigitalInput digitalinput.DigitalInput   `json:"DigitalInput,omitempty"`

	handle func(context.Context, *events.MessageAccepted, func(prop string, value float64, ts time.Time) error) (bool, error)

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
	logger := logging.GetFromContext(ctx).With(slog.String("function_id", f.ID()))
	ctx = logging.NewContextWithLogger(ctx, logger)

	onchange := func(prop string, value float64, ts time.Time) error {
		logger.Debug(fmt.Sprintf("property %s changed to %f with time %s", prop, value, ts.Format(time.RFC3339Nano)))

		err := f.storage.Add(ctx, f.ID(), prop, value, ts)
		if err != nil {
			logger.Error("failed to add values to database", "err", err.Error())
			return err
		}

		return nil
	}

	changed, err := f.handle(ctx, e, onchange)
	if err != nil {
		return err
	}

	logger.Debug("handled accepted message", "changed", changed)

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

	if changed || f.OnUpdate {
		fumsg := NewFunctionUpdatedMessage(f)
		logger.Debug("publishing message", "body", string(fumsg.Body()), "topic", fumsg.TopicName())
		msgctx.PublishOnTopic(ctx, fumsg)
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

func NewFunctionUpdatedMessage(f *fnct) messaging.TopicMessage {
	m, _ := messaging.NewTopicMessageJSON("function.updated", "application/json", *f)
	return m
}

type LogValue struct {
	Value     float64   `json:"v"`
	Timestamp time.Time `json:"ts"`
}

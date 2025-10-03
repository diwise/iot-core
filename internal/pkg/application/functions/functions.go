package functions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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
	"github.com/diwise/senml"
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
	ID_       string    `json:"id"`
	Name_     string    `json:"name"`
	Type      string    `json:"type"`
	SubType   string    `json:"subtype"`
	DeviceID  string    `json:"deviceID"`
	Location  *location `json:"location,omitempty"`
	Tenant    string    `json:"tenant,omitempty"`
	Source    string    `json:"source,omitempty"`
	OnUpdate  bool      `json:"onupdate"`
	Timestamp time.Time `json:"timestamp,omitempty"`

	Counter      counters.Counter            `json:"counter,omitempty"`
	Level        levels.Level                `json:"level,omitempty"`
	Presence     presences.Presence          `json:"presence,omitempty"`
	Timer        timers.Timer                `json:"timer,omitempty"`
	WaterQuality waterqualities.WaterQuality `json:"waterquality,omitempty"`
	Building     buildings.Building          `json:"building,omitempty"`
	AirQuality   airquality.AirQuality       `json:"airQuality,omitempty"`
	Stopwatch    stopwatch.Stopwatch         `json:"stopwatch,omitempty"`
	DigitalInput digitalinput.DigitalInput   `json:"digitalInput,omitempty"`

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
	log := logging.GetFromContext(ctx)
	log = log.With(slog.String("function_id", f.ID()), slog.String("device_id", e.DeviceID()))
	ctx = logging.NewContextWithLogger(ctx, log)

	if e.Timestamp.After(time.Now()) {
		log.Error("ignoring messages that claim to have been accepted in the future", "timestamp", e.Timestamp.Format(time.RFC3339))
		return nil
	}

	f.DeviceID = e.DeviceID()

	onchange := func(prop string, value float64, ts time.Time) error {
		err := f.storage.Add(ctx, f.ID(), prop, value, ts)
		if err != nil {
			log.Error("failed to add values to database", "err", err.Error())
			return err
		}

		if ts.After(f.Timestamp) {
			f.Timestamp = ts.UTC()
		}

		return nil
	}

	changed := false

	// TODO: We need to be able to have tenant info before the first packet arrives,
	// 	     so this lazy init version wont work in the long run ...
	if tenant, ok := e.Pack().GetStringValue(senml.FindByName("tenant")); ok {
		// Temporary fix to force an update the first time a function is called
		if f.Tenant == "" {
			f.Tenant = tenant
			changed = true
		}
	}

	if source, ok := e.Pack().GetStringValue(senml.FindByName("source")); ok {
		// Temporary fix to force an update the first time a function is called
		if f.Source == "" {
			f.Source = source
			changed = true
		}
	}

	if lat, lon, ok := e.Pack().GetLatLon(); ok {
		if f.Location == nil {
			f.Location = &location{
				Latitude:  lat,
				Longitude: lon,
			}
			changed = true
		}
	}

	changed, err := f.handle(ctx, e, onchange)
	if err != nil {
		if errors.Is(err, events.ErrNoMatch) {
			return nil
		}
		return err
	}

	if changed || f.OnUpdate {
		if !changed {
			f.Timestamp = e.Timestamp.UTC()
		}

		err := msgctx.PublishOnTopic(ctx, newFunctionUpdatedMessage(f))
		if err != nil {
			return err
		}
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

func newFunctionUpdatedMessage(f *fnct) messaging.TopicMessage {
	subType := ""
	if len(f.SubType) > 0 {
		subType = fmt.Sprintf(".%s", f.SubType)
	}

	if f.Timestamp.IsZero() {
		f.Timestamp = time.Now().UTC()
	}

	contentType := strings.ToLower(fmt.Sprintf("application/vnd.diwise.%s%s+json", f.Type, subType))
	m, _ := messaging.NewTopicMessageJSON("function.updated", contentType, *f)

	return m
}

type LogValue struct {
	Value     float64   `json:"v"`
	Timestamp time.Time `json:"ts"`
}

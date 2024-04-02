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
	log := logging.GetFromContext(ctx)
	log = log.With(slog.String("function_id", f.ID()))
	ctx = logging.NewContextWithLogger(ctx, log)

	onchange := func(prop string, value float64, ts time.Time) error {
		log.Debug(fmt.Sprintf("property %s changed to %f with time %s", prop, value, ts.Format(time.RFC3339)))

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

	changed, err := f.handle(ctx, e, onchange)
	if err != nil {
		if errors.Is(err, events.ErrNoMatch) {
			log.Debug(fmt.Sprintf("%s function should not handle this message type (%s)", f.Type, e.ObjectID()))
			return nil
		}
		return err
	}

	log.Debug(fmt.Sprintf("function %s handled incoming message.accepted of type %s, change is %t", f.Type, e.ObjectID(), changed))

	if lat, lon, ok := e.Pack.GetLatLon(); ok {
		if f.Location == nil {
			log.Debug("add location to function")
			f.Location = &location{
				Latitude:  lat,
				Longitude: lon,
			}
			changed = true
		}
	}

	// TODO: We need to be able to have tenant info before the first packet arrives,
	// 	     so this lazy init version wont work in the long run ...
	if tenant, ok := e.Pack.GetStringValue(senml.FindByName("tenant")); ok {
		// Temporary fix to force an update the first time a function is called
		if f.Tenant == "" {
			log.Debug("add tenant to function")
			f.Tenant = tenant
			changed = true
		}
	}

	if source, ok := e.Pack.GetStringValue(senml.FindByName("source")); ok {
		// Temporary fix to force an update the first time a function is called
		if f.Source == "" {
			log.Debug("add source to function")
			f.Source = source
			changed = true
		}
	}

	if changed || f.OnUpdate {
		fumsg := NewFunctionUpdatedMessage(f)
		log.Debug("publishing message", slog.String("body", string(fumsg.Body())), slog.String("topic", fumsg.TopicName()), slog.String("content-type", fumsg.ContentType()), slog.Bool("changed", changed), slog.Bool("onupdate", f.OnUpdate))

		err := msgctx.PublishOnTopic(ctx, fumsg)
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

func NewFunctionUpdatedMessage(f *fnct) messaging.TopicMessage {
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

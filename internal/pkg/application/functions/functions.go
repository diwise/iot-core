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
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Function interface {
	ID() string
	Name() string
	Type() string
	SubType() string
	DeviceID() string
	Tenant() string

	Handle(context.Context, *events.MessageAccepted) (bool, []Value, error)
}

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Value struct {
	Name      string    `json:"n"`
	Value     float64   `json:"v"`
	Timestamp time.Time `json:"ts"`
}

type fnct struct {
	ID_       string    `json:"id"`
	Name_     string    `json:"name"`
	Type_     string    `json:"type"`
	SubType_  string    `json:"subtype"`
	DeviceID_ string    `json:"deviceID"`
	Location  *location `json:"location,omitempty"`
	Tenant_   string    `json:"tenant,omitempty"`
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
}

func (f *fnct) ID() string {
	return f.ID_
}

func (f *fnct) Name() string {
	return f.Name_
}

func (f *fnct) Type() string {
	return f.Type_
}

func (f *fnct) SubType() string {
	return f.SubType_
}

func (f *fnct) DeviceID() string {
	return f.DeviceID_
}

func (f *fnct) Tenant() string {
	return f.Tenant_
}

func (f *fnct) Handle(ctx context.Context, e *events.MessageAccepted) (bool, []Value, error) {
	log := logging.GetFromContext(ctx)
	log = log.With(slog.String("function_id", f.ID()), slog.String("device_id", e.DeviceID()))
	ctx = logging.NewContextWithLogger(ctx, log)

	changed := false
	changes := make([]Value, 0)

	if e.Timestamp.After(time.Now()) {
		log.Error("ignoring messages that claim to have been accepted in the future", "timestamp", e.Timestamp.Format(time.RFC3339))
		return false, nil, nil
	}

	f.DeviceID_ = e.DeviceID()

	onchange := func(prop string, value float64, ts time.Time) error {
		log.Debug(fmt.Sprintf("property %s changed to %f with time %s", prop, value, ts.Format(time.RFC3339)))

		changed = true
		changes = append(changes, Value{Name: prop, Value: value, Timestamp: ts})

		if ts.After(f.Timestamp) {
			f.Timestamp = ts.UTC()
		}

		return nil
	}

	changed, err := f.handle(ctx, e, onchange)
	if err != nil {
		if errors.Is(err, events.ErrNoMatch) {
			return false, nil, nil
		}
		return false, nil, err
	}

	// TODO: We need to be able to have tenant info before the first packet arrives,
	// 	     so this lazy init version wont work in the long run ...
	if tenant, ok := e.Pack().GetStringValue(senml.FindByName("tenant")); ok {
		// Temporary fix to force an update the first time a function is called
		if f.Tenant_ == "" {
			log.Debug("add tenant to function")
			f.Tenant_ = tenant
			changed = true
		}
	}

	if source, ok := e.Pack().GetStringValue(senml.FindByName("source")); ok {
		// Temporary fix to force an update the first time a function is called
		if f.Source == "" {
			log.Debug("add source to function")
			f.Source = source
			changed = true
		}
	}

	if lat, lon, ok := e.Pack().GetLatLon(); ok {
		if f.Location == nil {
			log.Debug("add location to function")
			f.Location = &location{
				Latitude:  lat,
				Longitude: lon,
			}
			changed = true
		}
	}

	return changed || f.OnUpdate, changes, nil
}

func NewFunctionUpdatedMessage(f Function) messaging.TopicMessage {
	subType := ""

	if len(f.SubType()) > 0 {
		subType = fmt.Sprintf(".%s", f.SubType())
	}

	contentType := strings.ToLower(fmt.Sprintf("application/vnd.diwise.%s%s+json", f.Type(), subType))
	m, _ := messaging.NewTopicMessageJSON("function.updated", contentType, f)

	return m
}

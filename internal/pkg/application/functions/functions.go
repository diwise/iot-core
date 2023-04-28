package functions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

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
	ID() string
	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
	History(context.Context) ([]LogValue, error)
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

	Counter      counters.Counter            `json:"counter,omitempty"`
	Level        levels.Level                `json:"level,omitempty"`
	Presence     presences.Presence          `json:"presence,omitempty"`
	Timer        timers.Timer                `json:"timer,omitempty"`
	WaterQuality waterqualities.WaterQuality `json:"waterquality,omitempty"`

	handle func(context.Context, *events.MessageAccepted, func(prop string, value float64)) (bool, error)

	history             map[string][]LogValue
	defaultHistoryLabel string
}

func (f *fnct) ID() string {
	return f.ID_
}

func (f *fnct) Handle(ctx context.Context, e *events.MessageAccepted, msgctx messaging.MsgContext) error {

	logger := logging.GetFromContext(ctx)

	timeWhenHandleWasCalled := time.Now().UTC()

	onchange := func(prop string, value float64) {
		logger.Debug().Msgf("property %s changed to %f", prop, value)

		// onchange may be called repeatedly based on a timer, so we need to adjust
		// the event time if that happens
		timeWhenOnchangeWasCalled := time.Now().UTC()
		timeSinceHandleWasCalled := timeWhenOnchangeWasCalled.Sub(timeWhenHandleWasCalled)

		now, _ := time.Parse(time.RFC3339, e.Timestamp)
		if timeSinceHandleWasCalled >= time.Second {
			now = now.Add(timeSinceHandleWasCalled)
		}

		// TODO: This should be persisted to a database instead
		if loggedValues, ok := f.history[prop]; ok {
			f.history[prop] = append(loggedValues, LogValue{Value: value, Timestamp: now})
		} else {
			logger.Debug().Msgf("new value was not saved to history")
		}
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

	if changed {
		body, _ := json.Marshal(f)
		logger.Debug().Str("body", string(body)).Msgf("publishing message to %s", f.TopicName())
		msgctx.PublishOnTopic(ctx, f)
	}

	return nil
}

func (f *fnct) History(context.Context) ([]LogValue, error) {
	if loggedValues, ok := f.history[f.defaultHistoryLabel]; ok {
		return loggedValues, nil
	}

	return nil, errors.New("no history")
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

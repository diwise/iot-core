package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions/counters"
	"github.com/diwise/iot-core/internal/pkg/application/functions/levels"
	"github.com/diwise/iot-core/internal/pkg/application/functions/stopwatch"
	"github.com/diwise/iot-core/internal/pkg/application/functions/timers"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"

	lwm2m "github.com/diwise/iot-agent/pkg/lwm2m"
)

func Transform(ctx context.Context, msgctx messaging.MsgContext, msg messaging.IncomingTopicMessage) error {
	log := logging.GetFromContext(ctx)

	f := struct {
		DeviceID  string    `json:"deviceID"`
		Type      string    `json:"type"`
		SubType   string    `json:"subType"`
		Location  *location `json:"location,omitempty"`
		Tenant    string    `json:"tenant"`
		Timestamp time.Time `json:"timestamp"`

		Counter *struct {
			Count   int            `json:"count"`
			Changes map[string]int `json:"changes"`
		} `json:"counter,omitempty"`

		Level *struct {
			Current float64  `json:"current"`
			Percent *float64 `json:"percent,omitempty"`
			Offset  *float64 `json:"offset,omitempty"`
		} `json:"level,omitempty"`

		Stopwatch *struct {
			CumulativeTime time.Duration `json:"cumulativeTime"`
			Count          int32         `json:"count"`
			State          bool          `json:"state"`
		} `json:"stopwatch,omitempty"`

		Timer *struct {
			TotalDuration time.Duration  `json:"totalDuration"`
			Duration      *time.Duration `json:"duration,omitempty"`
		} `json:"timer,omitempty"`
	}{}

	err := json.Unmarshal(msg.Body(), &f)
	if err != nil {
		return err
	}

	log.Debug(fmt.Sprintf("transform function.updated of type %s and subType %s for deviceID %s", f.Type, f.SubType, f.DeviceID))

	pub := func(obj lwm2m.Lwm2mObject, tenant string) error {
		log.Debug(fmt.Sprintf("pub transformed message, id: %s, urn: %s", obj.ID(), obj.ObjectURN()))

		d := []events.EventDecoratorFunc{events.Tenant(tenant)}
		if f.Location != nil {
			d = append(d, events.Lat(f.Location.Latitude), events.Lon(f.Location.Longitude))
		}

		mt := events.NewMessageTransformed(lwm2m.ToPack(obj), d...)

		return msgctx.PublishOnTopic(ctx, mt)
	}

	switch f.Type {
	case counters.FunctionTypeName:
		switch f.SubType {
		case "peoplecounter":
			peopleCounter := lwm2m.NewPeopleCounter(f.DeviceID, f.Counter.Count, f.Timestamp)
			err = pub(peopleCounter, f.Tenant)
		}
	case levels.FunctionTypeName:
		if f.Level.Percent == nil {
			log.Debug("could not transform fillingLevel, percent is missing")
			return nil
		}
		fillingLevel := lwm2m.NewFillingLevel(f.DeviceID, *f.Level.Percent, f.Timestamp)
		l := int64(f.Level.Current)
		fillingLevel.ActualFillingLevel = &l
		err = pub(fillingLevel, f.Tenant)
	case stopwatch.FunctionTypeName:
		stopwatch := lwm2m.NewStopwatch(f.DeviceID, f.Stopwatch.CumulativeTime.Seconds(), f.Timestamp)
		stopwatch.OnOff = &f.Stopwatch.State
		stopwatch.DigitalInputCounter = f.Stopwatch.Count
		err = pub(stopwatch, f.Tenant)
	case timers.FunctionTypeName:
		if f.Timer.Duration == nil {
			log.Debug("could not transform timer, duration is missing")
			return nil
		}
		timer := lwm2m.NewTimer(f.DeviceID, f.Timer.Duration.Seconds(), f.Timestamp)
		cumulativeTime := f.Timer.TotalDuration.Seconds()
		timer.CumulativeTime = &cumulativeTime
		err = pub(timer, f.Tenant)
	default:
		log.Debug("no function transformer found for function.updated message", slog.String("content_type", msg.ContentType()))
	}

	return err
}

package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/iot-core/internal/pkg/application/decorators"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/functions/engines"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type App interface {
	MessageAccepted(ctx context.Context, evt events.MessageAccepted) error
	MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
	FunctionUpdated(ctx context.Context, body []byte) error
}

type app struct {
	deviceManagement   client.DeviceManagementClient
	measurementsClient measurements.MeasurementsClient
	funcRegistry       functions.FuncRegistry
	ruleEngine         engines.RuleEngine
	mu                 sync.Mutex
	messenger          messaging.MsgContext
}

func New(client client.DeviceManagementClient, measurementsClient measurements.MeasurementsClient, functionRegistry functions.FuncRegistry, ruleEngine engines.RuleEngine, msgCtx messaging.MsgContext) App {
	return &app{
		deviceManagement:   client,
		funcRegistry:       functionRegistry,
		measurementsClient: measurementsClient,
		ruleEngine:         ruleEngine,
		messenger:          msgCtx,
	}
}

var ErrCouldNotFindDevice = fmt.Errorf("could not find device")

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	log := logging.GetFromContext(ctx)

	if msg.Error() != nil {
		log.Debug("received malformed message", "err", msg.Error().Error())
		return nil, msg.Error()
	}

	device, err := a.deviceManagement.FindDeviceFromInternalID(ctx, msg.DeviceID())
	if err != nil {
		return nil, ErrCouldNotFindDevice
	}

	clone := msg.Pack().Clone()

	decs := make([]events.EventDecoratorFunc, 0)
	decs = append(decs,
		events.Lat(device.Latitude()),
		events.Lon(device.Longitude()),
		events.Environment(device.Environment()),
		events.Source(device.Source()),
		events.Tenant(device.Tenant()))

	switch msg.ObjectID() {
	case "3":
		decs = append(decs, decorators.Device(ctx, decorators.GetMaxPowerSourceVoltage(ctx, a.measurementsClient, device.ID())))
	case "3200":
		decs = append(decs, decorators.DigitalInput(ctx, decorators.GetNumberOfTrueValues(ctx, a.measurementsClient, device.ID())))
	}

	validated, err := a.ruleEngine.ValidateMessageReceived(ctx, msg, log)

	for _, validation := range validated {
		if !validation.IsValid {
			log.Debug("message did not validate by it's rule", "device_id", device.ID())

			if validation.ShouldAbort {
				abortMessage := events.NewMessageAborted(clone, validation.Errors)
				err = a.messenger.PublishOnTopic(ctx, abortMessage)
				return nil, fmt.Errorf("message did not validate and is set to abort")
			}

			notValidatedMessage := events.NewMessageNotValidated(clone, validation.Errors)
			err = a.messenger.PublishOnTopic(ctx, notValidatedMessage)
		}
	}

	ma := events.NewMessageAccepted(clone, decs...)

	return ma, nil
}

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted) error {
	if evt.Error() != nil {
		return evt.Error()
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	matchingFunctions, _ := a.funcRegistry.Find(ctx, functions.MatchSensor(evt.DeviceID()))

	if len(matchingFunctions) == 0 {
		return nil
	}

	errs := []error{}

	for _, f := range matchingFunctions {
		err := f.Handle(ctx, &evt, a.messenger)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

type functionUpdated struct {
	DeviceID  string    `json:"deviceID"`
	Type      string    `json:"type"`
	Tenant    string    `json:"tenant"`
	Timestamp time.Time `json:"timestamp"`

	Level     *level     `json:"level"`
	Timer     *timer     `json:"timer"`
	Stopwatch *stopwatch `json:"stopwatch"`
}

type level struct {
	Current float64 `json:"current"`
	Percent float64 `json:"percent"`
}

type timer struct {
	StartTime     time.Time      `json:"startTime"`
	State         bool           `json:"state"`
	Duration      *time.Duration `json:"duration"`
	TotalDuration time.Duration  `json:"totalDuration"`
}

type stopwatch struct {
	StartTime      time.Time      `json:"startTime"`
	StopTime       time.Time      `json:"stopTime"`
	Duration       *time.Duration `json:"duration"`
	State          bool           `json:"state"`
	Count          int            `json:"count"`
	CumulativeTime time.Duration  `json:"cumulativeTime"`
}

func (a *app) FunctionUpdated(ctx context.Context, body []byte) error {
	log := logging.GetFromContext(ctx)

	fn := functionUpdated{}
	err := json.Unmarshal(body, &fn)
	if err != nil {
		return err
	}

	var evt *events.MessageReceived

	switch strings.ToLower(fn.Type) {
	case "level":
		if fn.Level == nil {
			return nil
		}

		fl := lwm2m.NewFillingLevel(fn.DeviceID, fn.Level.Percent, fn.Timestamp)
		afl := int64(fn.Level.Current * 100) // lwm2m distance is meters, fillingLevel is cm
		fl.ActualFillingLevel = &afl

		log.Debug("filling level function updated", slog.String("deviceID", fn.DeviceID), slog.Float64("percent", fn.Level.Percent), slog.Float64("current", fn.Level.Current))

		evt = events.NewMessageReceived(lwm2m.ToPack(fl))
	case "timer":
		if fn.Timer == nil {
			return nil
		}

		tmr := lwm2m.NewTimer(fn.DeviceID, fn.Timer.Duration.Seconds(), fn.Timestamp)
		tmr.OnOff = fn.Timer.State
		cumulative := float64(fn.Timer.TotalDuration.Seconds())
		tmr.CumulativeTime = &cumulative

		evt = events.NewMessageReceived(lwm2m.ToPack(tmr))
	case "stopwatch":
		if fn.Stopwatch == nil {
			return nil
		}

		sw := lwm2m.NewStopwatch(fn.DeviceID, float64(fn.Stopwatch.CumulativeTime.Seconds()), fn.Timestamp)
		sw.OnOff = &fn.Stopwatch.State
		sw.DigitalInputCounter = int32(fn.Stopwatch.Count)

		evt = events.NewMessageReceived(lwm2m.ToPack(sw))
	default:
		return nil
	}

	if evt != nil {
		err := a.messenger.SendCommandTo(ctx, evt, "iot-core")
		if err != nil {
			log.Error("could not send message.received from function update", slog.String("deviceID", fn.DeviceID), slog.String("function_type", fn.Type), slog.String("err", err.Error()))
			return err
		}
		log.Debug("sent message.received from function update", slog.String("deviceID", fn.DeviceID), slog.String("function_type", fn.Type))
	}

	return nil
}

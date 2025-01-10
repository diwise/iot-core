package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/diwise/iot-core/internal/pkg/application/decorators"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

var ErrCouldNotFindDevice = fmt.Errorf("could not find device")

//go:generate moq -rm -out iotcore_mock.go . App
type App interface {
	MessageAccepted(ctx context.Context, evt events.MessageAccepted) error
	MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)

	Register(ctx context.Context, s functions.Setting) error
	Query(ctx context.Context, params map[string]any) ([]functions.Function, error)
}

type app struct {
	devMgmtClient      client.DeviceManagementClient
	measurementsClient measurements.MeasurementsClient
	registry           functions.Registry
	msgCtx             messaging.MsgContext
}

func New(client client.DeviceManagementClient, measurementsClient measurements.MeasurementsClient, functionRegistry functions.Registry, msgCtx messaging.MsgContext) App {
	return &app{
		devMgmtClient:      client,
		registry:           functionRegistry,
		measurementsClient: measurementsClient,
		msgCtx:             msgCtx,
	}
}

func (a *app) Register(ctx context.Context, s functions.Setting) error {
	log := logging.GetFromContext(ctx)

	_, err := a.devMgmtClient.FindDeviceFromInternalID(ctx, s.DeviceID)
	if err != nil {
		log.Debug(fmt.Sprintf("could not find device with internalID %s", s.DeviceID), "err", err.Error())
		return fmt.Errorf("could not find device with internalID %s", s.DeviceID)
	}

	err = a.registry.Add(ctx, s)
	if err != nil {
		return err
	}

	return nil
}

func (a *app) Query(ctx context.Context, params map[string]any) ([]functions.Function, error) {
	matchers := make([]functions.RegistryMatcherFunc, 0)

	if len(params) == 0 {
		matchers = append(matchers, functions.MatchAll())
	}

	for k, v := range params {
		switch strings.ToLower(k) {
		case "deviceid":
			matchers = append(matchers, functions.MatchSensor(v.(string)))
		case "id":
			matchers = append(matchers, functions.MatchID(v.(string)))
		}
	}

	if len(matchers) == 0 {
		return []functions.Function{}, nil
	}

	return a.registry.Find(ctx, matchers...)
}

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	if msg.Error() != nil {
		return nil, msg.Error()
	}

	log := logging.GetFromContext(ctx)
	log.Debug(fmt.Sprintf("received message of type %s for device %s", msg.ContentType(), msg.DeviceID()), slog.String("body", string(msg.Body())))

	device, err := a.devMgmtClient.FindDeviceFromInternalID(ctx, msg.DeviceID())
	if err != nil {
		log.Debug(fmt.Sprintf("could not find device with internalID %s", msg.DeviceID()), "err", err.Error())
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

	ma := events.NewMessageAccepted(clone, decs...)

	log.Debug(fmt.Sprintf("message.accepted created for device %s with object type %s", ma.DeviceID(), ma.ObjectID()), slog.String("body", string(ma.Body())))

	return ma, nil
}

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted) error {
	if evt.Error() != nil {
		return evt.Error()
	}

	logger := logging.GetFromContext(ctx)

	matchingFunctions, _ := a.registry.Find(ctx, functions.MatchSensor(evt.DeviceID()))
	matchingCount := len(matchingFunctions)

	if matchingCount == 0 {
		logger.Debug("no matching functions found")
		return nil
	}

	var errs []error

	for _, f := range matchingFunctions {
		changed, changes, err := f.Handle(ctx, &evt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if !changed {
			continue
		}

		a.registry.Update(ctx, f.ID(), f)

		if len(changes) > 0 {
			for _, change := range changes {
				err := a.registry.AddValue(ctx, f.ID(), change.Name, change.Value, change.Timestamp)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}

		err = a.msgCtx.PublishOnTopic(ctx, functions.NewFunctionUpdatedMessage(f))
		if err != nil {
			logger.Error("failed to publish function updated message", "err", err.Error())
			errs = append(errs, err)
			continue
		}

		for _, pack := range functions.Transform(f) {
			mt := events.NewMessageTransformed(pack, events.Tenant(f.Tenant()))
			err = a.msgCtx.PublishOnTopic(ctx, mt)
			if err != nil {
				logger.Error("failed to publish transformed message", "err", err.Error())
				errs = append(errs, err)
				continue
			}
		}
	}

	return errors.Join(errs...)
}

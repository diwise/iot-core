package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/diwise/iot-core/internal/pkg/application/decorators"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type App interface {
	MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error
	MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
}

type app struct {
	client             client.DeviceManagementClient
	measurementsClient measurements.MeasurementsClient
	fnctRegistry       functions.Registry
	mu                 sync.Mutex
}

func New(client client.DeviceManagementClient, measurementsClient measurements.MeasurementsClient, functionRegistry functions.Registry) App {
	return &app{
		client:             client,
		fnctRegistry:       functionRegistry,
		measurementsClient: measurementsClient,
	}
}

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error {
	if evt.Error() != nil {
		return evt.Error()
	}

	logger := logging.GetFromContext(ctx)

	a.mu.Lock()
	defer a.mu.Unlock()

	matchingFunctions, _ := a.fnctRegistry.Find(ctx, functions.MatchSensor(evt.DeviceID()))
	matchingCount := len(matchingFunctions)

	if matchingCount == 0 {
		logger.Debug("no matching functions found")
		return nil
	}

	logger.Debug("found matching functions", "count", matchingCount)

	for _, f := range matchingFunctions {
		if err := f.Handle(ctx, &evt, msgctx); err != nil {
			return err
		}
	}

	return nil
}

var ErrCouldNotFindDevice = fmt.Errorf("could not find device")

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	if msg.Error() != nil {
		return nil, msg.Error()
	}

	log := logging.GetFromContext(ctx)
	log.Debug(fmt.Sprintf("received message of type %s for device %s", msg.ContentType(), msg.DeviceID()), slog.String("body", string(msg.Body())))

	device, err := a.client.FindDeviceFromInternalID(ctx, msg.DeviceID())
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

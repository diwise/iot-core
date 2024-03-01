package application

import (
	"context"
	"fmt"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
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
	client       client.DeviceManagementClient
	fnctRegistry functions.Registry
}

func New(client client.DeviceManagementClient, functionRegistry functions.Registry) App {
	return &app{
		client:       client,
		fnctRegistry: functionRegistry,
	}
}

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error {
	matchingFunctions, _ := a.fnctRegistry.Find(ctx, functions.MatchSensor(evt.DeviceID()))

	logger := logging.GetFromContext(ctx)
	matchingCount := len(matchingFunctions)

	if matchingCount > 0 {
		logger.Debug("found matching functions", "count", matchingCount)
	} else {
		logger.Debug("no matching functions found")
	}

	for _, f := range matchingFunctions {
		if err := f.Handle(ctx, &evt, msgctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {

	if msg.DeviceID() == "" {
		return nil, fmt.Errorf("message pack contains no DeviceID")
	}

	device, err := a.client.FindDeviceFromInternalID(ctx, msg.DeviceID())
	if err != nil {
		return nil, fmt.Errorf("could not find device with internalID %s, %w", msg.DeviceID(), err)
	}

	return events.NewMessageAccepted(msg.Pack().Clone(),
		events.Lat(device.Latitude()),
		events.Lon(device.Longitude()),
		events.Environment(device.Environment()),
		events.Source(device.Source()),
		events.Tenant(device.Tenant())), nil
}

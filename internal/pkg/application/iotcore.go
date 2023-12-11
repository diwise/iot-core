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
	MessageAccepted(ctx context.Context, evt events.MessageAccepted) error
	MessageReceived(ctx context.Context, msg events.MessageReceived) error
}

type app struct {
	msgCtx                 messaging.MsgContext
	fnReg                  functions.Registry
	deviceManagementClient client.DeviceManagementClient
}

func New(dmc client.DeviceManagementClient, functionRegistry functions.Registry, msgCtx messaging.MsgContext) App {
	return &app{
		msgCtx:                 msgCtx,
		deviceManagementClient: dmc,
		fnReg:                  functionRegistry,
	}
}

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted) error {
	matchingFunctions, _ := a.fnReg.Find(ctx, functions.MatchSensor(evt.DeviceID))

	logger := logging.GetFromContext(ctx)
	matchingCount := len(matchingFunctions)

	if matchingCount > 0 {
		logger.Debug("found matching functions", "count", matchingCount)
	} else {
		logger.Debug("no matching functions found")
	}

	for _, f := range matchingFunctions {
		if err := f.Handle(ctx, &evt, a.msgCtx); err != nil {
			return err
		}
	}

	return nil
}

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) error {
	if msg.DeviceID() == "" {
		return fmt.Errorf("message pack contains no DeviceID")
	}

	device, err := a.deviceManagementClient.FindDeviceFromInternalID(ctx, msg.DeviceID())
	if err != nil {
		return fmt.Errorf("could not find device with internalID %s, %w", msg.DeviceID(), err)
	}

	messageAccepted := events.NewMessageAccepted(device.ID(), msg.Pack.Clone(),
		events.Lat(device.Latitude()),
		events.Lon(device.Longitude()),
		events.Environment(device.Environment()),
		events.Source(device.Source()),
		events.Tenant(device.Tenant()))

	log := logging.GetFromContext(ctx)
	log.Debug("publishing message", "topic", messageAccepted.TopicName(), "content-type", messageAccepted.ContentType())

	return a.msgCtx.PublishOnTopic(ctx, messageAccepted)
}

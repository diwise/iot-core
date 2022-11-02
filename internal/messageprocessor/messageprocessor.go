package messageprocessor

import (
	"context"
	"fmt"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
)

//go:generate moq -rm -out messageprocessor_mock.go . MessageProcessor

type MessageProcessor interface {
	ProcessMessage(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
}

type messageProcessor struct {
	deviceManagementClient client.DeviceManagementClient
}

func NewMessageProcessor(d client.DeviceManagementClient) MessageProcessor {
	return &messageProcessor{
		deviceManagementClient: d,
	}
}

func (m *messageProcessor) ProcessMessage(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	if msg.DeviceID() == "" {
		return nil, fmt.Errorf("message pack contains no DeviceID")
	}

	device, err := m.deviceManagementClient.FindDeviceFromInternalID(ctx, msg.DeviceID())
	if err != nil {
		return nil, fmt.Errorf("could not find device from internalID %s, %w", msg.DeviceID(), err)
	}

	return events.NewMessageAccepted(device.ID(),
		events.Lat(device.Latitude()),
		events.Lon(device.Longitude()),
		events.Environment(device.Environment()),
		events.Tenant(device.Tenant())), nil
}

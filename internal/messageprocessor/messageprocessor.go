package messageprocessor

import (
	"context"
	"fmt"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/farshidtz/senml/v2"
)

//go:generate moq -rm -out messageprocessor_mock.go . MessageProcessor

type MessageProcessor interface {
	ProcessMessage(ctx context.Context, pack senml.Pack) (*events.MessageAccepted, error)
}

type messageProcessor struct {
	deviceManagementClient client.DeviceManagementClient
}

func NewMessageProcessor(d client.DeviceManagementClient) MessageProcessor {
	return &messageProcessor{
		deviceManagementClient: d,
	}
}

func (m *messageProcessor) ProcessMessage(ctx context.Context, pack senml.Pack) (*events.MessageAccepted, error) {
	internalID := getInternalIDFromPack(pack)
	if internalID == "" {
		return nil, fmt.Errorf("unable to get internalID from pack")
	}

	device, err := m.deviceManagementClient.FindDeviceFromInternalID(ctx, internalID)
	if err != nil {
		return nil, err
	}

	// TODO: Validate, process and enrich data

	pack = enrichEnv(pack, device.Environment())
	pack = enrichTenant(pack, device.Tenant())

	msg := events.NewMessageAccepted(device.ID(), pack).AtLocation(device.Latitude(), device.Longitude())

	return &msg, nil
}

func enrichEnv(p senml.Pack, env string) senml.Pack {
	if env == "" {
		return p
	}

	envRec := &senml.Record{
		Name:        "env",
		StringValue: env,
	}

	p = append(p, *envRec)

	return p
}

func enrichTenant(p senml.Pack, tenant string) senml.Pack {
	if tenant == "" {
		return p
	}

	envRec := &senml.Record{
		Name:        "tenant",
		StringValue: tenant,
	}

	p = append(p, *envRec)

	return p
}

func getInternalIDFromPack(p senml.Pack) string {
	if len(p) == 0 {
		return ""
	}

	if p[0].Name == "0" {
		return p[0].StringValue
	}

	return ""
}

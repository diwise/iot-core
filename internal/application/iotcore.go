package application

import (
	"context"

	"github.com/diwise/iot-core/internal/messageprocessor"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/rs/zerolog"
)

type IoTCoreApp interface {
	MessageAccepted(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
}

type iotCoreApp struct {
	messageProcessor messageprocessor.MessageProcessor
	log              zerolog.Logger
}

func NewIoTCoreApp(serviceName string, m messageprocessor.MessageProcessor, logger zerolog.Logger) IoTCoreApp {
	return &iotCoreApp{
		messageProcessor: m,
		log:              logger,
	}
}

func (a *iotCoreApp) MessageAccepted(ctx context.Context, rcvdMsg events.MessageReceived) (*events.MessageAccepted, error) {

	if err := rcvdMsg.Pack.Validate(); err != nil {
		a.log.Error().Err(err).Msg("failed to validate senML message")
		return nil, err
	}

	e, err := a.messageProcessor.ProcessMessage(ctx, rcvdMsg.Pack)
	if err != nil {
		a.log.Error().Err(err).Msg("failed to process message")
		return nil, err
	}

	return e, nil
}

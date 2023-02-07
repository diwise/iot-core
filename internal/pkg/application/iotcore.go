package application

import (
	"context"

	"github.com/diwise/iot-core/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/rs/zerolog"
)

type App interface {
	MessageAccepted(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
}

type app struct {
	messageProcessor messageprocessor.MessageProcessor
	log              zerolog.Logger
}

func NewIoTCoreApp(serviceName string, m messageprocessor.MessageProcessor, logger zerolog.Logger) App {
	return &app{
		messageProcessor: m,
		log:              logger,
	}
}

func (a *app) MessageAccepted(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	if messageAccepted, err := a.messageProcessor.ProcessMessage(ctx, msg); err == nil {
		return messageAccepted, nil
	} else {
		a.log.Error().Err(err).Msg("failed to process message")
		return nil, err
	}
}

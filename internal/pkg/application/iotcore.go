package application

import (
	"context"
	"fmt"

	"github.com/diwise/iot-core/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/rs/zerolog"
)

type App interface {
	MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
}

type app struct {
	messageProcessor messageprocessor.MessageProcessor
}

func NewIoTCoreApp(serviceName string, m messageprocessor.MessageProcessor, logger zerolog.Logger) App {
	return &app{
		messageProcessor: m,
	}
}

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	if messageAccepted, err := a.messageProcessor.ProcessMessage(ctx, msg); err == nil {
		return messageAccepted, nil
	} else {
		return nil, fmt.Errorf("failed to process message: %w", err)
	}
}

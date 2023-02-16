package application

import (
	"context"
	"fmt"

	"github.com/diwise/iot-core/internal/pkg/application/features"
	"github.com/diwise/iot-core/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type App interface {
	MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error
	MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
}

type app struct {
	msgproc_  messageprocessor.MessageProcessor
	features_ features.Registry
}

func New(msgproc messageprocessor.MessageProcessor, featureRegistry features.Registry) App {
	return &app{
		msgproc_:  msgproc,
		features_: featureRegistry,
	}
}

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error {
	matchingFeatures, _ := a.features_.Find(ctx, features.MatchSensor(evt.Sensor))

	logger := logging.GetFromContext(ctx)
	logger.Debug().Msgf("found %d features connected to sensor %s", len(matchingFeatures), evt.Sensor)

	for _, f := range matchingFeatures {
		if err := f.Handle(ctx, &evt, msgctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	messageAccepted, err := a.msgproc_.ProcessMessage(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to process message: %w", err)
	}

	return messageAccepted, nil
}

package application

import (
	"context"
	"encoding/json"

	"github.com/diwise/iot-core/internal/messageprocessor"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/farshidtz/senml/v2"
	"github.com/rs/zerolog"
)

type IoTCoreApp interface {
	MessageAccepted(ctx context.Context, msg []byte) (*events.MessageAccepted, error)
}

type iotCoreApp struct {	
	messageProcessor messageprocessor.MessageProcessor
	log zerolog.Logger
}

func NewIoTCoreApp(serviceName string, m messageprocessor.MessageProcessor, logger zerolog.Logger) IoTCoreApp {
	return &iotCoreApp{
		messageProcessor: m,
		log: logger,
	}
}

func (a *iotCoreApp) MessageAccepted(ctx context.Context, msg []byte) (*events.MessageAccepted, error) {
	var pack senml.Pack
	
	err := json.Unmarshal(msg, &pack)
	if err != nil {
		a.log.Error().Err(err).Msg("failed to decode senML message from json")
		return nil, err
	}

	if err := pack.Validate(); err != nil {
		a.log.Error().Err(err).Msg("failed to validate senML message")
		return nil, err
	}
	
	e, err := a.messageProcessor.ProcessMessage(ctx, pack)
	if err != nil {
		a.log.Error().Err(err).Msg("failed to process message")
		return nil, err
	}	

	return e, nil
}
package main

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"time"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/logging"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/tracing"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"go.opentelemetry.io/otel"

	"github.com/rs/zerolog"
)

const serviceName string = "iot-core"

var tracer = otel.Tracer(serviceName)

func main() {
	serviceVersion := version()

	ctx, logger := logging.NewLogger(context.Background(), serviceName, serviceVersion)
	logger.Info().Msg("starting up ...")

	cleanup, err := tracing.Init(ctx, logger, serviceName, serviceVersion)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init tracing")
	}
	defer cleanup()

	config := messaging.LoadConfiguration(serviceName, logger)
	messenger, err := messaging.Initialize(config)

	needToDecideThis := "application/json"
	messenger.RegisterCommandHandler(needToDecideThis, newCommandHandler(messenger))

	for {
		time.Sleep(1 * time.Second)
	}
}

func newCommandHandler(messenger messaging.MsgContext) messaging.CommandHandler {
	return func(ctx context.Context, wrapper messaging.CommandMessageWrapper, logger zerolog.Logger) error {
		var err error
		ctx, span := tracer.Start(ctx, "rcv-cmd")
		defer func() {
			if err != nil {
				span.RecordError(err)
			}
			span.End()
		}()

		cmd := struct {
			InternalID  string  `json:"internalID"`
			Type        string  `json:"type"`
			SensorValue float64 `json:"sensorValue"`
		}{}

		body := wrapper.Body()
		json.Unmarshal(body, &cmd)

		// TODO: Validate, process and enrich data

		msg := events.NewMessageAccepted(
			cmd.InternalID, "temperature/water",
			cmd.Type, cmd.SensorValue,
		).AtLocation(62.39160, 17.30723)

		logger.Info().Msgf("publishing message to %s", msg.TopicName())
		err = messenger.PublishOnTopic(ctx, &msg)
		if err != nil {
			logger.Error().Err(err).Msg("failed to publish message")
		}

		return err
	}
}

func version() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	buildSettings := buildInfo.Settings
	infoMap := map[string]string{}
	for _, s := range buildSettings {
		infoMap[s.Key] = s.Value
	}

	sha := infoMap["vcs.revision"]
	if infoMap["vcs.modified"] == "true" {
		sha += "+"
	}

	return sha
}

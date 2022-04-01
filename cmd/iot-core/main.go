package main

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"strings"
	"time"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/tracing"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"go.opentelemetry.io/otel"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var tracer = otel.Tracer("iot-core")

func main() {
	serviceName := "iot-core"
	serviceVersion := version()

	logger := log.With().Str("service", strings.ToLower(serviceName)).Str("version", serviceVersion).Logger()
	logger.Info().Msg("starting up ...")

	ctx := context.Background()

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

		logger.Info().Str("body", string(wrapper.Body())).Msgf("received command")

		msg := &events.MessageAccepted{
			Sensor:      cmd.InternalID,
			Type:        cmd.Type,
			SensorValue: cmd.SensorValue,
		}
		err = messenger.PublishOnTopic(ctx, msg)

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

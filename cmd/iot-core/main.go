package main

import (
	"context"
	"encoding/json"
	"os"
	"runtime/debug"
	"time"

	"github.com/diwise/iot-core/internal/pkg/domain"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/logging"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/tracing"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/farshidtz/senml/v2"
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
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init messaging")
	}

	dmURL := os.Getenv("DEV_MGMT_URL")
	dmClient := domain.NewDeviceManagementClient(dmURL)

	needToDecideThis := "application/json"
	messenger.RegisterCommandHandler(needToDecideThis, newCommandHandler(messenger, dmClient))

	for {
		time.Sleep(1 * time.Second)
	}
}

func newCommandHandler(messenger messaging.MsgContext, dmClient domain.DeviceManagementClient) messaging.CommandHandler {
	return func(ctx context.Context, wrapper messaging.CommandMessageWrapper, logger zerolog.Logger) error {
		var err error
		ctx, span := tracer.Start(ctx, "rcv-cmd")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		var pack senml.Pack
		body := wrapper.Body()
		err = json.Unmarshal(body, &pack)
		if err != nil {
			logger.Error().Err(err).Msg("failed to decode senML message from json")
			return err
		}

		if err := pack.Validate(); err != nil {
			logger.Error().Err(err).Msg("failed to validate senML message")
			return err
		}

		internalID := getInternalIDFromPack(pack)				
		device, err := dmClient.FindDeviceFromInternalID(ctx, internalID)
		if err != nil {
			return err
		}

		// TODO: Validate, process and enrich data

		pack = enrichEnv(pack, device.Environment())

		msg := events.NewMessageAccepted(device.ID(), pack).AtLocation(device.Latitude(), device.Longitude())

		logger.Info().Msgf("publishing message to %s", msg.TopicName())
		err = messenger.PublishOnTopic(ctx, &msg)
		if err != nil {
			logger.Error().Err(err).Msg("failed to publish message")
			return err
		}

		return nil
	}
}

func enrichEnv(p senml.Pack, env string) senml.Pack {
	envRec := &senml.Record{
		Name:        "environment",
		StringValue: env,
	}

	p = append(p, *envRec)

	return p
}

func getInternalIDFromPack(p senml.Pack) string {
	return p[0].StringValue
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

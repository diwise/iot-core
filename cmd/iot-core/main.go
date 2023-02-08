package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"

	"github.com/diwise/iot-core/internal/pkg/application"
	"github.com/diwise/iot-core/internal/pkg/application/features"
	"github.com/diwise/iot-core/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-core/internal/pkg/presentation/api"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

const serviceName string = "iot-core"

var tracer = otel.Tracer(serviceName)
var featuresConfigPath string

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	flag.StringVar(&featuresConfigPath, "features", "/opt/diwise/config/features.csv", "configuration file for features")
	flag.Parse()

	dmClient := createDeviceManagementClientOrDie(ctx, logger)
	msgCtx := createMessagingContextOrDie(ctx, logger)

	var configFile *os.File
	var err error

	if featuresConfigPath != "" {
		configFile, err = os.Open(featuresConfigPath)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to open features config file")
		}
		defer configFile.Close()
	}

	_, api_, err := initialize(dmClient, msgCtx, configFile)
	if err != nil {
		logger.Fatal().Err(err).Msg("initialization failed")
	}

	servicePort := env.GetVariableOrDefault(logger, "SERVICE_PORT", "8080")
	err = http.ListenAndServe(":"+servicePort, api_.Router())
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start request router")
	}
}

func createDeviceManagementClientOrDie(ctx context.Context, logger zerolog.Logger) client.DeviceManagementClient {
	dmURL := env.GetVariableOrDie(logger, "DEV_MGMT_URL", "url to iot-device-mgmt")
	tokenURL := env.GetVariableOrDie(logger, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	clientID := env.GetVariableOrDie(logger, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	clientSecret := env.GetVariableOrDie(logger, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")

	dmClient, err := client.New(ctx, dmURL, tokenURL, clientID, clientSecret)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create device managagement client")
	}

	return dmClient
}

func createMessagingContextOrDie(ctx context.Context, logger zerolog.Logger) messaging.MsgContext {
	config := messaging.LoadConfiguration(serviceName, logger)
	messenger, err := messaging.Initialize(config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init messaging")
	}

	return messenger
}

func initialize(dmClient client.DeviceManagementClient, msgctx messaging.MsgContext, fconfig io.Reader) (application.App, api.API, error) {

	msgproc := messageprocessor.NewMessageProcessor(dmClient)

	featuresRegistry, err := features.NewRegistry(fconfig)
	if err != nil {
		return nil, nil, err
	}

	app := application.New(msgproc, featuresRegistry)

	needToDecideThis := "application/json"
	msgctx.RegisterCommandHandler(needToDecideThis, newCommandHandler(msgctx, app))

	routingKey := "message.accepted"
	msgctx.RegisterTopicMessageHandler(routingKey, newTopicMessageHandler(msgctx, app))

	return app, api.New(featuresRegistry), nil
}

func newCommandHandler(messenger messaging.MsgContext, app application.App) messaging.CommandHandler {

	return func(ctx context.Context, wrapper messaging.CommandMessageWrapper, logger zerolog.Logger) error {
		var err error

		ctx, span := tracer.Start(ctx, "receive-command")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		evt := events.MessageReceived{}
		err = json.Unmarshal(wrapper.Body(), &evt)
		if err != nil {
			logger.Error().Err(err).Msg("failed to decode message from json")
			return err
		}

		messageAccepted, err := app.MessageReceived(ctx, evt)
		if err != nil {
			logger.Error().Err(err).Msg("message not accepted")
			return err
		}

		logger.Info().Msgf("publishing message to %s", messageAccepted.TopicName())
		err = messenger.PublishOnTopic(ctx, messageAccepted)
		if err != nil {
			logger.Error().Err(err).Msg("failed to publish message")
			return err
		}

		return nil
	}
}

func newTopicMessageHandler(messenger messaging.MsgContext, app application.App) messaging.TopicMessageHandler {

	return func(ctx context.Context, msg amqp.Delivery, logger zerolog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "receive-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		ctx = logging.NewContextWithLogger(ctx, logger)
		logger.Info().Str("body", string(msg.Body)).Msg("received message")

		evt := events.MessageAccepted{}

		err = json.Unmarshal(msg.Body, &evt)
		if err != nil {
			logger.Error().Err(err).Msg("unable to unmarshal incoming message")
			return
		}

		err = evt.Error()
		if err != nil {
			logger.Warn().Err(err).Msg("received malformed topic message")
			return
		}

		err = app.MessageAccepted(ctx, evt, messenger)
	}
}

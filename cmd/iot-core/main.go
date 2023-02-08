package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"

	"github.com/diwise/iot-core/internal/pkg/application"
	"github.com/diwise/iot-core/internal/pkg/application/features"
	"github.com/diwise/iot-core/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/cors"
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

	flag.StringVar(&featuresConfigPath, "features", "", "configuration file for features")
	flag.Parse()

	dmURL := env.GetVariableOrDie(logger, "DEV_MGMT_URL", "url to iot-device-mgmt")
	tokenURL := env.GetVariableOrDie(logger, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	clientID := env.GetVariableOrDie(logger, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	clientSecret := env.GetVariableOrDie(logger, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")

	dmClient, err := client.New(ctx, dmURL, tokenURL, clientID, clientSecret)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create device managagement client")
	}

	m := messageprocessor.NewMessageProcessor(dmClient)

	config := messaging.LoadConfiguration(serviceName, logger)
	messenger, err := messaging.Initialize(config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init messaging")
	}

	app := application.NewIoTCoreApp(serviceName, m, logger)

	needToDecideThis := "application/json"
	messenger.RegisterCommandHandler(needToDecideThis, newCommandHandler(messenger, m, app))

	if featuresConfigPath != "" {
		configFile, err := os.Open(featuresConfigPath)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to open features config file")
		}
		defer configFile.Close()

		freg, err := features.NewRegistry(configFile)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create features registry")
		}

		routingKey := "message.accepted"
		messenger.RegisterTopicMessageHandler(routingKey, newTopicMessageHandler(messenger, app, freg))
	}

	setupRouterAndWaitForConnections(logger)
}

func newCommandHandler(messenger messaging.MsgContext, m messageprocessor.MessageProcessor, app application.App) messaging.CommandHandler {
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

func newTopicMessageHandler(messenger messaging.MsgContext, app application.App, freg features.Registry) messaging.TopicMessageHandler {

	return func(ctx context.Context, msg amqp.Delivery, logger zerolog.Logger) {
		ctx = logging.NewContextWithLogger(ctx, logger)
		logger.Info().Str("body", string(msg.Body)).Msg("received message")

		messageAccepted := &events.MessageAccepted{}
		if err := json.Unmarshal(msg.Body, messageAccepted); err == nil {
			matchingFeatures, _ := freg.Find(ctx, messageAccepted.Sensor)
			for _, f := range matchingFeatures {
				f.Handle(ctx, messageAccepted, messenger)
			}
		} else {
			logger.Error().Err(err).Msg("unable to unmarshal incoming message")
		}
	}
}

func setupRouterAndWaitForConnections(logger zerolog.Logger) {
	r := chi.NewRouter()
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start router")
	}
}

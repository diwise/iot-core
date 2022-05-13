package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/diwise/iot-core/internal/application"
	"github.com/diwise/iot-core/internal/messageprocessor"
	"github.com/diwise/iot-core/internal/pkg/domain"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

const serviceName string = "iot-core"

var tracer = otel.Tracer(serviceName)

func main() {
	serviceVersion := buildinfo.SourceVersion()
	_, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	dmURL := env.GetVariableOrDie(logger, "DEV_MGMT_URL", "url to iot-device-mgmt")
	dmClient := domain.NewDeviceManagementClient(dmURL)
	m := messageprocessor.NewMessageProcessor(dmClient)

	config := messaging.LoadConfiguration(serviceName, logger)
	messenger, err := messaging.Initialize(config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init messaging")
	}

	app := application.NewIoTCoreApp(serviceName, m, logger)

	needToDecideThis := "application/json"
	messenger.RegisterCommandHandler(needToDecideThis, newCommandHandler(messenger, m, app))

	setupRouterAndWaitForConnections(logger)
}

func newCommandHandler(messenger messaging.MsgContext, m messageprocessor.MessageProcessor, app application.IoTCoreApp) messaging.CommandHandler {
	return func(ctx context.Context, wrapper messaging.CommandMessageWrapper, logger zerolog.Logger) error {
		var err error

		ctx, span := tracer.Start(ctx, "rcv-cmd")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		rcvdMsg := events.MessageReceived{}
		err = json.Unmarshal(wrapper.Body(), &rcvdMsg)
		if err != nil {
			logger.Error().Err(err).Msg("failed to decode message from json")
			return err
		}

		e, err := app.MessageAccepted(ctx, rcvdMsg)
		if err != nil {
			return err
		}

		logger.Info().Msgf("publishing message to %s", e.TopicName())
		err = messenger.PublishOnTopic(ctx, e)
		if err != nil {
			logger.Error().Err(err).Msg("failed to publish message")
			return err
		}

		return nil
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

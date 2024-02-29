package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/diwise/iot-core/internal/pkg/application"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/iot-core/internal/pkg/presentation/api"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

const serviceName string = "iot-core"

var tracer = otel.Tracer(serviceName)
var functionsConfigPath string

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, _, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	flag.StringVar(&functionsConfigPath, "functions", "/opt/diwise/config/functions.csv", "configuration file for functions")
	flag.Parse()

	var err error

	dmClient := createDeviceManagementClientOrDie(ctx)
	defer dmClient.Close(ctx)

	msgCtx := createMessagingContextOrDie(ctx)
	defer msgCtx.Close()

	storage := createDatabaseConnectionOrDie(ctx)

	var configFile *os.File

	if functionsConfigPath != "" {
		configFile, err = os.Open(functionsConfigPath)
		if err != nil {
			fatal(ctx, "failed to open functions config file", err)
		}
		defer configFile.Close()
	}

	_, api_, err := initialize(ctx, dmClient, msgCtx, configFile, storage)
	if err != nil {
		fatal(ctx, "initialization failed", err)
	}

	servicePort := env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080")
	err = http.ListenAndServe(":"+servicePort, api_.Router())
	if err != nil {
		fatal(ctx, "failed to start request router", err)
	}
}

func createDeviceManagementClientOrDie(ctx context.Context) client.DeviceManagementClient {
	dmURL := env.GetVariableOrDie(ctx, "DEV_MGMT_URL", "url to iot-device-mgmt")
	tokenURL := env.GetVariableOrDie(ctx, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	clientID := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	clientSecret := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")

	dmClient, err := client.New(ctx, dmURL, tokenURL, clientID, clientSecret)
	if err != nil {
		fatal(ctx, "failed to create device managagement client", err)
	}

	return dmClient
}

func createMessagingContextOrDie(ctx context.Context) messaging.MsgContext {
	logger := logging.GetFromContext(ctx)

	config := messaging.LoadConfiguration(ctx, serviceName, logger)
	messenger, err := messaging.Initialize(ctx, config)
	if err != nil {
		fatal(ctx, "failed to init messaging", err)
	}
	messenger.Start()

	return messenger
}

func createDatabaseConnectionOrDie(ctx context.Context) database.Storage {
	storage, err := database.Connect(ctx, database.LoadConfiguration(ctx))
	if err != nil {
		fatal(ctx, "database connect failed", err)
	}
	err = storage.Initialize(ctx)
	if err != nil {
		fatal(ctx, "database initialize failed", err)
	}
	return storage
}

func initialize(ctx context.Context, dmClient client.DeviceManagementClient, msgctx messaging.MsgContext, fconfig io.Reader, storage database.Storage) (application.App, api.API, error) {
	msgproc := messageprocessor.NewMessageProcessor(dmClient)

	functionsRegistry, err := functions.NewRegistry(ctx, fconfig, storage)
	if err != nil {
		return nil, nil, err
	}

	app := application.New(msgproc, functionsRegistry)

	needToDecideThis := "application/json"
	msgctx.RegisterCommandHandler(messaging.MatchContentType(needToDecideThis), newCommandHandler(msgctx, app))

	routingKey := "message.accepted"
	msgctx.RegisterTopicMessageHandler(routingKey, newTopicMessageHandler(msgctx, app))

	return app, api.New(ctx, functionsRegistry), nil
}

func newCommandHandler(messenger messaging.MsgContext, app application.App) messaging.CommandHandler {
	return func(ctx context.Context, wrapper messaging.IncomingCommand, logger *slog.Logger) error {
		var err error

		ctx, span := tracer.Start(ctx, "receive-command")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		evt := events.MessageReceived{}
		err = json.Unmarshal(wrapper.Body(), &evt)
		if err != nil {
			logger.Error("failed to decode message from json", "err", err.Error())
			return err
		}

		logger = logger.With(slog.String("device_id", evt.Device))
		ctx = logging.NewContextWithLogger(ctx, logger)

		messageAccepted, err := app.MessageReceived(ctx, evt)
		if err != nil {
			logger.Error("message not accepted", "err", err.Error())
			return err
		}

		logger.Info("publishing message", "topic", messageAccepted.TopicName())
		err = messenger.PublishOnTopic(ctx, messageAccepted)
		if err != nil {
			logger.Error("failed to publish message", "err", err.Error())
			return err
		}

		return nil
	}
}

func newTopicMessageHandler(messenger messaging.MsgContext, app application.App) messaging.TopicMessageHandler {
	return func(ctx context.Context, msg messaging.IncomingTopicMessage, logger *slog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "receive-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		logger.Debug("received message", "body", string(msg.Body()))

		evt := events.MessageAccepted{}

		err = json.Unmarshal(msg.Body(), &evt)
		if err != nil {
			logger.Error("unable to unmarshal incoming message", "err", err.Error())
			return
		}

		err = evt.Error()
		if err != nil {
			logger.Warn("received malformed topic message", "err", err.Error())
			return
		}

		logger = logger.With(slog.String("device_id", evt.DeviceID))
		ctx = logging.NewContextWithLogger(ctx, logger)

		err = app.MessageAccepted(ctx, evt, messenger)
		if err != nil {
			logger.Error("failed to handle message", "err", err.Error())
		}
	}
}

func fatal(ctx context.Context, msg string, err error) {
	logger := logging.GetFromContext(ctx)
	logger.Error(msg, "err", err.Error())
	os.Exit(1)
}

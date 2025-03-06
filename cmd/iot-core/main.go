package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/diwise/iot-core/internal/pkg/application"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"
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
	ctx, _, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion, "json")
	defer cleanup()

	flag.StringVar(&functionsConfigPath, "functions", "/opt/diwise/config/functions.csv", "configuration file for functions")
	flag.Parse()

	var err error

	dmClient := createDeviceManagementClientOrDie(ctx)
	defer dmClient.Close(ctx)

	measurementsClient := createMeasurementsClientOrDie(ctx)

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

	_, api_, err := initialize(ctx, dmClient, measurementsClient, msgCtx, configFile, storage)
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

func createMeasurementsClientOrDie(ctx context.Context) measurements.MeasurementsClient {
	dmURL := env.GetVariableOrDie(ctx, "MEASUREMENTS_URL", "url to measurements service")
	tokenURL := env.GetVariableOrDie(ctx, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	clientID := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	clientSecret := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")

	measurementsClient, err := measurements.NewMeasurementsClient(ctx, dmURL, tokenURL, clientID, clientSecret)
	if err != nil {
		fatal(ctx, "failed to create measurements client", err)
	}

	return measurementsClient
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

func initialize(ctx context.Context, dmClient client.DeviceManagementClient, mClient measurements.MeasurementsClient, msgctx messaging.MsgContext, fconfig io.Reader, storage database.Storage) (application.App, api.API, error) {
	functionsRegistry, err := functions.NewRegistry(ctx, fconfig, storage)
	if err != nil {
		return nil, nil, err
	}

	app := application.New(dmClient, mClient, functionsRegistry)

	msgctx.RegisterCommandHandler(func(m messaging.Message) bool {
		return strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m")
	}, newCommandHandler(msgctx, app))

	msgctx.RegisterTopicMessageHandler("message.accepted", newTopicMessageHandler(msgctx, app))
	msgctx.RegisterTopicMessageHandler("function.updated", newFunctionUpdatedTopicMessageHandler(msgctx))

	return app, api.New(ctx, functionsRegistry), nil
}

func newCommandHandler(messenger messaging.MsgContext, app application.App) messaging.CommandHandler {
	return func(ctx context.Context, wrapper messaging.IncomingCommand, logger *slog.Logger) error {
		var err error

		ctx, span := tracer.Start(ctx, "receive-command")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		evt := events.MessageReceived{}
		err = json.Unmarshal(wrapper.Body(), &evt)
		if err != nil {
			logger.Error("failed to decode message from json", "err", err.Error())
			return err
		}

		logger = logger.With(slog.String("device_id", evt.DeviceID()))
		ctx = logging.NewContextWithLogger(ctx, logger)

		m, err := app.MessageReceived(ctx, evt)
		if err != nil {
			if errors.Is(err, application.ErrCouldNotFindDevice) {
				logger.Debug("could not find device, message not accepted")
				return nil
			}

			logger.Error("message not accepted", "err", err.Error())
			return err
		}

		logger.Debug("publishing message", slog.String("device_id", m.DeviceID()), slog.String("object_id", m.ObjectID()), slog.String("topic", m.TopicName()))

		err = messenger.PublishOnTopic(ctx, m)
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
		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

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

		logger.Debug(fmt.Sprintf("handling topic message for %s with type %s and content-type %s", evt.DeviceID(), evt.ObjectID(), evt.ContentType()))

		logger = logger.With(slog.String("device_id", evt.DeviceID()), slog.String("object_id", evt.ObjectID()))
		ctx = logging.NewContextWithLogger(ctx, logger)

		err = app.MessageAccepted(ctx, evt, messenger)
		if err != nil {
			logger.Error("failed to handle message", "err", err.Error())
		}
	}
}

func newFunctionUpdatedTopicMessageHandler(messenger messaging.MsgContext) messaging.TopicMessageHandler {
	return func(ctx context.Context, msg messaging.IncomingTopicMessage, logger *slog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "receive-function.updated")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		err = functions.Transform(ctx, messenger, msg)
		if err != nil {
			logger.Error("failed to transform message", "err", err.Error())
		}
	}
}

func fatal(ctx context.Context, msg string, err error) {
	logger := logging.GetFromContext(ctx)
	logger.Error(msg, "err", err.Error())
	os.Exit(1)
}

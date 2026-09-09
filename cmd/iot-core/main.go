package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/diwise/iot-core/internal/application"
	"github.com/diwise/iot-core/internal/application/functions"
	"github.com/diwise/iot-core/internal/application/measurements"
	"github.com/diwise/iot-core/internal/infrastructure/database"
	"github.com/diwise/iot-core/internal/presentation/api"
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

const defaultFunctionsConfigPath = "/opt/diwise/config/functions.csv"

var functionsConfigPath string

func main() {
	ctx, flags := parseExternalConfig(context.Background(), defaultFlags())
	functionsConfigPath = flags[flagFunctionsPath]

	serviceVersion := buildinfo.SourceVersion()
	ctx, _, cleanup := o11y.Init(ctx, serviceName, serviceVersion, "json")
	defer cleanup()

	if err := run(ctx); err != nil {
		fatal(ctx, "iot-core failed", err)
	}
}

// run performs startup and serving, returning errors to main so that
// deferred cleanup of already-acquired resources runs before the
// process exit code is decided. Only main decides the exit code.
func run(ctx context.Context) error {
	var err error

	dmClient, err := createDeviceManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device management client: %w", err)
	}
	defer dmClient.Close(ctx)

	measurementsClient, err := createMeasurementsClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create measurements client: %w", err)
	}
	defer measurementsClient.Close()

	msgCtx, err := createMessagingContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to init messaging: %w", err)
	}
	defer msgCtx.Close()

	storage, err := createDatabaseConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer storage.Close()

	var configFile *os.File

	if functionsConfigPath != "" {
		configFile, err = os.Open(functionsConfigPath)
		if err != nil {
			return fmt.Errorf("failed to open functions config file: %w", err)
		}
		defer configFile.Close()
	}

	_, api_, err := initialize(ctx, dmClient, measurementsClient, msgCtx, configFile, storage)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	if err := http.ListenAndServe(":"+servicePort(ctx), api_.Router()); err != nil {
		return fmt.Errorf("failed to start request router: %w", err)
	}

	return nil
}

func requireEnv(ctx context.Context, key, description string) (string, error) {
	value := env.GetVariableOrDefault(ctx, key, "")
	if value == "" {
		return "", fmt.Errorf("missing required environment variable %s (%s)", key, description)
	}

	return value, nil
}

func createDeviceManagementClient(ctx context.Context) (client.DeviceManagementClient, error) {
	dmURL, err := requireEnv(ctx, "DEV_MGMT_URL", "url to iot-device-mgmt")
	if err != nil {
		return nil, err
	}
	tokenURL, err := requireEnv(ctx, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	if err != nil {
		return nil, err
	}
	clientID, err := requireEnv(ctx, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	if err != nil {
		return nil, err
	}
	clientSecret, err := requireEnv(ctx, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")
	if err != nil {
		return nil, err
	}

	return client.New(ctx, dmURL, tokenURL, oauthRealmInsecure(ctx), clientID, clientSecret)
}

// servicePort is the minimal production seam for the public server
// port: tests target this function, not the env library, so a changed
// default or lookup breaks them.
func servicePort(ctx context.Context) string {
	return env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080")
}

// oauthRealmInsecure is the minimal production seam for the shared
// TLS-verification toggle. Only the exact string "true" disables
// verification; every other value (including "TRUE" or "1") keeps it.
func oauthRealmInsecure(ctx context.Context) bool {
	return env.GetVariableOrDefault(ctx, "OAUTH2_REALM_INSECURE", "false") == "true"
}

func createMeasurementsClient(ctx context.Context) (measurements.MeasurementsClient, error) {
	measurementsURL, err := requireEnv(ctx, "MEASUREMENTS_URL", "url to measurements service")
	if err != nil {
		return nil, err
	}
	tokenURL, err := requireEnv(ctx, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	if err != nil {
		return nil, err
	}
	clientID, err := requireEnv(ctx, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	if err != nil {
		return nil, err
	}
	clientSecret, err := requireEnv(ctx, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")
	if err != nil {
		return nil, err
	}

	return measurements.NewMeasurementsClient(ctx, measurementsURL, tokenURL, clientID, clientSecret, oauthRealmInsecure(ctx))
}

func createMessagingContext(ctx context.Context) (messaging.MsgContext, error) {
	logger := logging.GetFromContext(ctx)

	config := messaging.LoadConfiguration(ctx, serviceName, logger)
	messenger, err := messaging.Initialize(ctx, config)
	if err != nil {
		return nil, err
	}
	messenger.Start()

	return messenger, nil
}

func createDatabaseConnection(ctx context.Context) (database.Storage, error) {
	storage, err := database.Connect(ctx, database.LoadConfiguration(ctx))
	if err != nil {
		return nil, fmt.Errorf("database connect failed: %w", err)
	}
	if err := storage.Initialize(ctx); err != nil {
		storage.Close()
		return nil, fmt.Errorf("database initialize failed: %w", err)
	}

	return storage, nil
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

		logger.Debug("message.received", "device_id", evt.DeviceID(), "object_id", evt.ObjectID())

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

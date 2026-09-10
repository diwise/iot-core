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
	"sync"

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
	k8shandlers "github.com/diwise/service-chassis/pkg/infrastructure/net/http/handlers"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/diwise/service-chassis/pkg/infrastructure/servicerunner"
	"go.opentelemetry.io/otel"
)

const serviceName string = "iot-core"

var tracer = otel.Tracer(serviceName)

const defaultFunctionsConfigPath = "/opt/diwise/config/functions.csv"

func main() {
	ctx, flags := parseExternalConfig(context.Background(), defaultFlags())

	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(ctx, serviceName, serviceVersion, "json")
	defer cleanup()

	logging.SetLogLevel(parseLogLevel(flags[flagLogLevel]))

	// functions.csv öppnas i main och ägs av initialize: OnInit bygger
	// registret ur den och stänger den därefter. Tom sökväg betyder
	// inget filinnehåll, samma semantik som tidigare run().
	var configFile *os.File

	if path := flags[flagFunctionsPath]; path != "" {
		f, err := os.Open(path)
		exitIf(err, logger, "failed to open functions config file")
		configFile = f
	}

	cfg := loadServiceConfig(flags)

	runner, err := initialize(ctx, flags, &cfg, configFile)
	exitIf(err, logger, "failed to initialize service runner")

	err = runner.Run(ctx)
	exitIf(err, logger, "failed to start service runner")
}

// initialize bygger servicerunnern: konstruktion i OnInit,
// messaging-start och handlerregistrering i OnStarting, deterministisk
// stängning i OnShutdown. Kontrollservern bär endast liveness här;
// readiness-stubbar införs i CORE-005. Den publika servern behåller
// befintliga routes via registerPublicRoutes.
func initialize(ctx context.Context, flags flagMap, cfg *serviceConfig, fconfig io.Reader) (servicerunner.Runner[serviceConfig], error) {
	logger := logging.GetFromContext(ctx)

	var dmClient client.DeviceManagementClient
	var measurementsClient measurements.MeasurementsClient
	var messenger messaging.MsgContext
	var storage database.Storage
	var app application.App
	var api_ api.API

	owned := &ownedResources{}

	_, runner := servicerunner.New(ctx, *cfg,
		webserver("control", listen(flags[flagListenAddress]), port(flags[flagControlPort]),
			pprof(), liveness(func() error { return nil }), readiness(readinessProbes()),
		),
		webserver("public", listen(flags[flagListenAddress]), port(flags[flagServicePort]), withTracing(tracingEnabled(flags)),
			muxinit(func(ctx context.Context, identifier string, port string, cfg *serviceConfig, handler *http.ServeMux) error {
				return registerPublicRoutes(handler, &api_)
			}),
		),
		oninit(func(ctx context.Context, cfg *serviceConfig) error {
			logger.Debug("initializing servicerunner")

			if c, ok := fconfig.(io.Closer); ok {
				defer c.Close()
			}

			var err error

			dmClient, err = createDeviceManagementClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create device management client: %w", err)
			}
			owned.dmClient = dmClient

			measurementsClient, err = createMeasurementsClient(ctx)
			if err != nil {
				owned.close(ctx)
				return fmt.Errorf("failed to create measurements client: %w", err)
			}
			owned.measurementsClient = measurementsClient

			messenger, err = createMessagingContext(ctx)
			if err != nil {
				owned.close(ctx)
				return fmt.Errorf("failed to init messaging: %w", err)
			}
			owned.messenger = messenger

			storage, err = createDatabaseConnection(ctx)
			if err != nil {
				owned.close(ctx)
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			owned.storage = storage

			app, api_, err = buildApplication(ctx, dmClient, measurementsClient, fconfig, storage)
			if err != nil {
				owned.close(ctx)
				return fmt.Errorf("initialization failed: %w", err)
			}

			return nil
		}),
		onstarting(func(ctx context.Context, cfg *serviceConfig) (err error) {
			logger.Debug("starting servicerunner")

			// OnStarting failures bypass OnShutdown in the runner, so
			// clean up acquired resources on error below.
			defer func() {
				if err != nil {
					owned.close(ctx)
				}
			}()

			if err := messenger.Start(ctx); err != nil {
				return fmt.Errorf("failed to start messenger: %w", err)
			}

			if err := registerHandlers(messenger, app); err != nil {
				return err
			}

			return nil
		}),
		onshutdown(func(ctx context.Context, cfg *serviceConfig) error {
			logger.Debug("shutting down servicerunner")

			owned.close(ctx)

			return nil
		}),
	)

	return runner, nil
}

// readinessProbes returns the named readiness stubs. Per harmonization
// standard they always report OK and never call RabbitMQ, PostgreSQL,
// device management, measurements, OAuth or the function registry.
// Probe names match the sibling services for external probe definitions.
func readinessProbes() map[string]k8shandlers.ServiceProber {
	return map[string]k8shandlers.ServiceProber{
		"rabbitmq":  func(context.Context) (string, error) { return "ok", nil },
		"timescale": func(context.Context) (string, error) { return "ok", nil },
	}
}

// registerPublicRoutes monterar funktions-API:t på runnerns publika mux
// utan att ändra någon route. Monteringen är lat: runnern bygger muxar
// redan vid konstruktion, före OnInit, så vidarebefordran slås upp per
// request när api_ är byggt.
func registerPublicRoutes(handler *http.ServeMux, api_ *api.API) error {
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		(*api_).Router().ServeHTTP(w, r)
	})
	return nil
}

// ownedResources tracks the resources created during OnInit so shutdown
// stops inflow before clients and storage, exactly once. Shutdown is
// nil-safe (partial OnInit) and idempotent via the sync.Once guard, so
// the messenger is shut down at most once.
type ownedResources struct {
	once               sync.Once
	messenger          messaging.MsgContext
	measurementsClient measurements.MeasurementsClient
	dmClient           client.DeviceManagementClient
	storage            database.Storage
}

func (o *ownedResources) close(ctx context.Context) {
	o.once.Do(func() {
		if o.messenger != nil {
			if err := o.messenger.Shutdown(ctx); err != nil {
				logging.GetFromContext(ctx).Debug("failed to shut down messenger", "err", err.Error())
			}
		}
		if o.measurementsClient != nil {
			o.measurementsClient.Close()
		}
		if o.dmClient != nil {
			o.dmClient.Close(ctx)
		}
		if o.storage != nil {
			o.storage.Close()
		}
	})
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
// default or lookup breaks them. Since CORE-004 the runner reads the
// port from flags (same env name and default); this seam remains as the
// env-contract lock.
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

	config, err := messaging.LoadConfiguration(ctx, serviceName, logger)
	if err != nil {
		return nil, fmt.Errorf("messaging configuration error: %w", err)
	}
	messenger, err := messaging.Initialize(ctx, config)
	if err != nil {
		return nil, err
	}

	// Start sker i OnStarting, efter att handlerregistrering kan
	// felkontrolleras; tidigare startade fabriken loopen direkt.
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

// buildApplication constructs the function registry, application and API
// without touching the network. Messaging start and handler registration
// live in OnStarting via registerHandlers so failures surface with context.
func buildApplication(ctx context.Context, dmClient client.DeviceManagementClient, mClient measurements.MeasurementsClient, fconfig io.Reader, storage database.Storage) (application.App, api.API, error) {
	functionsRegistry, err := functions.NewRegistry(ctx, fconfig, storage)
	if err != nil {
		return nil, nil, err
	}

	app := application.New(dmClient, mClient, functionsRegistry)

	return app, api.New(ctx, functionsRegistry), nil
}

// registerHandlers registers the agent command handler and the topic
// handlers with unchanged filters, routing keys and payload handling.
// Every registration error aborts startup.
func registerHandlers(msgctx messaging.MsgContext, app application.App) error {
	if err := msgctx.RegisterCommandHandler(func(m messaging.Message) bool {
		return strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m")
	}, newCommandHandler(msgctx, app)); err != nil {
		return fmt.Errorf("failed to register command handler: %w", err)
	}

	if err := msgctx.RegisterTopicMessageHandler("message.accepted", newTopicMessageHandler(msgctx, app)); err != nil {
		return fmt.Errorf("failed to register message.accepted handler: %w", err)
	}

	if err := msgctx.RegisterTopicMessageHandler("function.updated", newFunctionUpdatedTopicMessageHandler(msgctx)); err != nil {
		return fmt.Errorf("failed to register function.updated handler: %w", err)
	}

	return nil
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
	return func(ctx context.Context, msg messaging.IncomingTopicMessage, logger *slog.Logger) error {
		var err error

		ctx, span := tracer.Start(ctx, "receive-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		evt := events.MessageAccepted{}

		err = json.Unmarshal(msg.Body(), &evt)
		if err != nil {
			logger.Error("unable to unmarshal incoming message", "err", err.Error())
			return messaging.Permanent(err)
		}

		err = evt.Error()
		if err != nil {
			logger.Warn("received malformed topic message", "err", err.Error())
			return messaging.Permanent(err)
		}

		logger.Debug(fmt.Sprintf("handling topic message for %s with type %s and content-type %s", evt.DeviceID(), evt.ObjectID(), evt.ContentType()))

		logger = logger.With(slog.String("device_id", evt.DeviceID()), slog.String("object_id", evt.ObjectID()))
		ctx = logging.NewContextWithLogger(ctx, logger)

		// Bevarad semantik: hanteringsfel loggas och ackas. Klassificering
		// till Temporary/Permanent kräver verifierad idempotens (TODO steg 1.2).
		err = app.MessageAccepted(ctx, evt, messenger)
		if err != nil {
			logger.Error("failed to handle message", "err", err.Error())
		}
		return nil
	}
}

func newFunctionUpdatedTopicMessageHandler(messenger messaging.MsgContext) messaging.TopicMessageHandler {
	return func(ctx context.Context, msg messaging.IncomingTopicMessage, logger *slog.Logger) error {
		var err error

		ctx, span := tracer.Start(ctx, "receive-function.updated")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		// Bevarad semantik: transformeringsfel loggas och ackas, se ovan.
		err = functions.Transform(ctx, messenger, msg)
		if err != nil {
			logger.Error("failed to transform message", "err", err.Error())
		}
		return nil
	}
}

func exitIf(err error, logger *slog.Logger, msg string, args ...any) {
	if err != nil {
		logger.With(args...).Error(msg, "err", err.Error())
		os.Exit(1)
	}
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

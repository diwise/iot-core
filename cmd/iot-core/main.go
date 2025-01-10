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
	clientMock "github.com/diwise/iot-device-mgmt/pkg/test"
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

func defaultFlags() FlagMap {
	return FlagMap{
		listenAddress:   "0.0.0.0",
		servicePort:     "8080",
		controlPort:     "8000",
		configFilePath:  "/opt/diwise/config/functions.csv",
		devMgmtUrl:      "http://iot-device-mgmt:8080",
		measurementsUrl: "http://iot-events:8080",
		policiesFile:    "/opt/diwise/config/authz.rego",
		dbHost:          "",
		dbUser:          "",
		dbPassword:      "",
		dbPort:          "5432",
		dbName:          "",
		dbSslMode:       "disable",
		devMode:         "false",
	}
}

const (
	serviceName string = "iot-core"
)

var tracer = otel.Tracer(serviceName)

func main() {
	ctx, flags := parseExternalConfig(context.Background(), defaultFlags())

	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(ctx, serviceName, serviceVersion, "json")
	defer cleanup()

	configFile, err := os.Open(flags[configFilePath])
	exitIf(err, logger, "unable to open functions configuration file")
	defer configFile.Close()

	policies, err := os.Open(flags[policiesFile])
	exitIf(err, logger, "unable to open opa policy file")
	defer policies.Close()

	storage, err := newStorage(ctx, flags)
	exitIf(err, logger, "failed to connect to database")

	registry, err := functions.NewRegistry(ctx, configFile, storage)
	exitIf(err, logger, "failed to create function registry")

	dmClient, err := newDevMgmtClient(ctx, flags)
	exitIf(err, logger, "failed to create device management client")
	defer dmClient.Close(ctx)

	measurementsClient, err := newMeasurementsClient(ctx, flags)
	exitIf(err, logger, "failed to create measurements client")

	msgCtx, err := messaging.Initialize(
		ctx, messaging.LoadConfiguration(ctx, serviceName, logger),
	)
	exitIf(err, logger, "failed to init messenger")

	cfg := &AppConfig{
		messenger:          msgCtx,
		devMgmtClient:      dmClient,
		measurementsClient: measurementsClient,
		storage:            storage,
		registry:           registry,
	}

	runner, _ := initialize(ctx, flags, cfg, policies)

	err = runner.Run(ctx)
	exitIf(err, logger, "failed to start service runner")
}

func newStorage(ctx context.Context, flags FlagMap) (database.Storage, error) {
	storage, err := database.Connect(ctx, database.NewConfig(flags[dbHost], flags[dbUser], flags[dbPassword], flags[dbPort], flags[dbName], flags[dbSslMode]))
	if err != nil {
		return nil, err
	}
	err = storage.Initialize(ctx)
	if err != nil {
		return nil, err
	}
	return storage, err
}

func newMeasurementsClient(ctx context.Context, flags FlagMap) (measurements.MeasurementsClient, error) {
	if flags[devMode] == "true" {
		return &measurements.MeasurementsClientMock{}, nil
	}

	mClient, err := measurements.NewMeasurementsClient(ctx, flags[measurementsUrl], flags[tokenUrl], flags[clientId], flags[clientSecret])
	return mClient, err
}

func newDevMgmtClient(ctx context.Context, flags FlagMap) (client.DeviceManagementClient, error) {
	if flags[devMode] == "true" {
		return &clientMock.DeviceManagementClientMock{}, nil
	}

	dmClient, err := client.New(ctx, flags[devMgmtUrl], flags[tokenUrl], flags[clientId], flags[clientSecret])
	return dmClient, err
}

func initialize(ctx context.Context, flags FlagMap, cfg *AppConfig, policies io.ReadCloser) (servicerunner.Runner[AppConfig], error) {
	probes := map[string]k8shandlers.ServiceProber{
		"rabbitmq":  func(context.Context) (string, error) { return "ok", nil },
		"timescale": func(context.Context) (string, error) { return "ok", nil },
	}

	app := application.New(cfg.devMgmtClient, cfg.measurementsClient, cfg.registry, cfg.messenger)

	_, runner := servicerunner.New(ctx, *cfg,
		webserver("control", listen(flags[listenAddress]), port(flags[controlPort]),
			pprof(), liveness(func() error { return nil }), readiness(probes),
		), onstarting(func(ctx context.Context, svcCfg *AppConfig) (err error) {
			return nil
		}),
		webserver("public", listen(flags[listenAddress]), port(flags[servicePort]),
			muxinit(func(ctx context.Context, identifier string, port string, svcCfg *AppConfig, handler *http.ServeMux) error {
				api.RegisterHandlers(ctx, serviceName, handler, app, policies)
				return nil
			}),
		),
		onstarting(func(ctx context.Context, svcCfg *AppConfig) error {
			svcCfg.messenger.Start()

			svcCfg.messenger.RegisterCommandHandler(func(m messaging.Message) bool {
				return strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m")
			}, newCommandHandler(svcCfg.messenger, app))

			svcCfg.messenger.RegisterTopicMessageHandler("message.accepted", newTopicMessageHandler(app))

			return nil
		}),
		onshutdown(func(ctx context.Context, svcCfg *AppConfig) error {
			svcCfg.messenger.Close()
			return nil
		}))

	return runner, nil
}

func parseExternalConfig(ctx context.Context, flags FlagMap) (context.Context, FlagMap) {
	// Allow environment variables to override certain defaults
	envOrDef := env.GetVariableOrDefault

	flags[controlPort] = envOrDef(ctx, "CONTROL_PORT", flags[controlPort])
	flags[servicePort] = envOrDef(ctx, "SERVICE_PORT", flags[servicePort])
	flags[devMgmtUrl] = envOrDef(ctx, "DEV_MGMT_URL", flags[devMgmtUrl])
	flags[measurementsUrl] = envOrDef(ctx, "MEASUREMENTS_URL", flags[measurementsUrl])
	flags[tokenUrl] = envOrDef(ctx, "OAUTH2_TOKEN_URL", flags[tokenUrl])
	flags[clientId] = envOrDef(ctx, "OAUTH2_CLIENT_ID", flags[clientId])
	flags[clientSecret] = envOrDef(ctx, "OAUTH2_CLIENT_SECRET", flags[clientSecret])
	flags[dbHost] = envOrDef(ctx, "POSTGRES_HOST", flags[dbHost])
	flags[dbUser] = envOrDef(ctx, "POSTGRES_USER", flags[dbUser])
	flags[dbPassword] = envOrDef(ctx, "POSTGRES_PASSWORD", flags[dbPassword])
	flags[dbPort] = envOrDef(ctx, "POSTGRES_PORT", flags[dbPort])
	flags[dbName] = envOrDef(ctx, "POSTGRES_DBNAME", flags[dbName])
	flags[dbSslMode] = envOrDef(ctx, "POSTGRES_SSLMODE", flags[dbSslMode])

	apply := func(f FlagType) func(string) error {
		return func(value string) error {
			flags[f] = value
			return nil
		}
	}

	// Allow command line arguments to override defaults and environment variables
	flag.Func("functions", "path to functions configuration file", apply(configFilePath))
	flag.Func("policies", "an authorization policy file", apply(policiesFile))
	flag.BoolFunc("devmode", "run in development mode", apply(devMode))
	flag.Parse()

	return ctx, flags
}

func exitIf(err error, logger *slog.Logger, msg string, args ...any) {
	if err != nil {
		logger.With(args...).Error(msg, "err", err.Error())
		os.Exit(1)
	}
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

func newTopicMessageHandler(app application.App) messaging.TopicMessageHandler {
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

		err = app.MessageAccepted(ctx, evt)
		if err != nil {
			logger.Error("failed to handle message", "err", err.Error())
		}
	}
}

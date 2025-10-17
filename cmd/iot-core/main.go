package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application"
	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/functions/engines"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/repository"
	"github.com/diwise/iot-core/internal/pkg/presentation/api"
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

func defaultFlags() flagMap {
	return flagMap{
		listenAddress: "0.0.0.0",
		servicePort:   "8080",
		controlPort:   "8000",

		functionsFile:       "/opt/diwise/config/functions.csv",
		deviceManagementUrl: "http://iot-device-mgmt",
		measurementsUrl:     "http://iot-events",

		oauth2TokenUrl:     "",
		oauth2ClientId:     "",
		oauth2ClientSecret: "",
		oauth2InsecureURL:  "true",

		dbHost:     "",
		dbUser:     "",
		dbPassword: "",
		dbPort:     "5432",
		dbName:     "diwise",
		dbSSLMode:  "disable",
	}
}

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion, "json")
	defer cleanup()

	ctx, flags := parseExternalConfig(ctx, defaultFlags())

	cfg := &appConfig{}

	runner, err := initialize(ctx, flags, cfg)
	exitIf(err, logger, "failed to initialize service runner")

	err = runner.Run(ctx)
	exitIf(err, logger, "failed to start service runner")
}

func initialize(ctx context.Context, flags flagMap, cfg *appConfig) (servicerunner.Runner[appConfig], error) {
	probes := map[string]k8shandlers.ServiceProber{
		"rabbitmq": func(context.Context) (string, error) { return "ok", nil },
	}

	log := logging.GetFromContext(ctx)

	var err error
	var dmClient client.DeviceManagementClient
	var msgCtx messaging.MsgContext
	var mClient measurements.MeasurementsClient
	var ruleStorage rules.Storage
	var funcStorage database.FuncStorage
	var funcRegistry functions.FuncRegistry
	var app application.App

	_, runner := servicerunner.New(ctx, *cfg,
		webserver("control", listen(flags[listenAddress]), port(flags[controlPort]),
			pprof(), liveness(func() error { return nil }), readiness(probes),
		),
		webserver("public", listen(flags[listenAddress]), port(flags[servicePort]), withtracing(true),
			muxinit(func(ctx context.Context, identifier string, port string, appCfg *appConfig, handler *http.ServeMux) error {
				api.RegisterHandlers(ctx, handler, app)
				return nil
			}),
		),
		oninit(func(ctx context.Context, ac *appConfig) error {
			dmClient, err = client.New(ctx, flags[deviceManagementUrl], flags[oauth2TokenUrl], flags[oauth2InsecureURL] == "true", flags[oauth2ClientId], flags[oauth2ClientSecret])
			if err != nil {
				return err
			}

			config := messaging.LoadConfiguration(ctx, serviceName, log)
			msgCtx, err = messaging.Initialize(ctx, config)
			if err != nil {
				return err
			}

			mClient, err = measurements.NewMeasurementsClient(ctx, flags[measurementsUrl], flags[oauth2TokenUrl], flags[oauth2ClientId], flags[oauth2ClientSecret])
			if err != nil {
				return err
			}

			dbConfig := database.NewConfig(flags[dbHost], flags[dbUser], flags[dbPassword], flags[dbPort], flags[dbName], flags[dbSSLMode])
			conn, err := database.GetConnection(ctx, dbConfig)

			if err != nil {
				return err
			}

			ruleStorage = rules.Connect(conn)
			ruleRepository := repository.New(ruleStorage)

			funcStorage = database.Connect(conn)

			f, _ := os.Open(flags[functionsFile])
			if f != nil {
				defer f.Close()
			}

			funcRegistry, err = functions.NewFuncRegistry(ctx, f, funcStorage)
			if err != nil {
				return err
			}

			ruleEngine := engines.New(ruleRepository)

			app = application.New(dmClient, mClient, funcRegistry, ruleEngine, msgCtx)

			return nil
		}),
		onstarting(func(ctx context.Context, svcCfg *appConfig) error {
			msgCtx.Start()

			msgCtx.RegisterCommandHandler(func(m messaging.Message) bool {
				return strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m")
			}, newMessageReceivedCommandHandler(msgCtx, app))

			msgCtx.RegisterTopicMessageHandler("message.accepted", newMessageAcceptedHandler(app))
			msgCtx.RegisterTopicMessageHandler("function.updated", newFunctionUpdatedTopicMessageHandler(app))

			err = funcStorage.Initialize(ctx)
			if err != nil {
				return err
			}

			return nil
		}),
		onshutdown(func(ctx context.Context, svcCfg *appConfig) error {
			dmClient.Close(ctx)
			msgCtx.Close()

			return nil
		}))

	return runner, nil
}

func parseExternalConfig(ctx context.Context, flags flagMap) (context.Context, flagMap) {
	// Allow environment variables to override certain defaults
	envOrDef := env.GetVariableOrDefault

	flags[servicePort] = envOrDef(ctx, "SERVICE_PORT", flags[servicePort])
	flags[controlPort] = envOrDef(ctx, "CONTROL_PORT", flags[controlPort])

	flags[deviceManagementUrl] = envOrDef(ctx, "DEV_MGMT_URL", flags[deviceManagementUrl])

	flags[oauth2TokenUrl] = envOrDef(ctx, "OAUTH2_TOKEN_URL", flags[oauth2TokenUrl])
	flags[oauth2ClientId] = envOrDef(ctx, "OAUTH2_CLIENT_ID", flags[oauth2ClientId])
	flags[oauth2ClientSecret] = envOrDef(ctx, "OAUTH2_CLIENT_SECRET", flags[oauth2ClientSecret])
	flags[oauth2InsecureURL] = envOrDef(ctx, "OAUTH2_REALM_INSECURE", flags[oauth2InsecureURL])

	flags[dbHost] = envOrDef(ctx, "POSTGRES_HOST", flags[dbHost])
	flags[dbPort] = envOrDef(ctx, "POSTGRES_PORT", flags[dbPort])
	flags[dbName] = envOrDef(ctx, "POSTGRES_DBNAME", flags[dbName])
	flags[dbUser] = envOrDef(ctx, "POSTGRES_USER", flags[dbUser])
	flags[dbPassword] = envOrDef(ctx, "POSTGRES_PASSWORD", flags[dbPassword])
	flags[dbSSLMode] = envOrDef(ctx, "POSTGRES_SSLMODE", flags[dbSSLMode])

	apply := func(f flagType) func(string) error {
		return func(value string) error {
			flags[f] = value
			return nil
		}
	}

	flag.Func("functions", "configuration file for functions", apply(functionsFile))

	flag.Parse()

	return ctx, flags
}

func newMessageReceivedCommandHandler(messenger messaging.MsgContext, app application.App) messaging.CommandHandler {
	return func(ctx context.Context, wrapper messaging.IncomingCommand, logger *slog.Logger) error {
		var err error

		ctx, span := tracer.Start(ctx, "receive-command")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		evt := events.MessageReceived{}
		err = json.Unmarshal(wrapper.Body(), &evt)
		if err != nil {
			logger.Error("failed to unmarshal message.received command from json", "err", err.Error())
			return err
		}

		logger = logger.With(slog.String("device_id", evt.DeviceID())).With(slog.String("object_id", evt.ObjectID()))
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

		err = messenger.PublishOnTopic(ctx, m)
		if err != nil {
			logger.Error("failed to publish message", "err", err.Error())
			return err
		}

		logger.Debug("received message accepted", slog.String("content_type", m.ContentType()), slog.String("topic", m.TopicName()))

		return nil
	}
}

func newMessageAcceptedHandler(app application.App) messaging.TopicMessageHandler {
	return func(ctx context.Context, msg messaging.IncomingTopicMessage, logger *slog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "accept-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		evt := events.MessageAccepted{}

		err = json.Unmarshal(msg.Body(), &evt)
		if err != nil {
			logger.Error("unable to unmarshal incoming topic message message.accepted", "err", err.Error())
			return
		}

		logger = logger.With(slog.String("device_id", evt.DeviceID()), slog.String("object_id", evt.ObjectID()))
		ctx = logging.NewContextWithLogger(ctx, logger)

		err = evt.Error()
		if err != nil {
			logger.Warn("received malformed message.accepted message", "err", err.Error())
			return
		}

		err = app.MessageAccepted(ctx, evt)
		if err != nil {
			logger.Error("failed to handle message", "err", err.Error())
		}

		logger.Debug("message.accepted handled", slog.String("content_type", evt.ContentType()), slog.String("topic", evt.TopicName()))
	}
}

func newFunctionUpdatedTopicMessageHandler(app application.App) messaging.TopicMessageHandler {
	return func(ctx context.Context, msg messaging.IncomingTopicMessage, logger *slog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "receive-function.updated")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		err = app.FunctionUpdated(ctx, msg.Body())
		if err != nil {
			logger.Error("failed to transform message", "err", err.Error())
		}
	}
}

func exitIf(err error, logger *slog.Logger, msg string, args ...any) {
	if err != nil {
		logger.With(args...).Error(msg, "err", err.Error())
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
}

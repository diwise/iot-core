package main

import (
	"context"
	"flag"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/servicerunner"
)

// CORE-003: privat typad config ägd av cmd. FlagMap bär de externa
// strängvärdena (default < env < CLI) med oförändrade env-namn och
// defaults; serviceConfig grupperar dem typat för konstruktion.
// Factories behåller sina signaturer i denna uppgift; CORE-004
// kopplar konfigurationen till servicerunner och injicering.

type flagType int
type flagMap map[flagType]string

const (
	flagServicePort flagType = iota
	flagDevMgmtURL
	flagMeasurementsURL
	flagOAuthTokenURL
	flagOAuthClientID
	flagOAuthClientSecret
	flagOAuthInsecure
	flagFunctionsPath

	flagListenAddress
	flagControlPort
	flagEnableTracing
)

// serverConfig grupperar den publika serverns inställningar.
type serverConfig struct {
	port string
}

// oauthConfig grupperar OAuth2-klientens inställningar.
type oauthConfig struct {
	tokenURL     string
	clientID     string
	clientSecret string
	insecure     bool
}

// deviceMgmtConfig grupperar device management-klientens inställningar.
type deviceMgmtConfig struct {
	url string
}

// measurementsConfig grupperar measurements-klientens inställningar.
type measurementsConfig struct {
	url string
}

// serviceConfig är den typade samlingen av allt cmd äger.
// Storage- och messaging-bibliotekens egna Config laddas även
// fortsättningsvis via deras LoadConfiguration i factories.
type serviceConfig struct {
	server        serverConfig
	oauth         oauthConfig
	deviceMgmt    deviceMgmtConfig
	measurements  measurementsConfig
	functionsPath string
}

func defaultFlags() flagMap {
	return flagMap{
		flagServicePort:       "8080",
		flagDevMgmtURL:        "",
		flagMeasurementsURL:   "",
		flagOAuthTokenURL:     "",
		flagOAuthClientID:     "",
		flagOAuthClientSecret: "",
		flagOAuthInsecure:     "false",
		flagFunctionsPath:     defaultFunctionsConfigPath,

		flagListenAddress: "0.0.0.0",
		flagControlPort:   "8000",
		flagEnableTracing: "true",
	}
}

var oninit = servicerunner.OnInit[serviceConfig]
var onstarting = servicerunner.OnStarting[serviceConfig]
var onshutdown = servicerunner.OnShutdown[serviceConfig]
var webserver = servicerunner.WithHTTPServeMux[serviceConfig]
var muxinit = servicerunner.OnMuxInit[serviceConfig]
var listen = servicerunner.WithListenAddr[serviceConfig]
var port = servicerunner.WithPort[serviceConfig]
var pprof = servicerunner.WithPPROF[serviceConfig]
var liveness = servicerunner.WithK8SLivenessProbe[serviceConfig]

// withTracing bär servicerunners tracing-wrapper. Namnet avviker från
// syskontjänsternas `tracing` eftersom main även importerar
// o11y/tracing för handlerspans.
var withTracing = servicerunner.WithTracing[serviceConfig]

// tracingEnabled is the minimal production seam for the tracing toggle.
// Only the exact string "true" enables tracing; ParseBool spellings
// such as "TRUE" or "1" intentionally do not.
func tracingEnabled(flags flagMap) bool {
	return flags[flagEnableTracing] == "true"
}

func parseExternalConfig(ctx context.Context, flags flagMap) (context.Context, flagMap) {
	envOrDef := env.GetVariableOrDefault

	flags[flagServicePort] = envOrDef(ctx, "SERVICE_PORT", flags[flagServicePort])
	flags[flagDevMgmtURL] = envOrDef(ctx, "DEV_MGMT_URL", flags[flagDevMgmtURL])
	flags[flagMeasurementsURL] = envOrDef(ctx, "MEASUREMENTS_URL", flags[flagMeasurementsURL])
	flags[flagOAuthTokenURL] = envOrDef(ctx, "OAUTH2_TOKEN_URL", flags[flagOAuthTokenURL])
	flags[flagOAuthClientID] = envOrDef(ctx, "OAUTH2_CLIENT_ID", flags[flagOAuthClientID])
	flags[flagOAuthClientSecret] = envOrDef(ctx, "OAUTH2_CLIENT_SECRET", flags[flagOAuthClientSecret])
	flags[flagOAuthInsecure] = envOrDef(ctx, "OAUTH2_REALM_INSECURE", flags[flagOAuthInsecure])
	// functions.csv-sökvägen styrs idag endast av CLI-flaggan
	// -functions (defaults i defaultFunctionsConfigPath); inget
	// env-namn läses för den i denna uppgift.

	// CORE-004: nya externa ytor för servicerunner. Dokumenteras som
	// extern påverkan i CORE-007.
	flags[flagListenAddress] = envOrDef(ctx, "LISTEN_ADDRESS", flags[flagListenAddress])
	flags[flagControlPort] = envOrDef(ctx, "CONTROL_PORT", flags[flagControlPort])
	flags[flagEnableTracing] = envOrDef(ctx, "ENABLE_TRACING", flags[flagEnableTracing])

	apply := func(f flagType) func(string) error {
		return func(value string) error {
			flags[f] = value
			return nil
		}
	}

	flag.Func("functions", "configuration file for functions", apply(flagFunctionsPath))
	flag.Parse()

	return ctx, flags
}

// loadServiceConfig översätter en parsad flagMap till typad config.
// Endast exakta strängen "true" ger insecure=true, samma semantik
// som oauthRealmInsecure-seamen.
func loadServiceConfig(flags flagMap) serviceConfig {
	return serviceConfig{
		server: serverConfig{
			port: flags[flagServicePort],
		},
		oauth: oauthConfig{
			tokenURL:     flags[flagOAuthTokenURL],
			clientID:     flags[flagOAuthClientID],
			clientSecret: flags[flagOAuthClientSecret],
			insecure:     flags[flagOAuthInsecure] == "true",
		},
		deviceMgmt: deviceMgmtConfig{
			url: flags[flagDevMgmtURL],
		},
		measurements: measurementsConfig{
			url: flags[flagMeasurementsURL],
		},
		functionsPath: flags[flagFunctionsPath],
	}
}

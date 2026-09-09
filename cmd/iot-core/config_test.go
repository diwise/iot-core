package main

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/matryer/is"
)

func withCleanFlags(t *testing.T, args []string) {
	t.Helper()

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})

	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	os.Args = args
}

// withUnsetEnv removes a variable for the test duration, restoring any
// ambient value afterwards. Ambient developer/CI environments must not
// leak into default assertions.
func withUnsetEnv(t *testing.T, key string) {
	t.Helper()

	v, ok := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, v)
		}
	})
}

// REV-015: tests target the production seam functions, not the env
// library, so a changed default or lookup breaks them.
func TestServicePortSeam(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	withUnsetEnv(t, "SERVICE_PORT")
	is.Equal(servicePort(ctx), "8080")

	t.Setenv("SERVICE_PORT", "9090")
	is.Equal(servicePort(ctx), "9090")

	// The env library maps both unset and empty to the default.
	t.Setenv("SERVICE_PORT", "")
	is.Equal(servicePort(ctx), "8080")
}

// REV-015: only the exact string "true" disables TLS verification.
// ParseBool-style spellings ("TRUE", "1", ...) intentionally do not.
func TestOAuthRealmInsecureSeam(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value *string
		want  bool
	}{
		{"unset uses secure default", nil, false},
		{"empty uses secure default", strptr(""), false},
		{"exact true disables verification", strptr("true"), true},
		{"uppercase TRUE keeps verification", strptr("TRUE"), false},
		{"numeric 1 keeps verification", strptr("1"), false},
		{"false keeps verification", strptr("false"), false},
		{"invalid keeps verification", strptr("bogus"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			if tc.value == nil {
				withUnsetEnv(t, "OAUTH2_REALM_INSECURE")
			} else {
				t.Setenv("OAUTH2_REALM_INSECURE", *tc.value)
			}

			is.Equal(oauthRealmInsecure(context.Background()), tc.want)
		})
	}
}

func strptr(s string) *string { return &s }

// REV-015: the functions file default lives in one constant used by
// flag registration; changing it breaks this test.
func TestDefaultFunctionsConfigPath(t *testing.T) {
	is := is.New(t)

	is.Equal(defaultFunctionsConfigPath, "/opt/diwise/config/functions.csv")
}

// CORE-003: locks all current defaults in defaultFlags. Storage- och
// messaging-bibliotekens egna defaults (POSTGRES_*, RABBITMQ_*) ägs av
// biblioteken och låses inte här.
func TestDefaultFlags(t *testing.T) {
	is := is.New(t)

	for _, key := range []string{
		"SERVICE_PORT",
		"DEV_MGMT_URL",
		"MEASUREMENTS_URL",
		"OAUTH2_TOKEN_URL",
		"OAUTH2_CLIENT_ID",
		"OAUTH2_CLIENT_SECRET",
		"OAUTH2_REALM_INSECURE",
	} {
		withUnsetEnv(t, key)
	}

	flags := defaultFlags()

	is.Equal(flags[flagServicePort], "8080")
	is.Equal(flags[flagDevMgmtURL], "")
	is.Equal(flags[flagMeasurementsURL], "")
	is.Equal(flags[flagOAuthTokenURL], "")
	is.Equal(flags[flagOAuthClientID], "")
	is.Equal(flags[flagOAuthClientSecret], "")
	is.Equal(flags[flagOAuthInsecure], "false")
	is.Equal(flags[flagFunctionsPath], defaultFunctionsConfigPath)
}

// CORE-003: env vinner över defaults med oförändrade env-namn.
func TestEnvOverridesFlags(t *testing.T) {
	is := is.New(t)
	withCleanFlags(t, []string{"iot-core"})

	t.Setenv("SERVICE_PORT", "9090")
	t.Setenv("DEV_MGMT_URL", "http://dm:8080")
	t.Setenv("MEASUREMENTS_URL", "http://meas:8080")
	t.Setenv("OAUTH2_TOKEN_URL", "http://auth/token")
	t.Setenv("OAUTH2_CLIENT_ID", "id")
	t.Setenv("OAUTH2_CLIENT_SECRET", "secret")
	t.Setenv("OAUTH2_REALM_INSECURE", "true")

	_, flags := parseExternalConfig(context.Background(), defaultFlags())

	is.Equal(flags[flagServicePort], "9090")
	is.Equal(flags[flagDevMgmtURL], "http://dm:8080")
	is.Equal(flags[flagMeasurementsURL], "http://meas:8080")
	is.Equal(flags[flagOAuthTokenURL], "http://auth/token")
	is.Equal(flags[flagOAuthClientID], "id")
	is.Equal(flags[flagOAuthClientSecret], "secret")
	is.Equal(flags[flagOAuthInsecure], "true")
	is.Equal(flags[flagFunctionsPath], defaultFunctionsConfigPath)
}

// CORE-003: CLI-flaggan -functions vinner över default.
func TestCLIFunctionsPathOverridesDefault(t *testing.T) {
	is := is.New(t)
	withCleanFlags(t, []string{"iot-core", "-functions=/tmp/custom.csv"})

	_, flags := parseExternalConfig(context.Background(), defaultFlags())

	is.Equal(flags[flagFunctionsPath], "/tmp/custom.csv")
}

// CORE-003: tom SERVICE_PORT faller tillbaka på default (env-biblioteket
// mappar både unset och empty till default).
func TestEmptyServicePortFallsBackToDefault(t *testing.T) {
	is := is.New(t)
	withCleanFlags(t, []string{"iot-core"})

	t.Setenv("SERVICE_PORT", "")

	_, flags := parseExternalConfig(context.Background(), defaultFlags())

	is.Equal(flags[flagServicePort], "8080")
}

// CORE-003: loadServiceConfig mappar flagMap till typad config med samma
// exakta "true"-semantik som oauthRealmInsecure-seamen.
func TestLoadServiceConfig(t *testing.T) {
	is := is.New(t)

	flags := defaultFlags()
	flags[flagServicePort] = "9090"
	flags[flagDevMgmtURL] = "http://dm:8080"
	flags[flagMeasurementsURL] = "http://meas:8080"
	flags[flagOAuthTokenURL] = "http://auth/token"
	flags[flagOAuthClientID] = "id"
	flags[flagOAuthClientSecret] = "secret"
	flags[flagOAuthInsecure] = "true"
	flags[flagFunctionsPath] = "/tmp/f.csv"

	cfg := loadServiceConfig(flags)

	is.Equal(cfg.server.port, "9090")
	is.Equal(cfg.deviceMgmt.url, "http://dm:8080")
	is.Equal(cfg.measurements.url, "http://meas:8080")
	is.Equal(cfg.oauth.tokenURL, "http://auth/token")
	is.Equal(cfg.oauth.clientID, "id")
	is.Equal(cfg.oauth.clientSecret, "secret")
	is.True(cfg.oauth.insecure)
	is.Equal(cfg.functionsPath, "/tmp/f.csv")

	flags[flagOAuthInsecure] = "TRUE"
	is.True(!loadServiceConfig(flags).oauth.insecure)
}

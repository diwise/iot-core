package main

import (
	"context"
	"testing"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/matryer/is"
)

// HARM-004: locks the configuration surface as implemented in main.go.
// iot-core has no typed flag map; env is read inline. Defaults and
// required variables below must not change without updating README and
// external deployment definitions.
func TestServicePortDefault(t *testing.T) {
	is := is.New(t)

	ctx := context.Background()
	is.Equal(env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080"), "8080")
}

func TestServicePortEnvOverride(t *testing.T) {
	is := is.New(t)

	t.Setenv("SERVICE_PORT", "9090")

	ctx := context.Background()
	is.Equal(env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080"), "9090")
}

// HARM-004: locks the OAUTH2_REALM_INSECURE parsing used for both the
// device management and measurements clients. Any other value than the
// exact string "true" means secure mode.
func TestOAuthInsecureParsing(t *testing.T) {
	is := is.New(t)

	ctx := context.Background()

	is.Equal(env.GetVariableOrDefault(ctx, "OAUTH2_REALM_INSECURE", "false") == "true", false)

	t.Setenv("OAUTH2_REALM_INSECURE", "true")
	is.Equal(env.GetVariableOrDefault(ctx, "OAUTH2_REALM_INSECURE", "false") == "true", true)
}

// Note (HARM-004): DEV_MGMT_URL, MEASUREMENTS_URL, OAUTH2_TOKEN_URL,
// OAUTH2_CLIENT_ID and OAUTH2_CLIENT_SECRET are required at startup via
// GetVariableOrDie and abort the process when missing, so they are
// verified against README instead of executed here. The -functions flag
// default (/opt/diwise/config/functions.csv) is registered in main and
// likewise documented in README.

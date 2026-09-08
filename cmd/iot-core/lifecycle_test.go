package main

import (
	"context"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func unsetRequiredEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"DEV_MGMT_URL",
		"MEASUREMENTS_URL",
		"OAUTH2_TOKEN_URL",
		"OAUTH2_CLIENT_ID",
		"OAUTH2_CLIENT_SECRET",
	} {
		t.Setenv(key, "")
	}
}

// REV-010: run returns startup errors to main so deferred cleanup of
// acquired resources executes before the exit code is decided.
func TestRunFailsWithoutRequiredEnv(t *testing.T) {
	is := is.New(t)
	unsetRequiredEnv(t)

	err := run(context.Background())
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "DEV_MGMT_URL"))
}

// BASE-012: startup helpers must return errors to main instead of
// exiting the process. Missing required variables fail fast with an
// error.
func TestCreateClientsRequireEnv(t *testing.T) {
	is := is.New(t)
	unsetRequiredEnv(t)

	ctx := context.Background()

	_, err := requireEnv(ctx, "DEV_MGMT_URL", "url to iot-device-mgmt")
	is.True(err != nil)

	_, err = createDeviceManagementClient(ctx)
	is.True(err != nil)

	_, err = createMeasurementsClient(ctx)
	is.True(err != nil)
}

// BASE-012: unreachable backends surface as returned errors, not exits.
func TestCreateClientsReturnBackendErrors(t *testing.T) {
	is := is.New(t)

	t.Setenv("DEV_MGMT_URL", "http://127.0.0.1:1")
	t.Setenv("MEASUREMENTS_URL", "http://127.0.0.1:1")
	t.Setenv("OAUTH2_TOKEN_URL", "http://127.0.0.1:1/token")
	t.Setenv("OAUTH2_CLIENT_ID", "id")
	t.Setenv("OAUTH2_CLIENT_SECRET", "secret")
	t.Setenv("POSTGRES_HOST", "postgres-invalid-host")

	ctx := context.Background()

	_, err := createDeviceManagementClient(ctx)
	is.True(err != nil)

	_, err = createMeasurementsClient(ctx)
	is.True(err != nil)

	_, err = createDatabaseConnection(ctx)
	is.True(err != nil)
}

// BASE-012: with messaging disabled the context is created without any
// broker connection.
func TestCreateMessagingContextDisabled(t *testing.T) {
	is := is.New(t)

	t.Setenv("RABBITMQ_DISABLED", "true")

	msgCtx, err := createMessagingContext(context.Background())
	is.NoErr(err)
	is.True(msgCtx != nil)
	msgCtx.Close()
}

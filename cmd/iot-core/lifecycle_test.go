package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/application/measurements"
	"github.com/diwise/iot-core/internal/infrastructure/database"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	k8shandlers "github.com/diwise/service-chassis/pkg/infrastructure/net/http/handlers"
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

// CORE-004: runner construction touches no network. OnInit runs at
// Run-time, so initialize must succeed even without required env;
// missing env fails first inside OnInit via the unchanged factories
// (covered by TestCreateClientsRequireEnv).
func TestInitializeBuildsRunnerWithoutNetwork(t *testing.T) {
	is := is.New(t)
	unsetRequiredEnv(t)

	t.Setenv("RABBITMQ_DISABLED", "true")

	flags := defaultFlags()
	cfg := loadServiceConfig(flags)

	runner, err := initialize(context.Background(), flags, &cfg, nil)
	is.NoErr(err)
	is.True(runner != nil)
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
	is.NoErr(msgCtx.Shutdown(context.Background()))
}

type fakeMeasurementsClient struct {
	closes  int
	onClose func()
}

func (f *fakeMeasurementsClient) GetMaxValue(ctx context.Context, measurementID string) (float64, error) {
	return 0, nil
}

func (f *fakeMeasurementsClient) GetCountTrueValues(ctx context.Context, measurementID string, timeAt, endTimeAt time.Time) (float64, error) {
	return 0, nil
}

func (f *fakeMeasurementsClient) Close() {
	f.closes++
	if f.onClose != nil {
		f.onClose()
	}
}

var _ measurements.MeasurementsClient = &fakeMeasurementsClient{}

type fakeStorage struct {
	closes  int
	onClose func()
}

func (f *fakeStorage) Initialize(ctx context.Context) error { return nil }

func (f *fakeStorage) Add(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
	return nil
}

func (f *fakeStorage) AddFnct(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
	return nil
}

func (f *fakeStorage) History(ctx context.Context, id, label string, lastN int) ([]database.LogValue, error) {
	return nil, nil
}

func (f *fakeStorage) Close() {
	f.closes++
	if f.onClose != nil {
		f.onClose()
	}
}

var _ database.Storage = &fakeStorage{}

// bytesReader builds the functions.csv content used across lifecycle tests.
func bytesReader(s string) io.Reader { return bytes.NewBufferString(s) }

// CORE-004: shutdown stops inflow before clients and storage, exactly
// once per owned resource, even when invoked twice.
func TestShutdownIsOrderedAndIdempotent(t *testing.T) {
	is := is.New(t)

	var order []string
	messenger := &messaging.MsgContextMock{
		ShutdownFunc: func(context.Context) error { order = append(order, "messenger"); return nil },
	}
	mClient := &fakeMeasurementsClient{onClose: func() { order = append(order, "measurements") }}
	dmClient := &dmctest.DeviceManagementClientMock{
		CloseFunc: func(context.Context) { order = append(order, "dm") },
	}
	store := &fakeStorage{onClose: func() { order = append(order, "storage") }}

	owned := &ownedResources{
		messenger:          messenger,
		measurementsClient: mClient,
		dmClient:           dmClient,
		storage:            store,
	}

	ctx := context.Background()
	owned.close(ctx)
	owned.close(ctx)

	is.Equal(order, []string{"messenger", "measurements", "dm", "storage"})
	is.Equal(mClient.closes, 1)
	is.Equal(store.closes, 1)
	is.Equal(len(messenger.ShutdownCalls()), 1)
	is.Equal(len(dmClient.CloseCalls()), 1)
}

// CORE-004: shutdown with no initialized resources (e.g. failed OnInit)
// must be a safe no-op.
func TestShutdownWithoutResourcesIsSafe(t *testing.T) {
	owned := &ownedResources{}

	owned.close(context.Background())
	owned.close(context.Background())
}

// CORE-004: handler registration keeps the command target, and every
// registration error aborts startup. The function framework is unhooked:
// no topic handlers are registered (message.accepted is produced, not
// consumed, by core).
func TestRegisterHandlersRegistersAllHandlers(t *testing.T) {
	is := is.New(t)
	_, dmClient, msgCtx := testSetup(t)

	fconf := bytesReader("fid1;name;counter;overflow;internalID;false")
	app, _, err := buildApplication(context.Background(), dmClient, nil, fconf, &fakeStorage{})
	is.NoErr(err)

	is.NoErr(registerHandlers(msgCtx, app))
	is.Equal(len(msgCtx.RegisterCommandHandlerCalls()), 1)
	is.Equal(len(msgCtx.RegisterTopicMessageHandlerCalls()), 0)
}

func TestRegisterHandlersPropagatesError(t *testing.T) {
	is := is.New(t)
	_, dmClient, _ := testSetup(t)

	fconf := bytesReader("fid1;name;counter;overflow;internalID;false")
	app, _, err := buildApplication(context.Background(), dmClient, nil, fconf, &fakeStorage{})
	is.NoErr(err)

	failing := &messaging.MsgContextMock{
		RegisterCommandHandlerFunc: func(messaging.MessageFilter, messaging.CommandHandler) error {
			return errors.New("broker unavailable")
		},
	}

	err = registerHandlers(failing, app)
	is.True(err != nil)
}

// CORE-005: readiness-stubbarna rapporterar alltid OK utan att röra
// något beroende.
func TestReadinessStubsAlwaysOK(t *testing.T) {
	is := is.New(t)

	probes := readinessProbes()
	is.Equal(len(probes), 2)

	for _, name := range []string{"rabbitmq", "timescale"} {
		status, err := probes[name](context.Background())
		is.NoErr(err)
		is.Equal(status, "ok")
	}
}

// CORE-005: kontrollserverns probe-handlers svarar OK för de namngivna
// stubbarna. Sökvägarna (/readyz, /readyz/{check}) ägs av runnern;
// här verifieras våra prober genom samma handlers.
func TestControlProbeHandlersRespondOK(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	probes := readinessProbes()

	readyz := k8shandlers.NewReadinessHandler(ctx, probes)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	readyz(rec, req)
	is.Equal(rec.Code, http.StatusNoContent)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz/{check}", k8shandlers.NewSingleReadinessHandler(ctx, probes))

	for _, name := range []string{"rabbitmq", "timescale"} {
		req := httptest.NewRequest(http.MethodGet, "/readyz/"+name, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		is.Equal(rec.Code, http.StatusNoContent)
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz/unknown", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusNotFound)
}

// CORE-004: the runner-owned public mux serves the unchanged API routes.
func TestPublicMountPreservesRoutes(t *testing.T) {
	is := is.New(t)
	_, dmClient, _ := testSetup(t)

	fconf := bytesReader("fid1;name;counter;overflow;internalID;false")
	_, api_, err := buildApplication(context.Background(), dmClient, nil, fconf, &fakeStorage{})
	is.NoErr(err)

	mux := http.NewServeMux()
	is.NoErr(registerPublicRoutes(mux, &api_))

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, _ := testRequest(server, http.MethodGet, "/api/functions", nil)
	is.Equal(resp.StatusCode, http.StatusOK)

	// CORE-005: publik /health är borttagen.
	resp, _ = testRequest(server, http.MethodGet, "/health", nil)
	is.Equal(resp.StatusCode, http.StatusNotFound)
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/infrastructure/database"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	diwisepkg "github.com/diwise/senml/diwise"
	"github.com/matryer/is"
)

func TestAPIfunctionsReturns200OK(t *testing.T) {
	is, dmClient, _ := testSetup(t)

	fconf := bytes.NewBufferString("fid1;name;counter;overflow;internalID;false")
	_, api, err := buildApplication(context.Background(), dmClient, nil, fconf, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
		InitializeFunc: func(contextMoqParam context.Context) error {
			return nil
		},
	})
	is.NoErr(err)

	server := httptest.NewServer(api.Router())
	defer server.Close()

	resp, _ := testRequest(server, http.MethodGet, "/api/functions", nil)
	is.Equal(resp.StatusCode, http.StatusOK)
}

// fakeIncomingCommand is a minimal messaging.IncomingCommand for driving the
// command handler without a broker.
type fakeIncomingCommand struct {
	body        []byte
	contentType string
}

func (c fakeIncomingCommand) Body() []byte             { return c.body }
func (c fakeIncomingCommand) ContentType() string      { return c.contentType }
func (c fakeIncomingCommand) Context() context.Context { return context.Background() }
func (c fakeIncomingCommand) RespondWith(context.Context, messaging.Response) error {
	return nil
}
func (c fakeIncomingCommand) MessageID() string       { return "test-message-id" }
func (c fakeIncomingCommand) CorrelationID() string   { return "test-message-id" }
func (c fakeIncomingCommand) Redelivered() bool       { return false }
func (c fakeIncomingCommand) RoutingKey() string      { return "iot-core" }
func (c fakeIncomingCommand) ReplyTo() string         { return "" }
func (c fakeIncomingCommand) Headers() map[string]any { return nil }

func commandTestApp(t *testing.T, is *is.I, dmClient *dmctest.DeviceManagementClientMock, msgCtx *messaging.MsgContextMock) messaging.CommandHandler {
	t.Helper()

	fconf := bytesReader("fid1;name;counter;overflow;internalID;false")
	app, _, err := buildApplication(context.Background(), dmClient, nil, fconf, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
		InitializeFunc: func(contextMoqParam context.Context) error {
			return nil
		},
	})
	is.NoErr(err)
	is.NoErr(registerHandlers(msgCtx, app))

	return msgCtx.RegisterCommandHandlerCalls()[0].Handler
}

func acceptedFromPublish(t *testing.T, is *is.I, msgCtx *messaging.MsgContextMock, idx int) events.MessageAccepted {
	t.Helper()
	is.Equal(len(msgCtx.PublishOnTopicCalls()), idx+1)
	var accepted events.MessageAccepted
	is.NoErr(json.Unmarshal(msgCtx.PublishOnTopicCalls()[idx].Message.Body(), &accepted))
	return accepted
}

const legacyTempCommand = `{
	"pack":[
		{"bn":"internalID/3303/","bt":1675805579,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},
		{"n":"5700","u":"Cel","v":21.5}
	],
	"timestamp":"2023-02-07T21:32:59.682607Z"
}`

const multiObservationCommand = `{
	"pack":[
		{"bn":"internalID/3303/","bt":1720000000,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},
		{"n":"5700","u":"Cel","v":21.5},
		{"bn":"internalID/3304/","bt":1720000000,"n":"0","vs":"urn:oma:lwm2m:ext:3304"},
		{"n":"5700","u":"%RH","v":55.0},
		{"bn":"internalID/3301/","bt":1720000000,"n":"0","vs":"urn:oma:lwm2m:ext:3301"},
		{"n":"5700","u":"lux","v":320.0}
	],
	"timestamp":"2024-07-03T09:46:40Z"
}`

// Fas B: kommandovägen validerar och berikar; en legacy-enobjektsrapport ger
// ett message.accepted med kvalificerad tenantmetadata och oförändrad typad
// content-type.
func TestCommandHandlerAcceptsLegacySingleTemp(t *testing.T) {
	is, dmClient, msgCtx := testSetup(t)
	handler := commandTestApp(t, is, dmClient, msgCtx)

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := handler(context.Background(), fakeIncomingCommand{
		body:        []byte(legacyTempCommand),
		contentType: "application/vnd.oma.lwm2m.ext.3303+json",
	}, l)
	is.NoErr(err)

	accepted := acceptedFromPublish(t, is, msgCtx, 0)
	if got := msgCtx.PublishOnTopicCalls()[0].Message.ContentType(); got != "application/vnd.oma.lwm2m.ext.3303+json" {
		t.Fatalf("content type = %q", got)
	}
	tenant, ok := accepted.Pack().GetStringValue(senml.FindByName("internalID/tenant"))
	if !ok || tenant != "default" {
		t.Fatalf("qualified tenant = %q, %v", tenant, ok)
	}
	// Naken legacy-metadata får inte förekomma i accepterade pack.
	if _, ok := accepted.Pack().GetStringValue(senml.FindByName("tenant")); ok {
		t.Fatal("bare tenant record present in accepted pack")
	}
}

// Fas B: en rapport med tre observationer ger ett message.accepted med alla
// observationer bevarade och generisk content-type.
func TestCommandHandlerAcceptsMultiObservation(t *testing.T) {
	is, dmClient, msgCtx := testSetup(t)
	handler := commandTestApp(t, is, dmClient, msgCtx)

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := handler(context.Background(), fakeIncomingCommand{
		body:        []byte(multiObservationCommand),
		contentType: "application/vnd.oma.lwm2m+json",
	}, l)
	is.NoErr(err)

	if got := msgCtx.PublishOnTopicCalls()[0].Message.ContentType(); got != events.GenericLwM2MContentType {
		t.Fatalf("content type = %q", got)
	}
	accepted := acceptedFromPublish(t, is, msgCtx, 0)
	parsed, err := diwisepkg.Parse(accepted.Pack(), time.Now().UTC())
	is.NoErr(err)
	if got := len(parsed.Objects()); got != 3 {
		t.Fatalf("observations = %d", got)
	}
	if err := parsed.ValidateAccepted(); err != nil {
		t.Fatalf("ValidateAccepted: %v", err)
	}
}

// Fas B: strukturskada kan aldrig läka vid retry och ska därför vara permanent.
func TestCommandHandlerRejectsMalformedPermanently(t *testing.T) {
	is, dmClient, msgCtx := testSetup(t)
	handler := commandTestApp(t, is, dmClient, msgCtx)

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := handler(context.Background(), fakeIncomingCommand{
		body:        []byte("{not-json"),
		contentType: "application/vnd.oma.lwm2m.ext.3303+json",
	}, l)
	is.True(err != nil)
	is.True(messaging.IsPermanent(err))
	is.Equal(len(msgCtx.PublishOnTopicCalls()), 0)
}

func testRequest(ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, _ := http.NewRequest(method, ts.URL+path, body)
	resp, _ := http.DefaultClient.Do(req)
	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	return resp, string(respBody)
}

func testSetup(t *testing.T) (*is.I, *dmctest.DeviceManagementClientMock, *messaging.MsgContextMock) {
	is := is.New(t)

	dmc := &dmctest.DeviceManagementClientMock{
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (client.Device, error) {
			res := &dmctest.DeviceMock{
				IDFunc:          func() string { return "internalID" },
				EnvironmentFunc: func() string { return "water" },
				SourceFunc:      func() string { return "test-source" },
				LongitudeFunc:   func() float64 { return 16 },
				LatitudeFunc:    func() float64 { return 32 },
				TenantFunc:      func() string { return "default" },
			}
			return res, nil
		},
	}

	msgctx := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			return nil
		},
		RegisterCommandHandlerFunc: func(messaging.MessageFilter, messaging.CommandHandler) error {
			return nil
		},
		RegisterTopicMessageHandlerFunc: func(string, messaging.TopicMessageHandler) error {
			return nil
		},
	}

	return is, dmc, msgctx
}

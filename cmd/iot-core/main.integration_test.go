package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

func TestAPIfunctionsReturns200OK(t *testing.T) {
	is, dmClient, msgCtx := testSetup(t)

	fconf := bytes.NewBufferString("fid1;name;counter;overflow;internalID;false")
	_, api, err := initialize(context.Background(), dmClient, msgCtx, fconf, &database.StorageMock{
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

	resp, _ := testRequest(is, server, http.MethodGet, "/api/functions", nil)
	is.Equal(resp.StatusCode, http.StatusOK)
}

func TestReceiveDigitalInputUpdateMessage(t *testing.T) {
	is, dmClient, msgCtx := testSetup(t)
	sID := "internalID"

	fconf := bytes.NewBufferString("fid1;name;counter;overflow;" + sID + ";false")
	_, _, err := initialize(context.Background(), dmClient, msgCtx, fconf, &database.StorageMock{
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

	topicMessageHandler := msgCtx.RegisterTopicMessageHandlerCalls()[0].Handler

	ctx := context.Background()
	l := slog.New(slog.NewTextHandler(io.Discard, nil))

	topicMessageHandler(ctx, &messaging.IncomingTopicMessageMock{
		BodyFunc: func() []byte { return newStateJSON(sID, true) },
	}, l)
	topicMessageHandler(ctx, &messaging.IncomingTopicMessageMock{
		BodyFunc: func() []byte { return newStateJSON(sID, false) },
	}, l)
	topicMessageHandler(ctx, &messaging.IncomingTopicMessageMock{
		BodyFunc: func() []byte { return newStateJSON(sID, true) },
	}, l)

	is.Equal(len(msgCtx.PublishOnTopicCalls()), 3) // should have been called three times

	b := msgCtx.PublishOnTopicCalls()[2].Message.Body()

	const expectation string = `{"id":"fid1","name":"name","type":"counter","subtype":"overflow","onupdate":false,"counter":{"count":2,"state":true}}`
	is.Equal(string(b), expectation)
}

func testRequest(is *is.I, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
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

func newStateJSON(sensorID string, on bool) []byte {
	return []byte(fmt.Sprintf(messageJSONFormat, sensorID, sensorID, on))
}

const messageJSONFormat string = `{
	"sensorID":"%s",
	"pack":[
		{"bn":"urn:oma:lwm2m:ext:3200","bt":1675805579,"n":"0","vs":"%s"},
		{"n":"5500","vb":%t}
	],
	"timestamp":"2023-02-07T21:32:59.682607Z"
}`

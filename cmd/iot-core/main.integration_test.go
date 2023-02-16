package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

func TestAPIFeaturesReturns200OK(t *testing.T) {
	is, dmClient, msgCtx := testSetup(t)

	fconf := bytes.NewBufferString("fid1;counter;overflow;internalID")
	_, api, err := initialize(context.Background(), dmClient, msgCtx, fconf)
	is.NoErr(err)

	server := httptest.NewServer(api.Router())
	defer server.Close()

	resp, _ := testRequest(is, server, http.MethodGet, "/api/features", nil)
	is.Equal(resp.StatusCode, http.StatusOK)
}

func TestReceiveDigitalInputUpdateMessage(t *testing.T) {
	is, dmClient, msgCtx := testSetup(t)
	sID := "internalID"

	fconf := bytes.NewBufferString("fid1;counter;overflow;" + sID)
	_, _, err := initialize(context.Background(), dmClient, msgCtx, fconf)
	is.NoErr(err)

	topicMessageHandler := msgCtx.RegisterTopicMessageHandlerCalls()[0].Handler

	ctx := context.Background()
	l := zerolog.Logger{}

	topicMessageHandler(ctx, amqp091.Delivery{Body: newStateJSON(sID, true)}, l)
	topicMessageHandler(ctx, amqp091.Delivery{Body: newStateJSON(sID, false)}, l)
	topicMessageHandler(ctx, amqp091.Delivery{Body: newStateJSON(sID, true)}, l)

	is.Equal(len(msgCtx.PublishOnTopicCalls()), 3) // should have been called three times
	msg := msgCtx.PublishOnTopicCalls()[1].Message
	b, _ := json.Marshal(msg)

	const expectation string = `{"id":"fid1","type":"counter","subtype":"overflow","counter":{"count":2,"state":true}}`
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
		RegisterCommandHandlerFunc: func(string, messaging.CommandHandler) error {
			return nil
		},
		RegisterTopicMessageHandlerFunc: func(string, messaging.TopicMessageHandler) {
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

package application

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

func TestWaterMeterMessageReceived(t *testing.T) {
	is, ctx, app, _, _, _, _ := testSetup(t)

	var evt events.MessageReceived
	err := json.Unmarshal([]byte(waterMeterMessage), &evt)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evt)

	is.Equal(7, len(ma.Pack()))

	is.NoErr(err)
}

func TestWaterMeterMessageAccepted(t *testing.T) {
	is, ctx, app, _, _, _, _ := testSetup(t)
	var evt events.MessageReceived
	err := json.Unmarshal([]byte(waterMeterMessage), &evt)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evt)
	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma)
	is.NoErr(err)
}

func TestDistanceMessageReceived(t *testing.T) {
	is, ctx, app, _, _, _, _ := testSetup(t)
	var evt events.MessageReceived
	err := json.Unmarshal([]byte(distanceMessage), &evt)
	is.NoErr(err)

	_, err = app.MessageReceived(ctx, evt)
	is.NoErr(err)
}

func TestDistanceMessageAccepted(t *testing.T) {
	is, ctx, app, _, _, _, _ := testSetup(t)
	var evt events.MessageReceived
	err := json.Unmarshal([]byte(distanceMessage), &evt)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evt)
	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma)
	is.NoErr(err)
}

func TestFunctionUpdatedWithDistance(t *testing.T) {
	is, ctx, app, _, _, _, _ := testSetup(t)
	err := app.FunctionUpdated(ctx, []byte(functionUpdatedWithLevel))
	is.NoErr(err)
}

func testSetup(t *testing.T) (*is.I, context.Context, App, client.DeviceManagementClient, *measurements.MeasurementsClientMock, functions.Registry, messaging.MsgContext) {
	is := is.New(t)
	d := &test.DeviceManagementClientMock{
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (client.Device, error) {
			return &test.DeviceMock{
				LatitudeFunc:    func() float64 { return 60.0 },
				LongitudeFunc:   func() float64 { return 10.0 },
				EnvironmentFunc: func() string { return "" },
				SourceFunc:      func() string { return "" },
				TenantFunc:      func() string { return "default" },
			}, nil
		},
	}

	ctx := context.Background()
	m := &measurements.MeasurementsClientMock{}
	mctx := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			return nil
		},
	}
	s := &database.StorageMock{
		HistoryFunc: func(ctx context.Context, id, label string, lastN int) ([]database.LogValue, error) {
			return []database.LogValue{}, nil
		},
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
	}
	r, _ := functions.NewRegistry(ctx, io.NopCloser(strings.NewReader(functionsFileContent)), s)

	a := New(d, m, r, mctx)

	return is, ctx, a, d, m, r, mctx
}

const functionsFileContent string = `
functionID;name;type;subtype;sensorID;onupdate;args
lvl-1;level;level;distance;internal-id-for-device;true;maxd=5.0,maxl=10.0,mean=5.0,offset=0.0`

const waterMeterMessage string = `
{
  "pack":
  [
    {"bn":"internal-id-for-device/3424/","bt":1563735600, "n":"0","vs":"urn:oma:lwm2m:ext:3424"},
    {"n":"1","u":"m3","v":10.727},
    {"n":"3","vs":"w1e"},
    {"n":"10","vb":true}
  ],
  "timestamp":"2025-10-02T08:42:07.505884256Z"
}`

const distanceMessage string = `
{
  "pack": [
    {
      "bn": "internal-id-for-device/3330/",
      "bt": 1713865679,
      "n": "0",
      "vs": "urn:oma:lwm2m:ext:3330"
    },
    {
      "n": "5700",
      "u": "m",
      "v": 1.80952
    },
    {
      "n": "5701",
      "vs": "metre"
    }
  ],
  "timestamp": "2025-10-02T12:02:34.633266567Z"
}`

const functionUpdatedWithLevel string = `
{
  "id": "lvl-1",
  "name": "level",
  "type": "level",
  "subtype": "distance",
  "deviceID": "internal-id-for-device",
  "location": {
    "latitude": 60,
    "longitude": 10
  },
  "tenant": "default",
  "onupdate": true,
  "timestamp": "2024-04-23T09:47:59Z",
  "level": {
    "current": 3.19,
    "percent": 31.9,
    "offset": -1.81
  }
}`

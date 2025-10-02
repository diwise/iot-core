package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/matryer/is"
)

func TestWaterMeterMessage(t *testing.T) {
	is, ctx, app, _, _, _ := testSetup(t)

	var evt events.MessageReceived
	err := json.Unmarshal([]byte(waterMeterMessage), &evt)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evt)

	is.Equal(7, len(ma.Pack()))

	is.NoErr(err)
}

func testSetup(t *testing.T) (*is.I, context.Context, App, client.DeviceManagementClient, measurements.MeasurementsClient, functions.Registry) {
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
	m := &measurements.MeasurementsClientMock{}
	f := &functions.RegistryMock{}
	a := New(d, m, f)
	ctx := context.Background()

	return is, ctx, a, d, m, f
}

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

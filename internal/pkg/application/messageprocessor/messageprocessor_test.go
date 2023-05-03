package messageprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/farshidtz/senml/v2/codec"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatProcessMessageReadsSenMLPackProperly(t *testing.T) {
	is, d, _ := testSetup(t)

	pack, _ := codec.DecodeJSON([]byte(co2))

	m := NewMessageProcessor(d)
	msg, err := m.ProcessMessage(context.Background(), events.MessageReceived{
		Device:    "devID",
		Pack:      pack,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	})

	is.True(msg != nil)
	is.NoErr(err)
	is.True(msg.Sensor == "internalID")
}

func testSetup(t *testing.T) (*is.I, *dmctest.DeviceManagementClientMock, zerolog.Logger) {
	is := is.New(t)

	dmc := &dmctest.DeviceManagementClientMock{
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (client.Device, error) {
			res := &dmctest.DeviceMock{
				IDFunc:          func() string { return "internalID" },
				EnvironmentFunc: func() string { return "water" },
				LongitudeFunc:   func() float64 { return 16 },
				LatitudeFunc:    func() float64 { return 32 },
				TenantFunc:      func() string { return "default" },
				SourceFunc:      func() string { return "source" },
			}
			return res, nil
		},
	}

	return is, dmc, zerolog.Logger{}
}

const co2 string = `[{"bn":"urn:oma:lwm2m:ext:3428","n":"0","vs":"internalID","bt":1234567},{"n":"CO2","v":22}]`

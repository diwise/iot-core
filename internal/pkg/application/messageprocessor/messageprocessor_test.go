package messageprocessor

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/farshidtz/senml/v2/codec"
	"github.com/matryer/is"
)

func TestThatProcessMessageReadsSenMLPackProperly(t *testing.T) {
	is, d, _ := testSetup(t)

	pack, _ := codec.DecodeJSON([]byte(co2))

	m := NewMessageProcessor(d)
	msg, err := m.ProcessMessage(context.Background(), events.NewMessageReceived(pack))

	is.True(msg != nil)
	is.NoErr(err)
	is.True(msg.DeviceID() == "internalID")
}

func testSetup(t *testing.T) (*is.I, *dmctest.DeviceManagementClientMock, *slog.Logger) {
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

	return is, dmc, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const co2 string = `[{"bn":"urn:oma:lwm2m:ext:3428","n":"0","vs":"internalID","bt":1234567},{"n":"CO2","v":22}]`

package messageprocessor

import (
	"context"
	"testing"

	"github.com/diwise/iot-core/internal/pkg/domain"
	"github.com/farshidtz/senml/v2/codec"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatProcessMessageReadsSenMLPackProperly(t *testing.T) {
	is, d, _ := testSetup(t)

	pack, _ := codec.DecodeJSON([]byte(co2))

	m := NewMessageProcessor(d)
	msg, err := m.ProcessMessage(context.Background(), pack)

	is.True(msg != nil)
	is.NoErr(err)
	is.True(msg.Sensor == "internalID")
}

func testSetup(t *testing.T) (*is.I, *domain.DeviceManagementClientMock, zerolog.Logger) {
	is := is.New(t)

	dmc := &domain.DeviceManagementClientMock{
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (domain.Device, error) {
			return domain.NewDevice("internalID", "water", 16, 32), nil
		},
	}

	return is, dmc, zerolog.Logger{}
}

const co2 string = `[{"bn":"urn:oma:lwm2m:ext:3428","n":"0","vs":"internalID","bt":1234567},{"n":"CO2","v":22}]`

package application

import (
	"context"
	"testing"

	"github.com/diwise/iot-core/internal/messageprocessor"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThat(t *testing.T) {
	is, m, log := testSetup(t)

	app := NewIoTCoreApp("", m, log)
	e, err := app.MessageAccepted(context.Background(), []byte(co2))

	is.NoErr(err)
	is.True(e != nil)
}

func testSetup(t *testing.T) (*is.I, *messageprocessor.MessageProcessorMock, zerolog.Logger) {
	is := is.New(t)

	m := &messageprocessor.MessageProcessorMock{
		ProcessMessageFunc: func(ctx context.Context, pack senml.Pack) (*events.MessageAccepted, error){			
			return events.NewMessageAccepted("internalID", pack), nil
		},
	}

	return is, m, zerolog.Logger{}
}

const co2 string = `[{"bn":"urn:oma:lwm2m:ext:3428","n":"0","vs":"internalID"},{"n":"CO2","v":22}]`
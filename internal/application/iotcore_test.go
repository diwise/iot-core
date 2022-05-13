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

func TestThatMessageAcceptedReturnsProperMessageAccepted(t *testing.T) {
	is, m, msgRcvd := testSetup(t)

	app := NewIoTCoreApp("", m, zerolog.Logger{})
	e, err := app.MessageAccepted(context.Background(), msgRcvd)

	is.NoErr(err)
	is.True(e != nil)
	is.Equal(e.Pack[0].BaseName, "urn:oma:lwm2m:ext:3428")
}

func testSetup(t *testing.T) (*is.I, *messageprocessor.MessageProcessorMock, events.MessageReceived) {
	is := is.New(t)

	m := &messageprocessor.MessageProcessorMock{
		ProcessMessageFunc: func(ctx context.Context, pack senml.Pack) (*events.MessageAccepted, error) {
			return events.NewMessageAccepted("internalID", pack), nil
		},
	}

	val := 22.0

	msgRcvd := events.MessageReceived{
		Device: "deviceID",
		Pack: senml.Pack{
			{BaseName: "urn:oma:lwm2m:ext:3428", Name: "0", StringValue: "internalID", BaseTime: 1234567},
			{Name: "CO2", Value: &val},
		},
		Timestamp: "",
	}

	return is, m, msgRcvd
}

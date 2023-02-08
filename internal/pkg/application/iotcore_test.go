package application

import (
	"context"
	"testing"

	"github.com/diwise/iot-core/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatProcessMessageIsCalled(t *testing.T) {
	is, m := testSetup(t)

	app := NewIoTCoreApp("", m, zerolog.Logger{})
	e, err := app.MessageReceived(context.Background(), events.MessageReceived{})

	is.NoErr(err)
	is.True(e != nil)
	is.True(len(m.ProcessMessageCalls()) == 1)
}

func testSetup(t *testing.T) (*is.I, *messageprocessor.MessageProcessorMock) {
	is := is.New(t)

	m := &messageprocessor.MessageProcessorMock{
		ProcessMessageFunc: func(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
			return events.NewMessageAccepted("internalID", senml.Pack{}), nil
		},
	}

	return is, m
}

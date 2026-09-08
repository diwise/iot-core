package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/topics"
	"github.com/diwise/senml"
	"github.com/matryer/is"
)

// HARM-003: locks the topic names shared by core, agent, events, things
// and transform-fiware. A rename requires a compatibility/migration plan.
func TestTopicNameConstants(t *testing.T) {
	is := is.New(t)

	is.Equal(topics.MessageReceived, "message.received")
	is.Equal(topics.MessageAccepted, "message.accepted")
	is.Equal(topics.MessageTransformed, "message.transformed")
}

// HARM-003: locks topic name, content type and JSON shape of the three
// core message envelopes. Consumers decode these; changes require a
// compatibility/migration plan.
func TestMessageEnvelopeWireContracts(t *testing.T) {
	is := is.New(t)

	pack := lwm2m.ToPack(lwm2m.NewTemperature("aaa-bbb-ccc/0", 10.0, time.Now()))

	received := NewMessageReceived(pack)
	is.Equal(received.TopicName(), "message.received")
	is.Equal(received.ContentType(), "application/vnd.oma.lwm2m.ext.3303+json")

	accepted := NewMessageAccepted(pack)
	is.Equal(accepted.TopicName(), "message.accepted")
	is.Equal(accepted.ContentType(), "application/vnd.oma.lwm2m.ext.3303+json")

	transformed := NewMessageTransformed(pack)
	is.Equal(transformed.TopicName(), "message.transformed")
	is.Equal(transformed.ContentType(), "application/vnd.oma.lwm2m.ext.3303+json")

	for _, body := range [][]byte{received.Body(), accepted.Body(), transformed.Body()} {
		var decoded map[string]any
		is.NoErr(json.Unmarshal(body, &decoded))

		_, ok := decoded["pack"]
		is.True(ok)
		_, ok = decoded["timestamp"]
		is.True(ok)
	}
}

// HARM-003: locks the validation rules consumers rely on when
// distinguishing malformed messages from valid ones.
func TestMessageValidationContract(t *testing.T) {
	is := is.New(t)

	empty := NewMessageAccepted(senml.Pack{})
	is.True(empty.Error() != nil)

	noDevice := NewMessageAccepted(senml.Pack{senml.Record{Name: "tenant"}})
	is.True(noDevice.Error() != nil)

	valid := NewMessageAccepted(lwm2m.ToPack(lwm2m.NewTemperature("aaa-bbb-ccc/0", 10.0, time.Now())))
	is.NoErr(valid.Error())
}

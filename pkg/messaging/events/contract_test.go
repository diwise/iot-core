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

// REV-016: locks the complete wire representation of a message.accepted
// body produced through the real producer path (agent LwM2M types plus
// core decorators) with fixed measurement time and a non-default
// tenant. Any change to payload values, tenant, time format or field
// set fails this test.
func TestMessageAcceptedGoldenBody(t *testing.T) {
	is := is.New(t)

	ts := time.Date(2025, 4, 10, 11, 44, 1, 0, time.UTC)
	pack := lwm2m.ToPack(lwm2m.NewTemperature("device-42/0", 21.5, ts))
	m := NewMessageAccepted(pack, Tenant("acme"), Source("test"), Lat(62.39), Lon(17.31))
	m.Timestamp = ts

	const golden = `{"pack":[{"bn":"device-42/0/3303/","bt":1744285441,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},{"n":"5700","u":"Cel","v":21.5},{"n":"tenant","vs":"acme"},{"n":"source","vs":"test"},{"u":"lat","v":62.39},{"u":"lon","v":17.31}],"timestamp":"2025-04-10T11:44:01Z"}`
	is.Equal(string(m.Body()), golden)

	is.Equal(m.TopicName(), "message.accepted")
	is.Equal(m.ContentType(), "application/vnd.oma.lwm2m.ext.3303+json")

	// Consumer-side decoding of the exact golden bytes.
	var decoded MessageAccepted
	is.NoErr(json.Unmarshal([]byte(golden), &decoded))
	is.NoErr(decoded.Error())
	is.Equal(decoded.DeviceID(), "device-42")
	is.Equal(decoded.Tenant(), "acme")
	is.Equal(decoded.ObjectID(), "3303")
	is.Equal(decoded.TopicName(), "message.accepted")
	is.Equal(decoded.ContentType(), "application/vnd.oma.lwm2m.ext.3303+json")
}

// REV-016: mutations to tenant, time, device or values must change the
// wire representation instead of passing silently.
func TestMessageAcceptedGoldenMutations(t *testing.T) {
	is := is.New(t)

	ts := time.Date(2025, 4, 10, 11, 44, 1, 0, time.UTC)
	newEnvelope := func() *MessageAccepted {
		pack := lwm2m.ToPack(lwm2m.NewTemperature("device-42/0", 21.5, ts))
		m := NewMessageAccepted(pack, Tenant("acme"))
		m.Timestamp = ts
		return m
	}

	baseline := string(newEnvelope().Body())

	otherTenant := NewMessageAccepted(
		lwm2m.ToPack(lwm2m.NewTemperature("device-42/0", 21.5, ts)),
		Tenant("other"),
	)
	otherTenant.Timestamp = ts
	is.True(string(otherTenant.Body()) != baseline)

	otherDevice := NewMessageAccepted(
		lwm2m.ToPack(lwm2m.NewTemperature("device-99/0", 21.5, ts)),
		Tenant("acme"),
	)
	otherDevice.Timestamp = ts
	is.True(string(otherDevice.Body()) != baseline)
	is.Equal(otherDevice.DeviceID(), "device-99")

	otherValue := NewMessageAccepted(
		lwm2m.ToPack(lwm2m.NewTemperature("device-42/0", 18.0, ts)),
		Tenant("acme"),
	)
	otherValue.Timestamp = ts
	is.True(string(otherValue.Body()) != baseline)

	otherTime := newEnvelope()
	otherTime.Timestamp = ts.Add(time.Hour)
	is.True(string(otherTime.Body()) != baseline)
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

package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/topics"
	"github.com/diwise/senml"
)

type MessageReceived struct {
	Pack      senml.Pack `json:"pack"`
	Timestamp time.Time  `json:"timestamp"`
}

func NewMessageReceived(pack senml.Pack) MessageReceived {
	return MessageReceived{
		Pack:      pack,
		Timestamp: time.Now().UTC(),
	}
}

func (m MessageReceived) DeviceID() string {
	return GetDeviceID(m.Pack)
}

func (m MessageReceived) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}

func (m MessageReceived) ContentType() string {
	return fmt.Sprintf("application/vnd.oma.lwm2m.ext.%s+json", GetObjectID(m.Pack))
}

func (m MessageReceived) Error() error {
	if GetDeviceID(m.Pack) == "" {
		return errors.New("device id is missing")
	}

	if m.Timestamp.IsZero() {
		return errors.New("timestamp is mising")
	}

	if len(m.Pack) == 0 {
		return errors.New("pack is empty")
	}

	return nil
}

type MessageAccepted struct {
	Pack      senml.Pack `json:"pack"`
	Timestamp time.Time  `json:"timestamp"`
}

func NewMessageAccepted(pack senml.Pack, decorators ...EventDecoratorFunc) *MessageAccepted {
	m := &MessageAccepted{
		Pack:      pack,
		Timestamp: time.Now().UTC(),
	}

	for _, d := range decorators {
		d(m)
	}

	return m
}

func (m MessageAccepted) DeviceID() string {
	return GetDeviceID(m.Pack)
}

func (m MessageAccepted) ObjectID() string {
	return GetObjectID(m.Pack)
}

func (m MessageAccepted) Tenant() string {
	s, ok := m.Pack.GetStringValue(senml.FindByName("tenant"))
	if !ok {
		return ""
	}
	return s
}

func (m MessageAccepted) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}

func (m MessageAccepted) ContentType() string {
	return fmt.Sprintf("application/vnd.oma.lwm2m.ext.%s+json", GetObjectID(m.Pack))
}

func (m MessageAccepted) TopicName() string {
	return topics.MessageAccepted
}

func (m MessageAccepted) Error() error {
	if GetDeviceID(m.Pack) == "" {
		return errors.New("device id is missing")
	}

	if m.Timestamp.IsZero() {
		return errors.New("timestamp is mising")
	}

	if len(m.Pack) == 0 {
		return errors.New("pack is empty")
	}

	return nil
}

type MessageTransformed struct {
	Pack      senml.Pack `json:"pack"`
	Timestamp time.Time  `json:"timestamp"`
}

func NewMessageTransformed(pack senml.Pack, tenant string) *MessageTransformed {
	_, ok := pack.GetStringValue(senml.FindByName("tenant"))
	if !ok {
		pack = append(pack, senml.Record{Name: "tenant", StringValue: tenant})
	}

	m := &MessageTransformed{
		Pack:      pack,
		Timestamp: time.Now().UTC(),
	}

	return m
}

func (m MessageTransformed) DeviceID() string {
	return GetDeviceID(m.Pack)
}

func (m MessageTransformed) ObjectID() string {
	return GetObjectID(m.Pack)
}

func (m MessageTransformed) Tenant() string {
	s, ok := m.Pack.GetStringValue(senml.FindByName("tenant"))
	if !ok {
		return ""
	}
	return s
}

func (m MessageTransformed) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}

func (m MessageTransformed) ContentType() string {
	return fmt.Sprintf("application/vnd.oma.lwm2m.ext.%s+json", GetObjectID(m.Pack))
}

func (m MessageTransformed) TopicName() string {
	return topics.MessageTransformed
}

func (m MessageTransformed) Error() error {
	if GetDeviceID(m.Pack) == "" {
		return errors.New("device id is missing")
	}

	if m.Timestamp.IsZero() {
		return errors.New("timestamp is missing")
	}

	if len(m.Pack) == 0 {
		return errors.New("pack is empty")
	}

	return nil
}

var ErrBadTimestamp = fmt.Errorf("bad timestamp")
var ErrNoMatch = fmt.Errorf("event mismatch")

func Matches(m MessageAccepted, objectURN string) bool {
	return (GetObjectURN(m.Pack) == objectURN)
}

func GetDeviceID(m senml.Pack) string {
	r, ok := m.GetRecord(senml.FindByName("0"))
	if !ok {
		return ""
	}
	return strings.Split(r.Name, "/")[0]
}

func GetObjectURN(m senml.Pack) string {
	r, ok := m.GetStringValue(senml.FindByName("0"))
	if !ok {
		return ""
	}
	return r
}

func GetObjectID(m senml.Pack) string {
	urn := GetObjectURN(m)
	if urn == "" {
		return ""
	}

	if !strings.Contains(urn, ":") {
		return ""
	}

	parts := strings.Split(urn, ":")
	return parts[len(parts)-1]
}

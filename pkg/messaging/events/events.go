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

var ErrBadTimestamp = fmt.Errorf("bad timestamp")
var ErrNoMatch = fmt.Errorf("event mismatch")

type Message interface {
	DeviceID() string
	ObjectID() string
	Pack() senml.Pack
	Append(r senml.Record)
	Replace(r senml.Record, find func(senml.Record) bool)
	Tenant() string
	Error() error
}

type MessageReceived struct {
	Pack_     senml.Pack `json:"pack"`
	Timestamp time.Time  `json:"timestamp"`
}
type MessageAccepted struct {
	Pack_     senml.Pack `json:"pack"`
	Timestamp time.Time  `json:"timestamp"`
}

func NewMessageReceived(pack senml.Pack, decorators ...EventDecoratorFunc) *MessageReceived {
	mr := &MessageReceived{
		Pack_:     pack,
		Timestamp: time.Now().UTC(),
	}
	for _, d := range decorators {
		d(mr)
	}
	return mr
}
func NewMessageAccepted(pack senml.Pack, decorators ...EventDecoratorFunc) *MessageAccepted {
	ma := &MessageAccepted{
		Pack_:     pack,
		Timestamp: time.Now().UTC(),
	}
	for _, d := range decorators {
		d(ma)
	}
	return ma
}


func (m MessageReceived) DeviceID() string {
	return GetDeviceID(m.Pack_)
}
func (m MessageReceived) ObjectID() string {
	return GetObjectID(m.Pack_)
}
func (m MessageReceived) Pack() senml.Pack {
	return m.Pack_
}
func (m *MessageReceived) Append(r senml.Record) {
	m.Pack_ = append(m.Pack_, r)
}
func (m *MessageReceived) Replace(r senml.Record, find func(senml.Record) bool) {
	for i, rec := range m.Pack_ {
		if find(rec) {
			m.Pack_[i] = r
			return
		}
	}
}
func (m MessageReceived) Tenant() string {
	s, ok := m.Pack().GetStringValue(senml.FindByName("tenant"))
	if !ok {
		return ""
	}
	return s
}
func (m MessageReceived) Error() error {
	if len(m.Pack()) == 0 {
		return errors.New("pack is empty")
	}
	if m.DeviceID() == "" {
		return errors.New("device id is missing")
	}
	if m.Timestamp.IsZero() {
		return errors.New("timestamp is mising")
	}

	return nil
}
func (m MessageReceived) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}
func (m MessageReceived) ContentType() string {
	return fmt.Sprintf("application/vnd.oma.lwm2m.ext.%s+json", m.ObjectID())
}
func (m MessageReceived) TopicName() string {
	return topics.MessageReceived
}

/*------------*/

func (m MessageAccepted) DeviceID() string {
	return GetDeviceID(m.Pack_)
}
func (m MessageAccepted) ObjectID() string {
	return GetObjectID(m.Pack_)
}
func (m MessageAccepted) Pack() senml.Pack {
	return m.Pack_
}
func (m *MessageAccepted) Append(r senml.Record) {
	m.Pack_ = append(m.Pack_, r)
}
func (m *MessageAccepted) Replace(r senml.Record, find func(senml.Record) bool) {
	for i, rec := range m.Pack_ {
		if find(rec) {
			m.Pack_[i] = r
			return
		}
	}
}
func (m MessageAccepted) Tenant() string {
	s, ok := m.Pack().GetStringValue(senml.FindByName("tenant"))
	if !ok {
		return ""
	}
	return s
}
func (m MessageAccepted) Error() error {
	if len(m.Pack()) == 0 {
		return errors.New("pack is empty")
	}
	if m.DeviceID() == "" {
		return errors.New("device id is missing")
	}
	if m.Timestamp.IsZero() {
		return errors.New("timestamp is mising")
	}

	return nil
}
func (m MessageAccepted) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}
func (m MessageAccepted) ContentType() string {
	return fmt.Sprintf("application/vnd.oma.lwm2m.ext.%s+json", m.ObjectID())
}
func (m MessageAccepted) TopicName() string {
	return topics.MessageAccepted
}

/*------------*/

func Matches(m Message, objectURN string) bool {
	return (GetObjectURN(m.Pack()) == objectURN)
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

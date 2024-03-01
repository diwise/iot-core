package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/topics"
	"github.com/farshidtz/senml/v2"
)

type SenMLMessage interface {
	DeviceID() string
	Pack() senml.Pack
	Timestamp() time.Time
}

type MessageReceived struct {
	Pack_      senml.Pack `json:"pack"`
	Timestamp_ time.Time  `json:"timestamp"`
}

func NewMessageReceived(pack senml.Pack) MessageReceived {
	return MessageReceived{
		Pack_:      pack,
		Timestamp_: time.Now().UTC(),
	}
}

func (m MessageReceived) DeviceID() string {
	return deviceID(m)
}

func (m MessageReceived) Pack() senml.Pack {
	return m.Pack_
}

func (m MessageReceived) Timestamp() time.Time {
	return m.Timestamp_
}

func (m MessageReceived) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}

func (m MessageReceived) ContentType() string {
	return fmt.Sprintf("application/vnd.oma.lwm2m.ext.%s+json", objectID(m))
}

func (m MessageReceived) Error() error {
	if m.DeviceID() == "" {
		return errors.New("device id is missing")
	}

	if m.Timestamp().IsZero() {
		return errors.New("timestamp is mising")
	}

	if len(m.Pack()) == 0 {
		return errors.New("pack is empty")
	}

	return nil
}

type MessageAccepted struct {
	Pack_      senml.Pack `json:"pack"`
	Timestamp_ time.Time  `json:"timestamp"`
}

func NewMessageAccepted(pack senml.Pack, decorators ...EventDecoratorFunc) *MessageAccepted {
	m := &MessageAccepted{
		Pack_:      pack,
		Timestamp_: time.Now().UTC(),
	}

	for _, d := range decorators {
		d(m)
	}

	return m
}

func (m MessageAccepted) DeviceID() string {
	return deviceID(m)
}

func (m MessageAccepted) Pack() senml.Pack {
	return m.Pack_
}

func (m MessageAccepted) Timestamp() time.Time {
	return m.Timestamp_
}

func (m MessageAccepted) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}

func (m MessageAccepted) ContentType() string {
	return fmt.Sprintf("application/vnd.oma.lwm2m.ext.%s+json", objectID(m))
}

func (m MessageAccepted) TopicName() string {
	return topics.MessageAccepted
}

func (m MessageAccepted) Error() error {
	if m.DeviceID() == "" {
		return errors.New("device id is missing")
	}

	if m.Timestamp().IsZero() {
		return errors.New("timestamp is mising")
	}

	if len(m.Pack_) == 0 {
		return errors.New("pack is empty")
	}

	return nil
}

func GetLatLon(m SenMLMessage) (float64, float64, bool) {
	lat := math.SmallestNonzeroFloat64
	lon := math.SmallestNonzeroFloat64

	for _, r := range m.Pack() {
		if r.Unit == senml.UnitLon {
			lon = *r.Value
		}
		if r.Unit == senml.UnitLat {
			lat = *r.Value
		}
	}

	return lat, lon, (lat != math.SmallestNonzeroFloat64 && lon != math.SmallestNonzeroFloat64)
}

func GetV(m SenMLMessage, name string) (float64, bool) {
	if r, ok := GetR(m, name); ok {
		if r.Value != nil {
			return *r.Value, true
		}
		return 0, false
	}
	return 0, false
}

func GetVS(m SenMLMessage, name string) (string, bool) {
	if r, ok := GetR(m, name); ok {
		return r.StringValue, true
	}
	return "", false
}

func GetVB(m SenMLMessage, name string) (bool, bool) {
	if r, ok := GetR(m, name); ok {
		if r.BoolValue != nil {
			return *r.BoolValue, true
		}
		return false, false
	}
	return false, false
}

func GetT(m SenMLMessage, name string) (time.Time, bool) {
	clone := m.Pack().Clone()
	bn := clone[0].BaseName
	n := fmt.Sprintf("%s%s", bn, name)

	clone.Normalize()

	for _, r := range clone {
		if strings.EqualFold(n, r.Name) {
			return time.Unix(int64(r.Time), 0).UTC(), true
		}
	}

	return time.Time{}, false
}

func GetR(m SenMLMessage, name string) (senml.Record, bool) {
	for _, r := range m.Pack() {
		if strings.EqualFold(r.Name, name) {
			return r, true
		}
	}
	return senml.Record{}, false
}

func ObjectURNMatches(m SenMLMessage, objectURN string) bool {
	return (urn(m) == objectURN)
}

func Get[T float64 | string | bool](m SenMLMessage, id int) (T, bool) {

	n := fmt.Sprint(id)
	t := *new(T)

	switch reflect.TypeOf(t).Kind() {
	case reflect.Float64:
		if v, ok := GetV(m, n); ok {
			if r, ok := reflect.ValueOf(v).Interface().(T); ok {
				return r, true
			}
		}
	case reflect.Bool:
		if vb, ok := GetVB(m, n); ok {
			if r, ok := reflect.ValueOf(vb).Interface().(T); ok {
				return r, true
			}
		}
	case reflect.String:
		if vs, ok := GetVS(m, n); ok {
			if r, ok := reflect.ValueOf(vs).Interface().(T); ok {
				return r, true
			}
		}

	default:
		return *new(T), false
	}

	return *new(T), false
}

func deviceID(m SenMLMessage) string {
	if m.Pack()[0].Name != "0" {
		return ""
	}
	parts := strings.Split(m.Pack()[0].BaseName, "/")

	return parts[0]
}

func urn(m SenMLMessage) string {
	if m.Pack()[0].Name != "0" {
		return ""
	}
	return m.Pack()[0].StringValue
}

func objectID(m SenMLMessage) string {
	parts := strings.Split(urn(m), ":")
	return parts[len(parts)-1]
}

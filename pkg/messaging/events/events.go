package events

import (
	"github.com/diwise/iot-core/pkg/messaging/topics"
)

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type MessageAccepted struct {
	Sensor     string `json:"sensorID"`
	SensorType string `json:"sensorType"`

	Location *location `json:"location,omitempty"`

	Type        string  `json:"type"`
	SensorValue float64 `json:"sensorValue"`
}

func NewMessageAccepted(sensor, sensorType, valueType string, value float64) *MessageAccepted {
	msg := &MessageAccepted{
		Sensor:     sensor,
		SensorType: sensorType,

		Type:        valueType,
		SensorValue: value,
	}
	return msg
}

func (m *MessageAccepted) ContentType() string {
	return "application/json"
}

func (m *MessageAccepted) TopicName() string {
	return topics.MessageAccepted
}

func (m MessageAccepted) AtLocation(latitude, longitude float64) MessageAccepted {
	m.Location = &location{
		Latitude:  latitude,
		Longitude: longitude,
	}
	return m
}

func (m MessageAccepted) IsLocated() bool {
	return m.Location != nil
}

func (m MessageAccepted) Latitude() float64 {
	if m.Location == nil {
		return 0
	}

	return m.Location.Latitude
}

func (m MessageAccepted) Longitude() float64 {
	if m.Location == nil {
		return 0
	}

	return m.Location.Longitude
}

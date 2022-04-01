package events

import "github.com/diwise/iot-core/pkg/messaging/topics"

type MessageAccepted struct {
	Sensor      string  `json:"sensorID"`
	Type        string  `json:"type"`
	SensorValue float64 `json:"sensorValue"`
}

func (m *MessageAccepted) ContentType() string {
	return "application/json"
}

func (m *MessageAccepted) TopicName() string {
	return topics.MessageAccepted
}

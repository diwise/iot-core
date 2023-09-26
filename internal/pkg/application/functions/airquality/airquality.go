package airquality

import (
	"context"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "airquality"
)

type AirQuality interface {
	Handle(context.Context, *events.MessageAccepted, func(prop string, value float64, ts time.Time) error) (bool, error)
}

func New() AirQuality {
	aq := &airquality{
		Particulates: particulates{},
	}
	return aq
}

type airquality struct {
	Particulates particulates `json:"particulates"`
	Temperature  float64      `json:"temperature"`
	Timestamp    time.Time    `json:"timestamp"`
}

type particulates struct {
	PM1  float64 `json:"pm1"`
	PM10 float64 `json:"pm10"`
	PM25 float64 `json:"pm25"`
	NO   float64 `json:"no"`
	NO2  float64 `json:"no2"`
	CO2  float64 `json:"co2"`
}

func (aq airquality) Handle(context.Context, *events.MessageAccepted, func(prop string, value float64, ts time.Time) error) (bool, error) {
	return false, nil
}

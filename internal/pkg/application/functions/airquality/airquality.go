package airquality

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const (
	FunctionTypeName string = "airquality"
)

type AirQuality interface {
	Handle(context.Context, *events.MessageAccepted, bool, func(prop string, value float64, ts time.Time) error) (bool, error)
	Temperature() float64
}

func New() AirQuality {
	aq := &airquality{
		Particulates_: particulates{},
	}
	return aq
}

type airquality struct {
	Particulates_ particulates `json:"particulates"`
	Temperature_  float64      `json:"temperature"`
	Timestamp_    time.Time    `json:"timestamp"`
}

type particulates struct {
	PM1  float64 `json:"pm1"`
	PM10 float64 `json:"pm10"`
	PM25 float64 `json:"pm25"`
	NO   float64 `json:"no"`
	NO2  float64 `json:"no2"`
	CO2  float64 `json:"co2"`
}

func (aq *airquality) Temperature() float64 {
	return aq.Temperature_
}

const (
	lwm2mTemperature string = "5700"
	lwm2mPM1         string = "5"
	lwm2mPM10        string = "1"
	lwm2mPM25        string = "3"
	lwm2mNO          string = "19"
	lwm2mNO2         string = "15"
	lwm2mCO2         string = "17"
)

func (aq *airquality) Handle(ctx context.Context, e *events.MessageAccepted, onupdate bool, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	log := logging.GetFromContext(ctx)

	temp, tempOk := e.GetFloat64(lwm2mTemperature)
	pm1, pm1Ok := e.GetFloat64(lwm2mPM1)
	pm10, pm10Ok := e.GetFloat64(lwm2mPM10)
	pm25, pm25Ok := e.GetFloat64(lwm2mPM25)
	no, noOk := e.GetFloat64(lwm2mNO)
	no2, no2Ok := e.GetFloat64(lwm2mNO2)
	co2, co2Ok := e.GetFloat64(lwm2mCO2)

	hasChanged := false
	var errs []error

	if tempOk {
		if aq.Temperature_ != temp {
			aq.Temperature_ = temp
			errs = append(errs, onchange("temperature", temp, getTime(e, lwm2mTemperature)))
			hasChanged = true
		}
	}

	if pm1Ok {
		if aq.Particulates_.PM1 != pm1 {
			aq.Particulates_.PM1 = pm1
			errs = append(errs, onchange("pm1", pm1, getTime(e, lwm2mPM1)))
			hasChanged = true
		}
	}

	if pm10Ok {
		if aq.Particulates_.PM10 != pm10 {
			aq.Particulates_.PM10 = pm10
			errs = append(errs, onchange("pm10", pm10, getTime(e, lwm2mPM10)))
			hasChanged = true
		}
	}

	if pm25Ok {
		if aq.Particulates_.PM25 != pm25 {
			aq.Particulates_.PM25 = pm25
			errs = append(errs, onchange("pm25", pm25, getTime(e, lwm2mPM25)))
			hasChanged = true
		}
	}

	if noOk {
		if aq.Particulates_.NO != no {
			aq.Particulates_.NO = no
			errs = append(errs, onchange("no", no, getTime(e, lwm2mNO)))
			hasChanged = true
		}
	}

	if no2Ok {
		if aq.Particulates_.NO2 != no2 {
			aq.Particulates_.NO2 = no2
			errs = append(errs, onchange("no2", no2, getTime(e, lwm2mNO2)))
			hasChanged = true
		}
	}

	if co2Ok {
		if aq.Particulates_.CO2 != co2 {
			aq.Particulates_.CO2 = co2
			errs = append(errs, onchange("co2", co2, getTime(e, lwm2mCO2)))
			hasChanged = true
		}
	}

	if hasChanged {
		aq.Timestamp_ = time.Now().UTC()
	}

	b, _ := json.Marshal(aq)
	log.Debug(fmt.Sprintf("AirQuality changed: %t.\n%s", hasChanged, string(b)))

	return hasChanged, errors.Join(errs...)
}

func getTime(e *events.MessageAccepted, name string) time.Time {
	t, tOk := e.GetTimeForRec(name)
	if tOk {
		return t
	}
	return time.Now().UTC()
}

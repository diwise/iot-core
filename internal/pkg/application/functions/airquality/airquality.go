package airquality

import (
	"context"
	"errors"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FunctionTypeName string = "airquality"
)

type AirQuality interface {
	Handle(context.Context, *events.MessageAccepted, func(prop string, value float64, ts time.Time) error) (bool, any, error)
	Temperature() float64
}

func New() AirQuality {
	aq := &airquality{
		Particulates_: &particulates{},
	}
	return aq
}

type AQ struct{}

type airquality struct {
	Particulates_ *particulates `json:"particulates,omitempty"`
	Temperature_  *float64      `json:"temperature,omitempty"`
	Timestamp_    time.Time     `json:"timestamp"`
}

type particulates struct {
	PM1  *float64 `json:"pm1,omitempty"`
	PM10 *float64 `json:"pm10,omitempty"`
	PM25 *float64 `json:"pm25,omitempty"`
	NO   *float64 `json:"no,omitempty"`
	NO2  *float64 `json:"no2,omitempty"`
	CO2  *float64 `json:"co2,omitempty"`
}

func (aq *airquality) Temperature() float64 {
	return *aq.Temperature_
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

func changed(a *float64, b float64) bool {
	if a == nil {
		return true
	}
	return *a != b
}

func (aq *airquality) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, any, error) {

	temp, tempOk := e.GetFloat64(lwm2mTemperature)
	pm1, pm1Ok := e.GetFloat64(lwm2mPM1)
	pm10, pm10Ok := e.GetFloat64(lwm2mPM10)
	pm25, pm25Ok := e.GetFloat64(lwm2mPM25)
	no, noOk := e.GetFloat64(lwm2mNO)
	no2, no2Ok := e.GetFloat64(lwm2mNO2)
	co2, co2Ok := e.GetFloat64(lwm2mCO2)

	hasChanged := false
	var errs []error

	diff := airquality{
		Particulates_: &particulates{},
	}

	if tempOk {
		if changed(aq.Temperature_, temp) {
			err := onchange("temperature", temp, getTime(e, lwm2mTemperature))

			if err != nil {
				errs = append(errs, err)
			} else {
				aq.Temperature_ = &temp
				diff.Temperature_ = &temp
				hasChanged = true
			}
		}
	}

	if pm1Ok {
		if changed(aq.Particulates_.PM1, pm1) {
			err := onchange("pm1", pm1, getTime(e, lwm2mPM1))

			if err != nil {
				errs = append(errs, err)
			} else {
				aq.Particulates_.PM1 = &pm1
				diff.Particulates_.PM1 = &pm1
				hasChanged = true
			}
		}
	}

	if pm10Ok {
		if changed(aq.Particulates_.PM10, pm10) {
			err := onchange("pm10", pm10, getTime(e, lwm2mPM10))

			if err != nil {
				errs = append(errs, err)
			} else {
				aq.Particulates_.PM10 = &pm10
				diff.Particulates_.PM10 = &pm10
				hasChanged = true
			}
		}
	}

	if pm25Ok {
		if changed(aq.Particulates_.PM25, pm25) {
			err := onchange("pm25", pm25, getTime(e, lwm2mPM25))

			if err != nil {
				errs = append(errs, err)
			} else {
				aq.Particulates_.PM25 = &pm25
				diff.Particulates_.PM25 = &pm25
				hasChanged = true
			}
		}
	}

	if noOk {
		if changed(aq.Particulates_.NO, no) {
			err := onchange("no", no, getTime(e, lwm2mNO))
			if err != nil {
				errs = append(errs, err)
			} else {
				aq.Particulates_.NO = &no
				diff.Particulates_.NO = &no
				hasChanged = true
			}
		}
	}

	if no2Ok {
		if changed(aq.Particulates_.NO2, no2) {
			err := onchange("no2", no2, getTime(e, lwm2mNO2))

			if err != nil {
				errs = append(errs, err)
			} else {
				aq.Particulates_.NO2 = &no2
				diff.Particulates_.NO2 = &no2
				hasChanged = true
			}
		}
	}

	if co2Ok {
		if changed(aq.Particulates_.CO2, co2) {
			err := onchange("co2", co2, getTime(e, lwm2mCO2))

			if err != nil {
				errs = append(errs, err)
			} else {
				aq.Particulates_.CO2 = &co2
				diff.Particulates_.CO2 = &co2
				hasChanged = true
			}
		}
	}

	if hasChanged {
		aq.Timestamp_ = time.Now().UTC()
		diff.Timestamp_ = aq.Timestamp_
	}

	return hasChanged, diff, errors.Join(errs...)
}

func getTime(e *events.MessageAccepted, name string) time.Time {
	t, tOk := e.GetTimeForRec(name)
	if tOk {
		return t
	}
	return time.Now().UTC()
}

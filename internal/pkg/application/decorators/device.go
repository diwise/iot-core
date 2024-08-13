package decorators

import (
	"context"
	"fmt"
	"math"

	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type ValueFinder func() float64

const (
	BatteryLevel       string = "9"
	PowerSourceVoltage string = "7"
	DeviceObjectID     string = "3"
)

func GetMaxPowerSourceVoltage(ctx context.Context, maxValueFinder measurements.MaxValueFinder, deviceID string) ValueFinder {
	powerSourceVoltageMeasurementID := fmt.Sprintf("%s/%s/%s", deviceID, DeviceObjectID, PowerSourceVoltage)

	m, err := maxValueFinder.GetMaxValue(ctx, powerSourceVoltageMeasurementID)
	if err != nil {
		return func() float64 {
			return 0.0
		}
	}

	return func() float64 {
		return m
	}
}

func Device(ctx context.Context, max ValueFinder) events.EventDecoratorFunc {
	log := logging.GetFromContext(ctx)

	return func(m *events.MessageAccepted) {
		objID := events.GetObjectID(m.Pack)
		if objID != DeviceObjectID {
			return
		}

		_, ok := m.Pack.GetValue(senml.FindByName(BatteryLevel))
		if ok {
			log.Debug("battery level already set")
			return
		}

		vvd, ok := m.Pack.GetValue(senml.FindByName(PowerSourceVoltage))
		if !ok {
			log.Warn("no power source voltage found")
			return
		}

		percentage := math.RoundToEven((vvd / max()) * 100)

		if percentage < 0 {
			percentage = 0
		}

		if percentage > 100 {
			percentage = 100
		}

		m.Pack = append(m.Pack, senml.Record{
			Name:  BatteryLevel,
			Value: &percentage,
			Unit:  "%",
		})
	}
}

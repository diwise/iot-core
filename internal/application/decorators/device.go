package decorators

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/diwise/iot-core/internal/application/measurements"
	"github.com/diwise/iot-core/internal/infrastructure/cache"
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

var c *cache.Cache

var stopCacheCleanup func()

func init() {
	c = cache.NewCache()
	stopCacheCleanup = c.Cleanup(1 * time.Hour)
}

// StopCacheCleanup terminates the cache sweeper. It is safe to call
// more than once. Wired to service shutdown at the earliest runner
// migration that owns shutdown (CORE-004); until then the process
// lifetime bounds it.
func StopCacheCleanup() {
	if stopCacheCleanup != nil {
		stopCacheCleanup()
	}
}

func GetMaxPowerSourceVoltage(ctx context.Context, maxValueFinder measurements.MaxValueFinder, deviceID string) ValueFinder {
	log := logging.GetFromContext(ctx)

	powerSourceVoltageMeasurementID := fmt.Sprintf("%s/%s/%s", deviceID, DeviceObjectID, PowerSourceVoltage)

	if value, ok := c.Get(powerSourceVoltageMeasurementID); ok {
		log.Debug(fmt.Sprintf("max power source voltage found in cache for %s", powerSourceVoltageMeasurementID))
		return func() float64 {
			return value.(float64)
		}
	}

	m, err := maxValueFinder.GetMaxValue(ctx, powerSourceVoltageMeasurementID)
	if err != nil {
		return func() float64 {
			return 0.0
		}
	}

	c.Set(powerSourceVoltageMeasurementID, m, 7*24*time.Hour)
	log.Debug(fmt.Sprintf("set max power source voltage for %s to %f", powerSourceVoltageMeasurementID, m))

	return func() float64 {
		return m
	}
}

func Device(ctx context.Context, max ValueFinder) events.EventDecoratorFunc {
	log := logging.GetFromContext(ctx)

	return func(m events.Message) {
		objID := m.ObjectID()
		if objID != DeviceObjectID {
			return
		}

		_, ok := m.Pack().GetValue(senml.FindByName(BatteryLevel))
		if ok {
			log.Debug("battery level already set")
			return
		}

		vvd, ok := m.Pack().GetValue(senml.FindByName(PowerSourceVoltage))
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

		m.Append(senml.Record{
			Name:  BatteryLevel,
			Value: &percentage,
			Unit:  "%",
		})
	}
}

package decorators

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const (
	DigitalInputState    string = "5500"
	DigitalInputCounter  string = "5501"
	DigitalInputObjectID string = "3200"
)

func GetNumberOfTrueValues(ctx context.Context, countValueFinder measurements.CountBoolValueFinder, deviceID string) ValueFinder {
	digitalInputStateMeasurementID := fmt.Sprintf("%s/%s/%s", deviceID, DigitalInputObjectID, DigitalInputState)

	m, err := countValueFinder.GetCountTrueValues(ctx, digitalInputStateMeasurementID, time.Unix(0, 0).UTC(), time.Now().UTC())
	if err != nil {
		return func() float64 {
			return 0
		}
	}

	return func() float64 {
		return m
	}
}

func DigitalInput(ctx context.Context, count ValueFinder) events.EventDecoratorFunc {
	log := logging.GetFromContext(ctx)

	return func(m events.Message) {
		objID := m.ObjectID()
		if objID != DigitalInputObjectID {
			return
		}

		_, ok := m.Pack().GetValue(senml.FindByName(DigitalInputCounter))
		if ok {
			log.Debug("digital input counter already set")
			return
		}

		digitalInputCounter := math.Ceil(count())

		if digitalInputCounter < 0 {
			digitalInputCounter = 0
		}

		if vb, ok := m.Pack().GetBoolValue(senml.FindByName(DigitalInputState)); ok {
			if vb {
				digitalInputCounter++
			}
		}

		log.Debug("setting digital input counter", "value", digitalInputCounter)
		
		m.Append(senml.Record{
			Name:  DigitalInputCounter,
			Value: &digitalInputCounter,
		})
	}
}

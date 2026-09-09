package decorators

import (
	"context"
	"math"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const (
	ActualNumberOfPersons string = "1"
	DailyNumberOfPersons  string = "2"
	PeopleCounterObjectID string = "3434"
)

func PeopleCounter(ctx context.Context, count ValueFinder) events.EventDecoratorFunc {
	log := logging.GetFromContext(ctx)

	return func(m events.Message) {
		objID := m.ObjectID()
		if objID != PeopleCounterObjectID {
			return
		}

		_, ok := m.Pack().GetValue(senml.FindByName(DailyNumberOfPersons))
		if ok {
			log.Debug("daily number of persons already set")
			return
		}

		dailyNumberOfPersons := math.Ceil(count())

		if dailyNumberOfPersons < 0 {
			dailyNumberOfPersons = 0
		}

		log.Debug("setting daily number of persons", "value", dailyNumberOfPersons)

		m.Append(senml.Record{
			Name:  DailyNumberOfPersons,
			Value: &dailyNumberOfPersons,
		})
	}
}

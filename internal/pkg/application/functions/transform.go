package functions

import (
	"github.com/diwise/iot-core/internal/pkg/application/functions/counters"
	"github.com/diwise/iot-core/internal/pkg/application/functions/levels"
	"github.com/diwise/iot-core/internal/pkg/application/functions/stopwatch"
	"github.com/diwise/iot-core/internal/pkg/application/functions/timers"
	"github.com/diwise/senml"

	"github.com/diwise/iot-agent/pkg/lwm2m"
)

func Transform(f Function) []senml.Pack {
	packs := make([]senml.Pack, 0)

	fn, ok := f.(*fnct)
	if !ok {
		return packs
	}

	switch fn.Type_ {
	case counters.FunctionTypeName:
		if counter, err := toCounter(f); err == nil {
			packs = append(packs, counter)
		}
	case levels.FunctionTypeName:
		if fillingLevel, err := toFillingLevel(f); err == nil {
			packs = append(packs, fillingLevel)
		}
	case stopwatch.FunctionTypeName:
		if stopwatch, err := toStopwatch(f); err == nil {
			packs = append(packs, stopwatch)
		}
	case timers.FunctionTypeName:
		if timer, err := toTimer(f); err == nil {
			packs = append(packs, timer)
		}
	}

	return packs
}

func toCounter(f Function) (senml.Pack, error) {
	fn := f.(*fnct)

	n := fn.Counter.Count()

	counter := lwm2m.NewDigitalInput(fn.DeviceID_, fn.Counter.State(), fn.Timestamp)
	counter.DigitalInputCounter = &n

	return lwm2m.ToPack(counter), nil
}

func toStopwatch(f Function) (senml.Pack, error) {
	fn := f.(*fnct)

	state := fn.Stopwatch.State()

	sw := lwm2m.NewStopwatch(fn.DeviceID_, fn.Stopwatch.CumulativeTime().Seconds(), fn.Timestamp)
	sw.OnOff = &state
	sw.DigitalInputCounter = fn.Stopwatch.Count()

	return lwm2m.ToPack(sw), nil
}

func toFillingLevel(f Function) (senml.Pack, error) {
	fn := f.(*fnct)

	level := int64(fn.Level.Current())
	percent := fn.Level.Percent()

	fillingLevel := lwm2m.NewFillingLevel(fn.DeviceID_, percent, fn.Timestamp)
	fillingLevel.ActualFillingLevel = &level

	return lwm2m.ToPack(fillingLevel), nil
}

func toTimer(f Function) (senml.Pack, error) {
	fn := f.(*fnct)

	timer := lwm2m.NewTimer(fn.DeviceID_, fn.Timer.Duration().Seconds(), fn.Timestamp)
	cumulativeTime := fn.Timer.TotalDuration().Seconds()
	timer.CumulativeTime = &cumulativeTime

	return lwm2m.ToPack(timer), nil
}

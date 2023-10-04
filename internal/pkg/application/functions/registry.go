package functions

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/diwise/iot-core/internal/pkg/application/functions/airquality"
	"github.com/diwise/iot-core/internal/pkg/application/functions/buildings"
	"github.com/diwise/iot-core/internal/pkg/application/functions/counters"
	"github.com/diwise/iot-core/internal/pkg/application/functions/levels"
	"github.com/diwise/iot-core/internal/pkg/application/functions/presences"
	"github.com/diwise/iot-core/internal/pkg/application/functions/timers"
	"github.com/diwise/iot-core/internal/pkg/application/functions/waterqualities"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Registry interface {
	Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Function, error)
	Get(ctx context.Context, functionID string) (Function, error)
}

func NewRegistry(ctx context.Context, input io.Reader, storage database.Storage) (Registry, error) {

	r := &reg{
		f: make(map[string]Function),
	}

	var err error

	numErrors := 0
	numFunctions := 0

	logger := logging.GetFromContext(ctx)

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		tokens := strings.Split(line, ";")
		tokenCount := len(tokens)

		if tokenCount >= 4 {
			f := &fnct{
				fnctMetadata: fnctMetadata{
					ID_:     tokens[0],
					Name_:   tokens[1],
					Type:    tokens[2],
					SubType: tokens[3],
				},

				storage: storage,
			}

			if f.Type == counters.FunctionTypeName {
				f.Counter = counters.New()
				f.handle = f.Counter.Handle
				f.defaultHistoryLabel = "count"
			} else if f.Type == levels.FunctionTypeName {
				levelConfig := ""
				if tokenCount > 5 {
					levelConfig = tokens[5]
				}
				f.defaultHistoryLabel = "level"
				l := lastLogValue(ctx, storage, f)

				logger.Debug().Msgf("new level %s with value %f", f.ID_, l.Value)

				f.Level, err = levels.New(levelConfig, l.Value)
				if err != nil {
					return nil, err
				}

				f.handle = f.Level.Handle
			} else if f.Type == presences.FunctionTypeName {
				f.defaultHistoryLabel = "presence"
				l := lastLogValue(ctx, storage, f)

				logger.Debug().Msgf("new presence %s with value %f", f.ID_, l.Value)

				f.Presence = presences.New(l.Value)
				f.handle = f.Presence.Handle
			} else if f.Type == timers.FunctionTypeName {
				f.Timer = timers.New()
				f.handle = f.Timer.Handle
				f.defaultHistoryLabel = "time"
			} else if f.Type == waterqualities.FunctionTypeName {
				f.WaterQuality = waterqualities.New()
				f.handle = f.WaterQuality.Handle
				f.defaultHistoryLabel = "temperature"
			} else if f.Type == buildings.FunctionTypeName {
				f.Building = buildings.New()
				f.handle = f.Building.Handle
				f.defaultHistoryLabel = "power"
			} else if f.Type == airquality.FunctionTypeName {
				f.AirQuality = airquality.New()
				f.handle = f.AirQuality.Handle
				f.defaultHistoryLabel = "temperature"
			} else {
				numErrors++
				if numErrors > 1 {
					return nil, fmt.Errorf("unable to parse function config line: \"%s\"", line)
				}
				continue
			}

			storage.AddFnct(ctx, f.ID_, f.Type, f.SubType, f.Tenant, f.Source, 0, 0)

			r.f[tokens[4]] = f
			numFunctions++
		}
	}

	logger.Info().Msgf("loaded %d functions from config file", numFunctions)

	return r, nil
}

type reg struct {
	f map[string]Function
}

func (r *reg) Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Function, error) {

	if len(matchers) == 0 {
		return nil, fmt.Errorf("at least one matcher must be supplied to Find")
	}

	var result []Function

	// TODO: Handle multiple chained matchers
	for _, match := range matchers {
		result = match(r)
	}

	return result, nil
}

func (r *reg) Get(ctx context.Context, functionID string) (Function, error) {
	for _, f := range r.f {
		if f.ID() == functionID {
			return f, nil
		}
	}

	return nil, errors.New("no such function")
}

type RegistryMatcherFunc func(r *reg) []Function

func MatchAll() RegistryMatcherFunc {
	return func(r *reg) []Function {
		result := make([]Function, 0, len(r.f))
		for _, f := range r.f {
			result = append(result, f)
		}
		return result
	}
}

func MatchSensor(sensorId string) RegistryMatcherFunc {
	return func(r *reg) []Function {
		f, ok := r.f[sensorId]
		if !ok {
			return []Function{}
		}

		return []Function{f}
	}
}

func lastLogValue(ctx context.Context, s database.Storage, f *fnct) database.LogValue {
	lv, err := s.History(ctx, f.ID_, f.defaultHistoryLabel, 1)
	if err != nil {
		return database.LogValue{}
	}
	if len(lv) == 0 {
		return database.LogValue{}
	}
	return lv[0]
}

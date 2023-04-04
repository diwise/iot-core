package functions

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/diwise/iot-core/internal/pkg/application/functions/counters"
	"github.com/diwise/iot-core/internal/pkg/application/functions/levels"
	"github.com/diwise/iot-core/internal/pkg/application/functions/presences"
	"github.com/diwise/iot-core/internal/pkg/application/functions/waterqualities"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Registry interface {
	Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Function, error)
}

func NewRegistry(ctx context.Context, input io.Reader) (Registry, error) {

	r := &reg{
		f: make(map[string]Function),
	}

	var err error

	numErrors := 0
	numFunctions := 0

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		tokens := strings.Split(line, ";")
		tokenCount := len(tokens)

		if tokenCount >= 4 {
			f := &fnct{
				ID:      tokens[0],
				Type:    tokens[1],
				SubType: tokens[2],
			}

			if f.Type == counters.FeatureTypeName {
				f.Counter = counters.New()
				f.handle = f.Counter.Handle
			} else if f.Type == levels.FeatureTypeName {
				levelConfig := ""
				if tokenCount > 4 {
					levelConfig = tokens[4]
				}

				f.Level, err = levels.New(levelConfig)
				if err != nil {
					return nil, err
				}

				f.handle = f.Level.Handle
			} else if f.Type == presences.FeatureTypeName {
				f.Presence = presences.New()
				f.handle = f.Presence.Handle
			} else if f.Type == waterqualities.FeatureTypeName {
				f.WaterQuality = waterqualities.New()
				f.handle = f.WaterQuality.Handle
			} else {
				numErrors++
				if numErrors > 1 {
					return nil, fmt.Errorf("unable to parse feature config line: \"%s\"", line)
				}
				continue
			}

			r.f[tokens[3]] = f
			numFunctions++
		}
	}

	logger := logging.GetFromContext(ctx)
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

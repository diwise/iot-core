package features

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/diwise/iot-core/internal/pkg/application/features/counters"
	"github.com/diwise/iot-core/internal/pkg/application/features/levels"
	"github.com/diwise/iot-core/internal/pkg/application/features/presences"
	"github.com/diwise/iot-core/internal/pkg/application/features/waterqualities"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Registry interface {
	Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Feature, error)
	Get(ctx context.Context, featureID string) (Feature, error)
}

func NewRegistry(ctx context.Context, input io.Reader) (Registry, error) {

	r := &reg{
		f: make(map[string]Feature),
	}

	var err error

	numErrors := 0
	numFeatures := 0

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		tokens := strings.Split(line, ";")
		tokenCount := len(tokens)

		if tokenCount >= 4 {
			f := &feat{
				ID_:     tokens[0],
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
			numFeatures++
		}
	}

	logger := logging.GetFromContext(ctx)
	logger.Info().Msgf("loaded %d features from config file", numFeatures)

	return r, nil
}

type reg struct {
	f map[string]Feature
}

func (r *reg) Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Feature, error) {

	if len(matchers) == 0 {
		return nil, fmt.Errorf("at least one matcher must be supplied to Find")
	}

	var result []Feature

	// TODO: Handle multiple chained matchers
	for _, match := range matchers {
		result = match(r)
	}

	return result, nil
}

func (r *reg) Get(ctx context.Context, featureID string) (Feature, error) {
	for _, f := range r.f {
		if f.ID() == featureID {
			return f, nil
		}
	}

	return nil, errors.New("no such feature")
}

type RegistryMatcherFunc func(r *reg) []Feature

func MatchAll() RegistryMatcherFunc {
	return func(r *reg) []Feature {
		result := make([]Feature, 0, len(r.f))
		for _, f := range r.f {
			result = append(result, f)
		}
		return result
	}
}

func MatchSensor(sensorId string) RegistryMatcherFunc {
	return func(r *reg) []Feature {
		f, ok := r.f[sensorId]
		if !ok {
			return []Feature{}
		}

		return []Feature{f}
	}
}

package functions

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions/airquality"
	"github.com/diwise/iot-core/internal/pkg/application/functions/buildings"
	"github.com/diwise/iot-core/internal/pkg/application/functions/counters"
	"github.com/diwise/iot-core/internal/pkg/application/functions/digitalinput"
	"github.com/diwise/iot-core/internal/pkg/application/functions/levels"
	"github.com/diwise/iot-core/internal/pkg/application/functions/presences"
	"github.com/diwise/iot-core/internal/pkg/application/functions/stopwatch"
	"github.com/diwise/iot-core/internal/pkg/application/functions/timers"
	"github.com/diwise/iot-core/internal/pkg/application/functions/waterqualities"
	"github.com/google/uuid"
)

//go:generate moq -rm -out registry_mock.go . Registry
type Registry interface {
	Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Function, error)
	Get(ctx context.Context, id string) (Function, error)
	Update(ctx context.Context, id string, fn Function) error
	Add(ctx context.Context, id, label string, value float64, timestamp time.Time) error
}

//go:generate moq -rm -out registry_mock.go . RegistryStorer
type RegistryStorer interface {
	Add(ctx context.Context, id, label string, value float64, timestamp time.Time) error

	LoadState(ctx context.Context, id string) ([]byte, error)
	SaveState(ctx context.Context, id string, a any) error

	AddSetting(ctx context.Context, id string, s Setting) error
	GetSettings(ctx context.Context) ([]Setting, error)
}

type Setting struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	SubType  string `json:"subtype"`
	DeviceID string `json:"deviceID"`
	OnUpdate bool   `json:"onupdate"`
	Args     string `json:"args"`
}

type registry struct {
	funcs  map[string]Function
	storer RegistryStorer
}

func NewRegistry(ctx context.Context, input io.Reader, rs RegistryStorer) (Registry, error) {
	r := &registry{
		funcs:  make(map[string]Function),
		storer: rs,
	}

	err := r.seed(ctx, input)
	if err != nil {
		return nil, err
	}

	err = r.init(ctx)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *registry) Add(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
	return r.storer.Add(ctx, id, label, value, timestamp)
}

func (r *registry) Update(ctx context.Context, id string, fn Function) error {
	r.funcs[fn.DeviceID()] = fn
	return r.storer.SaveState(ctx, id, fn)
}

func (r *registry) seed(ctx context.Context, input io.Reader) error {
	n := 0
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {

		if n == 0 {
			n++
			continue
		}

		line := scanner.Text()

		tokens := strings.Split(line, ";")
		tokenCount := len(tokens)

		if tokenCount >= 4 {
			deviceID := strings.ToLower(tokens[4])

			fn := Setting{
				ID:       tokens[0],
				Name:     tokens[1],
				Type:     tokens[2],
				SubType:  tokens[3],
				DeviceID: deviceID,
			}

			if tokenCount == 7 {
				fn.Args = tokens[5]
				fn.OnUpdate = tokens[6] == "true"
			}

			if tokenCount == 6 {
				fn.OnUpdate = tokens[5] == "true"
			}

			r.storer.AddSetting(ctx, fn.ID, fn)
		}
	}

	return nil
}

func newFnct(s Setting) *fnct {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}

	return &fnct{
		ID_:       s.ID,
		Name_:     s.Name,
		Type_:     s.Type,
		SubType_:  s.SubType,
		DeviceID_: s.DeviceID,
		OnUpdate:  s.OnUpdate,
	}
}

func newCounter(s Setting) Function {
	f := newFnct(s)
	f.Counter = counters.New()
	f.handle = f.Counter.Handle
	//f.defaultHistoryLabel = "count"
	return f
}

func newLevel(s Setting) Function {
	var err error

	f := newFnct(s)

	//f.defaultHistoryLabel = "level"

	f.Level, err = levels.New(s.Args, 0)
	if err != nil {
		return nil
	}

	f.handle = f.Level.Handle
	return f
}

func newPresence(s Setting) Function {
	f := newFnct(s)
	//f.defaultHistoryLabel = "presence"
	f.Presence = presences.New(0)
	f.handle = f.Presence.Handle
	return f
}

func newTimer(s Setting) Function {
	f := newFnct(s)
	f.Timer = timers.New()
	f.handle = f.Timer.Handle
	//f.defaultHistoryLabel = "time"
	return f
}

func newWaterQuality(s Setting) Function {
	f := newFnct(s)
	f.WaterQuality = waterqualities.New()
	f.handle = f.WaterQuality.Handle
	//f.defaultHistoryLabel = "temperature"
	return f
}

func newBuilding(s Setting) Function {
	f := newFnct(s)
	f.Building = buildings.New()
	f.handle = f.Building.Handle
	//f.defaultHistoryLabel = "power"
	return f
}

func newAirQuality(s Setting) Function {
	f := newFnct(s)
	f.AirQuality = airquality.New()
	f.handle = f.AirQuality.Handle
	//f.defaultHistoryLabel = "temperature"
	return f
}

func newStopwatch(s Setting) Function {
	f := newFnct(s)
	f.Stopwatch = stopwatch.New()
	f.handle = f.Stopwatch.Handle
	//f.defaultHistoryLabel = "duration"
	return f
}

func newDigitalInput(s Setting) Function {
	f := newFnct(s)
	//f.defaultHistoryLabel = "digitalinput"
	f.DigitalInput = digitalinput.New(0)
	f.handle = f.DigitalInput.Handle
	return f
}

func (r *registry) init(ctx context.Context) error {

	settings, err := r.storer.GetSettings(ctx)
	if err != nil {
		return err
	}

	for _, s := range settings {

		switch s.Type {
		case counters.FunctionTypeName:
			r.funcs[s.DeviceID] = newCounter(s)
		case levels.FunctionTypeName:
			r.funcs[s.DeviceID] = newLevel(s)
		case presences.FunctionTypeName:
			r.funcs[s.DeviceID] = newPresence(s)
		case timers.FunctionTypeName:
			r.funcs[s.DeviceID] = newTimer(s)
		case waterqualities.FunctionTypeName:
			r.funcs[s.DeviceID] = newWaterQuality(s)
		case buildings.FunctionTypeName:
			r.funcs[s.DeviceID] = newBuilding(s)
		case airquality.FunctionTypeName:
			r.funcs[s.DeviceID] = newAirQuality(s)
		case stopwatch.FunctionTypeName:
			r.funcs[s.DeviceID] = newStopwatch(s)
		case digitalinput.FunctionTypeName:
			r.funcs[s.DeviceID] = newDigitalInput(s)
		}

		if r.funcs[s.DeviceID] != nil {
			state, err := r.storer.LoadState(ctx, s.ID)
			if err != nil {
				continue
			}
			err = json.Unmarshal(state, r.funcs[s.DeviceID])
			if err != nil {
				continue
			}
		}
	}

	return nil
}

func (r *registry) Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Function, error) {

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

func (r *registry) Get(ctx context.Context, functionID string) (Function, error) {
	for _, f := range r.funcs {
		if f.ID() == functionID {
			return f, nil
		}
	}

	return nil, errors.New("no such function")
}

type RegistryMatcherFunc func(r *registry) []Function

func MatchAll() RegistryMatcherFunc {
	return func(r *registry) []Function {
		result := make([]Function, 0, len(r.funcs))
		for _, f := range r.funcs {
			result = append(result, f)
		}
		return result
	}
}

func MatchSensor(sensorId string) RegistryMatcherFunc {
	return func(r *registry) []Function {
		f, ok := r.funcs[strings.ToLower(sensorId)]
		if !ok {
			return []Function{}
		}

		return []Function{f}
	}
}

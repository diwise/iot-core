package features

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/diwise/iot-core/internal/pkg/application/features/counters"
	"github.com/diwise/iot-core/internal/pkg/application/features/levels"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

type Registry interface {
	Find(ctx context.Context, sensorID string) ([]Feature, error)
}

type Feature interface {
	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
}

func NewRegistry(input io.Reader) (Registry, error) {

	r := &reg{
		f: make(map[string]Feature),
	}

	var err error

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		tokens := strings.Split(line, ";")
		tokenCount := len(tokens)

		if tokenCount >= 4 {
			f := &feat{
				ID:      tokens[0],
				Type:    tokens[1],
				SubType: tokens[2],
			}

			if f.Type == counters.FeatureTypeName {
				f.Counter = counters.New()
			} else if f.Type == levels.FeatureTypeName {
				levelConfig := ""
				if tokenCount > 4 {
					levelConfig = tokens[4]
				}
				f.Level, err = levels.New(levelConfig)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("unable to parse feature config line: \"%s\"", line)
			}

			r.f[tokens[3]] = f
		}
	}

	return r, nil
}

type reg struct {
	f map[string]Feature
}

func (r *reg) Find(ctx context.Context, sensorID string) ([]Feature, error) {
	f, ok := r.f[sensorID]
	if !ok {
		return []Feature{}, nil
	}

	return []Feature{f}, nil
}

type feat struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	SubType string `json:"subtype"`

	Counter counters.Counter `json:"counter,omitempty"`
	Level   levels.Level     `json:"level,omitempty"`
}

func (f *feat) Handle(ctx context.Context, e *events.MessageAccepted, msgctx messaging.MsgContext) error {
	if f.Counter != nil {
		changed, err := f.Counter.Handle(ctx, e)
		if err != nil {
			return err
		}

		if changed {
			msgctx.PublishOnTopic(ctx, f)
		}
	} else if f.Level != nil {
		changed, err := f.Level.Handle(ctx, e)
		if err != nil {
			return err
		}

		if changed {
			msgctx.PublishOnTopic(ctx, f)
		}
	}

	return nil
}

func (f *feat) ContentType() string {
	return "application/json"
}

func (f *feat) TopicName() string {
	return "features.updated"
}

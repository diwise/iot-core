package levels

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const (
	FunctionTypeName string = "level"
)

type Level interface {
	Handle(context.Context, *events.MessageAccepted, func(string, float64, time.Time) error) (bool, error)
	Current() float64
	Offset() float64
	Percent() float64
}

func New(config string, current float64) (Level, error) {

	lvl := &level{
		cosAlpha: 1.0,
	}

	config = strings.ReplaceAll(config, " ", "")
	settings := strings.Split(config, ",")

	var err error

	for _, s := range settings {
		pair := strings.Split(s, "=")
		if len(pair) == 2 {
			switch pair[0] {
			case "angle":
				angle, err := strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level angle \"%s\": %w", s, err)
				}
				if angle < 0 || angle >= 90.0 {
					return nil, fmt.Errorf("level angle %f not within allowed [0, 90) range", angle)
				}
				// precalculate the cosine of the mount angle (after conversion to radians)
				lvl.cosAlpha = math.Cos(angle * math.Pi / 180.0)
			case "maxd":
				lvl.maxDistance, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			case "maxl":
				lvl.maxLevel, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			case "mean":
				lvl.meanLevel, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			case "offset":
				lvl.offsetLevel, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse offset config \"%s\": %w", s, err)
				}
			default:
				return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
			}
		}
	}

	lvl.Current_ = current
	if isNotZero(lvl.maxLevel) {
		pct := math.Min((lvl.Current_*100.0)/lvl.maxLevel, 100.0)
		if pct < 0 {
			pct = 0
		}
		lvl.Percent_ = &pct
	}

	if isNotZero(lvl.meanLevel) {
		offset := lvl.Current_ - lvl.meanLevel
		lvl.Offset_ = &offset
	}

	return lvl, nil
}

type level struct {
	cosAlpha    float64
	maxDistance float64
	maxLevel    float64
	meanLevel   float64
	offsetLevel float64

	Current_ float64  `json:"current"`
	Percent_ *float64 `json:"percent,omitempty"`
	Offset_  *float64 `json:"offset,omitempty"`
}

func (l *level) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	match := false

	if events.Matches(e, lwm2m.Distance) {
		match = true
	}

	if !match {
		return false, events.ErrNoMatch
	}

	if events.Matches(e, lwm2m.Distance) {
		return l.handleDistance(ctx, e, onchange)
	}

	return false, nil
}

func (l *level) handleDistance(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	const SensorValue string = "5700"

	log := logging.GetFromContext(ctx)

	sensorValue, ok := e.Pack().GetRecord(senml.FindByName(SensorValue))
	if !ok {
		return false, fmt.Errorf("could not find record for sensor value in distance pack")
	}

	distance, ok := sensorValue.GetValue()
	if !ok {
		return false, fmt.Errorf("could not find distance value in distance pack")
	}

	distance += l.offsetLevel

	previousLevel := l.Current_
	previousPercent := l.Percent_

	var errs []error

	// Calculate the current level using the configured angle (if any) and round to two decimals
	l.Current_ = math.Round((l.maxDistance-distance)*l.cosAlpha*100) / 100.0

	if !hasChanged(previousLevel, l.Current_) {
		return false, nil
	}

	ts, ok := sensorValue.GetTime()
	if !ok {
		ts = time.Now().UTC()
	}

	errs = append(errs, onchange("level", l.Current_, ts))

	if isNotZero(l.maxLevel) {
		pct := math.Min((l.Current_*100.0)/l.maxLevel, 100.0)
		l.Percent_ = &pct

		if previousPercent == nil {
			errs = append(errs, onchange("percent", *l.Percent_, ts))
		}

		if previousPercent != nil {
			if hasChanged(*previousPercent, pct) {
				errs = append(errs, onchange("percent", *l.Percent_, ts))
			}
		}
	} else {
		log.Info("cannot calculate percent since maxLevel is not set")
	}

	if isNotZero(l.meanLevel) {
		offset := l.Current_ - l.meanLevel
		l.Offset_ = &offset
	}

	return true, errors.Join(errs...)

}

func (l *level) Current() float64 {
	return l.Current_
}

func (l *level) Offset() float64 {
	if l.Offset_ != nil {
		return *l.Offset_
	}

	return 0.0
}

func (l *level) Percent() float64 {
	if l.Percent_ != nil {
		return *l.Percent_
	}

	return 0.0
}

func hasChanged(prev, new float64) bool {
	return isNotZero(new - prev)
}

func isNotZero(value float64) bool {
	return (math.Abs(value) >= 0.001)
}

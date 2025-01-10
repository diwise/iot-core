package levels

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
		CosAlpha: 1.0,
	}

	config = strings.ReplaceAll(config, " ", "")
	settings := strings.Split(config, ",")

	var err error

	for _, s := range settings {
		pair := strings.Split(s, "=")
		if len(pair) == 2 {
			if pair[0] == "angle" {
				angle, err := strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level angle \"%s\": %w", s, err)
				}
				if angle < 0 || angle >= 90.0 {
					return nil, fmt.Errorf("level angle %f not within allowed [0, 90) range", angle)
				}
				// precalculate the cosine of the mount angle (after conversion to radians)
				lvl.CosAlpha = math.Cos(angle * math.Pi / 180.0)
			} else if pair[0] == "maxd" {
				lvl.MaxDistance, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			} else if pair[0] == "maxl" {
				lvl.MaxLevel, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			} else if pair[0] == "mean" {
				lvl.MeanLevel, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			} else if pair[0] == "offset" {
				lvl.OffsetLevel, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse offset config \"%s\": %w", s, err)
				}
			} else {
				return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
			}
		}
	}

	lvl.Current_ = current
	if isNotZero(lvl.MaxLevel) {
		pct := math.Min((lvl.Current_*100.0)/lvl.MaxLevel, 100.0)
		if pct < 0 {
			pct = 0
		}
		lvl.Percent_ = &pct
	}

	if isNotZero(lvl.MeanLevel) {
		offset := lvl.Current_ - lvl.MeanLevel
		lvl.Offset_ = &offset
	}

	return lvl, nil
}

type level struct {
	CosAlpha    float64 `json:"cosAlpha"`
	MaxDistance float64 `json:"maxDistance"`
	MaxLevel    float64 `json:"maxLevel"`
	MeanLevel   float64 `json:"meanLevel"`
	OffsetLevel float64 `json:"offsetLevel"`

	Current_ float64  `json:"current"`
	Percent_ *float64 `json:"percent,omitempty"`
	Offset_  *float64 `json:"offset,omitempty"`
}

func (l *level) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {
	match := false

	log := logging.GetFromContext(ctx)

	if events.Matches(e, lwm2m.Distance) {
		match = true
	}

	if events.Matches(e, lwm2m.FillingLevel) {
		match = true
	}

	if !match {
		log.Debug(fmt.Sprintf("%s is not a message for level function", e.ObjectID()))
		return false, events.ErrNoMatch
	}

	if events.Matches(e, lwm2m.Distance) {
		log.Debug("level function matches distance")
		return l.handleDistance(ctx, e, onchange)
	}

	if events.Matches(e, lwm2m.FillingLevel) {
		log.Debug("level function matches filling level")
		return l.handleFillingLevel(ctx, e, onchange)
	}

	return false, nil
}

func (l *level) handleFillingLevel(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {

	const (
		ActualFillingPercentage string = "2"
		ActualFillingLevel      string = "3"
		HighThreshold           string = "4"
	)

	percent, percentOk := e.Pack().GetRecord(senml.FindByName(ActualFillingPercentage))
	level, levelOk := e.Pack().GetRecord(senml.FindByName(ActualFillingLevel))
	highThreshold, highThresholdOk := e.Pack().GetValue(senml.FindByName(HighThreshold))

	if !percentOk && !levelOk {
		return false, fmt.Errorf("could not find record for actual filling percentage or actual filling level in fillingLevel pack")
	}

	log := logging.GetFromContext(ctx)

	if highThresholdOk {
		log.Debug("HighThreshold is included in pack, will adjust maxLevel configuration")
		if highThreshold > l.MaxLevel {
			l.MaxLevel = highThreshold
		}
	}

	var errs []error
	changed := false

	offsetLevelIsSet := isNotZero(l.OffsetLevel)
	if offsetLevelIsSet {
		log.Debug("offset is set, will not use percent value", slog.Float64("offset", l.OffsetLevel))
	}

	if percentOk && !offsetLevelIsSet {
		previousPercent := l.Percent_

		p, valueOk := percent.GetValue()
		if !valueOk {
			return false, fmt.Errorf("could not get percent value in fillingLevel pack")
		}

		ts, timeOk := percent.GetTime()
		if !timeOk {
			ts = time.Now().UTC()
		}

		if previousPercent == nil {
			l.Percent_ = &p
			return true, onchange("percent", *l.Percent_, ts)
		}

		if !hasChanged(*previousPercent, p) {
			return false, nil
		}

		l.Percent_ = &p

		changed = true
		errs = append(errs, onchange("percent", *l.Percent_, ts))
	}

	if levelOk {
		previousLevel := l.Current_

		v, valueOk := level.GetValue()
		if !valueOk {
			return false, fmt.Errorf("could not get level value in fillingLevel pack")
		}

		log.Debug("pack contains actual filling level", slog.Float64("actual_filling_level", v), slog.Float64("offset", l.OffsetLevel))

		v += l.OffsetLevel

		ts, timeOk := level.GetTime()
		if !timeOk {
			ts = time.Now().UTC()
		}

		if !hasChanged(previousLevel, v) {
			return false, nil
		}

		log.Debug("level is changed", slog.Float64("old_value", l.Current_), slog.Float64("new_value", v))

		l.Current_ = v

		if isNotZero(l.MaxLevel) {
			previousPercent := l.Percent_
			pct := math.Min((l.Current_*100.0)/l.MaxLevel, 100.0)
			l.Percent_ = &pct

			if previousPercent == nil {
				changed = true
				errs = append(errs, onchange("percent", *l.Percent_, ts))
			}

			if previousPercent != nil {
				if hasChanged(*previousPercent, pct) {
					changed = true
					errs = append(errs, onchange("percent", *l.Percent_, ts))
				}
			}
		} else {
			log.Info("cannot calculate percent since maxLevel is not set")
		}

		changed = true
		errs = append(errs, onchange("level", l.Current_, ts))
	}

	return changed, errors.Join(errs...)
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

	distance += l.OffsetLevel

	previousLevel := l.Current_
	previousPercent := l.Percent_

	var errs []error

	// Calculate the current level using the configured angle (if any) and round to two decimals
	l.Current_ = math.Round((l.MaxDistance-distance)*l.CosAlpha*100) / 100.0

	log.Debug("calculate level using distance", slog.Float64("max_distance", l.MaxDistance), slog.Float64("max_level", l.MaxLevel), slog.Float64("angle", l.CosAlpha), slog.Float64("distance", distance))

	if !hasChanged(previousLevel, l.Current_) {
		log.Debug(fmt.Sprintf("distance has not changed (%f meters)", previousLevel))
		return false, nil
	}

	ts, ok := sensorValue.GetTime()
	if !ok {
		log.Debug("could not get time from sensor value in distance pack, will use Now().UTC()")
		ts = time.Now().UTC()
	}

	errs = append(errs, onchange("level", l.Current_, ts))

	if isNotZero(l.MaxLevel) {
		pct := math.Min((l.Current_*100.0)/l.MaxLevel, 100.0)
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

	if isNotZero(l.MeanLevel) {
		offset := l.Current_ - l.MeanLevel
		l.Offset_ = &offset
	}

	log.Debug("level function handled incoming distance message")

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

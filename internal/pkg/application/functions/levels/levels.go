package levels

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
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
			if pair[0] == "angle" {
				angle, err := strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level angle \"%s\": %w", s, err)
				}
				if angle < 0 || angle >= 90.0 {
					return nil, fmt.Errorf("level angle %f not within allowed [0, 90) range", angle)
				}
				// precalculate the cosine of the mount angle (after conversion to radians)
				lvl.cosAlpha = math.Cos(angle * math.Pi / 180.0)
			} else if pair[0] == "maxd" {
				lvl.maxDistance, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			} else if pair[0] == "maxl" {
				lvl.maxLevel, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			} else if pair[0] == "mean" {
				lvl.meanLevel, err = strconv.ParseFloat(pair[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
				}
			} else {
				return nil, fmt.Errorf("failed to parse level config \"%s\": %w", s, err)
			}
		}
	}

	lvl.Current_ = current
	if isNotZero(lvl.maxLevel) {
		pct := math.Min((lvl.Current_*100.0)/lvl.maxLevel, 100.0)
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

	Current_ float64  `json:"current"`
	Percent_ *float64 `json:"percent,omitempty"`
	Offset_  *float64 `json:"offset,omitempty"`
}

func (l *level) Handle(ctx context.Context, e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {

	if events.ObjectURNMatches(e, lwm2m.Distance) {
		return l.handleDistance(e, onchange)
	}

	if events.ObjectURNMatches(e, lwm2m.FillingLevel) {
		return l.handleFillingLevel(e, onchange)
	}

	return false, nil
}

func (l *level) handleFillingLevel(e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {

	const (
		ActualFillingPercentage string = "2"
		HighThreshold           string = "4"
	)

	percent, percentOk := events.GetFloat(e, ActualFillingPercentage)
	highThreshold, highThresholdOk := events.GetFloat(e, HighThreshold)

	if highThresholdOk {
		l.maxLevel = highThreshold
	}

	if percentOk {
		previousPercent := *l.Percent_

		if !hasChanged(previousPercent, percent) {
			return false, nil
		}

		l.Percent_ = &percent

		return true, onchange("percent", *l.Percent_, time.Now().UTC()) //TODO: use time from package
	}

	return false, nil
}

func (l *level) handleDistance(e *events.MessageAccepted, onchange func(prop string, value float64, ts time.Time) error) (bool, error) {

	const SensorValue string = "5700"
	r, ok := events.GetRecord(e, SensorValue)
	ts, timeOk := events.GetTime(e, SensorValue)

	if ok && timeOk && r.Value != nil {
		distance := *r.Value
		previousLevel := l.Current_

		// Calculate the current level using the configured angle (if any) and round to two decimals
		l.Current_ = math.Round((l.maxDistance-distance)*l.cosAlpha*100) / 100.0

		if !hasChanged(previousLevel, l.Current_) {
			return false, nil
		}

		if isNotZero(l.maxLevel) {
			pct := math.Min((l.Current_*100.0)/l.maxLevel, 100.0)
			l.Percent_ = &pct
		}

		if isNotZero(l.meanLevel) {
			offset := l.Current_ - l.meanLevel
			l.Offset_ = &offset
		}

		return true, onchange("level", l.Current_, ts)
	}

	return false, nil
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

func hasChanged(previousLevel, newLevel float64) bool {
	return isNotZero(newLevel - previousLevel)
}

func isNotZero(value float64) bool {
	return (math.Abs(value) >= 0.001)
}

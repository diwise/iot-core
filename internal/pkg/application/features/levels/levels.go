package levels

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
)

const (
	FeatureTypeName string = "level"
)

type Level interface {
	Handle(ctx context.Context, e *events.MessageAccepted) (bool, error)
	Current() float64
	Offset() float64
	Percent() float64
}

func New(config string) (Level, error) {

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

func (l *level) Handle(ctx context.Context, e *events.MessageAccepted) (bool, error) {

	if !e.BaseNameMatches(lwm2m.Distance) {
		return false, nil
	}

	const SensorValue string = "5700"
	distance, ok := e.GetFloat64(SensorValue)

	if ok {
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

		return true, nil
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

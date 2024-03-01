package events

import (
	"strings"

	"github.com/farshidtz/senml/v2"
)

type EventDecoratorFunc func(m *MessageAccepted)

func Rec(n, vs string, v *float64, vb *bool, t float64, sum *float64) EventDecoratorFunc {
	return func(m *MessageAccepted) {
		for _, r := range m.Pack_ {
			if strings.EqualFold(r.Name, n) {
				r.StringValue = vs
				r.Value = v
				r.BoolValue = vb
				r.Time = t
				r.Sum = sum
				return
			}
		}

		rec := senml.Record{
			Name:        n,
			StringValue: vs,
			Value:       v,
			BoolValue:   vb,
			Time:        t,
			Sum:         sum,
		}

		m.Pack_ = append(m.Pack_, rec)
	}
}

func Lat(t float64) EventDecoratorFunc {
	return func(m *MessageAccepted) {
		for _, r := range m.Pack_ {
			if r.Unit == senml.UnitLat {
				r.Value = &t
				return
			}
		}

		lat := &senml.Record{
			Unit:  senml.UnitLat,
			Value: &t,
		}

		m.Pack_ = append(m.Pack_, *lat)
	}
}

func Lon(t float64) EventDecoratorFunc {
	return func(m *MessageAccepted) {
		for _, r := range m.Pack_ {
			if r.Unit == senml.UnitLon {
				r.Value = &t
				return
			}
		}

		lat := &senml.Record{
			Unit:  senml.UnitLon,
			Value: &t,
		}

		m.Pack_ = append(m.Pack_, *lat)
	}
}

func Environment(e string) EventDecoratorFunc {
	if strings.EqualFold(e, "") {
		return func(m *MessageAccepted) {}
	}
	return Rec("env", e, nil, nil, 0, nil)
}

func Source(s string) EventDecoratorFunc {
	if strings.EqualFold(s, "") {
		return func(m *MessageAccepted) {}
	}
	return Rec("source", s, nil, nil, 0, nil)
}

func Tenant(t string) EventDecoratorFunc {
	if strings.EqualFold(t, "") {
		t = "default"
	}
	return Rec("tenant", t, nil, nil, 0, nil)
}

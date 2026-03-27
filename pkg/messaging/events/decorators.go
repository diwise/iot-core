package events

import (
	"strings"

	"github.com/diwise/senml"
)

type EventDecoratorFunc func(m Message)

func Rec(n, vs string, v *float64, vb *bool, t float64, sum *float64) EventDecoratorFunc {
	return func(m Message) {
		rec := senml.Record{
			Name:        n,
			StringValue: vs,
			Value:       v,
			BoolValue:   vb,
			Time:        t,
			Sum:         sum,
		}

		for _, r := range m.Pack() {
			if strings.EqualFold(r.Name, n) {
				// use the record passed into the predicate (rr) instead of closing over loop var r
				m.Replace(rec, func(rr senml.Record) bool { return strings.EqualFold(rr.Name, n) })
				return
			}
		}

		m.Append(rec)
	}
}

func Lat(t float64) EventDecoratorFunc {
	return func(m Message) {
		lat := senml.Record{
			Unit:  senml.UnitLat,
			Value: &t,
		}

		for _, r := range m.Pack() {
			if r.Unit == senml.UnitLat {
				// use the record passed into the predicate (rr) instead of closing over loop var r
				m.Replace(lat, func(rr senml.Record) bool { return rr.Unit == senml.UnitLat })
				return
			}
		}

		m.Append(lat)
	}
}

func Lon(t float64) EventDecoratorFunc {
	return func(m Message) {
		lon := senml.Record{
			Unit:  senml.UnitLon,
			Value: &t,
		}

		for _, r := range m.Pack() {
			if r.Unit == senml.UnitLon {
				// use the record passed into the predicate (rr) instead of closing over loop var r
				m.Replace(lon, func(rr senml.Record) bool { return rr.Unit == senml.UnitLon })
				return
			}
		}

		m.Append(lon)
	}
}

func Environment(e string) EventDecoratorFunc {
	if strings.EqualFold(e, "") {
		return func(m Message) {}
	}
	return Rec("env", e, nil, nil, 0, nil)
}

func Source(s string) EventDecoratorFunc {
	if strings.EqualFold(s, "") {
		return func(m Message) {}
	}
	return Rec("source", s, nil, nil, 0, nil)
}

func Tenant(t string) EventDecoratorFunc {
	if strings.EqualFold(t, "") {
		t = "default"
	}
	return Rec("tenant", t, nil, nil, 0, nil)
}

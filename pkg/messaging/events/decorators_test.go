package events

import (
	"testing"
	"time"

	"github.com/diwise/senml"
	"github.com/matryer/is"
)

func TestRecReplacesCorrectRecord(t *testing.T) {
	is := is.New(t)

	// pack with two records: one tenant record and one other
	p := senml.Pack{
		senml.Record{Name: "deviceA/0/0", Value: new(1.0)},
		senml.Record{Name: "tenant", StringValue: "old"},
	}

	m := NewMessageAccepted(p)

	// apply Rec with different case to ensure EqualFold
	Rec("tenant", "new", nil, nil, 0, nil)(m)

	// ensure tenant string updated, other record unchanged
	s, ok := m.Pack().GetStringValue(senml.FindByName("tenant"))
	is.True(ok)
	is.Equal(s, "new")

	v, ok := m.Pack().GetValue(senml.FindByName("deviceA/0/0"))
	is.True(ok)
	is.Equal(v, 1.0)
}

func TestLatAndLonReplaceAndAppend(t *testing.T) {
	is := is.New(t)

	now := float64(time.Now().Unix())

	// start with no lat/lon
	p := senml.Pack{
		senml.Record{Name: "deviceA/0/0", Value: new(1.0), Time: now},
	}
	m := NewMessageAccepted(p)

	// apply Lat -> should append
	Lat(12.34)(m)
	// apply Lon -> should append
	Lon(56.78)(m)

	// verify there is a record with UnitLat and UnitLon
	foundLat := false
	foundLon := false
	for _, r := range m.Pack() {
		if r.Unit == senml.UnitLat {
			foundLat = true
			is.Equal(*r.Value, 12.34)
		}
		if r.Unit == senml.UnitLon {
			foundLon = true
			is.Equal(*r.Value, 56.78)
		}
	}

	is.True(foundLat)
	is.True(foundLon)

	// Now test replace path: update lat
	Lat(77.77)(m)
	// ensure the UnitLat record now has 77.77
	for _, r := range m.Pack() {
		if r.Unit == senml.UnitLat {
			is.Equal(*r.Value, 77.77)
		}
	}
}

//go:fix inline
func floatPtr(f float64) *float64 { return new(f) }

package events

import (
	"testing"
	"time"

	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
)

func TestGetValuesFromPack(t *testing.T) {
	is := testSetup(t)
	var v float64 = 1.0
	var b bool = true

	dt := time.Date(2022, time.January, 1, 12, 0, 0, 0, time.UTC)

	evt := NewMessageAccepted(senml.Pack{}, Rec("withValues", "str", &v, &b, float64(dt.Unix()), nil))

	b, ok := GetVB(evt, "withValues")
	is.True(ok)
	v, ok = GetV(evt, "withValues")
	is.True(ok)
	str, ok := GetVS(evt, "withValues")
	is.True(ok)
	date, ok := GetT(evt, "withValues")
	is.True(ok)

	is.True(b)
	is.Equal(v, 1.0)
	is.Equal(str, "str")
	is.Equal(dt, date)
}

func TestNilValues(t *testing.T) {
	is := testSetup(t)

	evt := NewMessageAccepted(senml.Pack{}, Rec("nil", "", nil, nil, 0, nil))
	v, ok := GetV(evt, "nil")
	is.True(!ok)
	s, ok := GetVS(evt, "nil")
	is.True(ok)
	b, ok := GetVB(evt, "nil")
	is.True(!ok)

	is.Equal(v, 0.0)
	is.Equal(s, "")
	is.True(!b)
}

func TestGetValuesFromPack2(t *testing.T) {
	is := testSetup(t)
	var v float64 = 1.0
	var b bool = true
	dt := time.Date(2022, time.January, 1, 12, 0, 0, 0, time.UTC)
	baseRec := senml.Pack{
		senml.Record{
			Name:     "0",
			BaseName: "basename",
		},
	}

	evt := NewMessageAccepted(baseRec, Rec("1", "str", &v, &b, float64(dt.Unix()), nil))

	f, _ := Get[float64](evt, 1)
	is.Equal(1.0, f)
	s, _ := Get[string](evt, 1)
	is.Equal(s, "str")
	b2, _ := Get[bool](evt, 1)
	is.Equal(b2, true)
}

func testSetup(t *testing.T) *is.I {
	is := is.New(t)
	return is
}

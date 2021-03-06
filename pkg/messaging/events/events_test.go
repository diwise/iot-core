package events

import (
	"testing"

	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
)

func TestGetValuesFromPack(t *testing.T) {
	is := testSetup(t)

	evt := NewMessageAccepted("sensor", newPack("stringValue", 1.0, true))

	b, ok := evt.GetBool("SomeBool")
	is.True(ok)
	v, ok := evt.GetFloat64("SomeFloat")
	is.True(ok)
	str, ok := evt.GetString("SomeString")
	is.True(ok)

	is.True(b)
	is.Equal(v, 1.0)
	is.Equal(str, "stringValue")
}

func TestNilValues(t *testing.T) {
	is := testSetup(t)

	evt := NewMessageAccepted("sensor", newPack("stringValue", 1.0, true))
	v, ok := evt.GetFloat64("SomeNilValues")
	is.True(!ok)
	s, ok := evt.GetString("SomeNilValues")
	is.True(ok)
	b, ok := evt.GetBool("SomeNilValues")
	is.True(!ok)

	is.Equal(v, 0.0)
	is.Equal(s, "")
	is.True(!b)
}

func newPack(vs string, v float64, vb bool) senml.Pack {
	pack := senml.Pack{
		senml.Record{
			Name:      "SomeBool",
			BoolValue: &vb,
		},
		senml.Record{
			Name:        "SomeString",
			StringValue: vs,
		},
		senml.Record{
			Name:  "SomeFloat",
			Value: &v,
		},
		senml.Record{
			Name: "SomeNilValues",
		},
	}
	return pack
}

func testSetup(t *testing.T) *is.I {
	is := is.New(t)
	return is
}

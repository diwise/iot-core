package events

import (
	"testing"

	"github.com/matryer/is"
)

func TestGetValuesFromPack(t *testing.T) {
	is := testSetup(t)
	var v float64 = 1.0
	var b bool = true

	evt := NewMessageAccepted("sensor", Rec("withValues", "str", &v, &b))

	b, ok := evt.GetBool("withValues")
	is.True(ok)
	v, ok = evt.GetFloat64("withValues")
	is.True(ok)
	str, ok := evt.GetString("withValues")
	is.True(ok)

	is.True(b)
	is.Equal(v, 1.0)
	is.Equal(str, "str")
}

func TestNilValues(t *testing.T) {
	is := testSetup(t)

	evt := NewMessageAccepted("sensor", Rec("nil", "", nil, nil))
	v, ok := evt.GetFloat64("nil")
	is.True(!ok)
	s, ok := evt.GetString("nil")
	is.True(ok)
	b, ok := evt.GetBool("nil")
	is.True(!ok)

	is.Equal(v, 0.0)
	is.Equal(s, "")
	is.True(!b)
}

func testSetup(t *testing.T) *is.I {
	is := is.New(t)
	return is
}

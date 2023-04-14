package functions

import (
	"bytes"
	"context"
	"testing"

	"github.com/matryer/is"
)

func TestCreateRegistry(t *testing.T) {
	is := is.New(t)
	sensorId := "testId"

	config := "functionID;counter;overflow;" + sensorId
	reg, err := NewRegistry(context.Background(), bytes.NewBufferString(config))
	is.NoErr(err)

	matches, err := reg.Find(context.Background(), MatchSensor(sensorId))
	is.NoErr(err)

	is.Equal(len(matches), 1) // should find one matching feature
}

func TestFindNonMatchingFeatureReturnsEmptySlice(t *testing.T) {
	is := is.New(t)

	config := "functionID;counter;overflow;sensorId"
	reg, err := NewRegistry(context.Background(), bytes.NewBufferString(config))
	is.NoErr(err)

	matches, err := reg.Find(context.Background(), MatchSensor("noSuchSensor"))
	is.NoErr(err)

	is.Equal(len(matches), 0) // should not find any matching functions
}

package functions

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/matryer/is"
)

func TestCreateRegistry(t *testing.T) {
	is := is.New(t)
	sensorId := "testId"

	config := "functionID;name;counter;overflow;" + sensorId + ";false"
	reg, err := NewFuncRegistry(context.Background(), bytes.NewBufferString(config), &database.FuncStorageMock{
		AddFnctFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
		InitializeFunc: func(contextMoqParam context.Context) error {
			return nil
		},
	})
	is.NoErr(err)

	matches, err := reg.Find(context.Background(), MatchSensor(sensorId))
	is.NoErr(err)

	is.Equal(len(matches), 1) // should find one matching function
}

func TestFindNonMatchingFunctionReturnsEmptySlice(t *testing.T) {
	is := is.New(t)

	config := "functionID;name;counter;overflow;sensorId;false"
	reg, err := NewFuncRegistry(context.Background(), bytes.NewBufferString(config), &database.FuncStorageMock{
		AddFnctFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
		InitializeFunc: func(contextMoqParam context.Context) error {
			return nil
		},
	})
	is.NoErr(err)

	matches, err := reg.Find(context.Background(), MatchSensor("noSuchSensor"))
	is.NoErr(err)

	is.Equal(len(matches), 0) // should not find any matching functions
}

package levels

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestLevel(t *testing.T) {
	is := is.New(t)

	lvl, err := New("maxd=4", 0)
	is.NoErr(err)

	lvl.Handle(context.Background(), newDistance(1.27), func(string, float64, time.Time) error { return nil })

	is.Equal(lvl.Current(), 2.73)
}

func TestLevelWithOffset(t *testing.T) {
	is := is.New(t)

	lvl, err := New("maxd=4,offset=1,maxl=4", 0)
	is.NoErr(err)

	lvl.Handle(context.Background(), newDistance(1.27), func(string, float64, time.Time) error { return nil })

	is.Equal(lvl.Current(), 1.73)
	is.Equal(lvl.Percent(), 43.25)
}

func TestLevelWithOffsetFillingLevel(t *testing.T) {
	is := is.New(t)

	lvl, err := New("maxd=4,offset=1", 0)
	is.NoErr(err)
	
	// Offset = 1
	// Level = 1
	// HighThreshold = 4

	lvl.Handle(context.Background(), newFillingLevel(0, 1, 4, false, false), func(string, float64, time.Time) error { return nil })

	// Current = HighThreshold - (Level + Offset) 

	is.Equal(lvl.Current(), 2.0)
	is.Equal(lvl.Percent(), 50.0)
}

func TestLevelWithKnownMax(t *testing.T) {
	is := is.New(t)

	lvl, err := New("maxd=4,maxl=3", 0)
	is.NoErr(err)

	lvl.Handle(context.Background(), newDistance(1.27), func(string, float64, time.Time) error { return nil })

	is.Equal(lvl.Percent(), 91.0)
}

func TestLevelWithOverflowCapsPctTo100(t *testing.T) {
	is := is.New(t)

	lvl, err := New("maxd=4,maxl=3", 0)
	is.NoErr(err)

	lvl.Handle(context.Background(), newDistance(0.5), func(string, float64, time.Time) error { return nil })

	is.Equal(lvl.Percent(), 100.0)
}

func TestFillingLevel(t *testing.T) {
	is := is.New(t)

	lvl, err := New("maxd=4,maxl=3", 0)
	is.NoErr(err)

	lvl.Handle(context.Background(), newFillingLevel(53,0, 80, false, false), func(string, float64, time.Time) error { return nil })

	is.Equal(lvl.Percent(), 53.0)
}

func TestLevelWithMaxDAndMaxL(t *testing.T) {
	is := is.New(t)
	lvl, err := New("maxd=0.94,maxl=0.79", 0)
	is.NoErr(err)
	lvl.Handle(context.Background(), newDistance(0.4), func(s string, f float64, t time.Time) error {
		return nil
	})
	is.Equal(lvl.Current(), (0.94 - 0.4))
	is.Equal(lvl.Percent(), 68.35443037974683)
}

func newDistance(distance float64) *events.MessageAccepted {
	e := &events.MessageAccepted{}

	json.Unmarshal([]byte(
		fmt.Sprintf(distanceJSONFormat, distance),
	), e)
	return e
}

func newFillingLevel(actualFillingPercentage, actualFillingLevel, highThreshold float64, containerFull, containerEmpty bool) *events.MessageAccepted {
	e := &events.MessageAccepted{}
	json.Unmarshal([]byte(
		fmt.Sprintf(fillingLevelJSONFormat, actualFillingPercentage, actualFillingLevel, highThreshold, containerFull, containerEmpty),
	), e)

	return e
}

const fillingLevelJSONFormat string = `{	
	"pack":[
		{"bn":"testid/3435/","bt":1675801037,"n":"0","vs":"urn:oma:lwm2m:ext:3435"},	
		{"n":"2","v":%f},
		{"n":"3","v":%f},
		{"n":"4","v":%f},
		{"n":"5","vb":%t},
		{"n":"7","vb":%t}
	],
	"timestamp":"2023-02-07T20:17:17.312028Z"
}`

const distanceJSONFormat string = `{	
	"pack":[
		{"bn":"testid/3330/","bt":1675801037,"n":"0","vs":"urn:oma:lwm2m:ext:3330"},		
		{"n":"5700","u":"m","v":%f}
	],
	"timestamp":"2023-02-07T20:17:17.312028Z"
}`

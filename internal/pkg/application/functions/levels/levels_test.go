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

const distanceJSONFormat string = `{	
	"pack":[
		{"bn":"testid/3330/","bt":1675801037,"n":"0","vs":"urn:oma:lwm2m:ext:3330"},		
		{"n":"5700","u":"m","v":%f}
	],
	"timestamp":"2023-02-07T20:17:17.312028Z"
}`

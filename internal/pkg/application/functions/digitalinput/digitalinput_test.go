package digitalinput

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestDigitalInputWithStateChange(t *testing.T) {
	is := is.New(t)

	di := New(0)
	b, err := di.Handle(context.Background(), newState(false, "2023-02-07T21:32:59.682607Z"), func(string, float64, time.Time) error { return nil })
	is.NoErr(err)
	is.True(!b)

	b, err = di.Handle(context.Background(), newState(true, "2023-02-07T21:32:59.682607Z"), func(string, float64, time.Time) error { return nil })
	is.NoErr(err)
	is.True(b)

	is.True(di.State() == true)
}

func newState(on bool, timestamp string) *events.MessageAccepted {
	e := &events.MessageAccepted{}
	json.Unmarshal([]byte(
		fmt.Sprintf(messageJSONFormat, on, timestamp),
	), e)
	return e
}

const messageJSONFormat string = `{	
	"pack":[
		{"bn":"testid/3200/","bt":1675805579,"n":"0","vs":"urn:oma:lwm2m:ext:3200"},
		{"n":"5500","vb":%t}
	],
	"timestamp":"%s"
}`

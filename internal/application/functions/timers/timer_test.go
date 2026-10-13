package timers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/matryer/is"
)

func TestTimer(t *testing.T) {
	is := is.New(t)

	tmr := New()
	tmr.Handle(context.Background(), newState(true, "2023-02-07T21:32:59.682607Z"), func(string, float64, time.Time) error { return nil })
	tmr.Handle(context.Background(), newState(false, "2023-02-07T23:32:59.682607Z"), func(string, float64, time.Time) error { return nil })

	is.True(tmr.State() == false)
}

// REV-010: Stop is safe before activation and more than once, and no
// tick callbacks fire after Stop.
func TestTimerStopTerminatesTicks(t *testing.T) {
	is := is.New(t)

	tmr := NewWithInterval(20 * time.Millisecond)

	// Safe before activation and idempotent.
	tmr.Stop()
	tmr.Stop()

	var mu sync.Mutex
	tickCalls := 0
	onchange := func(prop string, value float64, ts time.Time) error {
		if prop == "time" {
			mu.Lock()
			tickCalls++
			mu.Unlock()
		}
		return nil
	}

	// NOTE: the shared newState helper below never matches (its "0"
	// record carries "testId" instead of the object URN), so activation
	// needs a fixture with the URN in place.
	activated, err := tmr.Handle(context.Background(), newDigitalInputState(true, "2023-02-07T21:32:59.682607Z"), onchange)
	is.NoErr(err)
	is.True(activated)

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := tickCalls
		mu.Unlock()
		if n > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	before := tickCalls
	mu.Unlock()
	is.True(before > 0)

	tmr.Stop()
	tmr.Stop()

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	after := tickCalls
	mu.Unlock()
	is.Equal(after, before)
}

func newDigitalInputState(on bool, timestamp string) *events.MessageAccepted {
	e := &events.MessageAccepted{}
	json.Unmarshal(
		fmt.Appendf(nil, digitalInputMessageJSONFormat, on, timestamp), e)
	return e
}

const digitalInputMessageJSONFormat string = `{
	"pack":[
		{"bn":"testId/3200/","bt":1675805579,"n":"0","vs":"urn:oma:lwm2m:ext:3200"},
		{"n":"5500","vb":%t}
	],
	"timestamp":"%s"
}`

func newState(on bool, timestamp string) *events.MessageAccepted {
	e := &events.MessageAccepted{}
	json.Unmarshal(
		fmt.Appendf(nil, messageJSONFormat, on, timestamp), e)
	return e
}

const messageJSONFormat string = `{
	"sensorID":"testId",
	"pack":[
		{"bn":"urn:oma:lwm2m:ext:3200","bt":1675805579,"n":"0","vs":"testId"},
		{"n":"5500","vb":%t}
	],
	"timestamp":"%s"
}`

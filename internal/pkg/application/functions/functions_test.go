package functions

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/matryer/is"
)

func TestCounter(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	config := "functionID;name;counter;overflow;" + sensorId + ";false"
	input := bytes.NewBufferString(config)

	reg, _ := NewRegistry(ctx, input, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
	})

	f, _ := reg.Find(ctx, MatchSensor(sensorId))

	ts, _ := time.Parse(time.RFC3339, "2024-03-20T12:19:48+01:00")

	pack := NewSenMLPack(sensorId, lwm2m.DigitalInput, ts, BoolValue("5500", true, 0))
	acceptedMessage := events.NewMessageAccepted(pack)

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload := msgctx.PublishOnTopicCalls()[0].Message.Body()

	const expectation string = `{"id":"functionID","name":"name","type":"counter","subtype":"overflow","deviceID":"testId","onupdate":false,"timestamp":"2024-03-20T11:19:48Z","counter":{"count":1,"state":true}}`
	is.Equal(string(generatedMessagePayload), expectation)
}

func TestLevel(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	input := bytes.NewBufferString("functionID;name;level;sand;" + sensorId + ";false;maxd=3.5,maxl=2.5")

	reg, _ := NewRegistry(ctx, input, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
		HistoryFunc: func(ctx context.Context, id, label string, lastN int) ([]database.LogValue, error) {
			return []database.LogValue{}, nil
		},
	})

	ts, _ := time.Parse(time.RFC3339, "2024-03-20T12:19:48+01:00")

	v := 2.1
	pack := NewSenMLPack(sensorId, lwm2m.Distance, ts, Rec("5700", &v, nil, "", nil, senml.UnitMeter, nil))
	acceptedMessage := events.NewMessageAccepted(pack)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))
	is.Equal(len(f), 1) // should find one matching function

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload := msgctx.PublishOnTopicCalls()[0].Message.Body()

	const expectation string = `{"id":"functionID","name":"name","type":"level","subtype":"sand","deviceID":"testId","onupdate":false,"timestamp":"2024-03-20T11:19:48Z","level":{"current":-2.1}}`
	is.Equal(string(generatedMessagePayload), expectation)
}

func TestLevelFromAnAngle(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"
	input := bytes.NewBufferString("functionID;name;level;sand;" + sensorId + ";false;maxd=3.5,maxl=2.5,angle=30")
	reg, _ := NewRegistry(ctx, input, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
		HistoryFunc: func(ctx context.Context, id, label string, lastN int) ([]database.LogValue, error) {
			return []database.LogValue{}, nil
		},
	})

	ts, _ := time.Parse(time.RFC3339, "2024-03-20T12:19:48+01:00")

	v := 2.1
	pack := NewSenMLPack(sensorId, lwm2m.Distance, ts, Rec("5700", &v, nil, "", nil, senml.UnitMeter, nil))
	acceptedMessage := events.NewMessageAccepted(pack)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))
	is.Equal(len(f), 1) // should find one matching function

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload := msgctx.PublishOnTopicCalls()[0].Message.Body()

	const expectation string = `{"id":"functionID","name":"name","type":"level","subtype":"sand","deviceID":"testId","onupdate":false,"timestamp":"2024-03-20T11:19:48Z","level":{"current":-2.1}}`
	is.Equal(string(generatedMessagePayload), expectation)
}

func TestTimer(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	config := "functionID;name;timer;overflow;" + sensorId + ";false"
	input := bytes.NewBufferString(config)

	reg, _ := NewRegistry(ctx, input, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
	})

	f, _ := reg.Find(ctx, MatchSensor(sensorId))

	packTime, _ := time.Parse(time.RFC3339, "2024-03-20T12:19:48+01:00")
	pack := NewSenMLPack(sensorId, lwm2m.DigitalInput, packTime, BoolValue("5500", true, 0))
	acceptedMessage := events.NewMessageAccepted(pack)

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload := msgctx.PublishOnTopicCalls()[0].Message.Body()

	const expectationFmt string = `{"id":"functionID","name":"name","type":"timer","subtype":"overflow","deviceID":"testId","onupdate":false,"timestamp":"2024-03-20T11:19:48Z","timer":{"startTime":"%s","state":true}}`
	is.Equal(string(generatedMessagePayload), fmt.Sprintf(expectationFmt, packTime.UTC().Format(time.RFC3339)))
}

func TestWaterQuality(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	input := bytes.NewBufferString("functionID;name;waterquality;beach;" + sensorId + ";false")

	reg, _ := NewRegistry(ctx, input, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		}})

	v := 2.34
	ts, _ := time.Parse(time.RFC3339, "2023-06-05T11:26:57Z")
	pack := NewSenMLPack(sensorId, lwm2m.Temperature, ts, Rec("5700", &v, nil, "", nil, senml.UnitCelsius, nil))
	acceptedMessage := events.NewMessageAccepted(pack)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))
	is.Equal(len(f), 1) // should find one matching function

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload := msgctx.PublishOnTopicCalls()[0].Message.Body()

	const expectation string = `{"id":"functionID","name":"name","type":"waterquality","subtype":"beach","deviceID":"testId","onupdate":false,"timestamp":"2023-06-05T11:26:57Z","waterquality":{"temperature":2.3,"timestamp":"2023-06-05T11:26:57Z"}}`
	is.Equal(string(generatedMessagePayload), expectation)
}

func TestAddToHistory(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"
	input := bytes.NewBufferString("functionID;name;waterquality;beach;" + sensorId + ";false")
	store := make([]database.LogValue, 0)
	reg, _ := NewRegistry(ctx, input, &database.StorageMock{
		AddFnFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			store = append(store, database.LogValue{Timestamp: timestamp, Value: value})
			return nil
		},
		HistoryFunc: func(ctx context.Context, id, label string, lastN int) ([]database.LogValue, error) {
			return store, nil
		},
	})

	newMessageAccepted := func(v float64, t time.Time) *events.MessageAccepted {
		pack := NewSenMLPack(sensorId, lwm2m.Temperature, t, Rec("5700", &v, nil, "", nil, senml.UnitCelsius, nil))
		return events.NewMessageAccepted(pack)
	}

	f, _ := reg.Find(ctx, MatchSensor(sensorId))

	_ = f[0].Handle(ctx, newMessageAccepted(1.67, time.Now().Add(-6*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(2.67, time.Now().Add(-5*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(3.67, time.Now().Add(-4*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(4.67, time.Now().Add(-3*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(5.67, time.Now().Add(-2*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(6.67, time.Now().Add(-1*time.Hour)), msgctx)

	h, _ := f[0].History(ctx, "", 0)
	is.Equal(6, len(h))
}

func testSetup(t *testing.T) (*is.I, context.Context, *messaging.MsgContextMock) {
	is := is.New(t)
	msgctx := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			return nil
		},
	}

	return is, context.Background(), msgctx
}

type SenMLDecoratorFunc func(p *senML)

type senML struct {
	Pack senml.Pack
}

func NewSenMLPack(deviceID, baseName string, baseTime time.Time, decorators ...SenMLDecoratorFunc) senml.Pack {
	s := &senML{}

	parts := strings.Split(baseName, ":")

	s.Pack = append(s.Pack, senml.Record{
		BaseName:    fmt.Sprintf("%s/%s/", deviceID, parts[len(parts)-1]),
		BaseTime:    float64(baseTime.Unix()),
		Name:        "0",
		StringValue: baseName,
	})

	for _, d := range decorators {
		d(s)
	}

	return s.Pack
}

func BoolValue(n string, vb bool, unixTime int64) SenMLDecoratorFunc {
	var t *time.Time

	if unixTime == 0 {
		t = nil
	} else {
		ut := time.Unix(unixTime, 0)
		t = &ut
	}

	return Rec(n, nil, nil, "", t, "", &vb)
}

func Rec(n string, v, sum *float64, vs string, t *time.Time, u string, vb *bool) SenMLDecoratorFunc {
	var tm float64
	if t != nil {
		tm = float64(t.Unix())
	}

	return func(p *senML) {
		r := senml.Record{
			Name:        n,
			Unit:        u,
			Time:        tm,
			Value:       v,
			StringValue: vs,
			BoolValue:   vb,
			Sum:         sum,
		}
		p.Pack = append(p.Pack, r)
	}
}

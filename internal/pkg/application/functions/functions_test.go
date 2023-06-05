package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
)

func TestCounter(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	config := "functionID;counter;overflow;" + sensorId
	input := bytes.NewBufferString(config)

	reg, _ := NewRegistry(ctx, input)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))

	pack := NewSenMLPack(sensorId, lwm2m.DigitalInput, time.Now().UTC(), BoolValue("5500", true, time.Now().UTC()))
	acceptedMessage := events.NewMessageAccepted("sensorID", pack)

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload, _ := json.Marshal(msgctx.PublishOnTopicCalls()[0].Message)

	const expectation string = `{"id":"functionID","type":"counter","subtype":"overflow","counter":{"count":1,"state":true}}`
	is.Equal(string(generatedMessagePayload), expectation)
}

func TestLevel(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	input := bytes.NewBufferString("functionID;level;sand;" + sensorId + ";maxd=3.5,maxl=2.5")

	reg, _ := NewRegistry(ctx, input)

	v := 2.1
	pack := NewSenMLPack(sensorId, lwm2m.Distance, time.Now().UTC(), Rec("5700", &v, nil, "", nil, senml.UnitMeter, nil))
	acceptedMessage := events.NewMessageAccepted(sensorId, pack)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))
	is.Equal(len(f), 1) // should find one matching function

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload, _ := json.Marshal(msgctx.PublishOnTopicCalls()[0].Message)

	const expectation string = `{"id":"functionID","type":"level","subtype":"sand","level":{"current":1.4,"percent":56}}`
	is.Equal(string(generatedMessagePayload), expectation)
}

func TestLevelFromAnAngle(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"
	input := bytes.NewBufferString("functionID;level;sand;" + sensorId + ";maxd=3.5,maxl=2.5,angle=30")
	reg, _ := NewRegistry(ctx, input)

	v := 2.1
	pack := NewSenMLPack(sensorId, lwm2m.Distance, time.Now().UTC(), Rec("5700", &v, nil, "", nil, senml.UnitMeter, nil))
	acceptedMessage := events.NewMessageAccepted(sensorId, pack)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))
	is.Equal(len(f), 1) // should find one matching function

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload, _ := json.Marshal(msgctx.PublishOnTopicCalls()[0].Message)

	const expectation string = `{"id":"functionID","type":"level","subtype":"sand","level":{"current":1.21,"percent":48.4}}`
	is.Equal(string(generatedMessagePayload), expectation)
}

func TestTimer(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	config := "functionID;timer;overflow;" + sensorId
	input := bytes.NewBufferString(config)

	reg, _ := NewRegistry(ctx, input)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))

	packTime := time.Now().UTC()
	pack := NewSenMLPack(sensorId, lwm2m.DigitalInput, packTime, BoolValue("5500", true, packTime))
	acceptedMessage := events.NewMessageAccepted("sensorID", pack)

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload, _ := json.Marshal(msgctx.PublishOnTopicCalls()[0].Message)

	const expectationFmt string = `{"id":"functionID","type":"timer","subtype":"overflow","timer":{"startTime":"%s","state":true}}`
	is.Equal(string(generatedMessagePayload), fmt.Sprintf(expectationFmt, packTime.Format(time.RFC3339)))
}

func TestWaterQuality(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	input := bytes.NewBufferString("functionID;waterquality;beach;" + sensorId)

	reg, _ := NewRegistry(ctx, input)

	v := 2.34
	ts, _ := time.Parse(time.RFC3339Nano, "2023-06-05T11:26:57Z")
	pack := NewSenMLPack(sensorId, lwm2m.Temperature, ts, Rec("5700", &v, nil, "", nil, senml.UnitCelsius, nil))
	acceptedMessage := events.NewMessageAccepted(sensorId, pack)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))
	is.Equal(len(f), 1) // should find one matching function

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload, _ := json.Marshal(msgctx.PublishOnTopicCalls()[0].Message)

	const expectation string = `{"id":"functionID","type":"waterquality","subtype":"beach","waterquality":{"temperature":2.3,"timestamp":"2023-06-05T11:26:57Z"}}`
	is.Equal(string(generatedMessagePayload), expectation)
}

func TestLastN(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"
	input := bytes.NewBufferString("functionID;waterquality;beach;" + sensorId)
	reg, _ := NewRegistry(ctx, input)

	newMessageAccepted := func(v float64, t time.Time) *events.MessageAccepted {
		pack := NewSenMLPack(sensorId, lwm2m.Temperature, t, Rec("5700", &v, nil, "", nil, senml.UnitCelsius, nil))
		return events.NewMessageAccepted(sensorId, pack)
	}

	f, _ := reg.Find(ctx, MatchSensor(sensorId))

	_ = f[0].Handle(ctx, newMessageAccepted(1.67, time.Now().Add(1*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(2.67, time.Now().Add(2*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(3.67, time.Now().Add(3*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(4.67, time.Now().Add(4*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(5.67, time.Now().Add(5*time.Hour)), msgctx)
	_ = f[0].Handle(ctx, newMessageAccepted(6.67, time.Now().Add(6*time.Hour)), msgctx)

	h, _ := f[0].History(ctx, 0)
	is.Equal(6, len(h))

	h, _ = f[0].History(ctx, 4)
	is.Equal(4, len(h))
	is.Equal(3.7, h[0].Value)

	h, _ = f[0].History(ctx, 7)
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

	s.Pack = append(s.Pack, senml.Record{
		BaseName:    baseName,
		BaseTime:    float64(baseTime.Unix()),
		Name:        "0",
		StringValue: deviceID,
	})

	for _, d := range decorators {
		d(s)
	}

	return s.Pack
}

func BoolValue(n string, vb bool, t time.Time) SenMLDecoratorFunc {
	return Rec(n, nil, nil, "", &t, "", &vb)
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

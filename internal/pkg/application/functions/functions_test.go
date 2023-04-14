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

	pack := NewSenMLPack(sensorId, lwm2m.DigitalInput, time.Now().UTC(), BoolValue("5500", true))
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
	is.Equal(len(f), 1) // should find one matching feature

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
	is.Equal(len(f), 1) // should find one matching feature

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

	pack := NewSenMLPack(sensorId, lwm2m.DigitalInput, time.Now().UTC(), BoolValue("5500", true))
	acceptedMessage := events.NewMessageAccepted("sensorID", pack)
	startTime := acceptedMessage.Timestamp

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload, _ := json.Marshal(msgctx.PublishOnTopicCalls()[0].Message)

	const expectationFmt string = `{"id":"functionID","type":"timer","subtype":"overflow","timer":{"startTime":"%s","state":true}}`
	is.Equal(string(generatedMessagePayload), fmt.Sprintf(expectationFmt, startTime))
}

func TestWaterQuality(t *testing.T) {
	is, ctx, msgctx := testSetup(t)

	sensorId := "testId"

	input := bytes.NewBufferString("functionID;waterquality;beach;" + sensorId)

	reg, _ := NewRegistry(ctx, input)

	v := 2.34
	pack := NewSenMLPack(sensorId, lwm2m.Temperature, time.Now().UTC(), Rec("5700", &v, nil, "", nil, senml.UnitCelsius, nil))
	acceptedMessage := events.NewMessageAccepted(sensorId, pack)

	f, _ := reg.Find(ctx, MatchSensor(sensorId))
	is.Equal(len(f), 1) // should find one matching feature

	err := f[0].Handle(ctx, acceptedMessage, msgctx)
	is.NoErr(err)

	is.Equal(len(msgctx.PublishOnTopicCalls()), 1)
	generatedMessagePayload, _ := json.Marshal(msgctx.PublishOnTopicCalls()[0].Message)

	const expectation string = `{"id":"functionID","type":"waterquality","subtype":"beach","waterquality":{"temperature":2.3}}`
	is.Equal(string(generatedMessagePayload), expectation)
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

func BoolValue(n string, vb bool) SenMLDecoratorFunc {
	return Rec(n, nil, nil, "", nil, "", &vb)
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

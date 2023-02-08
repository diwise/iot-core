package features

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
)

func TestCreateRegistry(t *testing.T) {
	is := is.New(t)

	config := "featureId;counter;overflow;sensorId"

	reg, err := NewRegistry(bytes.NewBufferString(config))
	is.NoErr(err)

	matches, err := reg.Find(context.Background(), "sensorId")
	is.NoErr(err)

	is.Equal(len(matches), 1) // should find one matching feature
}

func TestFindNonMatchingFeatureReturnsEmptySlice(t *testing.T) {
	is := is.New(t)

	config := "featureId;counter;overflow;sensorId"
	reg, err := NewRegistry(bytes.NewBufferString(config))
	is.NoErr(err)

	matches, err := reg.Find(context.Background(), "noSuchSensor")
	is.NoErr(err)

	is.Equal(len(matches), 0) // should not find any matching features
}

func TestCounter(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	sensorId := "testId"

	config := "featureId;counter;overflow;" + sensorId
	input := bytes.NewBufferString(config)

	reg, _ := NewRegistry(input)
	messenger := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			b, _ := json.MarshalIndent(message, " ", " ")
			fmt.Println(string(b))
			return nil
		},
	}

	f, _ := reg.Find(ctx, sensorId)

	const digitalInput string = "urn:oma:lwm2m:ext:3200"
	pack := NewSenMLPack(sensorId, digitalInput, time.Now().UTC(), BoolValue("5500", true))
	acceptedMessage := events.NewMessageAccepted("sensorID", pack)

	f[0].Handle(ctx, acceptedMessage, messenger)

	is.True(true)
}

func TestLevel(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	sensorId := "testId"

	input := bytes.NewBufferString("featureId;level;sand;" + sensorId + ";maxd=3.5,maxl=2.5")

	reg, _ := NewRegistry(input)
	messenger := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			b, _ := json.MarshalIndent(message, " ", " ")
			fmt.Println(string(b))
			return nil
		},
	}

	const distance string = "urn:oma:lwm2m:ext:3300"
	v := 2.1
	pack := NewSenMLPack(sensorId, distance, time.Now().UTC(), Rec("5700", &v, nil, "", nil, senml.UnitMeter, nil))
	acceptedMessage := events.NewMessageAccepted(sensorId, pack)

	f, _ := reg.Find(ctx, sensorId)
	is.Equal(len(f), 1) // should find one matching feature

	err := f[0].Handle(ctx, acceptedMessage, messenger)

	is.NoErr(err)
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

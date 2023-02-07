package features

import (
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

func TestFeatures(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	sensorId := "testId"

	reg, _ := NewRegistry()
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

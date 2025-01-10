package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/iot-core/internal/pkg/application/measurements"
	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
	"github.com/google/uuid"

	"github.com/diwise/iot-device-mgmt/pkg/client"
	clientMock "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"

	"github.com/matryer/is"
)

func TestMessageReceived(t *testing.T) {

	sensorId := uuid.NewString()
	ts := time.Now()
	v := 0.25

	ctx, is, app := testSetup(t, sensorId)

	pack := newPack(sensorId, lwm2m.Distance, ts, rec("5700", &v, nil, "", nil, senml.UnitMeter, nil))
	ma, err := app.MessageReceived(ctx, *events.NewMessageReceived(pack))

	is.NoErr(err)
	is.Equal(ma.DeviceID(), sensorId)
}

func TestMessageAccepted(t *testing.T) {
	sensorId := uuid.NewString()
	ts := time.Now()
	v := 0.25

	ctx, is, app := testSetup(t, sensorId)

	pack := newPack(sensorId, lwm2m.Distance, ts, rec("5700", &v, nil, "", nil, senml.UnitMeter, nil))
	ma, err := app.MessageReceived(ctx, *events.NewMessageReceived(pack))

	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma)

	is.NoErr(err)
}

func testSetup(t *testing.T, sensorID string) (context.Context, *is.I, App) {
	is := is.New(t)

	ctx := context.Background()

	c := &clientMock.DeviceManagementClientMock{
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (client.Device, error) {
			return &clientMock.DeviceMock{
				IDFunc: func() string {
					return deviceID
				},
				EnvironmentFunc: func() string {
					return ""
				},
				SourceFunc: func() string {
					return ""
				},
				TenantFunc: func() string {
					return "default"
				},
				LatitudeFunc: func() float64 {
					return 0
				},
				LongitudeFunc: func() float64 {
					return 0
				},
			}, nil
		},
	}
	m := &measurements.MeasurementsClientMock{}

	storage := make(map[string]any)
	storer := &functions.RegistryStorerMock{
		AddSettingFunc: func(ctx context.Context, id string, s functions.Setting) error {
			storage["setting__"+id] = s
			return nil
		},
		GetSettingsFunc: func(ctx context.Context) ([]functions.Setting, error) {
			settings := make([]functions.Setting, 0)
			for k, v := range storage {
				if !strings.HasPrefix(k, "setting__") {
					continue
				}
				settings = append(settings, v.(functions.Setting))
			}
			return settings, nil
		},
		LoadStateFunc: func(ctx context.Context, id string) ([]byte, error) {
			if v, ok := storage["state__"+id]; ok {
				return v.([]byte), nil
			}
			return nil, nil
		},
		SaveStateFunc: func(ctx context.Context, id string, a any) error {
			b, _ := json.Marshal(a)
			storage["state__"+id] = b
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			storage["value__"+id+"__"+label] = value
			return nil
		},
	}
	r, _ := functions.NewRegistry(ctx, strings.NewReader(fmt.Sprintf("\nlevel:49;soptunna:49;level;overflow;%s;maxd=0.65,maxl=0.5;false", sensorID)), storer)

	mc := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			return nil
		},
	}

	app := New(c, m, r, mc)

	return ctx, is, app
}

type decoratorFunc func(p *senML)

type senML struct {
	Pack senml.Pack
}

func newPack(deviceID, baseName string, baseTime time.Time, decorators ...decoratorFunc) senml.Pack {
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

func rec(n string, v, sum *float64, vs string, t *time.Time, u string, vb *bool) decoratorFunc {
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

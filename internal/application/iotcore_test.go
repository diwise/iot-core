package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/diwise/iot-core/internal/application/functions"
	"github.com/diwise/iot-core/internal/application/functions/engines"
	"github.com/diwise/iot-core/internal/application/measurements"
	"github.com/diwise/iot-core/internal/infrastructure/database"
	"github.com/diwise/iot-core/internal/infrastructure/database/rules"
	"github.com/diwise/iot-core/internal/infrastructure/repository"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/matryer/is"
)

func captureLoggerContext(buf *bytes.Buffer) context.Context {
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return logging.NewContextWithLogger(context.Background(), logger)
}

// CORE-006: sensorpayloads får aldrig loggas rutinmässigt. MessageReceived
// loggar identitet och typ, aldrig body. Provet använder en unik
// markörsträng som endast finns i bodyn. Identiteten här är det interna
// device-ID:t (sensor_id/devEUI finns inte i core); därför loggas
// fältet device_id.
func TestMessageReceivedDoesNotLogBody(t *testing.T) {
	is := is.New(t)

	const marker = "LOGPROBE-UNIQUE-7f3a"

	body := `{
		"pack":[
			{"bn":"device-log-probe-1/3200/","bt":1675805579,"n":"0","vs":"urn:oma:lwm2m:ext:3200"},
			{"n":"5500","vb":true},
			{"n":"5850","vs":"` + marker + `"}
		],
		"timestamp":"2023-02-07T21:32:59.682607Z"
	}`

	var msg events.MessageReceived
	is.NoErr(json.Unmarshal([]byte(body), &msg))
	is.True(msg.Error() == nil)

	dm := &dmctest.DeviceManagementClientMock{
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (client.Device, error) {
			return nil, errors.New("unknown device")
		},
	}

	var buf bytes.Buffer
	a := New(dm, nil, nil)

	_, err := a.MessageReceived(captureLoggerContext(&buf), msg)
	is.True(errors.Is(err, ErrCouldNotFindDevice))

	out := buf.String()
	is.True(!strings.Contains(out, marker))
	is.True(strings.Contains(out, "device-log-probe-1"))
	is.True(strings.Contains(out, "device_id"))
}

func TestWaterMeterMessageReceived(t *testing.T) {
	is, ctx, app, _, _, _, _ := testSetup(t)

	var evt events.MessageReceived
	err := json.Unmarshal([]byte(waterMeterMessage), &evt)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evt)

	is.Equal(7, len(ma.Pack()))

	is.NoErr(err)
}

func TestWaterMeterMessageAccepted(t *testing.T) {
	is, ctx, app, _, _, _, mctx := testSetup(t)
	var evt events.MessageReceived
	err := json.Unmarshal([]byte(waterMeterMessage), &evt)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evt)
	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma, mctx)
	is.NoErr(err)
}

func TestDistanceMessageReceived(t *testing.T) {
	is, ctx, app, _, _, _, _ := testSetup(t)
	var evt events.MessageReceived
	err := json.Unmarshal([]byte(distanceMessage), &evt)
	is.NoErr(err)

	_, err = app.MessageReceived(ctx, evt)
	is.NoErr(err)
}

func TestDistanceMessageAccepted(t *testing.T) {
	is, ctx, app, _, _, _, mctx := testSetup(t)
	var evt events.MessageReceived
	err := json.Unmarshal([]byte(distanceMessage), &evt)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evt)
	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma, mctx)
	is.NoErr(err)
}

func TestDigitalInputMessages(t *testing.T) {
	is, ctx, app, _, _, _, mctx := testSetup(t)
	var evtTrue events.MessageReceived
	err := json.Unmarshal([]byte(digitalInputTrueMessage), &evtTrue)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evtTrue)
	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma, mctx)
	is.NoErr(err)

	var evtFalse events.MessageReceived
	err = json.Unmarshal([]byte(digitalInputFalseMessage), &evtFalse)
	is.NoErr(err)

	ma, err = app.MessageReceived(ctx, evtFalse)
	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma, mctx)
	is.NoErr(err)

	err = app.FunctionUpdated(ctx, []byte(functionUpdatedWithTimer))
	is.NoErr(err)
}

func TestDigitalInputStopwatch(t *testing.T) {
	is, ctx, app, _, _, _, mctx := testSetup(t)

	trueMessage := strings.ReplaceAll(digitalInputTrueMessage, "internal-id-for-device-2", "internal-id-for-device-3")
	falseMessage := strings.ReplaceAll(digitalInputFalseMessage, "internal-id-for-device-2", "internal-id-for-device-3")

	var evtTrue events.MessageReceived
	err := json.Unmarshal([]byte(trueMessage), &evtTrue)
	is.NoErr(err)

	ma, err := app.MessageReceived(ctx, evtTrue)
	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma, mctx)
	is.NoErr(err)

	var evtFalse events.MessageReceived
	err = json.Unmarshal([]byte(falseMessage), &evtFalse)
	is.NoErr(err)

	ma, err = app.MessageReceived(ctx, evtFalse)
	is.NoErr(err)

	err = app.MessageAccepted(ctx, *ma, mctx)
	is.NoErr(err)

	err = app.FunctionUpdated(ctx, []byte(functionUpdatedWithStopwatch))
	is.NoErr(err)
}

func TestFunctionUpdatedWithDistance(t *testing.T) {
	is, ctx, app, _, _, _, _ := testSetup(t)
	err := app.FunctionUpdated(ctx, []byte(functionUpdatedWithLevel))
	is.NoErr(err)
}

func testSetup(t *testing.T) (*is.I, context.Context, App, client.DeviceManagementClient, *measurements.MeasurementsClientMock, functions.FuncRegistry, messaging.MsgContext) {
	is := is.New(t)
	d := &dmctest.DeviceManagementClientMock{
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (client.Device, error) {
			return &dmctest.DeviceMock{
				IDFunc:          func() string { return deviceID },
				LatitudeFunc:    func() float64 { return 60.0 },
				LongitudeFunc:   func() float64 { return 10.0 },
				EnvironmentFunc: func() string { return "" },
				SourceFunc:      func() string { return "" },
				TenantFunc:      func() string { return "default" },
			}, nil
		},
	}

	ctx := context.Background()
	m := &measurements.MeasurementsClientMock{
		GetCountTrueValuesFunc: func(ctx context.Context, measurmentID string, timeAt, endTimeAt time.Time) (float64, error) {
			return 0, nil
		},
	}
	mctx := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			fmt.Println(string(message.Body()))
			return nil
		},
		SendCommandToFunc: func(ctx context.Context, command messaging.Command, key string) error {
			fmt.Println(string(command.Body()))
			return nil
		},
	}
	s := &database.FuncStorageMock{
		HistoryFunc: func(ctx context.Context, id, label string, lastN int) ([]database.LogValue, error) {
			return []database.LogValue{}, nil
		},
		AddFnctFunc: func(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
			return nil
		},
		AddFunc: func(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
			return nil
		},
		CloseFunc: func() {},
	}
	r, _ := functions.NewFuncRegistry(ctx, io.NopCloser(strings.NewReader(functionsFileContent)), s)

	engine := &engines.RuleEngineMock{
		ValidationResultsFunc: func(ctx context.Context, msg events.MessageReceived) ([]engines.RuleValidation, error) {
			return nil, nil
		},
		ValidateRecordFunc: func(record senml.Record, rule rules.Rule) engines.RuleValidation {
			return engines.RuleValidation{}
		},
	}

	ruleRepository := &repository.RuleRepositoryMock{}

	a := NewWithRules(d, m, r, engine, ruleRepository, mctx)

	return is, ctx, a, d, m, r, mctx
}

const functionsFileContent string = `
functionID;name;type;subtype;sensorID;onupdate;args
lvl-1;level;level;distance;internal-id-for-device;true;maxd=5.0,maxl=10.0,mean=5.0,offset=0.0
tmr-1;timer;timer;;internal-id-for-device-2;true;
sw-1;stopwatch;stopwatch;;internal-id-for-device-3;true;`

const waterMeterMessage string = `
{
  "pack":[
    {"bn":"internal-id-for-device/3424/","bt":1563735600, "n":"0","vs":"urn:oma:lwm2m:ext:3424"},
    {"n":"1","u":"m3","v":10.727},
    {"n":"3","vs":"w1e"},
    {"n":"10","vb":true}
  ],
  "timestamp":"2025-10-02T08:42:07.505884256Z"
}`

const digitalInputTrueMessage string = `
{
  "pack":[
    {"bn": "internal-id-for-device-2/3200/","bt": 1759298400,"n": "0","vs": "urn:oma:lwm2m:ext:3200"},
    {"n":"5500","vb":true}   
  ],
  "timestamp": "2025-10-01T08:00:00.00Z"
}`

const digitalInputFalseMessage string = `
{
  "pack":[
    {"bn": "internal-id-for-device-2/3200/","bt": 1759302000,"n": "0","vs": "urn:oma:lwm2m:ext:3200"},
    {"n":"5500","vb":false}   
  ],
  "timestamp": "2025-10-01T09:00:00.00Z"
}`

const distanceMessage string = `
{
  "pack":[
    {"bn": "internal-id-for-device/3330/","bt": 1713865679,"n": "0","vs": "urn:oma:lwm2m:ext:3330"},
    {"n": "5700","u": "m","v": 1.80952},
    {"n": "5701","vs": "metre"}
  ],
  "timestamp": "2025-10-02T12:02:34.633266567Z"
}`

const functionUpdatedWithLevel string = `
{
  "id": "lvl-1",
  "name": "level",
  "type": "level",
  "subtype": "distance",
  "deviceID": "internal-id-for-device",
  "location": {
    "latitude": 60,
    "longitude": 10
  },
  "tenant": "default",
  "onupdate": true,
  "timestamp": "2024-04-23T09:47:59Z",
  "level": {
    "current": 3.19,
    "percent": 31.9,
    "offset": -1.81
  }
}`

const functionUpdatedWithTimer string = `{"id":"tmr-1","name":"timer","type":"timer","subtype":"","deviceID":"internal-id-for-device-2","location":{"latitude":60,"longitude":10},"tenant":"default","onupdate":true,"timestamp":"2025-10-01T07:00:00Z","timer":{"startTime":"2025-10-01T06:00:00Z","endTime":"2025-10-01T07:00:00Z","duration":3600000000000,"state":false,"totalDuration":3600000000000}}`
const functionUpdatedWithStopwatch string = `{"id":"sw-1","name":"stopwatch","type":"stopwatch","subtype":"","deviceID":"internal-id-for-device-3","location":{"latitude":60,"longitude":10},"tenant":"default","onupdate":true,"timestamp":"2025-10-01T07:00:00Z","stopwatch":{"startTime":"2025-10-01T06:00:00Z","stopTime":"2025-10-01T07:00:00Z","duration":3600000000000,"state":false,"count":0,"cumulativeTime":3600000000000}}`

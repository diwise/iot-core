package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/matryer/is"
)

func captureLoggerContext(buf *bytes.Buffer) context.Context {
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return logging.NewContextWithLogger(context.Background(), logger)
}

// CORE-006: sensorpayloads får aldrig loggas rutinmässigt. MessageReceived
// loggar identitet och typ, aldrig body. Provet använder en unik
// markörsträng som endast finns i bodyn.
func TestMessageReceivedDoesNotLogBody(t *testing.T) {
	is := is.New(t)

	const marker = "LOGPROBE-UNIQUE-7f3a"

	body := `{
		"pack":[
			{"bn":"sensor-log-probe-1/3200/","bt":1675805579,"n":"0","vs":"urn:oma:lwm2m:ext:3200"},
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
	is.True(strings.Contains(out, "sensor-log-probe-1"))
	is.True(strings.Contains(out, "sensor_id"))
	is.True(!strings.Contains(out, "device_id"))
}

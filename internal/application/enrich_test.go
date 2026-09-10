package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/senml"
	"github.com/matryer/is"
)

func enrichDeviceMock() *dmctest.DeviceManagementClientMock {
	return &dmctest.DeviceManagementClientMock{
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (client.Device, error) {
			return &dmctest.DeviceMock{
				IDFunc:          func() string { return deviceID },
				EnvironmentFunc: func() string { return "air" },
				SourceFunc:      func() string { return "trusted-source" },
				LongitudeFunc:   func() float64 { return 17 },
				LatitudeFunc:    func() float64 { return 62 },
				TenantFunc:      func() string { return "trusted-tenant" },
			}, nil
		},
	}
}

func receivedFromPack(t *testing.T, is *is.I, pack string) events.MessageReceived {
	t.Helper()
	body := `{"pack":` + pack + `,"timestamp":"2024-07-03T09:46:40Z"}`
	var msg events.MessageReceived
	is.NoErr(json.Unmarshal([]byte(body), &msg))
	return msg
}

// Direktinmatning med naken metadata (dokumenterat format) ska fortsätta
// fungera för enobjektspack; payloadens tenant ersätts med enhetens.
func TestHoistedLegacyMetadata(t *testing.T) {
	is := is.New(t)
	a := New(enrichDeviceMock(), nil, nil)

	msg := receivedFromPack(t, is, `[
		{"bn":"dev1/3303/","bt":1720000000,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},
		{"n":"5700","u":"Cel","v":21.5},
		{"u":"lat","v":61.0},
		{"u":"lon","v":16.0},
		{"n":"tenant","vs":"payload-tenant"},
		{"n":"source","vs":"payload-source"}
	]`)

	ma, err := a.MessageReceived(context.Background(), msg)
	is.NoErr(err)

	tenant, ok := ma.Pack().GetStringValue(shortName("tenant"))
	is.True(ok)
	is.Equal(tenant, "trusted-tenant")

	source, ok := ma.Pack().GetStringValue(shortName("source"))
	is.True(ok)
	is.Equal(source, "trusted-source")
}

// Flerobjektspack med naken metadata är tvetydigt och ska avvisas permanent.
func TestBareMetadataInMultiPackIsRejected(t *testing.T) {
	is := is.New(t)
	a := New(enrichDeviceMock(), nil, nil)

	msg := receivedFromPack(t, is, `[
		{"bn":"dev1/3303/","bt":1720000000,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},
		{"n":"5700","u":"Cel","v":21.5},
		{"bn":"dev1/3304/","bt":1720000000,"n":"0","vs":"urn:oma:lwm2m:ext:3304"},
		{"n":"5700","u":"%RH","v":55.0},
		{"n":"tenant","vs":"payload-tenant"}
	]`)

	_, err := a.MessageReceived(context.Background(), msg)
	is.True(err != nil)
}

// shortName matchar både nakna och kvalificerade namn (testhjälp som
// speglar findShortName-regeln).
func shortName(n string) senml.RecordFinder {
	return func(r senml.Record) bool {
		if r.Name == n {
			return true
		}
		for i := len(r.Name) - 1; i >= 0; i-- {
			if r.Name[i] == '/' {
				return r.Name[i+1:] == n
			}
		}
		return false
	}
}

// Suffix-toleranta accessorer ska fungera på resolvade pack.
func TestAccessorsOnResolvedPack(t *testing.T) {
	is := is.New(t)

	pack := senml.Pack{
		{Name: "dev1/3303/0", StringValue: "urn:oma:lwm2m:ext:3303", Time: 1720000000},
		{Name: "dev1/3303/5700", Unit: "Cel", Value: ptr(21.5), Time: 1720000000},
		{Name: "dev1/tenant", StringValue: "acme"},
	}

	is.Equal(events.GetDeviceID(pack), "dev1")
	is.Equal(events.GetObjectURN(pack), "urn:oma:lwm2m:ext:3303")
	is.Equal(events.GetObjectID(pack), "3303")

	m := events.NewMessageAccepted(pack)
	is.Equal(m.Tenant(), "acme")
	is.Equal(m.ContentType(), "application/vnd.oma.lwm2m.ext.3303+json")
}

func ptr(f float64) *float64 { return &f }

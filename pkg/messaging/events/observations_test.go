package events

// Låser observationskontraktet (observations-contract.md) mot fixturerna i
// testdata/. Producent och samtliga konsumenter ska verifieras mot samma
// fixturer innan kontraktsändringar aktiveras.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/diwise/senml"
	diwisepkg "github.com/diwise/senml/diwise"
)

var observationReference = time.Unix(1720007200, 0).UTC()

func loadPack(t *testing.T, name string) senml.Pack {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var p senml.Pack
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func mustParse(t *testing.T, p senml.Pack) *diwisepkg.Pack {
	t.Helper()
	parsed, err := diwisepkg.Parse(p, observationReference)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return parsed
}

func mustValue(t *testing.T, o diwisepkg.Object, resource string) float64 {
	t.Helper()
	r, err := o.Get(resource)
	if err != nil {
		t.Fatalf("Get(%s): %v", resource, err)
	}
	if r.Value == nil {
		t.Fatalf("Get(%s): no numeric value", resource)
	}
	return *r.Value
}

func TestLegacySingleTemp(t *testing.T) {
	parsed := mustParse(t, loadPack(t, "legacy-single-temp.json"))

	if got := parsed.DeviceID(); got != "dev1" {
		t.Fatalf("DeviceID = %q", got)
	}
	objs := parsed.Objects()
	if len(objs) != 1 {
		t.Fatalf("objects = %d", len(objs))
	}
	o := objs[0]
	if got := o.URN(); got != "urn:oma:lwm2m:ext:3303" {
		t.Fatalf("URN = %q", got)
	}
	if v := mustValue(t, o, "5700"); v != 21.5 {
		t.Fatalf("5700 = %v", v)
	}
}

func TestMultiTempHumidityLight(t *testing.T) {
	parsed := mustParse(t, loadPack(t, "multi-temp-humidity-light.json"))

	if got := parsed.DeviceID(); got != "dev1" {
		t.Fatalf("DeviceID = %q", got)
	}
	want := map[string]float64{
		"urn:oma:lwm2m:ext:3303": 21.5,
		"urn:oma:lwm2m:ext:3304": 55.0,
		"urn:oma:lwm2m:ext:3301": 320.0,
	}
	objs := parsed.Objects()
	if len(objs) != len(want) {
		t.Fatalf("objects = %d", len(objs))
	}
	for _, o := range objs {
		v, ok := want[o.URN()]
		if !ok {
			t.Fatalf("unexpected URN %q", o.URN())
		}
		if got := mustValue(t, o, "5700"); got != v {
			t.Fatalf("%s/5700 = %v, want %v", o.URN(), got, v)
		}
		delete(want, o.URN())
	}
	if len(want) != 0 {
		t.Fatalf("missing URNs: %v", want)
	}
}

func TestMultiChannelTemp(t *testing.T) {
	parsed := mustParse(t, loadPack(t, "multi-channel-temp.json"))

	objs := parsed.Objects()
	if len(objs) != 2 {
		t.Fatalf("objects = %d", len(objs))
	}
	got := map[string]float64{}
	for _, o := range objs {
		if o.URN() != "urn:oma:lwm2m:ext:3303" {
			t.Fatalf("URN = %q", o.URN())
		}
		got[o.Prefix()] = mustValue(t, o, "5700")
	}
	if got["dev1/0/3303/"] != 21.5 || got["dev1/1/3303/"] != 19.0 {
		t.Fatalf("channel values = %v", got)
	}
}

func TestRepeatedObservation(t *testing.T) {
	parsed := mustParse(t, loadPack(t, "repeated-observation.json"))

	objs := parsed.Objects()
	if len(objs) != 2 {
		t.Fatalf("observations = %d, want both preserved", len(objs))
	}
	if _, err := objs[0].Get("5700"); err != nil {
		t.Fatalf("Get(5700) on single observation: %v", err)
	}
}

func TestRepeatedResourceWithinObservationIsAmbiguous(t *testing.T) {
	p := senml.Pack{
		{BaseName: "dev1/3303/", BaseTime: 1720000000, Name: "0", StringValue: "urn:oma:lwm2m:ext:3303"},
		{Name: "5700", Unit: "Cel", Value: ptr(21.5)},
		{Name: "5700", Unit: "Cel", Value: ptr(21.7)},
	}
	// Kompakt form utan upprepad header: båda resurserna tillhör samma
	// observation och Get ska vägra gissa.
	parsed := mustParse(t, p)
	o := parsed.Objects()[0]
	if _, err := o.Get("5700"); !errors.Is(err, senml.ErrAmbiguous) {
		t.Fatalf("Get(5700) = %v, want ErrAmbiguous", err)
	}
	if len(o.Find("5700")) != 2 {
		t.Fatalf("Find(5700) should return both records")
	}
}

func TestMultiObjectDifferentTimes(t *testing.T) {
	parsed := mustParse(t, loadPack(t, "multi-object-different-times.json"))

	times := map[string]float64{}
	for _, o := range parsed.Objects() {
		r, err := o.Get("5700")
		if err != nil {
			t.Fatal(err)
		}
		ts, ok := r.GetTime()
		if !ok {
			t.Fatalf("%s: no time", o.URN())
		}
		times[o.URN()] = float64(ts.Unix())
	}
	if times["urn:oma:lwm2m:ext:3303"] != 1720000000 || times["urn:oma:lwm2m:ext:3304"] != 1720003600 {
		t.Fatalf("times = %v", times)
	}
}

func TestRejects(t *testing.T) {
	for _, name := range []string{
		"conflicting-tenant.json",
		"multi-device.json",
		"missing-header.json",
		"empty-pack.json",
	} {
		if _, err := diwisepkg.Parse(loadPack(t, name), observationReference); err == nil {
			t.Fatalf("%s: expected rejection", name)
		}
	}
}

func TestContentTypeForPacks(t *testing.T) {
	single := loadPack(t, "legacy-single-temp.json")
	if got := ContentTypeFor(single); got != "application/vnd.oma.lwm2m.ext.3303+json" {
		t.Fatalf("single = %q", got)
	}
	if got := NewMessageReceived(single).ContentType(); got != "application/vnd.oma.lwm2m.ext.3303+json" {
		t.Fatalf("received single = %q", got)
	}
	multi := loadPack(t, "multi-temp-humidity-light.json")
	if got := ContentTypeFor(multi); got != GenericLwM2MContentType {
		t.Fatalf("multi = %q", got)
	}
	if got := NewMessageAccepted(multi).ContentType(); got != GenericLwM2MContentType {
		t.Fatalf("accepted multi = %q", got)
	}
	// Oparsebart pack faller tillbaka på legacy-härledning.
	broken := senml.Pack{{Name: "5700", Value: ptr(1)}}
	if got := ContentTypeFor(broken); got != "application/vnd.oma.lwm2m.ext.+json" {
		t.Fatalf("broken = %q", got)
	}
}

func TestLegacyAccessorsDocumentFirstHeaderBehavior(t *testing.T) {
	// Dokumenterar befintligt beteende för flerobjektspack: singulära
	// accessorer följer första headern. Kontraktsbrott att förlita sig på
	// dessa för flerobjektspack – använd diwise-API:t.
	p := loadPack(t, "multi-temp-humidity-light.json")
	if got := GetDeviceID(p); got != "dev1" {
		t.Fatalf("GetDeviceID = %q", got)
	}
	if got := GetObjectURN(p); got != "urn:oma:lwm2m:ext:3303" {
		t.Fatalf("GetObjectURN = %q (first header wins)", got)
	}
}

func ptr(f float64) *float64 { return &f }

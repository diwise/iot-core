package application

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/diwise/iot-core/internal/application/decorators"
	"github.com/diwise/iot-core/internal/application/functions"
	"github.com/diwise/iot-core/internal/application/measurements"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	diwisepkg "github.com/diwise/senml/diwise"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type App interface {
	MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error
	MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
}

type app struct {
	client             client.DeviceManagementClient
	measurementsClient measurements.MeasurementsClient
	fnctRegistry       functions.Registry
	mu                 sync.Mutex
}

func New(client client.DeviceManagementClient, measurementsClient measurements.MeasurementsClient, functionRegistry functions.Registry) App {
	return &app{
		client:             client,
		fnctRegistry:       functionRegistry,
		measurementsClient: measurementsClient,
	}
}

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error {
	if evt.Error() != nil {
		return evt.Error()
	}

	logger := logging.GetFromContext(ctx)

	a.mu.Lock()
	defer a.mu.Unlock()

	matchingFunctions, _ := a.fnctRegistry.Find(ctx, functions.MatchSensor(evt.DeviceID()))
	matchingCount := len(matchingFunctions)

	if matchingCount == 0 {
		logger.Debug("no matching functions found")
		return nil
	}

	logger.Debug("found matching functions", "count", matchingCount)

	for _, f := range matchingFunctions {
		if err := f.Handle(ctx, &evt, msgctx); err != nil {
			return err
		}
	}

	return nil
}

var ErrCouldNotFindDevice = fmt.Errorf("could not find device")

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	if msg.Error() != nil {
		return nil, msg.Error()
	}

	log := logging.GetFromContext(ctx)

	// Referenstid för relativa SenML-tider: mottagningstid. Varje
	// observations mättidpunkt bevaras av Resolve.
	ref := time.Now().UTC()

	parsed, err := diwisepkg.Parse(msg.Pack(), ref)
	if err != nil {
		return nil, err
	}

	log.Debug("received message", "device_id", parsed.DeviceID(), "content_type", msg.ContentType(), "observations", len(parsed.Objects()))

	device, err := a.client.FindDeviceFromInternalID(ctx, parsed.DeviceID())
	if err != nil {
		log.Debug("could not find device", "device_id", parsed.DeviceID(), "err", err.Error())
		return nil, ErrCouldNotFindDevice
	}

	enriched := enrichPack(ctx, parsed, ref, device, a.measurementsClient)

	final, err := diwisepkg.Parse(enriched, ref)
	if err != nil {
		return nil, fmt.Errorf("enriched pack failed validation: %w", err)
	}
	if err := final.ValidateAccepted(); err != nil {
		return nil, fmt.Errorf("enriched pack missing tenant metadata: %w", err)
	}

	ma := events.NewMessageAccepted(final.Pack())

	log.Debug("message.accepted created", "device_id", ma.DeviceID(), "content_type", ma.ContentType(), "observations", len(final.Objects()))

	return ma, nil
}

// enrichPack berikar ett tolkat pack med kvalificerad metadata från betrodd
// enhetsdata samt härledda observationsposter. Allt som läggs till använder
// fullständiga namn så att resultatet förblir parsebart och varje
// observation behåller sin identitet, sina värden och sin mättidpunkt.
func enrichPack(ctx context.Context, parsed *diwisepkg.Pack, ref time.Time, device client.Device, measurementsClient measurements.MeasurementsClient) senml.Pack {
	records := parsed.Pack()
	deviceID := parsed.DeviceID()

	tenant := device.Tenant()
	if tenant == "" {
		tenant = "default"
	}
	records = append(records, senml.Record{Name: deviceID + "/tenant", StringValue: tenant})
	if s := device.Source(); s != "" {
		records = append(records, senml.Record{Name: deviceID + "/source", StringValue: s})
	}
	if e := device.Environment(); e != "" {
		records = append(records, senml.Record{Name: deviceID + "/env", StringValue: e})
	}
	lat, lon := device.Latitude(), device.Longitude()
	records = append(records,
		senml.Record{Name: deviceID + "/lat", Unit: senml.UnitLat, Value: &lat},
		senml.Record{Name: deviceID + "/lon", Unit: senml.UnitLon, Value: &lon},
	)

	if hasDigitalInput(parsed) {
		records = appendDigitalInputCounters(records, parsed, ref,
			decorators.GetNumberOfTrueValues(ctx, measurementsClient, device.ID()))
	}

	return records
}

func hasDigitalInput(parsed *diwisepkg.Pack) bool {
	for _, o := range parsed.Objects() {
		if o.URN() == "urn:oma:lwm2m:ext:"+decorators.DigitalInputObjectID {
			return true
		}
	}
	return false
}

// appendDigitalInputCounters härleder en 5501-räknare per 3200-observation
// som saknar en, enligt samma regel som tidigare gällde globalt: historiskt
// antal sanna tillstånd, minst 0, plus ett om observationens eget 5500 är sant.
func appendDigitalInputCounters(records senml.Pack, parsed *diwisepkg.Pack, ref time.Time, count decorators.ValueFinder) senml.Pack {
	for _, o := range parsed.Objects() {
		if o.URN() != "urn:oma:lwm2m:ext:"+decorators.DigitalInputObjectID {
			continue
		}
		if len(o.Find(decorators.DigitalInputCounter)) > 0 {
			continue
		}
		counter := math.Ceil(count())
		if counter < 0 {
			counter = 0
		}
		for _, r := range o.Find(decorators.DigitalInputState) {
			if r.BoolValue != nil && *r.BoolValue {
				counter++
				break
			}
		}
		obsTime := float64(ref.Unix())
		if header := o.Header(); header.Name != "" {
			if ts, ok := header.GetTime(); ok {
				obsTime = float64(ts.UnixNano()) / 1e9
			}
		}
		records = append(records, senml.Record{
			Name:  o.Prefix() + decorators.DigitalInputCounter,
			Value: &counter,
			Time:  obsTime,
		})
	}
	return records
}

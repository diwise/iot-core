package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/iot-core/internal/application/decorators"
	"github.com/diwise/iot-core/internal/application/functions"
	"github.com/diwise/iot-core/internal/application/functions/engines"
	"github.com/diwise/iot-core/internal/application/measurements"
	"github.com/diwise/iot-core/internal/infrastructure/database/rules"
	"github.com/diwise/iot-core/internal/infrastructure/repository"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	diwisepkg "github.com/diwise/senml/diwise"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/google/uuid"
)

type App interface {
	MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error
	MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
	FunctionUpdated(ctx context.Context, body []byte) error
	CreateRule(ctx context.Context, rule *rules.Rule) error
	GetRulesByDevice(ctx context.Context, deviceID string) ([]*rules.Rule, error)
	GetRule(ctx context.Context, ruleID string) (*rules.Rule, error)
	UpdateRule(ctx context.Context, rule *rules.Rule) error
	DeleteRule(ctx context.Context, ruleID string) error
}

type app struct {
	client             client.DeviceManagementClient
	measurementsClient measurements.MeasurementsClient
	fnctRegistry       functions.Registry
	ruleEngine         engines.RuleEngine
	ruleRepository     repository.RuleRepository
	messenger          messaging.MsgContext
	mu                 sync.Mutex
}

func New(client client.DeviceManagementClient, measurementsClient measurements.MeasurementsClient, functionRegistry functions.Registry) App {
	return &app{
		client:             client,
		fnctRegistry:       functionRegistry,
		measurementsClient: measurementsClient,
	}
}

// NewWithRules bevarar develop:s konstruktor (rules-engine) vid sidan av
// labs-1.27:s New. labs-1.27:s treargumentare är normerande för kärnflödet;
// denna variant kopplar dessutom regelverk och messenger utan att ändra
// validerings-/berikningssemantiken när de är nil.
func NewWithRules(client client.DeviceManagementClient, measurementsClient measurements.MeasurementsClient, functionRegistry functions.Registry, ruleEngine engines.RuleEngine, ruleRepository repository.RuleRepository, msgCtx messaging.MsgContext) App {
	return &app{
		client:             client,
		fnctRegistry:       functionRegistry,
		measurementsClient: measurementsClient,
		ruleEngine:         ruleEngine,
		ruleRepository:     ruleRepository,
		messenger:          msgCtx,
	}
}

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error {
	if evt.Error() != nil {
		return evt.Error()
	}

	logger := logging.GetFromContext(ctx)

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.fnctRegistry == nil {
		logger.Debug("no function registry, skipping")
		return nil
	}

	matchingFunctions, _ := a.fnctRegistry.Find(ctx, functions.MatchSensor(evt.DeviceID()))
	matchingCount := len(matchingFunctions)

	if matchingCount == 0 {
		logger.Debug("no matching functions found")
		return nil
	}

	logger.Debug("found matching functions", "count", matchingCount)

	if msgctx == nil {
		msgctx = a.messenger
	}

	errs := []error{}
	for _, f := range matchingFunctions {
		if err := f.Handle(ctx, &evt, msgctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
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

	parsed, err := diwisepkg.Parse(hoistLegacyMetadata(msg.Pack(), log), ref)
	if err != nil {
		return nil, err
	}

	log.Debug("received message", "device_id", parsed.DeviceID(), "content_type", msg.ContentType(), "observations", len(parsed.Objects()))

	device, err := a.client.FindDeviceFromInternalID(ctx, parsed.DeviceID())
	if err != nil {
		log.Debug("could not find device", "device_id", parsed.DeviceID(), "err", err.Error())
		return nil, ErrCouldNotFindDevice
	}

	// Rules-validering från develop (opt-in): körs endast när en
	// regel-motor är kopplad. Icke-aborterande underkännanden publiceras
	// men blockerar inte berikningsflödet; aborterande returnerar fel.
	// Utan motor (labs-1.27:s standard, samt enhetstester med nil) är
	// beteendet oförändrat.
	if a.ruleEngine != nil {
		if err := a.validateRules(ctx, msg, device, log); err != nil {
			return nil, err
		}
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

func (a *app) validateRules(ctx context.Context, msg events.MessageReceived, device client.Device, log *slog.Logger) error {
	validated, err := a.ruleEngine.ValidationResults(ctx, msg)
	if err != nil {
		return err
	}

	clone := msg.Pack().Clone()

	for _, validation := range validated {
		if validation.IsValid {
			continue
		}
		log.Debug("message did not validate by it's rule", "device_id", device.ID())

		if validation.ShouldAbort {
			if a.messenger != nil {
				abortMessage := events.NewMessageAborted(clone, validation.ValidationMessages)
				if err := a.messenger.PublishOnTopic(ctx, abortMessage); err != nil {
					return err
				}
			}
			return fmt.Errorf("message did not validate and is set to abort")
		}

		if a.messenger != nil {
			notValidatedMessage := events.NewMessageNotValidated(clone, validation.ValidationMessages)
			if err := a.messenger.PublishOnTopic(ctx, notValidatedMessage); err != nil {
				return err
			}
		}
	}

	return nil
}

type functionUpdated struct {
	DeviceID  string    `json:"deviceID"`
	Type      string    `json:"type"`
	Tenant    string    `json:"tenant"`
	Timestamp time.Time `json:"timestamp"`

	Level     *level     `json:"level"`
	Timer     *timer     `json:"timer"`
	Stopwatch *stopwatch `json:"stopwatch"`
}

type level struct {
	Current float64 `json:"current"`
	Percent float64 `json:"percent"`
}

type timer struct {
	StartTime     time.Time      `json:"startTime"`
	State         bool           `json:"state"`
	Duration      *time.Duration `json:"duration"`
	TotalDuration time.Duration  `json:"totalDuration"`
}

type stopwatch struct {
	StartTime      time.Time      `json:"startTime"`
	StopTime       time.Time      `json:"stopTime"`
	Duration       *time.Duration `json:"duration"`
	State          bool           `json:"state"`
	Count          int            `json:"count"`
	CumulativeTime time.Duration  `json:"cumulativeTime"`
}

func (a *app) FunctionUpdated(ctx context.Context, body []byte) error {
	log := logging.GetFromContext(ctx)

	fn := functionUpdated{}
	err := json.Unmarshal(body, &fn)
	if err != nil {
		return err
	}

	var evt *events.MessageReceived

	switch strings.ToLower(fn.Type) {
	case "level":
		if fn.Level == nil {
			return nil
		}

		fl := lwm2m.NewFillingLevel(fn.DeviceID, fn.Level.Percent, fn.Timestamp)
		afl := int64(fn.Level.Current * 100) // lwm2m distance is meters, fillingLevel is cm
		fl.ActualFillingLevel = &afl

		log.Debug("filling level function updated", slog.String("deviceID", fn.DeviceID), slog.Float64("percent", fn.Level.Percent), slog.Float64("current", fn.Level.Current))

		evt = events.NewMessageReceived(lwm2m.ToPack(fl))
	case "timer":
		if fn.Timer == nil {
			return nil
		}

		tmr := lwm2m.NewTimer(fn.DeviceID, fn.Timer.Duration.Seconds(), fn.Timestamp)
		tmr.OnOff = fn.Timer.State
		cumulative := float64(fn.Timer.TotalDuration.Seconds())
		tmr.CumulativeTime = &cumulative

		evt = events.NewMessageReceived(lwm2m.ToPack(tmr))
	case "stopwatch":
		if fn.Stopwatch == nil {
			return nil
		}

		sw := lwm2m.NewStopwatch(fn.DeviceID, float64(fn.Stopwatch.CumulativeTime.Seconds()), fn.Timestamp)
		sw.OnOff = &fn.Stopwatch.State
		sw.DigitalInputCounter = int32(fn.Stopwatch.Count)

		evt = events.NewMessageReceived(lwm2m.ToPack(sw))
	default:
		return nil
	}

	if evt != nil {
		if a.messenger == nil {
			return fmt.Errorf("no messenger configured for function update")
		}
		err := a.messenger.SendCommandTo(ctx, evt, "iot-core")
		if err != nil {
			log.Error("could not send message.received from function update", slog.String("deviceID", fn.DeviceID), slog.String("function_type", fn.Type), slog.String("err", err.Error()))
			return err
		}
		log.Debug("sent message.received from function update", slog.String("deviceID", fn.DeviceID), slog.String("function_type", fn.Type))
	}

	return nil
}

func (a *app) CreateRule(ctx context.Context, rule *rules.Rule) error {
	log := logging.GetFromContext(ctx)

	if rule != nil && strings.TrimSpace(rule.ID) == "" {
		rule.ID = uuid.NewString()
	}

	if a.ruleRepository == nil {
		return fmt.Errorf("no rule repository configured")
	}

	err := a.ruleRepository.Add(ctx, *rule)
	if err != nil {
		log.Error("could not create rule", "rule_id", rule.ID, "err", err.Error())
		return err
	}

	log.Debug("rule created successfully", "rule_id", rule.ID)
	return nil
}

func (a *app) GetRulesByDevice(ctx context.Context, deviceID string) ([]*rules.Rule, error) {
	log := logging.GetFromContext(ctx)

	if a.ruleRepository == nil {
		return nil, fmt.Errorf("no rule repository configured")
	}

	deviceRules, err := a.ruleRepository.Get(ctx, deviceID)
	if err != nil {
		log.Error("could not get rules for device", "device_id", deviceID, "err", err.Error())
		return nil, err
	}

	log.Debug("retrieved rules for device", "device_id", deviceID, "count", len(deviceRules))

	// Convert []rules.Rule to []*rules.Rule
	result := make([]*rules.Rule, len(deviceRules))
	for i := range deviceRules {
		result[i] = &deviceRules[i]
	}

	return result, nil
}

func (a *app) GetRule(ctx context.Context, ruleID string) (*rules.Rule, error) {
	log := logging.GetFromContext(ctx)

	if a.ruleRepository == nil {
		return nil, fmt.Errorf("no rule repository configured")
	}

	rule, err := a.ruleRepository.GetRuleById(ctx, ruleID)
	if err != nil {
		log.Error("could not get rule", "rule_id", ruleID, "err", err.Error())
		return nil, err
	}

	log.Debug("retrieved rule", "rule_id", ruleID)
	return &rule, nil
}

func (a *app) UpdateRule(ctx context.Context, rule *rules.Rule) error {
	log := logging.GetFromContext(ctx)

	if a.ruleRepository == nil {
		return fmt.Errorf("no rule repository configured")
	}

	err := a.ruleRepository.Update(ctx, *rule)
	if err != nil {
		log.Error("could not update rule", "rule_id", rule.ID, "err", err.Error())
		return err
	}

	log.Debug("rule updated successfully", "rule_id", rule.ID)
	return nil
}

func (a *app) DeleteRule(ctx context.Context, ruleID string) error {
	log := logging.GetFromContext(ctx)

	if a.ruleRepository == nil {
		return fmt.Errorf("no rule repository configured")
	}

	err := a.ruleRepository.Delete(ctx, ruleID)
	if err != nil {
		log.Error("could not delete rule", "rule_id", ruleID, "err", err.Error())
		return err
	}

	log.Debug("rule deleted successfully", "rule_id", ruleID)
	return nil
}

// hoistLegacyMetadata lyfter nakna metadataposter (tenant/source/env utan
// deviceprefix samt namnlösa lat/lon-poster) till kvalificerad packnivå
// med packens enhet som prefix. Detta bevarar det dokumenterade
// direktinmatningsformatet för enobjektspack; flerobjektspack med naken
// metadata är tvetydiga och lämnas orörda så att Parse avvisar dem.
func hoistLegacyMetadata(pack senml.Pack, log *slog.Logger) senml.Pack {
	fullName := func(r senml.Record) string {
		if r.BaseName != "" && strings.HasPrefix(r.Name, r.BaseName) {
			return r.Name
		}
		return r.BaseName + r.Name
	}

	isHeader := func(r senml.Record) bool {
		name := fullName(r)
		if name != "0" && !strings.HasSuffix(name, "/0") {
			return false
		}
		return strings.HasPrefix(r.StringValue, "urn:oma:lwm2m:")
	}

	var device string
	headers := 0
	for _, r := range pack {
		if isHeader(r) {
			headers++
			if device == "" {
				device = strings.Split(fullName(r), "/")[0]
			}
		}
	}

	isBare := func(r senml.Record) (senml.Record, bool) {
		name := fullName(r)
		if name == "tenant" || name == "source" || name == "env" {
			if r.StringValue == "" {
				return senml.Record{}, false
			}
			r.BaseName = device + "/"
			return r, true
		}
		if name == "" && (r.Unit == senml.UnitLat || r.Unit == senml.UnitLon) {
			if r.Value == nil {
				return senml.Record{}, false
			}
			n := "lat"
			if r.Unit == senml.UnitLon {
				n = "lon"
			}
			r.BaseName = device + "/"
			r.Name = n
			return r, true
		}
		return senml.Record{}, false
	}

	hoistable := device != "" && headers == 1
	var out senml.Pack
	hoisted := 0
	for _, r := range pack {
		if hoistable {
			if h, ok := isBare(r); ok {
				out = append(out, h)
				hoisted++
				continue
			}
		}
		out = append(out, r)
	}
	if hoisted > 0 {
		log.Debug("hoisted legacy metadata to pack level", "device_id", device, "records", hoisted)
	}
	return out
}

// enrichPack berikar ett tolkat pack med kvalificerad metadata från betrodd
// enhetsdata samt härledda observationsposter. Allt som läggs till använder
// fullständiga namn så att resultatet förblir parsebart och varje
// observation behåller sin identitet, sina värden och sin mättidpunkt.
//
// Tenant från payload ersätts alltid med enhetens tenant: payloaden får
// aldrig styra tenant. För source/env/lat/lon vinner enhetens värden när de
// är satta (legacy-paritet); annars behålls payloadens. Objektlokal
// metadata från payload lämnas orörd.
func enrichPack(ctx context.Context, parsed *diwisepkg.Pack, ref time.Time, device client.Device, measurementsClient measurements.MeasurementsClient) senml.Pack {
	deviceID := parsed.DeviceID()
	prefix := deviceID + "/"

	tenant := device.Tenant()
	if tenant == "" {
		tenant = "default"
	}
	source, hasSource := device.Source(), device.Source() != ""
	env, hasEnv := device.Environment(), device.Environment() != ""
	lat, lon := device.Latitude(), device.Longitude()

	records := make(senml.Pack, 0, len(parsed.Pack())+5)
	for _, r := range parsed.Pack() {
		name := r.Name
		if !strings.HasPrefix(name, prefix) {
			records = append(records, r)
			continue
		}
		trailing := strings.TrimPrefix(name, prefix)
		switch {
		case trailing == "tenant":
			continue // ersätts alltid nedan
		case trailing == "source" && hasSource,
			trailing == "env" && hasEnv,
			trailing == "lat" && r.Unit == senml.UnitLat,
			trailing == "lon" && r.Unit == senml.UnitLon:
			continue // ersätts alltid nedan
		default:
			records = append(records, r)
		}
	}

	records = append(records, senml.Record{Name: deviceID + "/tenant", StringValue: tenant})
	if hasSource {
		records = append(records, senml.Record{Name: deviceID + "/source", StringValue: source})
	}
	if hasEnv {
		records = append(records, senml.Record{Name: deviceID + "/env", StringValue: env})
	}
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

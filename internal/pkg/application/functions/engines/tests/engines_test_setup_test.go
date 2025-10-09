package engine

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/diwise/iot-core/internal/pkg/application/functions/engines"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/database/rules"
	"github.com/diwise/iot-core/internal/pkg/infrastructure/repository"
	"github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	testCtx            context.Context
	testPool           *pgxpool.Pool
	testRuleStore      rules.Storage
	testRuleRepository repository.RuleRepository
	dbAvailable        bool
	lastSetupError     string
	log                *slog.Logger
)

func init() {
	fmt.Fprintln(os.Stderr, "[setup] init(): engines_test_setup.go loaded")
}

func TestMain(m *testing.M) {
	fmt.Fprintln(os.Stderr, "[setup] TestMain: start (tests/engines)")
	log = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	testCtx = context.Background()
	cfg := database.NewConfig("localhost", "diwise", "diwise", "5432", "diwise", "disable")

	pool, err := database.GetConnection(testCtx, cfg)
	if err != nil {
		lastSetupError = fmt.Sprintf("connect failed: %v", err)
		fmt.Fprintln(os.Stderr, "[setup]", lastSetupError)
		goto RUN
	}
	if pingErr := pool.Ping(testCtx); pingErr != nil {
		lastSetupError = fmt.Sprintf("ping failed: %v", pingErr)
		fmt.Fprintln(os.Stderr, "[setup]", lastSetupError)
		pool.Close()
		goto RUN
	}

	testPool = pool

	testRuleStore = rules.Connect(testPool)
	if initErr := testRuleStore.Initialize(testCtx); initErr != nil {
		lastSetupError = fmt.Sprintf("initialize failed: %v", initErr)
		fmt.Fprintln(os.Stderr, "[setup]", lastSetupError)
	} else {
		dbAvailable = true
		fmt.Fprintln(os.Stderr, "[setup] DB available: OK")
	}
	testRuleRepository = repository.NewRepository(testRuleStore)

RUN:
	code := m.Run()
	if testPool != nil {
		testPool.Close()
	}
	os.Exit(code)
}

func requireDB(t *testing.T) {
	t.Helper()
	if dbAvailable {
		return
	}
	msg := "skippas: ingen DB tillgänglig"
	if lastSetupError != "" {
		msg += " (" + lastSetupError + ")"
	}
	t.Skip(msg)
}

func cleanDB(t *testing.T) {
	t.Helper()
	requireDB(t)
	if _, err := testPool.Exec(testCtx, `TRUNCATE TABLE rules;`); err != nil {
		t.Fatalf("truncate rules: %v", err)
	}
}

/** Assert functions for pointer values **/

func assertFloatPtrEq(t *testing.T, got, want *float64) {
	t.Helper()
	if got == nil && want == nil {
		return
	}
	if (got == nil) != (want == nil) {
		t.Fatalf("nil mismatch: got=%v want=%v", got, want)
	}
	const eps = 1e-9
	diff := *got - *want
	if diff < -eps || diff > eps {
		t.Fatalf("float mismatch: got=%v want=%v", *got, *want)
	}
}

func assertStringPtrEq(t *testing.T, got, want *string) {
	t.Helper()
	if got == nil && want == nil {
		return
	}
	if (got == nil) != (want == nil) {
		t.Fatalf("nil mismatch: got=%v want=%v", got, want)
	}
	if *got != *want {
		t.Fatalf("string mismatch: got=%q want=%q", *got, *want)
	}
}

func assertBoolPtrEq(t *testing.T, got, want *bool) {
	t.Helper()
	if got == nil && want == nil {
		return
	}
	if (got == nil) != (want == nil) {
		t.Fatalf("nil mismatch: got=%v want=%v", got, want)
	}
	if *got != *want {
		t.Fatalf("bool mismatch: got=%v want=%v", *got, *want)
	}
}

func newTestRepository() repository.RuleRepository {
	return repository.NewRepository(testRuleStore)
}

func newTestEngine() engines.RuleEngine {
	return engines.NewEngine(testRuleRepository)
}

// {F: n:1, v:22.5} {FS: n:3, vs:w1e} {VB: n:10, vb:true}
func newMessageReceivedWithPacks(id string) events.MessageReceived {
	msg := events.MessageReceived{
		Pack_: senml.Pack{
			// Base record
			createBaseRecord(id),
			// Float record
			senml.Record{
				Name:  "1",
				Value: F64(22.5),
				Unit:  "m3",
			},
			// String record
			senml.Record{
				Name:        "3",
				StringValue: "w1e",
			},
			// Boolean record
			senml.Record{
				Name:      "10",
				BoolValue: B(true),
			},
		}}

	return msg
}

func createBaseRecord(id string) senml.Record {
	return senml.Record{
		XMLName:     nil,
		BaseName:    id,
		BaseTime:    1563735600,
		BaseUnit:    "",
		Name:        "0",
		Unit:        "",
		StringValue: "",
		DataValue:   "",
	}
}

func newMessageReceived(records []senml.Record) events.MessageReceived {
	msg := events.MessageReceived{
		Pack_: records,
	}
	return msg
}

func F64(v float64) *float64 { return &v }
func S(v string) *string     { return &v }
func B(v bool) *bool         { return &v }

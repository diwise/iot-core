package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSQL(t *testing.T) {
	// start TimescaleDB using 'docker compose -f deployments/docker-compose.yaml up'
	// Without IOT_TEST_DATABASE=1 the test skips explicitly instead of
	// passing silently; with it, a missing database fails the test.
	if os.Getenv("IOT_TEST_DATABASE") == "" {
		t.Skip("skipping database integration test; set IOT_TEST_DATABASE=1 with a running TimescaleDB to run it")
	}

	s, ctx, err := testSetup()
	if err != nil {
		t.Fatalf("IOT_TEST_DATABASE=1 requires a running database: %v", err)
	}

	err = s.AddFnct(ctx, "fnct-01", "waterquality", "beach", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}

	err = s.Add(ctx, "fnct-01", "temperature", 42.0, time.Now().UTC())
	if err != nil {
		t.Error(err)
	}

	err = s.Add(ctx, "fnct-01", "temperature", 45.0, time.Now().UTC().Add(5*time.Second))
	if err != nil {
		t.Error(err)
	}

	logValues, err := s.History(ctx, "fnct-01", "temperature", 10)
	if err != nil {
		t.Error(err)
	}
	if !(len(logValues) >= 2) {
		t.Fail()
	}
}

func testSetup() (Storage, context.Context, error) {
	cfg := Config{
		host:     "localhost",
		user:     "diwise",
		password: "diwise",
		port:     "5432",
		dbname:   "diwise",
		sslmode:  "disable",
	}

	ctx := context.Background()

	s, err := Connect(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	_ = s.Initialize(ctx)

	return s, ctx, nil
}

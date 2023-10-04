package database

import (
	"context"
	"testing"
	"time"
)

func TestSQL(t *testing.T) {
	// start TimescaleDB using 'docker compose -f deployments/docker-compose.yaml up'
	// test will PASS if no DB is running

	s, ctx, err := testSetup()
	if err != nil {
		return
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

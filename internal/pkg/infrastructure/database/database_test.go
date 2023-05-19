package database

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestSQL(t *testing.T) {
	s, ctx, err := testSetup()
	if err != nil {
		return
	}

	err = s.AddFn(ctx, "fnct-01", "waterquality", "beach", "", "", 0, 0)
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

	s, err := Connect(ctx, zerolog.Logger{}, cfg)
	if err != nil {
		return nil, nil, err
	}

	_ = s.Initialize(ctx)

	return s, ctx, nil
}

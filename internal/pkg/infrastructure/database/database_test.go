package database

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestXxx(t *testing.T) {
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

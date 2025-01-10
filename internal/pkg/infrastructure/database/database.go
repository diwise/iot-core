package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/diwise/iot-core/internal/pkg/application/functions"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate moq -rm -out storage_mock.go . Storage

type Storage interface {
	Initialize(context.Context) error
	functions.RegistryStorer
}

type impl struct {
	db *pgxpool.Pool
}

type Config struct {
	host     string
	user     string
	password string
	port     string
	dbname   string
	sslmode  string
}

func (c Config) ConnStr() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.user, c.password, c.host, c.port, c.dbname, c.sslmode)
}

func NewConfig(host, user, password, port, dbname, sslmode string) Config {
	return Config{
		host:     host,
		user:     user,
		password: password,
		port:     port,
		dbname:   dbname,
		sslmode:  sslmode,
	}
}

func LoadConfiguration(ctx context.Context) Config {
	return Config{
		host:     env.GetVariableOrDefault(ctx, "POSTGRES_HOST", ""),
		user:     env.GetVariableOrDefault(ctx, "POSTGRES_USER", ""),
		password: env.GetVariableOrDefault(ctx, "POSTGRES_PASSWORD", ""),
		port:     env.GetVariableOrDefault(ctx, "POSTGRES_PORT", "5432"),
		dbname:   env.GetVariableOrDefault(ctx, "POSTGRES_DBNAME", "diwise"),
		sslmode:  env.GetVariableOrDefault(ctx, "POSTGRES_SSLMODE", "disable"),
	}
}

func Connect(ctx context.Context, cfg Config) (Storage, error) {
	conn, err := pgxpool.New(ctx, cfg.ConnStr())
	if err != nil {
		return nil, err
	}

	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return &impl{
		db: conn,
	}, nil
}

func (i *impl) Initialize(ctx context.Context) error {
	ddl := `
		CREATE TABLE IF NOT EXISTS fnct (
			id 		TEXT PRIMARY KEY NOT NULL,
			data	JSONB NULL
	  	);

		CREATE TABLE IF NOT EXISTS fnct_history (
			row_id 	bigserial,
			time 	TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			fnct_id TEXT NOT NULL,
			label 	TEXT NOT NULL,
			value 	DOUBLE PRECISION NOT NULL,
			FOREIGN KEY (fnct_id) REFERENCES fnct (id)
	  	);

		CREATE TABLE IF NOT EXISTS fnct_state (
			id 		  	TEXT PRIMARY KEY NOT NULL,
			time 		TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,						
			state     	JSONB NULL,
			FOREIGN KEY (id) REFERENCES fnct (id)
	  	);

		CREATE INDEX IF NOT EXISTS fnct_history_fnct_id_label_idx ON fnct_history (fnct_id, label);`

	tx, err := i.db.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, ddl)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	var n int32
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) n
		FROM timescaledb_information.hypertables
		WHERE hypertable_name = 'fnct_history';`).Scan(&n)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	if n == 0 {
		_, err := tx.Exec(ctx, `SELECT create_hypertable('fnct_history', 'time');`)
		if err != nil {
			tx.Rollback(ctx)
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (i *impl) LoadState(ctx context.Context, id string) ([]byte, error) {

	var state json.RawMessage

	err := i.db.QueryRow(ctx, "SELECT state FROM fnct_state WHERE id = $1", id).Scan(&state)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return state, err
}

func (i *impl) SaveState(ctx context.Context, id string, a any) error {

	b, _ := json.Marshal(a)

	args := pgx.NamedArgs{
		"id":    id,
		"state": json.RawMessage(b),
	}

	_, err := i.db.Exec(ctx, "INSERT INTO fnct_state (id, state) VALUES (@id, @state) ON CONFLICT (id) DO UPDATE SET state = @state;", args)

	return err
}

func (i *impl) AddSetting(ctx context.Context, id string, s functions.Setting) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	args := pgx.NamedArgs{
		"id":   id,
		"data": json.RawMessage(b),
	}

	_, err = i.db.Exec(ctx, "INSERT INTO fnct (id, data) VALUES (@id, @data) ON CONFLICT (id) DO UPDATE SET data = @data;", args)

	return err
}

func (i *impl) GetSettings(ctx context.Context) ([]functions.Setting, error) {
	rows, err := i.db.Query(ctx, "SELECT data FROM fnct")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make([]functions.Setting, 0)

	for rows.Next() {
		var data json.RawMessage
		err := rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		s := functions.Setting{}
		err = json.Unmarshal(data, &s)
		if err != nil {
			return nil, err
		}

		settings = append(settings, s)
	}

	return settings, nil
}

/* values */

func (i *impl) Add(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
	_, err := i.db.Exec(ctx, `
		INSERT INTO fnct_history (time, fnct_id, label, value) VALUES ($1, $2, $3, $4);
	`, timestamp, id, label, value)

	return err
}

func (i *impl) History(ctx context.Context, id, label string, lastN int) ([]functions.Value, error) {
	rows, err := i.db.Query(ctx,
		`SELECT time, value FROM (
			SELECT time, value, row_id
			FROM fnct_history
			WHERE fnct_id=$1 AND label=$2
			ORDER BY time DESC, row_id DESC
			LIMIT $3
			) as history
		ORDER BY time ASC, row_id ASC`, id, label, lastN)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logValues := make([]functions.Value, 0)

	for rows.Next() {
		var t time.Time
		var v float64
		err := rows.Scan(&t, &v)
		if err != nil {
			return nil, err
		}
		logValues = append(logValues, functions.Value{Name: label, Timestamp: t, Value: v})
	}

	return logValues, nil
}

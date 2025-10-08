package database

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate moq -rm -out storage_mock.go . Storage
//go:generate moq -rm -out func_storage_mock.go . FuncStorage

type Storage interface {
	Initialize(context.Context) error
}

type FuncStorage interface {
	Storage
	Add(ctx context.Context, id, label string, value float64, timestamp time.Time) error
	AddFnct(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error
	History(ctx context.Context, id, label string, lastN int) ([]LogValue, error)
}

type impl struct {
	db *pgxpool.Pool
}

type LogValue struct {
	Value     float64   `json:"v"`
	Timestamp time.Time `json:"ts"`
}

type Config struct {
	host     string
	user     string
	password string
	port     string
	dbname   string
	sslmode  string
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

func (c Config) ConnStr() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.user, c.password, c.host, c.port, c.dbname, c.sslmode)
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

func GetConnection(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	conn, err := pgxpool.New(ctx, cfg.ConnStr())
	if err != nil {
		return nil, err
	}

	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
func Connect(conn *pgxpool.Pool) FuncStorage {
	return &impl{
		db: conn,
	}
}

func (i *impl) Initialize(ctx context.Context) error {
	return i.createTables(ctx)
}

func (i *impl) createTables(ctx context.Context) error {
	ddl := `
		CREATE TABLE IF NOT EXISTS fnct (
			id 		  TEXT PRIMARY KEY NOT NULL,
			type 	  TEXT NOT NULL,
			sub_type  TEXT NOT NULL,
			tenant 	  TEXT NOT NULL,
			source 	  TEXT NULL,
			latitude  NUMERIC(7, 5),
			longitude NUMERIC(7, 5)
	  	);

		CREATE TABLE IF NOT EXISTS fnct_history (
			row_id 	bigserial,
			time 	TIMESTAMPTZ NOT NULL,
			fnct_id TEXT NOT NULL,
			label 	TEXT NOT NULL,
			value 	DOUBLE PRECISION NOT NULL,
			FOREIGN KEY (fnct_id) REFERENCES fnct (id)
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

func (i *impl) AddFnct(ctx context.Context, id, fnType, subType, tenant, source string, lat, lon float64) error {
	_, err := i.db.Exec(ctx, `
		INSERT INTO fnct(id,type,sub_type,tenant,source,latitude,longitude) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING;
	`, id, fnType, subType, tenant, source, lat, lon)

	return err
}

func (i *impl) Add(ctx context.Context, id, label string, value float64, timestamp time.Time) error {
	_, err := i.db.Exec(ctx, `
		INSERT INTO fnct_history (time, fnct_id, label, value) VALUES ($1, $2, $3, $4);
	`, timestamp, id, label, value)

	return err
}

func (i *impl) History(ctx context.Context, id, label string, lastN int) ([]LogValue, error) {
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

	logValues := make([]LogValue, 0)

	for rows.Next() {
		var t time.Time
		var v float64
		err := rows.Scan(&t, &v)
		if err != nil {
			return nil, err
		}
		logValues = append(logValues, LogValue{Timestamp: t, Value: v})
	}

	return logValues, nil
}

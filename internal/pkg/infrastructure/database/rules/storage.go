package rules

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/diwise/iot-core/internal/pkg/infrastructure/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

type impl struct {
	db *pgxpool.Pool
}

//go:generate moq -rm -out storage_mock.go . Storage

type Storage interface {
	database.Storage
	Add(ctx context.Context, rule Rule) error
	Get(ctx context.Context, id string) ([]Rule, []error, error)
	Update(ctx context.Context, rule Rule) error
	Delete(ctx context.Context, id string) error
}

var ErrNotFound = errors.New("rule not found")

func (i *impl) Add(ctx context.Context, r Rule) error {

	vmin, vmax, vs, vb, err := NormalizedParams(r)
	if err != nil {
		return err
	}

	const q = `
			INSERT INTO rules (
				id, measurement_id, device_id, measurement_type, should_abort,
				v_min_value, v_max_value, vs_value, vb_value
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (id) DO UPDATE SET
				measurement_id   = EXCLUDED.measurement_id,
				device_id        = EXCLUDED.device_id,
				measurement_type = EXCLUDED.measurement_type,
				should_abort     = EXCLUDED.should_abort,
				v_min_value      = EXCLUDED.v_min_value,
				v_max_value      = EXCLUDED.v_max_value,
				vs_value         = EXCLUDED.vs_value,
				vb_value         = EXCLUDED.vb_value;`

	_, err = i.db.Exec(ctx, q,
		r.ID, r.MeasurementID, r.DeviceID, r.MeasurementType, r.ShouldAbort,
		vmin, vmax, vs, vb,
	)
	return err
}

func (i *impl) Get(ctx context.Context, id string) ([]Rule, []error, error) {

	var rules []Rule
	var rowErrors []error

	const q = `
		SELECT
			id, measurement_id, device_id, measurement_type, should_abort,
			v_min_value, v_max_value, vs_value, vb_value
		FROM rules
		WHERE device_id = $1;`

	rows, err := i.db.Query(ctx, q, id)
	if err != nil {
		return rules, nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var (
			r    Rule
			vmin sql.NullFloat64
			vmax sql.NullFloat64
			vs   sql.NullString
			vb   sql.NullBool
		)

		err := rows.Scan(
			&r.ID, &r.MeasurementID, &r.DeviceID, &r.MeasurementType, &r.ShouldAbort,
			&vmin, &vmax, &vs, &vb,
		)
		if err != nil {
			rowErrors = append(rowErrors, fmt.Errorf("scan error: %w", err))
			continue
		}

		if vmin.Valid || vmax.Valid {
			r.RuleValues.V = &RuleV{}
			if vmin.Valid {
				min := vmin.Float64
				r.RuleValues.V.MinValue = &min
			}
			if vmax.Valid {
				max := vmax.Float64
				r.RuleValues.V.MaxValue = &max
			}
		}

		if vs.Valid {
			s := vs.String
			r.RuleValues.Vs = &RuleVs{Value: &s}
		}

		if vb.Valid {
			b := vb.Bool
			r.RuleValues.Vb = &RuleVb{Value: &b}
		}

		rules = append(rules, r)
	}

	if err = rows.Err(); err != nil {
		return rules, rowErrors, err
	}

	return rules, rowErrors, nil
}

func (i *impl) Update(ctx context.Context, r Rule) error {

	vmin, vmax, vs, vb, err := NormalizedParams(r)
	if err != nil {
		return err
	}

	const q = `
				UPDATE rules
				SET
					measurement_id   = $2,
					device_id        = $3,
					measurement_type = $4,
					should_abort     = $5,
					v_min_value      = $6,
					v_max_value      = $7,
					vs_value         = $8,
					vb_value         = $9
				WHERE id = $1;`

	ct, err := i.db.Exec(ctx, q,
		r.ID, r.MeasurementID, r.DeviceID, r.MeasurementType, r.ShouldAbort,
		vmin, vmax, vs, vb,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (i *impl) Delete(ctx context.Context, id string) error {
	ct, err := i.db.Exec(ctx, `DELETE FROM rules WHERE id=$1;`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (i *impl) Initialize(ctx context.Context) error {
	return i.createTables(ctx)
}

func (i *impl) createTables(ctx context.Context) (err error) {
	const ddl = `
				CREATE TABLE IF NOT EXISTS rules (
					id               TEXT PRIMARY KEY,
					measurement_id   TEXT        NOT NULL,
					device_id        TEXT        NOT NULL,
					measurement_type INT         NOT NULL,
					should_abort     BOOLEAN     NOT NULL,
					v_min_value      DOUBLE PRECISION,
					v_max_value      DOUBLE PRECISION,
					vs_value         TEXT,
					vb_value         BOOLEAN,
					CONSTRAINT rules_one_kind_chk CHECK (
						(vs_value IS NOT NULL AND v_min_value IS NULL AND v_max_value IS NULL AND vb_value IS NULL)
						OR
						(vb_value IS NOT NULL AND v_min_value IS NULL AND v_max_value IS NULL AND vs_value IS NULL)
						OR
						((v_min_value IS NOT NULL OR v_max_value IS NOT NULL) AND vs_value IS NULL AND vb_value IS NULL)
					)
				);
				`
	tx, err := i.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, ddl); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func Connect(conn *pgxpool.Pool) Storage {
	return &impl{db: conn}
}

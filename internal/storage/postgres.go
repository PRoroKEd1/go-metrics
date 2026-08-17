package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	models "github.com/PRoroKEd1/go-metrics/internal/model"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresStorage struct {
	db *sql.DB
}

func isRetryablePostgresError(err error) bool {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure,
			"40001",
			"40P01",
			"57P01",
			"53300":
			return true
		}
	}

	return false
}

func execWithRetry(ctx context.Context, db *sql.DB, query string, args ...any) error {
	delays := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	for attempt := 0; attempt <= len(delays); attempt++ {

		_, err := db.ExecContext(ctx, query, args...)

		if err == nil {
			return nil
		}

		if !isRetryablePostgresError(err) {
			return err
		}

		if attempt < len(delays) {
			time.Sleep(delays[attempt])
		}
	}

	return errors.New("postgres retry attempts exceeded")
}

func NewPostgresStorage(dsn string) (*PostgresStorage, *sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, err
	}

	migrationPath, err := filepath.Abs("migrations")
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	sourceURL := "file://" + filepath.ToSlash(migrationPath)

	sourceDriver, err := (&file.File{}).Open(sourceURL)
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	m, err := migrate.NewWithSourceInstance(
		"file",
		sourceDriver,
		dsn,
	)
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		db.Close()
		return nil, nil, err
	}

	return &PostgresStorage{db: db}, db, nil
}

func (ps *PostgresStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	var value float64

	err := ps.db.QueryRowContext(
		ctx,
		`SELECT value FROM metrics WHERE id = $1 AND type = 'gauge'`,
		name,
	).Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		return 0, false
	}

	return value, true
}

func (ps *PostgresStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	var value int64

	err := ps.db.QueryRowContext(
		ctx,
		`SELECT delta FROM metrics WHERE id = $1 AND type = 'counter'`,
		name,
	).Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		return 0, false
	}

	return value, true
}

func (ps *PostgresStorage) GetAllGauges(ctx context.Context) map[string]float64 {
	rows, err := ps.db.QueryContext(
		ctx,
		`SELECT id, value FROM metrics WHERE type = 'gauge'`,
	)
	if err != nil {
		return map[string]float64{}
	}
	defer rows.Close()

	result := make(map[string]float64)

	for rows.Next() {
		var name string
		var value float64

		if err := rows.Scan(&name, &value); err == nil {
			result[name] = value
		}
	}

	if err := rows.Err(); err != nil {
		return map[string]float64{}
	}

	return result
}

func (ps *PostgresStorage) GetAllCounters(ctx context.Context) map[string]int64 {
	rows, err := ps.db.QueryContext(
		ctx,
		`SELECT id, delta FROM metrics WHERE type = 'counter'`,
	)
	if err != nil {
		return map[string]int64{}
	}
	defer rows.Close()

	result := make(map[string]int64)

	for rows.Next() {
		var name string
		var value int64

		if err := rows.Scan(&name, &value); err == nil {
			result[name] = value
		}
	}

	if err := rows.Err(); err != nil {
		return map[string]int64{}
	}

	return result
}

func (ps *PostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	return execWithRetry(
		ctx,
		ps.db,
		`INSERT INTO metrics (id, type, value)
		 VALUES ($1, 'gauge', $2)
		 ON CONFLICT (id, type)
		 DO UPDATE SET value = EXCLUDED.value`,
		name,
		value,
	)
}

func (ps *PostgresStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	return execWithRetry(
		ctx,
		ps.db,
		`INSERT INTO metrics (id, type, delta)
		 VALUES ($1, 'counter', $2)
		 ON CONFLICT (id, type)
		 DO UPDATE SET delta = metrics.delta + EXCLUDED.delta`,
		name,
		value,
	)
}

func (ps *PostgresStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	args := make([]any, 0, len(metrics)*2)
	values := make([]string, 0, len(metrics)*4)

	for i, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("value is required for gauge")
			}

			idIndex := i*2 + 1
			valueIndex := i*2 + 2

			values = append(values, fmt.Sprintf(
				"($%d, 'gauge', NULL, $%d)",
				idIndex,
				valueIndex,
			))

			args = append(args, metric.ID, *metric.Value)

		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("delta is required for counter")
			}

			idIndex := i*2 + 1
			deltaIndex := i*2 + 2

			values = append(values, fmt.Sprintf(
				"($%d, 'counter', $%d, NULL)",
				idIndex,
				deltaIndex,
			))

			args = append(args, metric.ID, *metric.Delta)

		default:
			return fmt.Errorf("unknown metric type")
		}
	}

	query := `
		INSERT INTO metrics (id, type, delta, value)
		VALUES ` + strings.Join(values, ", ") + `
		ON CONFLICT (id, type)
		DO UPDATE SET
			delta = CASE
				WHEN EXCLUDED.type = 'counter'
				THEN metrics.delta + EXCLUDED.delta
				ELSE metrics.delta
			END,
			value = CASE
				WHEN EXCLUDED.type = 'gauge'
				THEN EXCLUDED.value
				ELSE metrics.value
			END
	`

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return tx.Commit()
}

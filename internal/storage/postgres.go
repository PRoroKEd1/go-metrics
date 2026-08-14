package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

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
			pgerrcode.ConnectionFailure:
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

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{
		db: db,
	}
}

func (ps *PostgresStorage) GetGauge(name string) (float64, bool) {
	var value float64

	err := ps.db.QueryRowContext(
		context.Background(),
		`SELECT value FROM metrics WHERE id = $1 AND type = 'gauge'`,
		name,
	).Scan(&value)

	if err == sql.ErrNoRows {
		return 0, false
	}
	if err != nil {
		return 0, false
	}

	return value, true
}

func (ps *PostgresStorage) GetCounter(name string) (int64, bool) {
	var value int64

	err := ps.db.QueryRowContext(
		context.Background(),
		`SELECT delta FROM metrics WHERE id = $1 AND type = 'counter'`,
		name,
	).Scan(&value)

	if err == sql.ErrNoRows {
		return 0, false
	}
	if err != nil {
		return 0, false
	}

	return value, true
}

func (ps *PostgresStorage) GetAllGauges() map[string]float64 {
	rows, err := ps.db.QueryContext(
		context.Background(),
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

func (ps *PostgresStorage) GetAllCounters() map[string]int64 {
	rows, err := ps.db.QueryContext(
		context.Background(),
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

func (ps *PostgresStorage) UpdateGauge(name string, value float64) {
	err := execWithRetry(
		context.Background(),
		ps.db,
		`INSERT INTO metrics (id, type, value)
		 VALUES ($1, 'gauge', $2)
		 ON CONFLICT (id, type)
		 DO UPDATE SET value = EXCLUDED.value`,
		name,
		value,
	)

	if err != nil {
		panic(err)
	}
}

func (ps *PostgresStorage) UpdateCounter(name string, value int64) {
	err := execWithRetry(
		context.Background(),
		ps.db,
		`INSERT INTO metrics (id, type, delta)
		 VALUES ($1, 'counter', $2)
		 ON CONFLICT (id, type)
		 DO UPDATE SET delta = metrics.delta + EXCLUDED.delta`,
		name,
		value,
	)

	if err != nil {
		panic(err)
	}
}

func (ps *PostgresStorage) SaveToFile(string) error {
	return nil
}

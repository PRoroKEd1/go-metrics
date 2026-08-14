package storage

import (
	"context"
	"database/sql"
)

type PostgresStorage struct {
	db *sql.DB
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
	_, err := ps.db.ExecContext(
		context.Background(),
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
	_, _ = ps.db.ExecContext(
		context.Background(),
		`INSERT INTO metrics (id, type, delta)
		 VALUES ($1, 'counter', $2)
		 ON CONFLICT (id, type)
		 DO UPDATE SET delta = metrics.delta + EXCLUDED.delta`,
		name,
		value,
	)
}

func (ps *PostgresStorage) SaveToFile(string) error {
	return nil
}

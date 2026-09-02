package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	models "github.com/PRoroKEd1/go-metrics/internal/model"
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func NewFileStorage(filename string, restore bool) (*MemStorage, error) {
	store := NewMemStorage()

	if restore {
		if err := store.RestoreFromFile(filename); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (m *MemStorage) SaveToFile(filename string) error {
	var metrics []models.Metrics

	for name, value := range m.gauge {
		v := value
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}

	for name, delta := range m.counter {
		d := delta
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &d,
		})
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0666)
}

func (m *MemStorage) RestoreFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metrics []models.Metrics

	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				m.UpdateGauge(context.Background(), metric.ID, *metric.Value)
			}
		case "counter":
			if metric.Delta != nil {
				m.counter[metric.ID] = *metric.Delta
			}
		}
	}

	return nil
}

func (m *MemStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	gauges := make(map[string]float64, len(m.gauge))
	for k, v := range m.gauge {
		gauges[k] = v
	}

	counters := make(map[string]int64, len(m.counter))
	for k, v := range m.counter {
		counters[k] = v
	}

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("value is required for gauge")
			}
			gauges[metric.ID] = *metric.Value

		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("delta is required for counter")
			}
			counters[metric.ID] += *metric.Delta

		default:
			return fmt.Errorf("unknown metric type")
		}
	}

	m.gauge = gauges
	m.counter = counters

	return nil
}

func (m *MemStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	val, ok := m.gauge[name]
	return val, ok
}

func (m *MemStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	val, ok := m.counter[name]
	return val, ok
}

func (m *MemStorage) GetAllGauges(ctx context.Context) map[string]float64 {
	return m.gauge
}

func (m *MemStorage) GetAllCounters(ctx context.Context) map[string]int64 {
	return m.counter
}

func (m *MemStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	m.gauge[name] = value
	return nil
}

func (m *MemStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	m.counter[name] += value
	return nil
}

package storage

import (
	"encoding/json"
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

func (ms *MemStorage) SaveToFile(filename string) error {
	var metrics []models.Metrics

	for name, value := range ms.gauge {
		v := value
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}

	for name, delta := range ms.counter {
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

func (ms *MemStorage) RestoreFromFile(filename string) error {
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

	for _, m := range metrics {
		switch m.MType {
		case "gauge":
			if m.Value != nil {
				ms.UpdateGauge(m.ID, *m.Value)
			}
		case "counter":
			if m.Delta != nil {
				ms.UpdateCounter(m.ID, *m.Delta)
			}
		}
	}

	return nil
}

func (m *MemStorage) GetGauge(name string) (float64, bool) {
	val, ok := m.gauge[name]
	return val, ok
}

func (m *MemStorage) GetCounter(name string) (int64, bool) {
	val, ok := m.counter[name]
	return val, ok
}

func (m *MemStorage) GetAllGauges() map[string]float64 {
	return m.gauge
}

func (m *MemStorage) GetAllCounters() map[string]int64 {
	return m.counter
}

func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.gauge[name] = value
}

func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.counter[name] += value
}

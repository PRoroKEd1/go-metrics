package agent

import (
	"sync"

	models "github.com/PRoroKEd1/go-metrics/internal/model"
)

type MemStorage struct {
	mu             sync.RWMutex
	gaugeMetrics   map[string]float64
	counterMetrics map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gaugeMetrics:   make(map[string]float64),
		counterMetrics: make(map[string]int64),
	}
}

func (ms *MemStorage) SetGauge(name string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gaugeMetrics[name] = value
}

func (ms *MemStorage) IncrementCounter(name string, value int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counterMetrics[name] += value
}

func (ms *MemStorage) GetMetricsSnapshotAndReset() []models.Metrics {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	metrics := make([]models.Metrics, 0, len(ms.gaugeMetrics)+len(ms.counterMetrics))

	for name, value := range ms.gaugeMetrics {
		v := value
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}

	for name, delta := range ms.counterMetrics {
		d := delta
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &d,
		})
	}

	ms.counterMetrics["PollCount"] = 0
	return metrics
}

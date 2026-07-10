package storage

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

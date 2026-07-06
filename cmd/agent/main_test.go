package main

import (
	"testing"
)

func TestFlagMemStorage(t *testing.T) {
	storage := NewMemStorage()

	if storage.gaugeMetrics == nil {
		t.Errorf("Ошибка: мапа gaugeMetrics не инициализирована (равна nil)")
	}

	if storage.counterMetrics == nil {
		t.Errorf("Ошибка: мапа counterMetrics не инициализирована (равна nil)")
	}
}

func TestFlagAgentMetrics(t *testing.T) {
	storage := NewMemStorage()

	storage.gaugeMetrics["RandomValue"] = 0.123
	storage.counterMetrics["PollCount"]++

	storage.counterMetrics["PollCount"]++

	if val, ok := storage.gaugeMetrics["RandomValue"]; !ok || val != 0.123 {
		t.Errorf("Ожидалось значение RandomValue 0.123, получено %v", val)
	}

	if val, ok := storage.counterMetrics["PollCount"]; !ok || val != 2 {
		t.Errorf("Ожидалось значение PollCount 2, получено %v", val)
	}
}

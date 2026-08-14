package storage

import (
	"os"
	"testing"
)

func TestMemStorage_GetAllGauges(t *testing.T) {
	store := NewMemStorage()

	gauges := store.GetAllGauges()
	if len(gauges) != 0 {
		t.Errorf("Expected empty map, got %d elements", len(gauges))
	}

	store.UpdateGauge("TestMetric", 123.45)

	gauges = store.GetAllGauges()
	if len(gauges) != 1 {
		t.Errorf("Expected 1 element, got %d elements", len(gauges))
	}

	if gauges["TestMetric"] != 123.45 {
		t.Errorf("Expected 123.45, got %v", gauges["TestMetric"])
	}
}

func TestMemStorage_Counter(t *testing.T) {
	store := NewMemStorage()

	store.UpdateCounter("Requests", 10)
	store.UpdateCounter("Requests", 5)

	value, ok := store.GetCounter("Requests")

	if !ok {
		t.Fatal("counter не найден")
	}

	if value != 15 {
		t.Fatalf(
			"ожидалось 15, получили %d",
			value,
		)
	}
}

func TestFileStorage_SaveRestore(t *testing.T) {

	filename := "test_metrics.json"

	defer os.Remove(filename)

	store := NewMemStorage()

	store.UpdateGauge(
		"CPU",
		55.5,
	)

	store.UpdateCounter(
		"PollCount",
		100,
	)

	err := store.SaveToFile(filename)

	if err != nil {
		t.Fatal(err)
	}

	newStore := NewMemStorage()

	err = newStore.RestoreFromFile(filename)

	if err != nil {
		t.Fatal(err)
	}

	gauge, ok := newStore.GetGauge("CPU")

	if !ok || gauge != 55.5 {
		t.Fatalf(
			"gauge восстановлен неправильно: %v",
			gauge,
		)
	}

	counter, ok := newStore.GetCounter("PollCount")

	if !ok || counter != 100 {
		t.Fatalf(
			"counter восстановлен неправильно: %d",
			counter,
		)
	}
}

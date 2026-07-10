package storage

import (
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

package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMemStorage_GetAllGauges(t *testing.T) {
	store := NewMemStorage()
	ctx := context.Background()

	gauges := store.GetAllGauges(ctx)
	if len(gauges) != 0 {
		t.Errorf("Expected empty map, got %d elements", len(gauges))
	}

	if err := store.UpdateGauge(ctx, "TestMetric", 123.45); err != nil {
		t.Fatal(err)
	}

	gauges = store.GetAllGauges(ctx)
	if len(gauges) != 1 {
		t.Errorf("Expected 1 element, got %d elements", len(gauges))
	}

	if gauges["TestMetric"] != 123.45 {
		t.Errorf("Expected 123.45, got %v", gauges["TestMetric"])
	}
}

func TestMemStorage_Counter(t *testing.T) {
	store := NewMemStorage()
	ctx := context.Background()

	if err := store.UpdateCounter(ctx, "Requests", 10); err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateCounter(ctx, "Requests", 5); err != nil {
		t.Fatal(err)
	}

	value, ok := store.GetCounter(ctx, "Requests")

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
	ctx := context.Background()

	if err := store.UpdateGauge(
		ctx,
		"CPU",
		55.5,
	); err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateCounter(
		ctx,
		"PollCount",
		100,
	); err != nil {
		t.Fatal(err)
	}

	err := store.SaveToFile(filename)

	if err != nil {
		t.Fatal(err)
	}

	newStore := NewMemStorage()

	err = newStore.RestoreFromFile(filename)

	if err != nil {
		t.Fatal(err)
	}

	gauge, ok := newStore.GetGauge(ctx, "CPU")

	if !ok || gauge != 55.5 {
		t.Fatalf(
			"gauge восстановлен неправильно: %v",
			gauge,
		)
	}

	counter, ok := newStore.GetCounter(ctx, "PollCount")

	if !ok || counter != 100 {
		t.Fatalf(
			"counter восстановлен неправильно: %d",
			counter,
		)
	}
}

func BenchmarkMemStorage_Update(b *testing.B) {
	store := NewMemStorage()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.UpdateGauge(ctx, "Alloc", float64(i))
		_ = store.UpdateCounter(ctx, "PollCount", int64(i))
	}
}

func TestMemStorage_SaveAndRestore(t *testing.T) {
	ctx := context.Background()
	store := NewMemStorage()
	_ = store.UpdateGauge(ctx, "Alloc", 42.5)
	_ = store.UpdateCounter(ctx, "PollCount", 10)

	tmpFile := filepath.Join(t.TempDir(), "metrics.json")

	if err := store.SaveToFile(tmpFile); err != nil {
		t.Fatalf("ошибка сохранения в файл: %v", err)
	}

	newStore := NewMemStorage()
	if err := newStore.RestoreFromFile(tmpFile); err != nil {
		t.Fatalf("ошибка восстановления из файла: %v", err)
	}

	val, ok := newStore.GetGauge(ctx, "Alloc")
	if !ok || val != 42.5 {
		t.Errorf("ожидался gauge 42.5, получен %v", val)
	}

	cnt, ok := newStore.GetCounter(ctx, "PollCount")
	if !ok || cnt != 10 {
		t.Errorf("ожидался counter 10, получен %v", cnt)
	}
}

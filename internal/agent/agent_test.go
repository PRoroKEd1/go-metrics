package agent

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "github.com/PRoroKEd1/go-metrics/internal/model"
)

func TestNewMemStorage(t *testing.T) {
	storage := NewMemStorage()

	snapshot := storage.GetMetricsSnapshot()
	if snapshot == nil {
		t.Errorf("Ошибка: GetMetricsSnapshot вернул nil, хранилище не инициализировано корректно")
	}
}

func TestAgentMetrics(t *testing.T) {
	storage := NewMemStorage()
	storage.SetGauge("RandomValue", 0.123)
	storage.IncrementCounter("PollCount", 1)
	storage.IncrementCounter("PollCount", 1)

	snapshot := storage.GetMetricsSnapshot()

	var foundGauge bool
	var foundCounter bool

	for _, m := range snapshot {
		if m.ID == "RandomValue" && m.MType == "gauge" {
			foundGauge = true
			if m.Value == nil || *m.Value != 0.123 {
				t.Errorf("Ожидалось значение RandomValue 0.123, получено %v", *m.Value)
			}
		}
		if m.ID == "PollCount" && m.MType == "counter" {
			foundCounter = true
			if m.Delta == nil || *m.Delta != 2 {
				t.Errorf("Ожидалось значение PollCount 2, получено %v", *m.Delta)
			}
		}
	}

	if !foundGauge {
		t.Errorf("Метрика RandomValue не найдена в снапшоте")
	}
	if !foundCounter {
		t.Errorf("Метрика PollCount не найдена в снапшоте")
	}
}

func TestCompress(t *testing.T) {

	data := []byte("hello metrics")

	result, err := compress(data)

	if err != nil {
		t.Fatal(err)
	}

	reader, err := gzip.NewReader(
		bytes.NewReader(result),
	)

	if err != nil {
		t.Fatal(err)
	}

	defer reader.Close()

	buf := new(bytes.Buffer)

	_, err = buf.ReadFrom(reader)

	if err != nil {
		t.Fatal(err)
	}

	if buf.String() != string(data) {
		t.Fatalf(
			"ожидалось %s получили %s",
			data,
			buf.String(),
		)
	}
}

func TestSendMetrics(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				if r.URL.Path != "/updates/" {
					t.Fatal("неверный путь")
				}

				w.WriteHeader(http.StatusOK)
			},
		),
	)

	defer server.Close()

	addr := strings.TrimPrefix(
		server.URL,
		"http://",
	)

	value := 10.0

	metrics := []models.Metrics{
		{
			ID:    "Test",
			MType: "gauge",
			Value: &value,
		},
	}

	sendMetrics(
		addr,
		metrics,
		"",
	)
}

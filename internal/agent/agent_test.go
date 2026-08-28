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

	if storage.gaugeMetrics == nil {
		t.Errorf("Ошибка: мапа gaugeMetrics не инициализирована (равна nil)")
	}

	if storage.counterMetrics == nil {
		t.Errorf("Ошибка: мапа counterMetrics не инициализирована (равна nil)")
	}
}

func TestAgentMetrics(t *testing.T) {
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

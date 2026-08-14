package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/PRoroKEd1/go-metrics/internal/model"
	"github.com/PRoroKEd1/go-metrics/internal/storage"
)

func TestUpdatesJSONHandler(t *testing.T) {
	store := storage.NewMemStorage()

	h := NewHandler(
		store,
		false,
		"",
		nil,
	)

	metrics := []models.Metrics{
		{
			ID:    "TestGauge",
			MType: "gauge",
			Value: func() *float64 {
				v := 10.5
				return &v
			}(),
		},
		{
			ID:    "TestCounter",
			MType: "counter",
			Delta: func() *int64 {
				v := int64(5)
				return &v
			}(),
		},
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/updates/",
		bytes.NewBuffer(body),
	)

	rec := httptest.NewRecorder()

	h.UpdatesJSONHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"ожидался статус 200, получили %d",
			rec.Code,
		)
	}

	gauge, ok := store.GetGauge("TestGauge")
	if !ok {
		t.Fatal("gauge не найден")
	}

	if gauge != 10.5 {
		t.Fatalf(
			"ожидалось 10.5, получили %v",
			gauge,
		)
	}

	counter, ok := store.GetCounter("TestCounter")
	if !ok {
		t.Fatal("counter не найден")
	}

	if counter != 5 {
		t.Fatalf(
			"ожидалось 5, получили %d",
			counter,
		)
	}
}

func TestUpdatesJSONHandlerEmpty(t *testing.T) {

	store := storage.NewMemStorage()

	h := NewHandler(
		store,
		false,
		"",
		nil,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/updates/",
		bytes.NewBuffer([]byte(`[]`)),
	)

	rec := httptest.NewRecorder()

	h.UpdatesJSONHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"ожидался 200, получили %d",
			rec.Code,
		)
	}

}

func TestUpdateJSONHandler_InvalidType(t *testing.T) {

	store := storage.NewMemStorage()

	h := NewHandler(
		store,
		false,
		"",
		nil,
	)

	body := []byte(`
{
"id":"test",
"type":"unknown"
}
`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/update/",
		bytes.NewBuffer(body),
	)

	rec := httptest.NewRecorder()

	h.UpdateJSONHandler(
		rec,
		req,
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"ожидался 400, получили %d",
			rec.Code,
		)
	}
}

func TestValueJSONHandler(t *testing.T) {

	store := storage.NewMemStorage()

	store.UpdateGauge(
		"Temperature",
		25.5,
	)

	h := NewHandler(
		store,
		false,
		"",
		nil,
	)

	body := []byte(`
{
"id":"Temperature",
"type":"gauge"
}
`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/value/",
		bytes.NewBuffer(body),
	)

	rec := httptest.NewRecorder()

	h.ValueJSONHandler(
		rec,
		req,
	)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"ожидался 200, получили %d",
			rec.Code,
		)
	}
}

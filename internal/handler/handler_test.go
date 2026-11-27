package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "github.com/PRoroKEd1/go-metrics/internal/model"
	"github.com/PRoroKEd1/go-metrics/internal/storage"
)

func TestUpdatesJSONHandler(t *testing.T) {
	store := storage.NewMemStorage()

	h := NewHandler(store, false, "", nil, nil)

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

	gauge, ok := store.GetGauge(context.Background(), "TestGauge")
	if !ok {
		t.Fatal("gauge не найден")
	}

	if gauge != 10.5 {
		t.Fatalf(
			"ожидалось 10.5, получили %v",
			gauge,
		)
	}

	counter, ok := store.GetCounter(context.Background(), "TestCounter")
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

	h := NewHandler(store, false, "", nil, nil)

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

	h := NewHandler(store, false, "", nil, nil)

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
		context.Background(),
		"Temperature",
		25.5,
	)

	h := NewHandler(store, false, "", nil, nil)

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

func TestJSONAndExtraHandlers(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewHandler(store, false, "", nil, nil)

	// 1. Тест UpdateJSONHandler (все ветки: успех и ошибки валидации)
	for _, body := range []string{
		`{"id":"Alloc","type":"gauge","value":123.45}`,
		`{"id":"PollCount","type":"counter","delta":10}`,
		`{"id":"Alloc","type":"gauge"}`,
		`{"id":"PollCount","type":"counter"}`,
		`{"id":"Unknown","type":"unknown"}`,
		`invalid_json`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.UpdateJSONHandler(rec, req)
	}

	// 2. Тест UpdatesJSONHandler (батч: успех и ошибки)
	for _, body := range []string{
		`[{"id":"Alloc","type":"gauge","value":55.5},{"id":"PollCount","type":"counter","delta":5}]`,
		`[{"id":"BadGauge","type":"gauge"},{"id":"BadCounter","type":"counter"},{"id":"BadType","type":"unknown"}]`,
		`[]`,
		`invalid_json`,
	} {
		reqBatch := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
		recBatch := httptest.NewRecorder()
		h.UpdatesJSONHandler(recBatch, reqBatch)
	}

	// 3. Тест ValueJSONHandler (все ветки: найденные, 404 и ошибки)
	for _, body := range []string{
		`{"id":"Alloc","type":"gauge"}`,
		`{"id":"PollCount","type":"counter"}`,
		`{"id":"MissingGauge","type":"gauge"}`,
		`{"id":"MissingCounter","type":"counter"}`,
		`{"id":"UnknownType","type":"unknown"}`,
		`invalid_json`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.ValueJSONHandler(rec, req)
	}

	// 4. Тест HTML-страницы и PingDBHandler
	reqAll := httptest.NewRequest(http.MethodGet, "/", nil)
	recAll := httptest.NewRecorder()
	h.GetAllMetricsHandler(recAll, reqAll)

	reqPing := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recPing := httptest.NewRecorder()
	h.PingDBHandler(recPing, reqPing)
}

package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/PRoroKEd1/go-metrics/internal/storage"
)

// демонстрирует обновление метрики типа gauge через JSON API.
func ExampleHandler_UpdateJSONHandler() {
	store := storage.NewMemStorage()
	h := NewHandler(store, false, "", nil, nil)

	body := `{"id":"Alloc","type":"gauge","value":123.45}`
	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.UpdateJSONHandler(rec, req)

	fmt.Println(rec.Code)
	fmt.Print(rec.Body.String())

	// Output:
	// 200
	// {"id":"Alloc","type":"gauge","value":123.45}
}

// демонстрирует получение значения метрики через JSON API.
func ExampleHandler_ValueJSONHandler() {
	store := storage.NewMemStorage()
	h := NewHandler(store, false, "", nil, nil)

	// Сначала сохраняем метрику
	updateReq := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(`{"id":"PollCount","type":"counter","delta":5}`))
	h.UpdateJSONHandler(httptest.NewRecorder(), updateReq)

	// Запрашиваем её значение
	valReq := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(`{"id":"PollCount","type":"counter"}`))
	rec := httptest.NewRecorder()
	h.ValueJSONHandler(rec, valReq)

	fmt.Println(rec.Code)
	fmt.Print(rec.Body.String())

	// Output:
	// 200
	// {"id":"PollCount","type":"counter","delta":5}
}

// демонстрирует пакетное обновление метрик
func ExampleHandler_UpdatesJSONHandler() {
	store := storage.NewMemStorage()
	h := NewHandler(store, false, "", nil, nil)

	batch := `[{"id":"Alloc","type":"gauge","value":100},{"id":"PollCount","type":"counter","delta":1}]`
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(batch))
	rec := httptest.NewRecorder()

	h.UpdatesJSONHandler(rec, req)

	fmt.Println(rec.Code)

	// Output:
	// 200
}

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PRoroKEd1/go-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func TestGetMetricHandler(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewHandler(store)
	store.UpdateGauge("CPU", 42.5)
	store.UpdateCounter("Clicks", 10)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", h.GetMetricHandler)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Положительный: gauge существует",
			url:            "/value/gauge/CPU",
			expectedStatus: http.StatusOK,
			expectedBody:   "42.5",
		},
		{
			name:           "Негативный: gauge не существует",
			url:            "/value/gauge/RAM",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Metric not found\n",
		},
		{
			name:           "Негативный: неизвестный тип metric",
			url:            "/value/asd/qwe",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Unknown metric type\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Ожидался статус %d, получено %d", tt.expectedStatus, res.StatusCode)
			}

			bodyBytes, _ := io.ReadAll(res.Body)
			bodyString := string(bodyBytes)

			if bodyString != tt.expectedBody {
				t.Errorf("Ожидалось %q, получено %q", tt.expectedBody, bodyString)
			}
		})
	}
}

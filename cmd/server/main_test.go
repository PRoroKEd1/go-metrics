package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestGetMetricHandler(t *testing.T) {
	storage := NewMemStorage()
	storage.UpdateGauge("CPU", 42.5)
	storage.UpdateCounter("Clicks", 10)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", storage.getMetricHandler)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Positive: get existing gauge",
			url:            "/value/gauge/CPU",
			expectedStatus: http.StatusOK,
			expectedBody:   "42.5",
		},
		{
			name:           "Negative: get non-existent gauge",
			url:            "/value/gauge/RAM",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Metric not found\n",
		},
		{
			name:           "Negative: unknown metric type",
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
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			bodyBytes, _ := io.ReadAll(res.Body)
			bodyString := string(bodyBytes)

			if bodyString != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, bodyString)
			}
		})
	}
}

func TestMemStorage_GetAllGauges(t *testing.T) {
	storage := NewMemStorage()

	gauges := storage.GetAllGauges()
	if len(gauges) != 0 {
		t.Errorf("Expected empty map, got %d elements", len(gauges))
	}

	storage.UpdateGauge("TestMetric", 123.45)

	gauges = storage.GetAllGauges()
	if len(gauges) != 1 {
		t.Errorf("Expected 1 element, got %d elements", len(gauges))
	}

	if gauges["TestMetric"] != 123.45 {
		t.Errorf("Expected 123.45, got %v", gauges["TestMetric"])
	}
}

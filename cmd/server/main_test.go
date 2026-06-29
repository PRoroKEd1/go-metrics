package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateHandler_Success(t *testing.T) {
	storage := NewMemStorage()

	request := httptest.NewRequest(http.MethodPost, "/update/counter/RandomMetric/111", nil)
	request.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()
	storage.updateHandler(recorder, request)
	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		t.Errorf("Ожидался статус %d, но получен %d", http.StatusOK, result.StatusCode)
	}
}

func TestUpdateHandler_Fail(t *testing.T) {
	storage := NewMemStorage()

	request := httptest.NewRequest(http.MethodGet, "/update/counter/RandomMetric/111", nil)
	request.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()
	storage.updateHandler(recorder, request)
	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Ожидался статус %d, но получен %d", http.StatusMethodNotAllowed, result.StatusCode)
	}
}

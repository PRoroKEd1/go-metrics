package logging

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitializeAndRequestLogger(t *testing.T) {
	if err := Initialize("info"); err != nil {
		t.Fatalf("ошибка инициализации логгера: %v", err)
	}

	if err := Initialize("invalid_level"); err == nil {
		t.Errorf("ожидалась ошибка для невалидного уровня логирования")
	}

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("logged"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("ожидался статус 201, получен %d", rec.Code)
	}
}

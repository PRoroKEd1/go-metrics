package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzipMiddleware(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte(`{"id":"Alloc","type":"gauge"}`))
	zw.Close()

	req := httptest.NewRequest(http.MethodPost, "/update/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ожидался статус 200, получен %d", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("ожидался заголовок Content-Encoding: gzip")
	}

	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("ошибка создания gzip reader: %v", err)
	}
	defer zr.Close()

	uncompressed, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("ошибка чтения распакованного ответа: %v", err)
	}
	if string(uncompressed) != `{"id":"Alloc","type":"gauge"}` {
		t.Errorf("неожиданное тело ответа: %s", string(uncompressed))
	}
}

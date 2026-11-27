package audit

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditor_NotifyAll(t *testing.T) {
	tmpDir := t.TempDir()
	auditFile := filepath.Join(tmpDir, "audit.log")

	var receivedURLCall bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURLCall = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	auditor := NewAuditor()
	auditor.Register(NewFileObserver(auditFile))
	auditor.Register(NewURLObserver(srv.URL))

	auditor.NotifyAll([]string{"Alloc", "PollCount"}, "192.168.0.42:12345")

	data, err := os.ReadFile(auditFile)
	if err != nil {
		t.Fatalf("не удалось прочитать файл аудита: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "192.168.0.42") || !strings.Contains(content, "Alloc") {
		t.Errorf("неожиданное содержимое файла аудита: %s", content)
	}

	if !receivedURLCall {
		t.Errorf("ожидался HTTP-запрос на сервер аудита, но он не пришел")
	}
}

package hash

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSign(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		key      string
		expected string
	}{
		{
			name:     "Обычный текст",
			data:     []byte("hello world"),
			key:      "secret",
			expected: "734cc62f32841568f45715aeb9f4d7891324e6d948e4c6c60c0621cdac48623a",
		},
		{
			name:     "Пустая строка",
			data:     []byte(""),
			key:      "secret",
			expected: "f9e66e179b6747ae54108f82f8ade8b3c25d76fd30afde6c395822c530196169",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sign(tt.data, tt.key)
			if result != tt.expected {
				t.Errorf("ожидалось %s, получено %s", tt.expected, result)
			}
		})
	}
}

func TestHashMiddleware(t *testing.T) {
	key := "test-secret"
	bodyData := []byte("test body")
	validHash := Sign(bodyData, key)

	tests := []struct {
		name           string
		key            string
		headerHash     string
		expectedStatus int
	}{
		{
			name:           "Валидный хеш",
			key:            key,
			headerHash:     validHash,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Невалидный хеш",
			key:            key,
			headerHash:     "wrong-hash",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Отсутствует хеш при наличии ключа",
			key:            key,
			headerHash:     "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Ключ не задан (пропуск)",
			key:            "",
			headerHash:     "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyData))
			if tt.headerHash != "" {
				req.Header.Set("HashSHA256", tt.headerHash)
			}

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			})

			handlerToTest := HashMiddleware(tt.key)(nextHandler)
			recorder := httptest.NewRecorder()
			handlerToTest.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("ожидался статус %d, получен %d", tt.expectedStatus, recorder.Code)
			}
		})
	}
}

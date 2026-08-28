package hash

import (
	"bytes"
	"io"
	"net/http"
)

type hashResponseWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
}

func (hw *hashResponseWriter) Write(b []byte) (int, error) {
	return hw.buf.Write(b)
}

func (hw *hashResponseWriter) WriteHeader(statusCode int) {
	hw.statusCode = statusCode
}

func HashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			clientHash := r.Header.Get("HashSHA256")
			if clientHash != "" && r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Failed to read body", http.StatusInternalServerError)
					return
				}

				expectedHash := Sign(bodyBytes, key)
				if clientHash != expectedHash {
					http.Error(w, "Invalid HashSHA256", http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			hw := &hashResponseWriter{
				ResponseWriter: w,
				buf:            &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(hw, r)

			responseBytes := hw.buf.Bytes()
			responseHash := Sign(responseBytes, key)

			w.Header().Set("HashSHA256", responseHash)
			w.WriteHeader(hw.statusCode)
			w.Write(responseBytes)
		})
	}
}

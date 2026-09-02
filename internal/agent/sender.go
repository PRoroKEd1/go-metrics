package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/PRoroKEd1/go-metrics/internal/hash"
	models "github.com/PRoroKEd1/go-metrics/internal/model"
)

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestSpeed)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации сжатия: %v", err)
	}
	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("ошибка записи данных для сжатия: %v", err)
	}
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("ошибка закрытия gzip writer: %v", err)
	}
	return b.Bytes(), nil
}

type retryTransport struct {
	base http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	delays := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	for attempt := 0; attempt <= len(delays); attempt++ {
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}

		resp, err := t.base.RoundTrip(req)
		if err == nil {
			return resp, nil
		}
		if attempt == len(delays) {
			return nil, err
		}

		log.Println("Ошибка HTTP-запроса, повтор:", err)
		time.Sleep(delays[attempt])
	}
	return nil, fmt.Errorf("не удалось выполнить HTTP-запрос")
}

func sendMetrics(addr string, metrics []models.Metrics, key string) {
	if len(metrics) == 0 {
		return
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		log.Println("Ошибка сериализации:", err)
		return
	}

	compressedBody, err := compress(body)
	if err != nil {
		log.Println("Ошибка сжатия:", err)
		return
	}

	url := fmt.Sprintf("http://%s/updates/", addr)

	req, err := http.NewRequest("POST", url, bytes.NewReader(compressedBody))
	if err != nil {
		log.Println("Ошибка создания запроса:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	client := &http.Client{
		Transport: &retryTransport{
			base: http.DefaultTransport,
		},
	}

	if key != "" {
		hashValue := hash.Sign(compressedBody, key)
		req.Header.Set("HashSHA256", hashValue)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Ошибка отправки батча:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Статус ответа:", resp.Status)
}

func worker(id int, jobs <-chan []models.Metrics, addr, key string) {
	for metrics := range jobs {
		log.Printf("Воркер %d начал отправку %d метрик\n", id, len(metrics))
		sendMetrics(addr, metrics, key)
	}
}

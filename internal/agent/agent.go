package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"time"

	"github.com/PRoroKEd1/go-metrics/internal/hash"
	models "github.com/PRoroKEd1/go-metrics/internal/model"
)

type MemStorage struct {
	gaugeMetrics   map[string]float64
	counterMetrics map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gaugeMetrics:   make(map[string]float64),
		counterMetrics: make(map[string]int64),
	}
}

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

	req, err := http.NewRequest(
		"POST",
		url,
		bytes.NewReader(compressedBody),
	)
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

func Run(addr string, pollInterval int, reportInterval int, key string) {
	store := NewMemStorage()
	pollDuration := time.Duration(pollInterval) * time.Second
	reportDuration := time.Duration(reportInterval) * time.Second
	var memStats runtime.MemStats
	ticks := 0
	for {
		ticks++
		runtime.ReadMemStats(&memStats)
		store.gaugeMetrics["Alloc"] = float64(memStats.Alloc)
		store.gaugeMetrics["BuckHashSys"] = float64(memStats.BuckHashSys)
		store.gaugeMetrics["Frees"] = float64(memStats.Frees)
		store.gaugeMetrics["GCCPUFraction"] = float64(memStats.GCCPUFraction)
		store.gaugeMetrics["GCSys"] = float64(memStats.GCSys)
		store.gaugeMetrics["HeapAlloc"] = float64(memStats.HeapAlloc)
		store.gaugeMetrics["HeapIdle"] = float64(memStats.HeapIdle)
		store.gaugeMetrics["HeapInuse"] = float64(memStats.HeapInuse)
		store.gaugeMetrics["HeapObjects"] = float64(memStats.HeapObjects)
		store.gaugeMetrics["HeapReleased"] = float64(memStats.HeapReleased)
		store.gaugeMetrics["HeapSys"] = float64(memStats.HeapSys)
		store.gaugeMetrics["LastGC"] = float64(memStats.LastGC)
		store.gaugeMetrics["Lookups"] = float64(memStats.Lookups)
		store.gaugeMetrics["MCacheInuse"] = float64(memStats.MCacheInuse)
		store.gaugeMetrics["MCacheSys"] = float64(memStats.MCacheSys)
		store.gaugeMetrics["MSpanInuse"] = float64(memStats.MSpanInuse)
		store.gaugeMetrics["MSpanSys"] = float64(memStats.MSpanSys)
		store.gaugeMetrics["Mallocs"] = float64(memStats.Mallocs)
		store.gaugeMetrics["NextGC"] = float64(memStats.NextGC)
		store.gaugeMetrics["NumForcedGC"] = float64(memStats.NumForcedGC)
		store.gaugeMetrics["NumGC"] = float64(memStats.NumGC)
		store.gaugeMetrics["OtherSys"] = float64(memStats.OtherSys)
		store.gaugeMetrics["PauseTotalNs"] = float64(memStats.PauseTotalNs)
		store.gaugeMetrics["StackInuse"] = float64(memStats.StackInuse)
		store.gaugeMetrics["StackSys"] = float64(memStats.StackSys)
		store.gaugeMetrics["Sys"] = float64(memStats.Sys)
		store.gaugeMetrics["TotalAlloc"] = float64(memStats.TotalAlloc)

		if ticks == int(reportDuration/pollDuration) {
			ticks = 0

			metrics := make([]models.Metrics, 0, len(store.gaugeMetrics)+len(store.counterMetrics))

			for name, value := range store.gaugeMetrics {
				v := value

				metrics = append(metrics, models.Metrics{
					ID:    name,
					MType: "gauge",
					Value: &v,
				})
			}

			for name, delta := range store.counterMetrics {
				d := delta

				metrics = append(metrics, models.Metrics{
					ID:    name,
					MType: "counter",
					Delta: &d,
				})
			}

			sendMetrics(addr, metrics, key)
			store.counterMetrics["PollCount"] = 0
		}
		store.counterMetrics["PollCount"]++
		store.gaugeMetrics["RandomValue"] = rand.Float64()
		time.Sleep(pollDuration)

	}
}

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
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/PRoroKEd1/go-metrics/internal/hash"
	models "github.com/PRoroKEd1/go-metrics/internal/model"
)

type MemStorage struct {
	mu             sync.RWMutex
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

func worker(id int, jobs <-chan []models.Metrics, addr, key string) {
	for metrics := range jobs {
		log.Printf("Воркер %d начал отправку %d метрик\n", id, len(metrics))
		sendMetrics(addr, metrics, key)
	}
}

func (ms *MemStorage) SetGauge(name string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gaugeMetrics[name] = value
}

func (ms *MemStorage) IncrementCounter(name string, value int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counterMetrics[name] += value
}

func (ms *MemStorage) ResetCounter(name string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counterMetrics[name] = 0
}

func (ms *MemStorage) GetMetricsSnapshot() []models.Metrics {
	ms.mu.RLock() // RLock - блокировка только для чтения
	defer ms.mu.RUnlock()

	metrics := make([]models.Metrics, 0, len(ms.gaugeMetrics)+len(ms.counterMetrics))

	for name, value := range ms.gaugeMetrics {
		v := value
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}

	for name, delta := range ms.counterMetrics {
		d := delta
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &d,
		})
	}
	return metrics
}

func collectExtraMetrics(store *MemStorage, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		v, err := mem.VirtualMemory()
		if err == nil {
			store.SetGauge("TotalMemory", float64(v.Total))
			store.SetGauge("FreeMemory", float64(v.Free))
		}

		c, err := cpu.Percent(0, true)
		if err == nil {
			for i, percent := range c {
				metricName := fmt.Sprintf("CPUutilization%d", i+1)
				store.SetGauge(metricName, percent)
			}
		}
	}
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

func collectRuntimeMetrics(store *MemStorage, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	var memStats runtime.MemStats

	for range ticker.C {
		runtime.ReadMemStats(&memStats)

		store.SetGauge("Alloc", float64(memStats.Alloc))
		store.SetGauge("BuckHashSys", float64(memStats.BuckHashSys))
		store.SetGauge("Frees", float64(memStats.Frees))
		store.SetGauge("GCCPUFraction", float64(memStats.GCCPUFraction))
		store.SetGauge("GCSys", float64(memStats.GCSys))
		store.SetGauge("HeapAlloc", float64(memStats.HeapAlloc))
		store.SetGauge("HeapIdle", float64(memStats.HeapIdle))
		store.SetGauge("HeapInuse", float64(memStats.HeapInuse))
		store.SetGauge("HeapObjects", float64(memStats.HeapObjects))
		store.SetGauge("HeapReleased", float64(memStats.HeapReleased))
		store.SetGauge("HeapSys", float64(memStats.HeapSys))
		store.SetGauge("LastGC", float64(memStats.LastGC))
		store.SetGauge("Lookups", float64(memStats.Lookups))
		store.SetGauge("MCacheInuse", float64(memStats.MCacheInuse))
		store.SetGauge("MCacheSys", float64(memStats.MCacheSys))
		store.SetGauge("MSpanInuse", float64(memStats.MSpanInuse))
		store.SetGauge("MSpanSys", float64(memStats.MSpanSys))
		store.SetGauge("Mallocs", float64(memStats.Mallocs))
		store.SetGauge("NextGC", float64(memStats.NextGC))
		store.SetGauge("NumForcedGC", float64(memStats.NumForcedGC))
		store.SetGauge("NumGC", float64(memStats.NumGC))
		store.SetGauge("OtherSys", float64(memStats.OtherSys))
		store.SetGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
		store.SetGauge("StackInuse", float64(memStats.StackInuse))
		store.SetGauge("StackSys", float64(memStats.StackSys))
		store.SetGauge("Sys", float64(memStats.Sys))
		store.SetGauge("TotalAlloc", float64(memStats.TotalAlloc))

		store.SetGauge("RandomValue", rand.Float64())
		store.IncrementCounter("PollCount", 1)
	}
}

func Run(addr string, pollInterval int, reportInterval int, key string, rateLimit int) {
	store := NewMemStorage()
	pollDuration := time.Duration(pollInterval) * time.Second
	reportDuration := time.Duration(reportInterval) * time.Second
	go collectRuntimeMetrics(store, pollDuration)
	go collectExtraMetrics(store, pollDuration)

	jobs := make(chan []models.Metrics, rateLimit)

	for i := 1; i <= rateLimit; i++ {
		go worker(i, jobs, addr, key)
	}

	ticker := time.NewTicker(reportDuration)
	defer ticker.Stop()

	for range ticker.C {
		snapshot := store.GetMetricsSnapshot()
		store.ResetCounter("PollCount")
		jobs <- snapshot
	}
}

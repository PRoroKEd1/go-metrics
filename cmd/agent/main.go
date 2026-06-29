package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"
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

func main() {
	storage := NewMemStorage()
	pollInterval := 2 * time.Second
	reportInterval := 10 * time.Second
	fmt.Println("Агент запущен...")
	var memStats runtime.MemStats

	ticks := 0
	for {
		ticks++
		runtime.ReadMemStats(&memStats)
		storage.gaugeMetrics["Alloc"] = float64(memStats.Alloc)
		storage.gaugeMetrics["BuckHashSys"] = float64(memStats.BuckHashSys)
		storage.gaugeMetrics["Frees"] = float64(memStats.Frees)
		storage.gaugeMetrics["GCCPUFraction"] = float64(memStats.GCCPUFraction)
		storage.gaugeMetrics["GCSys"] = float64(memStats.GCSys)
		storage.gaugeMetrics["HeapAlloc"] = float64(memStats.HeapAlloc)
		storage.gaugeMetrics["HeapIdle"] = float64(memStats.HeapIdle)
		storage.gaugeMetrics["HeapInuse"] = float64(memStats.HeapInuse)
		storage.gaugeMetrics["HeapObjects"] = float64(memStats.HeapObjects)
		storage.gaugeMetrics["HeapReleased"] = float64(memStats.HeapReleased)
		storage.gaugeMetrics["HeapSys"] = float64(memStats.HeapSys)
		storage.gaugeMetrics["LastGC"] = float64(memStats.LastGC)
		storage.gaugeMetrics["Lookups"] = float64(memStats.Lookups)
		storage.gaugeMetrics["MCacheInuse"] = float64(memStats.MCacheInuse)
		storage.gaugeMetrics["MCacheSys"] = float64(memStats.MCacheSys)
		storage.gaugeMetrics["MSpanInuse"] = float64(memStats.MSpanInuse)
		storage.gaugeMetrics["MSpanSys"] = float64(memStats.MSpanSys)
		storage.gaugeMetrics["Mallocs"] = float64(memStats.Mallocs)
		storage.gaugeMetrics["NextGC"] = float64(memStats.NextGC)
		storage.gaugeMetrics["NumForcedGC"] = float64(memStats.NumForcedGC)
		storage.gaugeMetrics["NumGC"] = float64(memStats.NumGC)
		storage.gaugeMetrics["OtherSys"] = float64(memStats.OtherSys)
		storage.gaugeMetrics["PauseTotalNs"] = float64(memStats.PauseTotalNs)
		storage.gaugeMetrics["StackInuse"] = float64(memStats.StackInuse)
		storage.gaugeMetrics["StackSys"] = float64(memStats.StackSys)
		storage.gaugeMetrics["Sys"] = float64(memStats.Sys)
		storage.gaugeMetrics["TotalAlloc"] = float64(memStats.TotalAlloc)

		if ticks == int(reportInterval/pollInterval) {

			ticks = 0

			for name, value := range storage.gaugeMetrics {
				url := fmt.Sprintf("http://localhost:8080/update/gauge/%s/%f", name, value)

				resp, err := http.Post(
					url,
					"text/plain",
					nil,
				)
				if err != nil {
					fmt.Println("Ошибка запроса:", err)
					continue
				}
				resp.Body.Close()

				fmt.Println("Статус ответа:", resp.Status)
			}

			for name, value := range storage.counterMetrics {
				url := fmt.Sprintf("http://localhost:8080/update/counter/%s/%d", name, value)

				resp, err := http.Post(
					url,
					"text/plain",
					nil,
				)
				if err != nil {
					fmt.Println("Ошибка запроса:", err)
					continue
				}
				resp.Body.Close()

				fmt.Println("Статус ответа:", resp.Status)
			}
		}
		storage.counterMetrics["PollCount"]++
		storage.gaugeMetrics["RandomValue"] = rand.Float64()
		time.Sleep(pollInterval)
	}
}

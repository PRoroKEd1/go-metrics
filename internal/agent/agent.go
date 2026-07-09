package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
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

func Run(addr string, pollInterval int, reportInterval int) {
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

			for name, value := range store.gaugeMetrics {
				url := fmt.Sprintf("http://%s/update/gauge/%s/%f", addr, name, value)

				resp, err := http.Post(
					url,
					"text/plain",
					nil,
				)
				if err != nil {
					log.Println("Ошибка запроса:", err)
					continue
				}
				resp.Body.Close()

				log.Println("Статус ответа:", resp.Status)
			}

			for name, value := range store.counterMetrics {
				url := fmt.Sprintf("http://%s/update/counter/%s/%d", addr, name, value)

				resp, err := http.Post(
					url,
					"text/plain",
					nil,
				)
				if err != nil {
					log.Println("Ошибка запроса:", err)
					continue
				}
				resp.Body.Close()

				log.Println("Статус ответа:", resp.Status)
			}
			store.counterMetrics["PollCount"] = 0
		}
		store.counterMetrics["PollCount"]++
		store.gaugeMetrics["RandomValue"] = rand.Float64()
		time.Sleep(pollDuration)

	}
}

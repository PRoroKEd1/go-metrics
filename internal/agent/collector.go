package agent

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

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

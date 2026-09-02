package agent

import (
	"time"

	models "github.com/PRoroKEd1/go-metrics/internal/model"
)

type Config struct {
	Addr           string
	PollInterval   int
	ReportInterval int
	Key            string
	RateLimit      int
}

func Run(cfg Config) {
	store := NewMemStorage()
	pollDuration := time.Duration(cfg.PollInterval) * time.Second
	reportDuration := time.Duration(cfg.ReportInterval) * time.Second

	go collectRuntimeMetrics(store, pollDuration)
	go collectExtraMetrics(store, pollDuration)

	jobs := make(chan []models.Metrics, cfg.RateLimit)

	for i := 1; i <= cfg.RateLimit; i++ {
		go worker(i, jobs, cfg.Addr, cfg.Key)
	}

	ticker := time.NewTicker(reportDuration)
	defer ticker.Stop()

	for range ticker.C {
		snapshot := store.GetMetricsSnapshotAndReset()
		jobs <- snapshot
	}
}

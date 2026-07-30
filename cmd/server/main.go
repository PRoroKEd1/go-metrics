package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/PRoroKEd1/go-metrics/internal/compress"
	"github.com/PRoroKEd1/go-metrics/internal/handler"
	"github.com/PRoroKEd1/go-metrics/internal/logging"
	"github.com/PRoroKEd1/go-metrics/internal/storage"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
)

type Config struct {
	Addr            string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

func parseConfig(args []string) (Config, error) {
	var cfg Config
	f := flag.NewFlagSet("сервер", flag.ContinueOnError)

	f.StringVar(&cfg.Addr, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	f.IntVar(&cfg.StoreInterval, "i", 300, "Интервал")
	f.BoolVar(&cfg.Restore, "r", true, "Восстанавливать ли данные")
	f.StringVar(&cfg.FileStoragePath, "f", "/tmp/metrics-db.json", "Путь файла")

	if err := f.Parse(args); err != nil {
		return cfg, err
	}

	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func main() {

	if err := logging.Initialize("info"); err != nil {
		slog.Error("Ошибка инициализации логгера", "err", err)
		os.Exit(1)
	}

	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		slog.Error("Ошибка инициализации конфига", "err", err)
		os.Exit(1)
	}

	store := storage.NewMemStorage()

	h := handler.NewHandler(store, cfg.StoreInterval == 0, cfg.FileStoragePath)

	r := chi.NewRouter()

	r.Use(logging.RequestLogger)

	r.Use(compress.GzipMiddleware)

	if cfg.Restore {
		if err := store.RestoreFromFile(cfg.FileStoragePath); err != nil {
			slog.Error("Ошибка загрузки метрик из файла", "err", err)
		} else {
			slog.Info("Метрики успешно загружены из файла", "path", cfg.FileStoragePath)
		}
	}

	if cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := store.SaveToFile(cfg.FileStoragePath); err != nil {
					slog.Error("Ошибка сохранения метрик", "err", err)
				}
			}
		}()
	}

	r.Post("/update/", h.UpdateJSONHandler)
	r.Post("/value/", h.ValueJSONHandler)

	// Старые эндпоинты
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)

	r.Get("/", h.GetAllMetricsHandler)

	slog.Info("Сервер запущен", "address", cfg.Addr)

	if err := http.ListenAndServe(cfg.Addr, r); err != nil {
		slog.Error("Ошибка при запуске сервера", "err", err)
		os.Exit(1)
	}
}

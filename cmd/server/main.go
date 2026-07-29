package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/PRoroKEd1/go-metrics/internal/compress"
	"github.com/PRoroKEd1/go-metrics/internal/handler"
	"github.com/PRoroKEd1/go-metrics/internal/logging"
	"github.com/PRoroKEd1/go-metrics/internal/storage"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
)

type Config struct {
	Addr string `env:"ADDRESS"`
}

func parseConfig(args []string) (Config, error) {
	var cfg Config
	f := flag.NewFlagSet("сервер", flag.ContinueOnError)

	f.StringVar(&cfg.Addr, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")

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

	store := storage.NewMemStorage()
	h := handler.NewHandler(store)

	r := chi.NewRouter()

	r.Use(logging.RequestLogger)

	r.Use(compress.GzipMiddleware)

	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		slog.Error("Ошибка инициализации конфига", "err", err)
		os.Exit(1)
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

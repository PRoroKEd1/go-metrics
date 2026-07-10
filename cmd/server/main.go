package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/PRoroKEd1/go-metrics/internal/handler"
	"github.com/PRoroKEd1/go-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Config struct {
	Addr string
}

func parseFlags(args []string) (Config, error) {
	var cfg Config
	f := flag.NewFlagSet("сервер", flag.ContinueOnError)

	f.StringVar(&cfg.Addr, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")

	err := f.Parse(args)
	return cfg, err
}

func main() {
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		log.Fatalf("Ошибка парсинга флагов: %v", err)
	}

	store := storage.NewMemStorage()
	h := handler.NewHandler(store)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)
	r.Get("/", h.GetAllMetricsHandler)

	log.Printf("Сервер запущен и слушает порт %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, r); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}

}

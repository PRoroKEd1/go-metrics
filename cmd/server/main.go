package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PRoroKEd1/go-metrics/internal/compress"
	"github.com/PRoroKEd1/go-metrics/internal/handler"
	"github.com/PRoroKEd1/go-metrics/internal/hash"
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
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
}

func parseConfig(args []string) (Config, error) {
	var cfg Config
	f := flag.NewFlagSet("сервер", flag.ContinueOnError)

	f.StringVar(&cfg.Addr, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	f.IntVar(&cfg.StoreInterval, "i", 300, "Интервал")
	f.BoolVar(&cfg.Restore, "r", true, "Восстанавливать ли данные")
	f.StringVar(&cfg.FileStoragePath, "f", "/tmp/metrics-db.json", "Путь файла")
	f.StringVar(&cfg.DatabaseDSN, "d", "", "Подключения к ДБ")
	f.StringVar(&cfg.Key, "k", "", "Ключ для подписи данных")

	if err := f.Parse(args); err != nil {
		return cfg, err
	}

	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}

	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = "/tmp/metrics-db.json"
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

	var db *sql.DB
	var store handler.Storage
	var fileStore *storage.MemStorage

	if cfg.DatabaseDSN != "" {
		var err error

		store, db, err = storage.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			slog.Error("Не удалось подключиться к БД", "err", err)
			os.Exit(1)
		}

		slog.Info("База данных подключена")
	} else {
		filePath := cfg.FileStoragePath

		if filePath != "" {
			fileStore, err = storage.NewFileStorage(filePath, cfg.Restore)
			if err != nil {
				slog.Error("Не удалось создать файловое хранилище", "err", err)
				os.Exit(1)
			}

			store = fileStore
			slog.Info("Используется файловое хранилище", "path", filePath)
		} else {
			store = storage.NewMemStorage()
			slog.Info("Используется хранилище в памяти")
		}
	}

	h := handler.NewHandler(store, cfg.StoreInterval == 0, cfg.FileStoragePath, db)

	r := chi.NewRouter()

	r.Use(logging.RequestLogger)
	r.Use(hash.HashMiddleware(cfg.Key))
	r.Use(compress.GzipMiddleware)

	if cfg.Restore && cfg.DatabaseDSN == "" {
		if memStore, ok := store.(*storage.MemStorage); ok {
			if err := memStore.RestoreFromFile(cfg.FileStoragePath); err != nil {
				slog.Error("Ошибка загрузки метрик из файла", "err", err)
			} else {
				slog.Info("Метрики успешно загружены из файла", "path", cfg.FileStoragePath)
			}
		}
	}

	if cfg.DatabaseDSN == "" && cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := fileStore.SaveToFile(cfg.FileStoragePath); err != nil {
					slog.Error("Ошибка сохранения метрик в файл", "err", err)
				}
			}
		}()
	}

	r.Post("/update/", h.UpdateJSONHandler)
	r.Post("/updates/", h.UpdatesJSONHandler)
	r.Post("/value/", h.ValueJSONHandler)

	// Старые эндпоинты
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)

	r.Get("/ping", h.PingDBHandler)
	r.Get("/", h.GetAllMetricsHandler)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: r,
	}
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

		<-sigint
		slog.Info("Получен сигнал завершения, останавливаем сервер...")

		if err := srv.Shutdown(context.Background()); err != nil {
			slog.Error("Ошибка при остановке сервера", "err", err)
		}
		if cfg.DatabaseDSN == "" && cfg.FileStoragePath != "" {
			if err := fileStore.SaveToFile(cfg.FileStoragePath); err != nil {
				slog.Error("Ошибка финального сохранения метрик", "err", err)
			} else {
				slog.Info("Финальное сохранение прошло успешно")
			}
		}

		close(idleConnsClosed)
	}()

	slog.Info("Сервер запущен", "address", cfg.Addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("Ошибка при запуске сервера", "err", err)
		os.Exit(1)
	}

	if db != nil {
		db.Close()
		slog.Info("Соединение с БД закрыто")
	}

	<-idleConnsClosed
	slog.Info("Сервер успешно завершил работу")
}

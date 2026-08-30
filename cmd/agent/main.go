package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"

	"github.com/PRoroKEd1/go-metrics/internal/agent"
)

type Config struct {
	Addr           string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func parseConfig(args []string) (Config, error) {
	var cfg Config
	f := flag.NewFlagSet("агент", flag.ContinueOnError)

	f.StringVar(&cfg.Addr, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	f.IntVar(&cfg.PollInterval, "p", 2, "частота опроса")
	f.IntVar(&cfg.ReportInterval, "r", 10, "частота отправки")
	f.StringVar(&cfg.Key, "k", "", "Ключ для подписи данных")
	f.IntVar(&cfg.RateLimit, "l", 1, "Количество одновременных запросов")

	if err := f.Parse(args); err != nil {
		return cfg, err
	}

	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		slog.Error("Ошибка парсинга конфига", "err", err)
		os.Exit(1)
	}

	if cfg.PollInterval <= 0 {
		slog.Error("Ошибка. Неподходящее число для PollInterval. Должно быть >= 1")
		os.Exit(1)
	}

	agentCfg := agent.Config{
		Addr:           cfg.Addr,
		PollInterval:   cfg.PollInterval,
		ReportInterval: cfg.ReportInterval,
		Key:            cfg.Key,
		RateLimit:      cfg.RateLimit,
	}

	slog.Info("Агент запущен...", "addr", cfg.Addr)
	agent.Run(agentCfg)
}

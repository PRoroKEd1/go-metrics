package main

import (
	"flag"
	"log"
	"os"

	"github.com/PRoroKEd1/go-metrics/internal/agent"
)

type Config struct {
	Addr           string
	PollInterval   int
	ReportInterval int
}

func parseFlags(args []string) (Config, error) {
	var cfg Config
	f := flag.NewFlagSet("агент", flag.ContinueOnError)

	f.StringVar(&cfg.Addr, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	f.IntVar(&cfg.PollInterval, "p", 2, "частота опроса")
	f.IntVar(&cfg.ReportInterval, "r", 10, "частота отправки")

	err := f.Parse(args)
	return cfg, err
}

func main() {
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		log.Fatalf("Ошибка парсинга флагов: %v", err)
	}

	if cfg.PollInterval <= 0 {
		log.Fatalf("Ошибка. Неподходящее число. Число не должно быть меньше 1")
	}
	log.Println("Агент запущен...")
	agent.Run(cfg.Addr, cfg.PollInterval, cfg.ReportInterval)
}

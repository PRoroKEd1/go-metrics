package main

import (
	"os"
	"testing"
)

func TestParseConfig_Server(t *testing.T) {
	os.Setenv("ADDRESS", "127.0.0.1:9090")
	os.Setenv("STORE_INTERVAL", "10")
	defer os.Unsetenv("ADDRESS")
	defer os.Unsetenv("STORE_INTERVAL")

	args := []string{"-a", "localhost:8080"}
	cfg, err := parseConfig(args)

	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9090" {
		t.Errorf("адрес: ожидали 127.0.0.1:9090, получили %s", cfg.Addr)
	}
	if cfg.StoreInterval != 10 {
		t.Errorf("интервал: ожидали 10, получили %d", cfg.StoreInterval)
	}
}

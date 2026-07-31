package main

import (
	"os"
	"testing"
)

func TestParseConfig_EnvPriority(t *testing.T) {
	os.Setenv("ADDRESS", "127.0.0.1:9090")
	defer os.Unsetenv("ADDRESS")

	args := []string{"-a", "localhost:8080"}
	cfg, err := parseConfig(args)

	if err != nil {
		t.Fatalf("ожидалось отсутствие ошибки, получили: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9090" {
		t.Errorf("ожидался адрес 127.0.0.1:9090, получили: %s", cfg.Addr)
	}
}

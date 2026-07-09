package main

import (
	"testing"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedAddr string
	}{
		{
			name:         "Дефолтные значения",
			args:         []string{},
			expectedAddr: "localhost:8080",
		},
		{
			name:         "Кастомный адресс",
			args:         []string{"-a", ":9090"},
			expectedAddr: ":9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseFlags(tt.args)
			if err != nil {
				t.Fatalf("Неожиданная ошибка: %v", err)
			}

			if cfg.Addr != tt.expectedAddr {
				t.Errorf("Ожидался адрес %s, получен %s", tt.expectedAddr, cfg.Addr)
			}
		})
	}
}

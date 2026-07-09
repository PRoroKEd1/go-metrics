package main

import "testing"

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig Config
	}{
		{
			name: "Дефолтные значения",
			args: []string{},
			expectedConfig: Config{
				Addr:           "localhost:8080",
				PollInterval:   2,
				ReportInterval: 10,
			},
		},
		{
			name: "Все кастомные значения",
			args: []string{"-a", ":9090", "-p", "5", "-r", "20"},
			expectedConfig: Config{
				Addr:           ":9090",
				PollInterval:   5,
				ReportInterval: 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseFlags(tt.args)
			if err != nil {
				t.Fatalf("Неожиданная ошибка: %v", err)
			}

			if cfg != tt.expectedConfig {
				t.Errorf("Ожидался конфиг %+v, получен %+v", tt.expectedConfig, cfg.Addr)
			}
		})
	}
}

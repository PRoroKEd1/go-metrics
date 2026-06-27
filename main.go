package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func main() {
	mux := http.NewServeMux()
	addr := ":8080"
	storage := NewMemStorage()
	mux.HandleFunc("/update/", storage.updateHandler)

	log.Printf("Сервер запущен и слушает порт %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)

	}

}

func (m *MemStorage) updateHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	path := r.URL.Path
	if len(path) == 0 || path[0] == '/' {
		path = path[1:]
	}
	parts := strings.Split(path, "/")

	if len(parts) != 4 || parts[0] != "update" {
		http.Error(w, "Invalid path format", http.StatusNotFound)
		return
	}

	metricType := parts[1]
	metricName := parts[2]
	metricValueStr := parts[3]

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValueStr, 64)
		if err != nil {
			http.Error(w, "Invalid gauge value format", http.StatusBadRequest)
			return
		}

		m.UpdateGauge(metricName, value)

	case "counter":
		value, err := strconv.ParseInt(metricValueStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value format", http.StatusBadRequest)
			return
		}
		m.UpdateCounter(metricName, value)

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "updated")
}

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage() MemStorage {
	return MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.gauge[name] = value
}

func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.counter[name] += value
}

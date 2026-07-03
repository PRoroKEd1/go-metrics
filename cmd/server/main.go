package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"html/template"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	addr := ":8080"
	storage := NewMemStorage()
	r.Post("/update/{type}/{name}/{value}", storage.updateHandler)
	r.Get("/value/{type}/{name}", storage.getMetricHandler)

	r.Get("/", storage.getAllMetricsHandler)

	log.Printf("Сервер запущен и слушает порт %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}

func (m *MemStorage) updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValueStr := chi.URLParam(r, "value")

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

func (m *MemStorage) GetGauge(name string) (float64, bool) {
	val, ok := m.gauge[name]
	return val, ok
}

func (m *MemStorage) GetCounter(name string) (int64, bool) {
	val, ok := m.counter[name]
	return val, ok
}

func (m *MemStorage) GetAllGauges() map[string]float64 {
	return m.gauge
}

func (m *MemStorage) GetAllCounters() map[string]int64 {
	return m.counter
}

func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.gauge[name] = value
}

func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.counter[name] += value
}

func (m *MemStorage) getMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	switch metricType {
	case "gauge":
		val, ok := m.GetGauge(metricName)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, "%v", val)

	case "counter":
		val, ok := m.GetCounter(metricName)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, "%v", val)

	default:
		http.Error(w, "Unknown metric type", http.StatusNotFound)
		return
	}
}

func (m *MemStorage) getAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	gauges := m.GetAllGauges()
	counters := m.GetAllCounters()

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   gauges,
		Counters: counters,
	}

	tmplText := `
<!DOCTYPE html>
<html>
<head>
	<title>Метрики</title>
</head>
<body>
	<h1>Gauge метрики</h1>
	<ul>
		{{range $name, $value := .Gauges}}
			<li>{{$name}}: {{$value}}</li>
		{{end}}
	</ul>

	<h1>Counter метрики</h1>
	<ul>
		{{range $name, $value := .Counters}}
			<li>{{$name}}: {{$value}}</li>
		{{end}}
	</ul>
</body>
</html>
`

	tmpl, err := template.New("metrics").Parse(tmplText)
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)

}

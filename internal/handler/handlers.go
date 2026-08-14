package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	models "github.com/PRoroKEd1/go-metrics/internal/model"
	"github.com/PRoroKEd1/go-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	storage  storage.Storage
	tmpl     *template.Template
	syncSave bool
	filePath string
	db       *sql.DB
}

func NewHandler(storage storage.Storage, syncSave bool, filePath string, db *sql.DB) *Handler {
	tmplText := `
<!DOCTYPE html>
<html>
<head><title>Метрики</title></head>
<body>
	<h1>Gauge метрики</h1>
	<ul>
		{{range $name, $value := .Gauges}}<li>{{$name}}: {{$value}}</li>{{end}}
	</ul>
	<h1>Counter метрики</h1>
	<ul>
		{{range $name, $value := .Counters}}<li>{{$name}}: {{$value}}</li>{{end}}
	</ul>
</body>
</html>`
	tmpl := template.Must(template.New("metrics").Parse(tmplText))

	return &Handler{
		storage:  storage,
		tmpl:     tmpl,
		syncSave: syncSave,
		filePath: filePath,
		db:       db,
	}
}

func (h *Handler) PingDBHandler(w http.ResponseWriter, r *http.Request) {

	if h.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
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
		h.storage.UpdateGauge(metricName, value)

	case "counter":
		value, err := strconv.ParseInt(metricValueStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value format", http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(metricName, value)

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	if h.syncSave {
		h.storage.SaveToFile(h.filePath)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "updated")
}

func (h *Handler) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	switch metricType {
	case "gauge":
		val, ok := h.storage.GetGauge(metricName)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		strVal := strconv.FormatFloat(val, 'f', -1, 64)
		w.Write([]byte(strVal))

	case "counter":
		val, ok := h.storage.GetCounter(metricName)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		strVal := strconv.FormatInt(val, 10)
		w.Write([]byte(strVal))

	default:
		http.Error(w, "Unknown metric type", http.StatusNotFound)
		return
	}
}

func (h *Handler) GetAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   gauges,
		Counters: counters,
	}

	var buf bytes.Buffer
	if err := h.tmpl.Execute(&buf, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func (h *Handler) UpdateJSONHandler(w http.ResponseWriter, r *http.Request) {
	var req models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.MType {
	case "gauge":
		if req.Value == nil {
			http.Error(w, "Value is required for gauge", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(req.ID, *req.Value)

	case "counter":
		if req.Delta == nil {
			http.Error(w, "Delta is required for counter", http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(req.ID, *req.Delta)

		newVal, _ := h.storage.GetCounter(req.ID)
		req.Delta = &newVal

	default:
		http.Error(w, "Unknown metric type", http.StatusBadRequest)
		return
	}

	if h.syncSave {
		h.storage.SaveToFile(h.filePath)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

func (h *Handler) ValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	var req models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.MType {
	case "gauge":
		val, ok := h.storage.GetGauge(req.ID)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		req.Value = &val

	case "counter":

		val, ok := h.storage.GetCounter(req.ID)
		if !ok {
			http.Error(w, "Counter not found", http.StatusNotFound)
			return
		}
		req.Delta = &val

	default:
		http.Error(w, "Unknown metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

func (h *Handler) UpdatesJSONHandler(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				http.Error(w, "Value is required for gauge", http.StatusBadRequest)
				return
			}

			h.storage.UpdateGauge(metric.ID, *metric.Value)

		case models.Counter:
			if metric.Delta == nil {
				http.Error(w, "Delta is required for counter", http.StatusBadRequest)
				return
			}

			h.storage.UpdateCounter(metric.ID, *metric.Delta)

		default:
			http.Error(w, "Unknown metric type", http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(metrics)
}

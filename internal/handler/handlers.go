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

	"github.com/PRoroKEd1/go-metrics/internal/audit"
	models "github.com/PRoroKEd1/go-metrics/internal/model"
	"github.com/go-chi/chi/v5"
)

// Storage определяет интерфейс хранилища метрик.
type Storage interface {
	GetGauge(ctx context.Context, name string) (float64, bool)
	GetCounter(ctx context.Context, name string) (int64, bool)
	GetAllGauges(ctx context.Context) map[string]float64
	GetAllCounters(ctx context.Context) map[string]int64
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, value int64) error
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
}

// Handler обрабатывает HTTP-запросы для работы с метриками.
type Handler struct {
	storage  Storage
	tmpl     *template.Template
	syncSave bool
	filePath string
	db       *sql.DB
	auditor  *audit.Auditor
}

// NewHandler создает и инициализирует новый экземпляр Handler.
func NewHandler(storage Storage, syncSave bool, filePath string, db *sql.DB, auditor *audit.Auditor) *Handler {
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
		auditor:  auditor,
	}
}

// PingDBHandler проверяет соединение с базой данных PostgreSQL.
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

// UpdateMetricHandler обрабатывает обновление метрики через URL-параметры.
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

		if err := h.storage.UpdateGauge(r.Context(), metricName, value); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	case "counter":
		value, err := strconv.ParseInt(metricValueStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value format", http.StatusBadRequest)
			return
		}

		if err := h.storage.UpdateCounter(r.Context(), metricName, value); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}
	if h.auditor != nil {
		h.auditor.NotifyAll([]string{metricName}, r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "updated")
}

// GetMetricHandler возвращает текущее значение метрики в текстовом формате.
func (h *Handler) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	switch metricType {
	case "gauge":
		val, ok := h.storage.GetGauge(r.Context(), metricName)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		strVal := strconv.FormatFloat(val, 'f', -1, 64)
		w.Write([]byte(strVal))

	case "counter":
		val, ok := h.storage.GetCounter(r.Context(), metricName)
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

// GetAllMetricsHandler возвращает HTML-страницу со списком всех метрик.
func (h *Handler) GetAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges(r.Context())
	counters := h.storage.GetAllCounters(r.Context())

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

// UpdateJSONHandler обрабатывает обновление одной метрики в формате JSON.
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

		if err := h.storage.UpdateGauge(r.Context(), req.ID, *req.Value); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	case "counter":
		if req.Delta == nil {
			http.Error(w, "Delta is required for counter", http.StatusBadRequest)
			return
		}

		if err := h.storage.UpdateCounter(r.Context(), req.ID, *req.Delta); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		newVal, _ := h.storage.GetCounter(r.Context(), req.ID)
		req.Delta = &newVal

	default:
		http.Error(w, "Unknown metric type", http.StatusBadRequest)
		return
	}

	if h.auditor != nil {
		h.auditor.NotifyAll([]string{req.ID}, r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

// ValueJSONHandler возвращает значение запрошенной метрики в формате JSON.
func (h *Handler) ValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	var req models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.MType {
	case "gauge":
		val, ok := h.storage.GetGauge(r.Context(), req.ID)
		if !ok {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		req.Value = &val

	case "counter":

		val, ok := h.storage.GetCounter(r.Context(), req.ID)
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

// UpdatesJSONHandler обрабатывает пакетное обновление метрик в формате JSON.
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

	if err := h.storage.UpdateMetrics(r.Context(), metrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.auditor != nil {
		names := make([]string, 0, len(metrics))
		for _, m := range metrics {
			names = append(names, m.ID)
		}
		h.auditor.NotifyAll(names, r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(metrics)
}

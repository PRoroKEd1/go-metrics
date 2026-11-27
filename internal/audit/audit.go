package audit

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// интерфейс Наблюдателя
type Observer interface {
	Notify(event Event) error
}

// Издатель, который хранит список наблюдателей и рассылает им события
type Auditor struct {
	observers []Observer
}

func NewAuditor() *Auditor {
	return &Auditor{
		observers: make([]Observer, 0),
	}
}

// добавляет нового наблюдателя
func (a *Auditor) Register(o Observer) {
	a.observers = append(a.observers, o)
}

// формирует событие и отправляет во все доступные приёмники
func (a *Auditor) NotifyAll(metrics []string, remoteAddr string) {
	if len(a.observers) == 0 || len(metrics) == 0 {
		return
	}

	//"127.0.0.1:54321" в "127.0.0.1"
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ip,
	}

	for _, observer := range a.observers {
		if err := observer.Notify(event); err != nil {
			slog.Error("Ошибка отправки события аудита", "err", err)
		}
	}
}

//ПРИЁМНИК 1: ФАЙЛ

type FileObserver struct {
	mu       sync.Mutex
	filePath string
}

func NewFileObserver(filePath string) *FileObserver {
	return &FileObserver{filePath: filePath}
}

func (f *FileObserver) Notify(event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

// ПРИЁМНИК 2: УДАЛЁННЫЙ СЕРВЕР

type URLObserver struct {
	url    string
	client *http.Client
}

func NewURLObserver(url string) *URLObserver {
	return &URLObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (u *URLObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, u.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

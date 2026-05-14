package service

import (
	"fmt"
	"sync"
	"time"

	"log-server/internal/config"
	"log-server/internal/model"
	"log-server/internal/storage"

	"github.com/google/uuid"
)

type LogService struct {
	storage  *storage.MemoryStorage
	logQueue chan model.LogEntry
	config   *config.Config

	clients     map[chan model.LogEntry]bool
	clientMutex sync.RWMutex

	stopCh chan struct{}
	doneCh chan struct{}
}

func NewLogService(cfg *config.Config, stor *storage.MemoryStorage) *LogService {
	svc := &LogService{
		storage:  stor,
		logQueue: make(chan model.LogEntry, cfg.Buffer.QueueSize),
		config:   cfg,
		clients:  make(map[chan model.LogEntry]bool),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}

	go svc.processQueue()

	return svc
}

func (s *LogService) processQueue() {
	defer close(s.doneCh)

	batch := make([]model.LogEntry, 0, 100)
	ticker := time.NewTicker(s.config.Buffer.FlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) > 0 {
			s.flushBatch(batch)
			batch = batch[:0]
		}
	}

	for {
		select {
		case entry := <-s.logQueue:
			batch = append(batch, entry)
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.stopCh:
			for {
				select {
				case entry := <-s.logQueue:
					batch = append(batch, entry)
					if len(batch) >= 100 {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

func (s *LogService) flushBatch(batch []model.LogEntry) {
	if err := s.storage.WriteBatch(batch); err != nil {
		fmt.Printf("Failed to write log batch: %v\n", err)
	}
}

func (s *LogService) WriteLog(title, level, tag, message string) (*model.LogEntry, error) {
	if title == "" {
		title = ""
	}

	normalizedLevel := normalizeLevel(level)
	if normalizedLevel == "" {
		return nil, fmt.Errorf("LogWrite: 不支持的 level -> %s", level)
	}

	entry := model.LogEntry{
		ID:        uuid.New().String(),
		Title:     title,
		Level:     normalizedLevel,
		Tag:       tag,
		Message:   message,
		Timestamp: time.Now(),
	}

	s.broadcast(entry)

	select {
	case s.logQueue <- entry:
		return &entry, nil
	default:
		if err := s.storage.Write(entry); err != nil {
			return nil, err
		}
		return &entry, nil
	}
}

func (s *LogService) Subscribe() chan model.LogEntry {
	client := make(chan model.LogEntry, 100)

	s.clientMutex.Lock()
	s.clients[client] = true
	s.clientMutex.Unlock()

	return client
}

func (s *LogService) Unsubscribe(client chan model.LogEntry) {
	s.clientMutex.Lock()
	if _, ok := s.clients[client]; ok {
		delete(s.clients, client)
		close(client)
	}
	s.clientMutex.Unlock()
}

func (s *LogService) broadcast(entry model.LogEntry) {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()

	for client := range s.clients {
		select {
		case client <- entry:
		default:
		}
	}
}

func (s *LogService) Close() {
	select {
	case <-s.doneCh:
		return
	default:
	}

	close(s.stopCh)
	<-s.doneCh

	s.clientMutex.Lock()
	for client := range s.clients {
		delete(s.clients, client)
		close(client)
	}
	s.clientMutex.Unlock()
}

func (s *LogService) QueryLogs(filter model.LogFilter) (*model.LogListResponse, error) {
	return s.storage.Query(filter)
}

func (s *LogService) GetTitles() (*model.TitleListResponse, error) {
	return s.storage.GetTitles()
}

func (s *LogService) GetStats() (*model.StatsResponse, error) {
	return s.storage.GetStats()
}

func (s *LogService) GetRecentLogs(count int) []model.LogEntry {
	return s.storage.GetRecentLogs(count)
}

func (s *LogService) ExportLogs(title string, filter model.LogFilter) ([]model.LogEntry, error) {
	return s.storage.ExportLogs(title, filter)
}

func (s *LogService) GetTags(title string) ([]string, error) {
	return s.storage.GetTags(title)
}

func (s *LogService) ClearLogs(title string) error {
	return s.storage.ClearLogs(title)
}

func normalizeLevel(level string) string {
	switch level {
	case "info", "warn", "error", "debug":
		return level
	default:
		return ""
	}
}

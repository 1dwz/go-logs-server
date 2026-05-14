package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"log-server/internal/config"
	"log-server/internal/model"
)

const currentLogFileName = "current.jsonl"

type FileStorage struct {
	config  *config.Config
	logs    []model.LogEntry
	logsMap map[string]model.LogEntry
	mu      sync.RWMutex
	maxLogs int64

	logFile *os.File
	writer  *bufio.Writer
}

func NewFileStorage(cfg *config.Config) (*FileStorage, error) {
	if err := os.MkdirAll(cfg.Storage.LogDir, 0755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(cfg.Storage.LogDir, currentLogFileName)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	store := &FileStorage{
		config:  cfg,
		logs:    make([]model.LogEntry, 0, cfg.Buffer.QueueSize),
		logsMap: make(map[string]model.LogEntry),
		maxLogs: int64(cfg.Buffer.QueueSize),
		logFile: file,
		writer:  bufio.NewWriter(file),
	}

	if err := store.loadFromFile(filePath); err != nil {
		_ = file.Close()
		return nil, err
	}

	return store, nil
}

func (s *FileStorage) loadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry model.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		s.appendMemory(entry)
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func (s *FileStorage) Write(entry model.LogEntry) error {
	return s.WriteBatch([]model.LogEntry{entry})
}

func (s *FileStorage) WriteBatch(entries []model.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range entries {
		s.appendMemory(entry)

		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		if _, err := s.writer.Write(data); err != nil {
			return err
		}
		if err := s.writer.WriteByte('\n'); err != nil {
			return err
		}
	}

	return s.writer.Flush()
}

func (s *FileStorage) appendMemory(entry model.LogEntry) {
	s.checkAndEvict()
	s.logs = append(s.logs, entry)
	s.logsMap[entry.ID] = entry
}

func (s *FileStorage) checkAndEvict() {
	if int64(len(s.logs)) < s.maxLogs {
		return
	}

	removeCount := len(s.logs) / 10
	if removeCount < 100 {
		removeCount = 100
	}
	if removeCount > len(s.logs) {
		removeCount = len(s.logs)
	}

	removed := s.logs[:removeCount]
	s.logs = s.logs[removeCount:]
	for _, e := range removed {
		delete(s.logsMap, e.ID)
	}
}

func (s *FileStorage) Query(filter model.LogFilter) (*model.LogListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filteredIdx := make([]int, 0)
	for i, entry := range s.logs {
		if s.matchFilter(entry, filter) {
			filteredIdx = append(filteredIdx, i)
		}
	}

	total := int64(len(filteredIdx))

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 50
	}

	start := (filter.Page - 1) * filter.PageSize
	end := start + filter.PageSize
	if start >= len(filteredIdx) {
		return &model.LogListResponse{
			Logs:     []model.LogEntry{},
			Total:    total,
			Page:     filter.Page,
			PageSize: filter.PageSize,
		}, nil
	}
	if end > len(filteredIdx) {
		end = len(filteredIdx)
	}

	pageLogs := make([]model.LogEntry, 0, end-start)
	for i := start; i < end; i++ {
		realIdx := filteredIdx[len(filteredIdx)-1-i]
		pageLogs = append(pageLogs, s.logs[realIdx])
	}

	return &model.LogListResponse{
		Logs:     pageLogs,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

func (s *FileStorage) matchFilter(entry model.LogEntry, filter model.LogFilter) bool {
	if filter.TitleProvided && entry.Title != filter.Title {
		return false
	}
	if filter.Level != "" && entry.Level != filter.Level {
		return false
	}
	if filter.Tag != "" {
		tags := strings.Split(filter.Tag, ",")
		found := false
		for _, t := range tags {
			t = strings.TrimSpace(t)
			if t == entry.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if filter.Keyword != "" && !containsFold(entry.Message, filter.Keyword) {
		return false
	}
	if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
		return false
	}
	return true
}

func containsFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		sc, tc := s[i], t[i]
		if sc == tc {
			continue
		}
		if sc >= 'A' && sc <= 'Z' {
			sc += 'a' - 'A'
		}
		if tc >= 'A' && tc <= 'Z' {
			tc += 'a' - 'A'
		}
		if sc != tc {
			return false
		}
	}
	return true
}

func (s *FileStorage) GetTitles() (*model.TitleListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	titleMap := make(map[string]*model.TitleInfo)
	unknownInfo := &model.TitleInfo{
		Name:      "未知",
		Count:     0,
		LastTime:  "",
		LevelDist: map[string]int64{"info": 0, "warn": 0, "error": 0, "debug": 0},
	}

	for _, entry := range s.logs {
		title := entry.Title
		if title == "" {
			title = "__unknown__"
		}

		if _, exists := titleMap[title]; !exists {
			titleMap[title] = &model.TitleInfo{
				Name:      title,
				Count:     0,
				LastTime:  entry.Timestamp.Format(time.RFC3339),
				LevelDist: map[string]int64{"info": 0, "warn": 0, "error": 0, "debug": 0},
			}
		}

		titleMap[title].Count++
		if entry.Timestamp.Format(time.RFC3339) > titleMap[title].LastTime {
			titleMap[title].LastTime = entry.Timestamp.Format(time.RFC3339)
		}
		if _, ok := titleMap[title].LevelDist[entry.Level]; ok {
			titleMap[title].LevelDist[entry.Level]++
		}
	}

	titles := make([]model.TitleInfo, 0)
	for _, info := range titleMap {
		if info.Name == "__unknown__" {
			info.Name = "未知"
			unknownInfo = info
		} else {
			titles = append(titles, *info)
		}
	}

	model.SortTitlesByName(titles)
	if unknownInfo.Count > 0 {
		titles = append(titles, *unknownInfo)
	}

	return &model.TitleListResponse{
		Titles: titles,
		Total:  int64(len(titles)),
	}, nil
}

func (s *FileStorage) GetStats() (*model.StatsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := time.Now().Format("2006-01-02")
	stats := &model.StatsResponse{
		TodayTotal: 0,
		ByLevel: map[string]int{
			"info":  0,
			"warn":  0,
			"error": 0,
			"debug": 0,
		},
	}

	for _, entry := range s.logs {
		if entry.Timestamp.Format("2006-01-02") == today {
			stats.TodayTotal++
		}
		if _, ok := stats.ByLevel[entry.Level]; ok {
			stats.ByLevel[entry.Level]++
		}
	}

	return stats, nil
}

func (s *FileStorage) GetRecentLogs(count int) []model.LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if count > len(s.logs) {
		count = len(s.logs)
	}
	if count <= 0 {
		count = len(s.logs)
	}

	result := make([]model.LogEntry, count)
	copy(result, s.logs[len(s.logs)-count:])
	return result
}

func (s *FileStorage) ExportLogs(title string, filter model.LogFilter) ([]model.LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filter.Title = title

	logs := make([]model.LogEntry, 0)
	for _, entry := range s.logs {
		if s.matchFilter(entry, filter) {
			logs = append(logs, entry)
		}
	}

	if len(logs) > 10000 {
		logs = logs[len(logs)-10000:]
	}

	return logs, nil
}

func (s *FileStorage) GetTags(title string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tagSet := make(map[string]bool)

	for _, entry := range s.logs {
		if entry.Title != title {
			continue
		}
		if entry.Tag != "" {
			tagSet[entry.Tag] = true
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	sort.Strings(tags)
	return tags, nil
}

func (s *FileStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer != nil {
		if err := s.writer.Flush(); err != nil {
			return err
		}
	}
	if s.logFile != nil {
		return s.logFile.Close()
	}
	return nil
}

func (s *FileStorage) GetLogCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.logs))
}

func (s *FileStorage) ClearLogs(title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	remaining := make([]model.LogEntry, 0)
	for _, entry := range s.logs {
		if entry.Title != title {
			remaining = append(remaining, entry)
		} else {
			delete(s.logsMap, entry.ID)
		}
	}
	s.logs = remaining

	if err := s.writer.Flush(); err != nil {
		return err
	}

	tmpPath := s.logFile.Name() + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	tmpWriter := bufio.NewWriter(tmpFile)
	for _, entry := range s.logs {
		data, err := json.Marshal(entry)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return err
		}
		if _, err := tmpWriter.Write(data); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return err
		}
		if err := tmpWriter.WriteByte('\n'); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return err
		}
	}
	if err := tmpWriter.Flush(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	tmpFile.Close()

	s.writer.Flush()
	s.logFile.Close()

	if err := os.Rename(tmpPath, s.logFile.Name()); err != nil {
		newFile, err := os.OpenFile(s.logFile.Name(), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		s.logFile = newFile
		s.writer = bufio.NewWriter(newFile)
		for _, entry := range s.logs {
			data, err := json.Marshal(entry)
			if err != nil {
				return err
			}
			if _, err := s.writer.Write(data); err != nil {
				return err
			}
			if err := s.writer.WriteByte('\n'); err != nil {
				return err
			}
		}
		return s.writer.Flush()
	}

	newFile, err := os.OpenFile(s.logFile.Name(), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	s.logFile = newFile
	s.writer = bufio.NewWriter(newFile)

	return nil
}

package model

import (
	"sort"
	"time"
)

type LogEntry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Level     string    `json:"level"`
	Tag       string    `json:"tag"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type LogFilter struct {
	Title         string
	TitleProvided bool
	Level         string
	Tag           string
	Keyword       string
	StartTime     *time.Time
	EndTime       *time.Time
	Page          int
	PageSize      int
}

type LogListResponse struct {
	Logs     []LogEntry `json:"logs"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}

type TitleInfo struct {
	Name      string           `json:"name"`
	Count     int64            `json:"count"`
	LastTime  string           `json:"lastTime"`
	LevelDist map[string]int64 `json:"levelDist"`
}

type TitleListResponse struct {
	Titles []TitleInfo `json:"titles"`
	Total  int64       `json:"total"`
}

type StatsResponse struct {
	TodayTotal int64          `json:"todayTotal"`
	ByLevel    map[string]int `json:"byLevel"`
}

type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func SuccessResponse(data interface{}) ApiResponse {
	return ApiResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

func ErrorResponse(message string) ApiResponse {
	return ApiResponse{
		Code:    -1,
		Message: message,
	}
}

func SortTitlesByName(titles []TitleInfo) {
	sort.Slice(titles, func(i, j int) bool {
		return titles[i].Name < titles[j].Name
	})
}

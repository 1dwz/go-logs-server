package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"log-server/internal/model"
	"log-server/internal/service"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	service *service.LogService
}

func NewLogHandler(svc *service.LogService) *LogHandler {
	return &LogHandler{
		service: svc,
	}
}

func (h *LogHandler) HandleLogWrite(c *gin.Context) {
	title := strings.TrimSpace(c.Query("title"))
	level := strings.TrimSpace(c.Query("level"))
	tag := strings.TrimSpace(c.Query("tag"))

	if title == "" {
		title = ""
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusRequestEntityTooLarge, "LogWrite: 请求体过大或读取失败")
		return
	}
	message := string(body)

	entry, err := h.service.WriteLog(title, level, tag, message)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "OK")
	_ = entry
}

func (h *LogHandler) HandleQueryLogs(c *gin.Context) {
	_, titleProvided := c.GetQuery("title")
	filter := model.LogFilter{
		Title:         c.Query("title"),
		TitleProvided: titleProvided,
		Level:         c.Query("level"),
		Tag:           c.Query("tag"),
		Keyword:       c.Query("keyword"),
		Page:          1,
		PageSize:      50,
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filter.Page = p
		}
	}

	if pageSize := c.Query("pageSize"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 200 {
			filter.PageSize = ps
		}
	}

	if startTime := c.Query("startTime"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = &t
		}
	}

	if endTime := c.Query("endTime"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = &t
		}
	}

	result, err := h.service.QueryLogs(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

func (h *LogHandler) HandleStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

func (h *LogHandler) HandleTitles(c *gin.Context) {
	titles, err := h.service.GetTitles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(titles))
}

func (h *LogHandler) HandleTags(c *gin.Context) {
	title := c.Query("title")

	tags, err := h.service.GetTags(title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(tags))
}

func (h *LogHandler) HandleClearLogs(c *gin.Context) {
	title := strings.TrimSpace(c.Query("title"))

	if title == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse("标题不能为空"))
		return
	}

	if err := h.service.ClearLogs(title); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(nil))
}

func (h *LogHandler) HandleExport(c *gin.Context) {
	title, titleProvided := c.GetQuery("title")
	level := c.Query("level")

	filter := model.LogFilter{
		TitleProvided: titleProvided,
		Level:         level,
	}

	logs, err := h.service.ExportLogs(title, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(err.Error()))
		return
	}

	filename := "logs"
	if title != "" {
		filename = title
	}
	filename += "_" + time.Now().Format("20060102_150405") + ".json"

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusOK, logs)
}

func (h *LogHandler) HandleLogStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	title, titleProvided := c.GetQuery("title")
	level := c.Query("level")
	tag := c.Query("tag")

	client := h.service.Subscribe()
	defer h.service.Unsubscribe(client)

	keepAlive := time.NewTicker(30 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-keepAlive.C:
			c.SSEvent("ping", gin.H{"time": time.Now().Format(time.RFC3339)})
			c.Writer.Flush()
		case entry, ok := <-client:
			if !ok {
				return
			}
			if titleProvided && entry.Title != title {
				continue
			}
			if level != "" && entry.Level != level {
				continue
			}
			if tag != "" && !matchesTagFilter(entry.Tag, tag) {
				continue
			}
			c.SSEvent("log", entry)
			c.Writer.Flush()
		}
	}
}

func matchesTagFilter(entryTag, filter string) bool {
	tags := strings.Split(filter, ",")
	for _, tag := range tags {
		if strings.TrimSpace(tag) == entryTag {
			return true
		}
	}
	return false
}

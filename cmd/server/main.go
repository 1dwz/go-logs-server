package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log-server/internal/config"
	"log-server/internal/handler"
	"log-server/internal/service"
	"log-server/internal/storage"
	"log-server/web"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.LoadConfig("config.toml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	store, err := storage.NewFileStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	logService := service.NewLogService(cfg, store)
	logHandler := handler.NewLogHandler(logService)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.POST("/log", logHandler.HandleLogWrite)

	api := router.Group("/api")
	{
		api.GET("/logs", logHandler.HandleQueryLogs)
		api.GET("/logs/stream", logHandler.HandleLogStream)
		api.GET("/stats", logHandler.HandleStats)
		api.GET("/titles", logHandler.HandleTitles)
		api.GET("/tags", logHandler.HandleTags)
		api.GET("/export", logHandler.HandleExport)
		api.DELETE("/logs", logHandler.HandleClearLogs)
	}

	registerWebRoutes(router)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Printf("日志服务器启动中，监听地址: %s", addr)
	log.Printf("Web 界面地址: http://localhost:%d/", cfg.Server.Port)
	log.Printf("API 地址: http://localhost:%d/api", cfg.Server.Port)
	log.Printf("日志写入地址: POST http://localhost:%d/log", cfg.Server.Port)
	log.Printf("日志持久化文件: %s", cfg.Storage.LogDir)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("服务器正在关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown failed: %v", err)
	}

	logService.Close()
	if err := store.Close(); err != nil {
		log.Printf("Storage close failed: %v", err)
	}

	log.Println("服务器已关闭")
}

func registerWebRoutes(router *gin.Engine) {
	staticFiles, err := fs.Sub(web.Files, ".")
	if err != nil {
		log.Fatalf("Failed to load embedded web files: %v", err)
	}

	router.StaticFS("/static", http.FS(staticFiles))
	router.GET("/", serveEmbeddedHTML("index.html"))
	router.GET("/view", serveEmbeddedHTML("view.html"))
}

func serveEmbeddedHTML(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		content, err := web.Files.ReadFile(name)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to read %s", name)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	}
}

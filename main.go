package main

import (
	"api-systemd/internal/pkg/config"
	"api-systemd/internal/pkg/logger"
	"api-systemd/internal/router"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var serverPort string

func main() {
	flag.StringVar(&serverPort, "port", ":8080", "server port")
	flag.Parse()

	// 加载配置
	cfg := config.Load()

	// 如果命令行指定了端口，优先使用命令行参数
	if serverPort != ":8080" {
		cfg.Server.Port = serverPort
	} else {
		serverPort = cfg.Server.Port
	}

	ctx := context.Background()
	logger.Info(ctx, "Starting API-Systemd server", "port", serverPort, "api_key_configured", cfg.Security.APIKey != "")

	// 创建路由器
	r := router.New(cfg)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         serverPort,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 优雅关闭处理
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info(ctx, "Shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error(ctx, "Server shutdown error", "error", err)
		}
	}()

	logger.Info(ctx, "Server started successfully", "port", serverPort)
	fmt.Printf("🚀 API-Systemd Server listening on %s...\n", serverPort)
	fmt.Printf("📋 API Documentation: https://github.com/rushteam/api-systemd\n")
	fmt.Printf("🩺 Health check: http://localhost%s/health\n", serverPort)
	fmt.Printf("🔍 Debug profiler: http://localhost%s/debug/\n", serverPort)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(ctx, "Server failed to start", "error", err)
		panic(err)
	}

	logger.Info(ctx, "Server stopped gracefully")
}

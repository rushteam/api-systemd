package main

import (
	"api-systemd/internal/app"
	"api-systemd/internal/middleware"
	"api-systemd/internal/pkg/config"
	"api-systemd/internal/pkg/logger"
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
	logger.Info(ctx, "Starting API-Systemd server", "port", serverPort, "config", cfg)

	app := app.New()

	// API路由配置
	http.HandleFunc("/services/deploy", middleware.Chain(app.Deploy,
		middleware.Recovery, middleware.Logging, middleware.CORS))
	http.HandleFunc("/services/start", middleware.Chain(app.StartService,
		middleware.Recovery, middleware.Logging, middleware.CORS))
	http.HandleFunc("/services/stop", middleware.Chain(app.Stop,
		middleware.Recovery, middleware.Logging, middleware.CORS))
	http.HandleFunc("/services/restart", middleware.Chain(app.Restart,
		middleware.Recovery, middleware.Logging, middleware.CORS))
	http.HandleFunc("/services/remove", middleware.Chain(app.Remove,
		middleware.Recovery, middleware.Logging, middleware.CORS))
	http.HandleFunc("/services/status", middleware.Chain(app.GetStatus,
		middleware.Recovery, middleware.Logging, middleware.CORS))
	http.HandleFunc("/services/logs", middleware.Chain(app.GetLogs,
		middleware.Recovery, middleware.Logging, middleware.CORS))

	// 配置管理接口
	http.HandleFunc("/configs/create", middleware.Chain(app.CreateConfig,
		middleware.Recovery, middleware.Logging, middleware.CORS))
	http.HandleFunc("/configs/delete", middleware.Chain(app.DeleteConfig,
		middleware.Recovery, middleware.Logging, middleware.CORS))

	// 系统健康检查
	http.HandleFunc("/health", middleware.Chain(app.HealthCheck,
		middleware.Recovery, middleware.Logging, middleware.CORS))

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         serverPort,
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
	fmt.Printf("API-Systemd Server listening on %s...\n", serverPort)
	fmt.Printf("Health check available at: http://localhost%s/health\n", serverPort)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(ctx, "Server failed to start", "error", err)
		panic(err)
	}

	logger.Info(ctx, "Server stopped gracefully")
}

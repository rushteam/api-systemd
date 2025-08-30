package app

import (
	"api-systemd/internal/pkg/logger"
	"api-systemd/internal/pkg/systemd"
	"api-systemd/internal/pkg/validator"
	"api-systemd/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type App struct {
	Service service.Service
}

func New() *App {
	return &App{
		Service: service.NewService(),
	}
}

// getServiceName 获取服务名称（支持URL参数和查询参数）
func getServiceName(r *http.Request) string {
	// 首先尝试从URL路径参数获取
	if serviceName := chi.URLParam(r, "serviceName"); serviceName != "" {
		return serviceName
	}
	// 然后从查询参数获取
	return r.URL.Query().Get("service")
}

// 启动 Systemd 服务
func (s *App) StartService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	service := r.URL.Query().Get("service")

	if err := validator.ValidateServiceName(service); err != nil {
		logger.Error(ctx, "StartService validation failed", "error", err, "service", service)
		apiResponse(w, -1, "validation failed", err.Error())
		return
	}

	err := s.Service.Start(ctx, service)
	if err != nil {
		logger.Error(ctx, "StartService failed", "error", err, "service", service)
		apiResponse(w, -1, "failed", err.Error())
		return
	}

	apiResponse(w, 0, "ok", map[string]string{"service": service, "status": "started"})
}

// CreateConfig 创建配置
func (s *App) CreateConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var configRequest struct {
		Service string `json:"service"`
		Config  string `json:"config"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&configRequest); err != nil {
		logger.Error(ctx, "Failed to decode config request", "error", err)
		apiResponse(w, -1, "invalid request format", err.Error())
		return
	}

	if err := validator.ValidateServiceName(configRequest.Service); err != nil {
		logger.Error(ctx, "CreateConfig validation failed", "error", err, "service", configRequest.Service)
		apiResponse(w, -1, "validation failed", err.Error())
		return
	}

	if configRequest.Config == "" {
		logger.Error(ctx, "Empty config content", "service", configRequest.Service)
		apiResponse(w, -1, "validation failed", "config content cannot be empty")
		return
	}

	filePath := fmt.Sprintf("/etc/systemd/system/%s.service", configRequest.Service)
	logger.Info(ctx, "Creating config file", "service", configRequest.Service, "file", filePath)

	err := os.WriteFile(filePath, []byte(configRequest.Config), 0644)
	if err != nil {
		logger.Error(ctx, "Failed to write config file", "error", err, "service", configRequest.Service, "file", filePath)
		apiResponse(w, -1, "failed", err.Error())
		return
	}

	// 重新加载 Systemd 配置
	logger.Info(ctx, "Reloading systemd daemon")
	err = systemd.ReloadDaemon()
	if err != nil {
		logger.Error(ctx, "Failed to reload systemd daemon", "error", err)
		apiResponse(w, -1, "failed", err.Error())
		return
	}

	apiResponse(w, 0, "ok", map[string]string{
		"service":     configRequest.Service,
		"config_file": filePath,
	})
}

// DeleteConfig 删除配置
func (s *App) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := getServiceName(r)

	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "DeleteConfig validation failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "validation failed", err.Error())
		return
	}

	filePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	logger.Info(ctx, "Deleting config file", "service", serviceName, "file", filePath)

	err := os.Remove(filePath)
	if err != nil {
		logger.Error(ctx, "Failed to delete config file", "error", err, "service", serviceName, "file", filePath)
		apiResponse(w, -1, "failed", err.Error())
		return
	}

	// 重新加载 Systemd 配置
	logger.Info(ctx, "Reloading systemd daemon")
	err = systemd.ReloadDaemon()
	if err != nil {
		logger.Error(ctx, "Failed to reload systemd daemon", "error", err)
		apiResponse(w, -1, "failed", err.Error())
		return
	}

	apiResponse(w, 0, "ok", map[string]string{
		"service":      serviceName,
		"deleted_file": filePath,
	})
}

// 获取服务状态
// GetStatus 获取服务状态接口
func (s *App) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := getServiceName(r)

	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "GetStatus validation failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "validation failed", err.Error())
		return
	}

	logger.Info(ctx, "GetStatus request received", "service", serviceName)

	status, err := s.Service.GetStatus(ctx, serviceName)
	if err != nil {
		logger.Error(ctx, "GetStatus failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "failed to get status", err.Error())
		return
	}

	logger.Info(ctx, "GetStatus completed successfully", "service", serviceName)
	apiResponse(w, 0, "ok", status)
}

// Stop 停止服务接口
func (s *App) Stop(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := getServiceName(r)

	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "Stop validation failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "validation failed", err.Error())
		return
	}

	logger.Info(ctx, "Stop request received", "service", serviceName)

	if err := s.Service.Stop(ctx, serviceName); err != nil {
		logger.Error(ctx, "Stop failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "stop failed", err.Error())
		return
	}

	logger.Info(ctx, "Stop completed successfully", "service", serviceName)
	apiResponse(w, 0, "ok", map[string]string{"service": serviceName, "status": "stopped"})
}

// Remove 移除服务接口
func (s *App) Remove(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := getServiceName(r)

	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "Remove validation failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "validation failed", err.Error())
		return
	}

	logger.Info(ctx, "Remove request received", "service", serviceName)

	if err := s.Service.Remove(ctx, serviceName); err != nil {
		logger.Error(ctx, "Remove failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "remove failed", err.Error())
		return
	}

	logger.Info(ctx, "Remove completed successfully", "service", serviceName)
	apiResponse(w, 0, "ok", map[string]string{"service": serviceName, "status": "removed"})
}

// Restart 重启服务接口
func (s *App) Restart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := getServiceName(r)

	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "Restart validation failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "validation failed", err.Error())
		return
	}

	logger.Info(ctx, "Restart request received", "service", serviceName)

	if err := s.Service.Restart(ctx, serviceName); err != nil {
		logger.Error(ctx, "Restart failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "restart failed", err.Error())
		return
	}

	logger.Info(ctx, "Restart completed successfully", "service", serviceName)
	apiResponse(w, 0, "ok", map[string]string{"service": serviceName, "status": "restarted"})
}

// GetLogs 获取服务日志接口
func (s *App) GetLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := getServiceName(r)
	linesStr := r.URL.Query().Get("lines")

	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "GetServiceLogs validation failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "validation failed", err.Error())
		return
	}

	lines := 50 // 默认50行
	if linesStr != "" {
		if parsedLines, err := strconv.Atoi(linesStr); err == nil && parsedLines > 0 {
			lines = parsedLines
		}
	}

	logger.Info(ctx, "GetServiceLogs request received", "service", serviceName, "lines", lines)

	logEntries, err := s.Service.GetLogs(ctx, serviceName, lines)
	if err != nil {
		logger.Error(ctx, "GetServiceLogs failed", "error", err, "service", serviceName)
		apiResponse(w, -1, "failed to get logs", err.Error())
		return
	}

	logger.Info(ctx, "GetServiceLogs completed successfully", "service", serviceName, "entries", len(logEntries))
	apiResponse(w, 0, "ok", map[string]interface{}{
		"service": serviceName,
		"lines":   len(logEntries),
		"logs":    logEntries,
	})
}

// HealthCheck 健康检查接口
func (s *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 检查系统基本信息
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"services": map[string]string{
			"systemd": "available",
		},
	}

	// 简单检查systemd是否可用
	if err := systemd.CheckSystemdAvailable(); err != nil {
		health["status"] = "degraded"
		health["services"].(map[string]string)["systemd"] = "unavailable"
		logger.Warn(ctx, "Systemd not available", "error", err)
	}

	logger.Debug(ctx, "Health check performed")
	apiResponse(w, 0, "ok", health)
}

// Deploy 部署服务接口
func (s *App) Deploy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var params service.DeployRequest
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		logger.Error(ctx, "Failed to decode deploy request", "error", err)
		apiResponse(w, -1, "invalid request format", err.Error())
		return
	}

	logger.Info(ctx, "Deploy request received", "service", params.Service, "url", params.PackageURL)

	if err := s.Service.Deploy(ctx, &params); err != nil {
		logger.Error(ctx, "Deploy failed", "error", err, "service", params.Service)
		apiResponse(w, -1, "deploy failed", err.Error())
		return
	}

	logger.Info(ctx, "Deploy completed successfully", "service", params.Service)
	apiResponse(w, 0, "ok", map[string]string{
		"service": params.Service,
		"status":  "deployed",
	})
}

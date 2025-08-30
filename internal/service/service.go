package service

import (
	download "api-systemd/internal/pkg/donwload"
	"api-systemd/internal/pkg/extract"
	"api-systemd/internal/pkg/hooks"
	"api-systemd/internal/pkg/logger"
	"api-systemd/internal/pkg/logs"
	"api-systemd/internal/pkg/systemd"
	"api-systemd/internal/pkg/telemetry"
	"api-systemd/internal/pkg/validator"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Service 服务管理接口
type Service interface {
	// Deploy 部署服务
	Deploy(ctx context.Context, params *DeployRequest) error
	// Start 启动服务
	Start(ctx context.Context, serviceName string) error
	// Stop 停止服务
	Stop(ctx context.Context, serviceName string) error
	// Restart 重启服务
	Restart(ctx context.Context, serviceName string) error
	// Remove 移除服务
	Remove(ctx context.Context, serviceName string) error
	// GetStatus 获取服务状态
	GetStatus(ctx context.Context, serviceName string) (*systemd.Unit, error)
	// GetLogs 获取服务日志
	GetLogs(ctx context.Context, serviceName string, lines int) ([]logs.LogEntry, error)
}

type service struct {
	mu           sync.RWMutex // 添加读写锁以支持并发控制
	hookExecutor hooks.HookExecutorInterface
	otelReporter *telemetry.OTELReporter
}

func NewService() Service {
	return &service{
		hookExecutor: hooks.NewHookExecutor(),
	}
}

// DeployRequest 部署请求
type DeployRequest struct {
	Service       string                    `json:"service"`                 // 服务名称
	Path          string                    `json:"path"`                    // 部署路径
	PackageURL    string                    `json:"package_url"`             // 包下载地址
	StartCommand  string                    `json:"start_command"`           // 启动命令
	Config        *hooks.ServiceConfig      `json:"config,omitempty"`        // 服务配置
	Hooks         []hooks.Hook              `json:"hooks,omitempty"`         // 生命周期钩子
	Notifications *hooks.NotificationConfig `json:"notifications,omitempty"` // 通知配置
}

// Deploy 部署服务（统一的增强版本）
func (s *service) Deploy(ctx context.Context, params *DeployRequest) error {
	// 参数验证
	if err := validator.ValidateServiceName(params.Service); err != nil {
		logger.Error(ctx, "Deploy validation failed", "error", err, "service", params.Service)
		return fmt.Errorf("validation failed: %w", err)
	}

	// 并发控制
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Info(ctx, "Starting deployment", "service", params.Service, "url", params.PackageURL)

	// 初始化OTEL报告器（如果配置了）
	if params.Notifications != nil && params.Notifications.OTEL != nil && params.Notifications.OTEL.Enabled {
		otelReporter, err := telemetry.NewOTELReporter(*params.Notifications.OTEL)
		if err != nil {
			logger.Warn(ctx, "Failed to initialize OTEL reporter", "error", err)
		} else {
			s.otelReporter = otelReporter
		}
	}

	// 执行pre-start钩子
	if len(params.Hooks) > 0 {
		events := s.hookExecutor.ExecuteHooks(ctx, params.Hooks, hooks.HookPreStart, params.Service, map[string]interface{}{
			"action": "deploy",
			"phase":  "pre_start",
		})

		// 检查关键钩子是否失败
		for _, event := range events {
			if event.Status == "failure" {
				logger.Error(ctx, "Pre-start hook failed", "service", params.Service, "hook", event.HookType, "error", event.Error)
				if s.otelReporter != nil {
					s.otelReporter.ReportHookExecution(ctx, event)
				}
				return fmt.Errorf("pre-start hook failed: %s", event.Error)
			}
			if s.otelReporter != nil {
				s.otelReporter.ReportHookExecution(ctx, event)
			}
		}
	}

	// 下载和解压
	tempFile, err := download.Download(params.PackageURL)
	if err != nil {
		logger.Error(ctx, "Failed to download file", "error", err, "url", params.PackageURL)
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(tempFile); removeErr != nil {
			logger.Warn(ctx, "Failed to remove temp file", "file", tempFile, "error", removeErr)
		}
	}()

	folders, err := extract.Extract(tempFile, params.Path)
	if err != nil {
		logger.Error(ctx, "Failed to extract file", "error", err, "file", tempFile)
		return fmt.Errorf("failed to extract file: %w", err)
	}

	var folder string
	if len(folders) > 0 {
		folder = folders[0]
	}
	if len(folder) == 0 {
		logger.Error(ctx, "No folders extracted from package")
		return fmt.Errorf("failed to extract folder name")
	}

	// 创建服务配置
	var config *hooks.ServiceConfig
	if params.Config != nil {
		config = params.Config
		config.ServiceName = params.Service
		config.WorkingDirectory = filepath.Join(params.Path, folder)
		config.ExecStart = filepath.Join(params.Path, folder, params.StartCommand)
	} else {
		config = &hooks.ServiceConfig{
			ServiceName:      params.Service,
			Description:      fmt.Sprintf("%s Service", params.Service),
			WorkingDirectory: filepath.Join(params.Path, folder),
			ExecStart:        filepath.Join(params.Path, folder, params.StartCommand),
			RestartPolicy:    "always",
			Hooks:            []hooks.Hook{},
		}
	}

	// 合并钩子配置
	if len(params.Hooks) > 0 {
		config.Hooks = append(config.Hooks, params.Hooks...)
	}

	// 写入systemd配置
	systemdFile := fmt.Sprintf("/etc/systemd/system/%s.service", params.Service)
	systemdConfig := NewSystemdConfig(params.Service, config.WorkingDirectory, params.StartCommand, config)

	if err := systemdConfig.WriteFile(systemdFile); err != nil {
		logger.Error(ctx, "Failed to write systemd config", "error", err, "file", systemdFile)
		return fmt.Errorf("failed to write systemd config: %w", err)
	}

	logger.Info(ctx, "Creating systemd config", "service", params.Service, "path", config.WorkingDirectory)

	// 重新加载systemd
	logger.Info(ctx, "Reloading systemd daemon")
	if err := systemd.ReloadDaemon(); err != nil {
		logger.Error(ctx, "Failed to reload systemd daemon", "error", err)
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// 启用和启动服务
	logger.Info(ctx, "Enabling service", "service", params.Service)
	if err := systemd.EnableUnit(params.Service); err != nil {
		logger.Error(ctx, "Failed to enable service", "error", err, "service", params.Service)
		return fmt.Errorf("failed to enable service: %w", err)
	}

	logger.Info(ctx, "Starting service", "service", params.Service)
	if err := systemd.Send(params.Service, "start", "replace"); err != nil {
		logger.Error(ctx, "Failed to start service", "error", err, "service", params.Service)
		return fmt.Errorf("failed to start service: %w", err)
	}

	// 执行post-start钩子
	if len(params.Hooks) > 0 {
		events := s.hookExecutor.ExecuteHooks(ctx, params.Hooks, hooks.HookPostStart, params.Service, map[string]interface{}{
			"action": "deploy",
			"phase":  "post_start",
		})

		for _, event := range events {
			if s.otelReporter != nil {
				s.otelReporter.ReportHookExecution(ctx, event)
			}
		}
	}

	// 发送服务部署事件通知
	if s.otelReporter != nil {
		s.otelReporter.ReportServiceEvent(ctx, params.Service, "deployed", map[string]interface{}{
			"package_url": params.PackageURL,
			"path":        params.Path,
		})
	}

	// 发送回调通知
	if params.Notifications != nil && params.Notifications.Callback != nil && params.Notifications.Callback.Enabled {
		go s.sendCallbackNotification(ctx, params.Service, "deployed", params.Notifications.Callback)
	}

	logger.Info(ctx, "Deployment completed successfully", "service", params.Service)
	return nil
}

func (s *service) Stop(ctx context.Context, serviceName string) error {
	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "Stop validation failed", "error", err, "service", serviceName)
		return fmt.Errorf("validation failed: %w", err)
	}

	logger.Info(ctx, "Stopping service", "service", serviceName)

	// Step 1: Stop the service
	err := systemd.Send(serviceName, "stop", "replace")
	if err != nil {
		logger.Error(ctx, "Failed to stop service", "error", err, "service", serviceName)
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Step 2: Disable the service
	err = systemd.DisableUnit(serviceName)
	if err != nil {
		logger.Error(ctx, "Failed to disable service", "error", err, "service", serviceName)
		return fmt.Errorf("failed to disable service: %w", err)
	}

	logger.Info(ctx, "Service stopped successfully", "service", serviceName)
	return nil
}

func (s *service) Remove(ctx context.Context, serviceName string) error {
	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "Remove validation failed", "error", err, "service", serviceName)
		return fmt.Errorf("validation failed: %w", err)
	}

	logger.Info(ctx, "Removing service", "service", serviceName)

	// Step 1: Stop the service
	err := systemd.Send(serviceName, "stop", "replace")
	if err != nil {
		logger.Error(ctx, "Failed to stop service", "error", err, "service", serviceName)
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Step 2: Disable the service
	err = systemd.DisableUnit(serviceName)
	if err != nil {
		logger.Error(ctx, "Failed to disable service", "error", err, "service", serviceName)
		return fmt.Errorf("failed to disable service: %w", err)
	}

	// Step 3: Remove the Systemd service file
	systemdFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	logger.Info(ctx, "Removing systemd service file", "file", systemdFile)

	if err := os.Remove(systemdFile); err != nil {
		logger.Error(ctx, "Failed to remove systemd service file", "error", err, "file", systemdFile)
		return fmt.Errorf("failed to remove systemd service file: %w", err)
	}

	// Step 4: Reload systemd daemon to apply changes
	logger.Info(ctx, "Reloading systemd daemon")
	err = systemd.ReloadDaemon()
	if err != nil {
		logger.Error(ctx, "Failed to reload systemd daemon", "error", err)
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	logger.Info(ctx, "Service removed successfully", "service", serviceName)
	return nil
}

func (s *service) Restart(ctx context.Context, serviceName string) error {
	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "Restart validation failed", "error", err, "service", serviceName)
		return fmt.Errorf("validation failed: %w", err)
	}

	logger.Info(ctx, "Restarting service", "service", serviceName)

	// Step 1: Restart the service
	err := systemd.Send(serviceName, "restart", "replace")
	if err != nil {
		logger.Error(ctx, "Failed to restart service", "error", err, "service", serviceName)
		return fmt.Errorf("failed to restart service: %w", err)
	}

	logger.Info(ctx, "Service restarted successfully", "service", serviceName)
	return nil
}

// GetStatus 获取服务状态
func (s *service) GetStatus(ctx context.Context, serviceName string) (*systemd.Unit, error) {
	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "GetStatus validation failed", "error", err, "service", serviceName)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	logger.Debug(ctx, "Getting service status", "service", serviceName)

	data, err := systemd.Load(serviceName)
	if err != nil {
		logger.Error(ctx, "Failed to load service status", "error", err, "service", serviceName)
		return nil, fmt.Errorf("failed to get service status: %w", err)
	}
	return data, nil
}

// Start 启动服务
func (s *service) Start(ctx context.Context, serviceName string) error {
	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "Start validation failed", "error", err, "service", serviceName)
		return fmt.Errorf("validation failed: %w", err)
	}

	logger.Info(ctx, "Starting service", "service", serviceName)

	err := systemd.Send(serviceName, "start", "replace")
	if err != nil {
		logger.Error(ctx, "Failed to start service", "error", err, "service", serviceName)
		return fmt.Errorf("failed to start service: %w", err)
	}

	logger.Info(ctx, "Service started successfully", "service", serviceName)
	return nil
}

// GetLogs 获取服务日志
func (s *service) GetLogs(ctx context.Context, serviceName string, lines int) ([]logs.LogEntry, error) {
	if err := validator.ValidateServiceName(serviceName); err != nil {
		logger.Error(ctx, "GetLogs validation failed", "error", err, "service", serviceName)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	logger.Debug(ctx, "Getting service logs", "service", serviceName, "lines", lines)

	logEntries, err := logs.GetServiceLogs(ctx, serviceName, lines)
	if err != nil {
		logger.Error(ctx, "Failed to get service logs", "error", err, "service", serviceName)
		return nil, fmt.Errorf("failed to get service logs: %w", err)
	}

	return logEntries, nil
}

// sendCallbackNotification 发送回调通知
func (s *service) sendCallbackNotification(ctx context.Context, serviceName, eventType string, config *hooks.CallbackConfig) {
	payload := map[string]interface{}{
		"service_name": serviceName,
		"event_type":   eventType,
		"timestamp":    time.Now(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error(ctx, "Failed to marshal callback payload", "error", err)
		return
	}

	client := &http.Client{Timeout: config.Timeout}
	req, err := http.NewRequestWithContext(ctx, config.Method, config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error(ctx, "Failed to create callback request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error(ctx, "Callback request failed", "error", err, "url", config.URL)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger.Info(ctx, "Callback notification sent successfully", "service", serviceName, "event", eventType)
	} else {
		logger.Warn(ctx, "Callback returned non-success status", "status", resp.StatusCode, "service", serviceName)
	}
}

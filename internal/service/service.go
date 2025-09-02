package service

import (
	"api-systemd/internal/pkg/artifact"
	"api-systemd/internal/pkg/hooks"
	"api-systemd/internal/pkg/logger"
	"api-systemd/internal/pkg/logs"
	"api-systemd/internal/pkg/systemd"
	"api-systemd/internal/pkg/telemetry"
	"api-systemd/internal/pkg/validator"
	"api-systemd/internal/pkg/workspace"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	// ListServices 获取服务列表
	ListServices(ctx context.Context) ([]ServiceInfo, error)
}

type service struct {
	mu           sync.RWMutex // 添加读写锁以支持并发控制
	hookExecutor hooks.HookExecutorInterface
	otelReporter *telemetry.OTELReporter
	workspaceMgr *workspace.Manager
	artifactMgr  *artifact.Manager
}

func NewService(workDir string) Service {
	workspaceMgr := workspace.NewManager(workDir)

	// 初始化工作空间
	if err := workspaceMgr.InitWorkspace(); err != nil {
		// 记录错误但不阻止服务启动
		fmt.Printf("Warning: failed to initialize workspace: %v\n", err)
	}

	return &service{
		hookExecutor: hooks.NewHookExecutor(),
		workspaceMgr: workspaceMgr,
		artifactMgr:  artifact.NewManager(),
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

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name        string `json:"name"`        // 服务名称
	Status      string `json:"status"`      // 服务状态
	Description string `json:"description"` // 服务描述
	Path        string `json:"path"`        // 服务路径
	Enabled     bool   `json:"enabled"`     // 是否启用
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

	// 创建服务和日志目录
	serviceDir, err := s.workspaceMgr.EnsureServiceDir(params.Service)
	if err != nil {
		logger.Error(ctx, "Failed to create service directory", "error", err, "service", params.Service)
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	logDir, err := s.workspaceMgr.EnsureLogDir(params.Service)
	if err != nil {
		logger.Error(ctx, "Failed to create log directory", "error", err, "service", params.Service)
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logger.Info(ctx, "Created service directories",
		"service", params.Service,
		"serviceDir", serviceDir,
		"logDir", logDir)

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

	// 验证URL格式
	if err := s.artifactMgr.ValidateURL(params.PackageURL); err != nil {
		logger.Error(ctx, "Invalid package URL", "error", err, "url", params.PackageURL)
		return fmt.Errorf("invalid package URL: %w", err)
	}

	// 下载并解压产物
	folders, err := s.artifactMgr.DownloadAndExtract(params.PackageURL, serviceDir)
	if err != nil {
		logger.Error(ctx, "Failed to download and extract artifact", "error", err, "url", params.PackageURL)
		return fmt.Errorf("failed to download and extract artifact: %w", err)
	}

	// 获取解压后的第一个文件夹
	folder := s.artifactMgr.GetFirstFolder(folders)
	if len(folder) == 0 {
		logger.Error(ctx, "No folders extracted from package")
		return fmt.Errorf("failed to extract folder name")
	}

	// 创建服务配置
	var config *hooks.ServiceConfig
	if params.Config != nil {
		config = params.Config
		config.ServiceName = params.Service
		config.WorkingDirectory = filepath.Join(serviceDir, folder)
		config.ExecStart = filepath.Join(serviceDir, folder, params.StartCommand)
	} else {
		config = &hooks.ServiceConfig{
			ServiceName:      params.Service,
			Description:      fmt.Sprintf("%s Service", params.Service),
			WorkingDirectory: filepath.Join(serviceDir, folder),
			ExecStart:        filepath.Join(serviceDir, folder, params.StartCommand),
			RestartPolicy:    "always",
			Hooks:            []hooks.Hook{},
		}
	}

	// 设置日志目录环境变量
	if config.Environment == nil {
		config.Environment = make(map[string]string)
	}
	config.Environment["LOG_DIR"] = logDir

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
			"service_dir": serviceDir,
			"log_dir":     logDir,
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

	// Step 5: Clean up service directories
	if err := s.workspaceMgr.CleanupService(serviceName); err != nil {
		logger.Warn(ctx, "Failed to cleanup service directories", "error", err, "service", serviceName)
		// 不返回错误，因为systemd服务已经成功删除
	} else {
		logger.Info(ctx, "Service directories cleaned up", "service", serviceName)
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

// ListServices 获取服务列表（过滤掉系统服务，只显示通过API部署的服务）
func (s *service) ListServices(ctx context.Context) ([]ServiceInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logger.Info(ctx, "Listing services")

	// 通过systemd D-Bus获取所有服务单元
	units, err := systemd.ListUnits(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to list systemd units", "error", err)
		return nil, fmt.Errorf("failed to list systemd units: %w", err)
	}

	var services []ServiceInfo

	// 过滤出通过API部署的服务（通常在/etc/systemd/system/目录下，且不是系统内置服务）
	for _, unit := range units {
		// 只处理.service类型的单元
		if !strings.HasSuffix(unit.Name, ".service") {
			continue
		}

		// 过滤掉系统服务，只保留用户部署的服务
		if isSystemService(unit.Name) {
			continue
		}

		// 检查服务文件是否在我们管理的目录中
		serviceName := strings.TrimSuffix(unit.Name, ".service")
		serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

		// 检查文件是否存在且可读
		if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
			continue
		}

		// 获取服务详细信息
		serviceInfo := ServiceInfo{
			Name:        serviceName,
			Status:      unit.ActiveState,
			Description: unit.Description,
			Path:        serviceFile,
			Enabled:     unit.UnitFileState == "enabled",
		}

		services = append(services, serviceInfo)
	}

	logger.Info(ctx, "Services filtered successfully", "total_units", len(units), "filtered_services", len(services))
	return services, nil
}

// isSystemService 判断是否为系统服务
func isSystemService(serviceName string) bool {
	// 系统内置服务列表（常见的系统服务）
	systemServices := []string{
		"systemd-", "dbus", "NetworkManager", "sshd", "chronyd", "rsyslog",
		"firewalld", "auditd", "crond", "atd", "cups", "avahi-daemon",
		"bluetooth", "wpa_supplicant", "ModemManager", "accounts-daemon",
		"polkit", "udisks2", "colord", "rtkit-daemon", "upower",
		"gdm", "lightdm", "sddm", "getty@", "user@", "session-",
		"NetworkManager-", "systemd", "kernel", "kthread", "migration",
		"rcu_", "watchdog", "ksoftirqd", "systemd-resolved", "systemd-networkd",
		"systemd-timesyncd", "systemd-logind", "systemd-machined", "systemd-importd",
		"systemd-hostnamed", "systemd-localed", "systemd-timedated",
		"api-systemd", // 排除自己
	}

	// 检查是否匹配系统服务前缀
	for _, prefix := range systemServices {
		if strings.HasPrefix(serviceName, prefix) {
			return true
		}
	}

	return false
}

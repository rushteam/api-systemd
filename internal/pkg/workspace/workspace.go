package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager 工作空间管理器
type Manager struct {
	workDir string
}

// NewManager 创建工作空间管理器
func NewManager(workDir string) *Manager {
	return &Manager{
		workDir: workDir,
	}
}

// InitWorkspace 初始化工作空间
func (m *Manager) InitWorkspace() error {
	// 创建根工作目录
	if err := os.MkdirAll(m.workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory %s: %w", m.workDir, err)
	}

	// 创建services目录
	servicesDir := filepath.Join(m.workDir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		return fmt.Errorf("failed to create services directory %s: %w", servicesDir, err)
	}

	// 创建logs目录
	logsDir := filepath.Join(m.workDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory %s: %w", logsDir, err)
	}

	return nil
}

// GetServiceDir 获取服务目录路径
func (m *Manager) GetServiceDir(serviceName string) string {
	return filepath.Join(m.workDir, "services", serviceName)
}

// GetLogDir 获取日志目录路径
func (m *Manager) GetLogDir(serviceName string) string {
	return filepath.Join(m.workDir, "logs", serviceName)
}

// EnsureServiceDir 确保服务目录存在
func (m *Manager) EnsureServiceDir(serviceName string) (string, error) {
	serviceDir := m.GetServiceDir(serviceName)
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create service directory %s: %w", serviceDir, err)
	}
	return serviceDir, nil
}

// EnsureLogDir 确保日志目录存在
func (m *Manager) EnsureLogDir(serviceName string) (string, error) {
	logDir := m.GetLogDir(serviceName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}
	return logDir, nil
}

// CleanupService 清理服务相关目录
func (m *Manager) CleanupService(serviceName string) error {
	serviceDir := m.GetServiceDir(serviceName)
	logDir := m.GetLogDir(serviceName)

	// 删除服务目录
	if err := os.RemoveAll(serviceDir); err != nil {
		return fmt.Errorf("failed to remove service directory %s: %w", serviceDir, err)
	}

	// 删除日志目录
	if err := os.RemoveAll(logDir); err != nil {
		return fmt.Errorf("failed to remove log directory %s: %w", logDir, err)
	}

	return nil
}

// GetWorkDir 获取工作目录根路径
func (m *Manager) GetWorkDir() string {
	return m.workDir
}

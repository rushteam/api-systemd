package artifact

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Manager 产物管理器
type Manager struct{}

// NewManager 创建产物管理器
func NewManager() *Manager {
	return &Manager{}
}

// DownloadAndExtract 下载并解压产物到指定目录
func (m *Manager) DownloadAndExtract(url, targetDir string) ([]string, error) {
	// 1. 下载文件
	tempFile, err := m.downloadFile(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer os.Remove(tempFile) // 清理临时文件

	// 2. 解压文件
	folders, err := m.extractFile(tempFile, targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract file: %w", err)
	}

	return folders, nil
}

// downloadFile 下载文件到临时目录
func (m *Manager) downloadFile(url string) (string, error) {
	// 发送HTTP请求
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download file from %s: %w", url, err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	// 从URL推断文件扩展名
	ext := ""
	if strings.Contains(url, ".tar.gz") {
		ext = ".tar.gz"
	} else if strings.Contains(url, ".zip") {
		ext = ".zip"
	} else if strings.Contains(url, ".tar") {
		ext = ".tar"
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "artifact-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// 将响应内容写入临时文件
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return tempFile.Name(), nil
}

// extractFile 解压文件到目标目录
func (m *Manager) extractFile(filePath, targetDir string) ([]string, error) {
	// 确保目标目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	// 根据文件扩展名选择解压方法
	if strings.HasSuffix(filePath, ".zip") {
		return m.extractZip(filePath, targetDir)
	} else if strings.HasSuffix(filePath, ".tar.gz") || strings.HasSuffix(filePath, ".tar") {
		return m.extractTar(filePath, targetDir)
	}

	return nil, fmt.Errorf("unsupported file format: %s", filePath)
}

// extractZip 解压ZIP文件
func (m *Manager) extractZip(filePath, targetDir string) ([]string, error) {
	return extractZipFile(filePath, targetDir)
}

// extractTar 解压TAR文件
func (m *Manager) extractTar(filePath, targetDir string) ([]string, error) {
	return extractTarFile(filePath, targetDir)
}

// GetFirstFolder 获取解压后的第一个文件夹名
func (m *Manager) GetFirstFolder(folders []string) string {
	if len(folders) > 0 {
		return folders[0]
	}
	return ""
}

// ValidateURL 验证URL格式
func (m *Manager) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	// 检查是否是支持的文件格式
	supportedFormats := []string{".zip", ".tar.gz", ".tar"}
	supported := false
	for _, format := range supportedFormats {
		if strings.Contains(url, format) {
			supported = true
			break
		}
	}

	if !supported {
		return fmt.Errorf("unsupported file format, supported formats: %v", supportedFormats)
	}

	return nil
}

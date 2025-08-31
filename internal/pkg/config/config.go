package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig    `json:"server"`
	Security  SecurityConfig  `json:"security"`
	Logging   LoggingConfig   `json:"logging"`
	Workspace WorkspaceConfig `json:"workspace"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port            string        `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableAuth    bool     `json:"enable_auth"`
	APIKey        string   `json:"api_key"`
	AllowedHosts  []string `json:"allowed_hosts"`
	RateLimitRPS  int      `json:"rate_limit_rps"`
	MaxUploadSize int64    `json:"max_upload_size"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputFile string `json:"output_file"`
}

// WorkspaceConfig 工作空间配置
type WorkspaceConfig struct {
	WorkDir string `json:"work_dir"` // 工作目录根路径
}

// Load 加载配置
func Load() *Config {
	apiKey := getEnv("API_KEY", "")

	// 如果API_KEY为空，生成临时的
	if apiKey == "" {
		apiKey = generateTempAPIKey()
		fmt.Printf("\n🔑 API_KEY 未设置，已生成临时密钥:\n")
		fmt.Printf("   API_KEY: %s\n", apiKey)
		fmt.Printf("   请使用此密钥进行API认证: Authorization: Bearer %s\n", apiKey)
		fmt.Printf("   建议在生产环境中设置固定的 API_KEY 环境变量\n\n")
	}

	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", ":8080"),
			ReadTimeout:     getDurationEnv("READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Security: SecurityConfig{
			EnableAuth:    true, // 强制启用认证
			APIKey:        apiKey,
			AllowedHosts:  getStringSliceEnv("ALLOWED_HOSTS", []string{"*"}),
			RateLimitRPS:  getIntEnv("RATE_LIMIT_RPS", 100),
			MaxUploadSize: getInt64Env("MAX_UPLOAD_SIZE", 100*1024*1024), // 100MB
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			OutputFile: getEnv("LOG_OUTPUT_FILE", ""),
		},
		Workspace: WorkspaceConfig{
			WorkDir: getEnv("WORK_DIR", "/opt/api-systemd"),
		},
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnv 获取布尔类型环境变量
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// getIntEnv 获取整数类型环境变量
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getInt64Env 获取int64类型环境变量
func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

// getDurationEnv 获取时间间隔类型环境变量
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

// getStringSliceEnv 获取字符串切片类型环境变量
func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// 简化实现，实际可以使用更复杂的分割逻辑
		return []string{value}
	}
	return defaultValue
}

// generateTempAPIKey 生成临时API密钥
func generateTempAPIKey() string {
	// 生成32字节的随机数据
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// 如果随机数生成失败，使用时间戳作为后备方案
		return fmt.Sprintf("tmp-api-key-%d", time.Now().Unix())
	}

	// 转换为十六进制字符串
	return "tmp-" + hex.EncodeToString(bytes)
}

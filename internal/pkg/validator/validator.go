package validator

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	ErrEmptyServiceName   = errors.New("service name cannot be empty")
	ErrInvalidServiceName = errors.New("service name contains invalid characters")
	ErrEmptyPath          = errors.New("path cannot be empty")
	ErrInvalidURL         = errors.New("invalid package URL")
	ErrEmptyStartCommand  = errors.New("start command cannot be empty")
)

// ValidateServiceName 验证服务名称
func ValidateServiceName(serviceName string) error {
	if strings.TrimSpace(serviceName) == "" {
		return ErrEmptyServiceName
	}

	// 服务名只允许字母、数字、连字符和下划线
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, serviceName)
	if !matched {
		return ErrInvalidServiceName
	}

	return nil
}

// ValidatePath 验证路径
func ValidatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return ErrEmptyPath
	}

	// 路径必须是绝对路径
	if !strings.HasPrefix(path, "/") {
		return errors.New("path must be absolute")
	}

	return nil
}

// ValidateURL 验证URL
func ValidateURL(rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return ErrInvalidURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL must use http or https scheme")
	}

	return nil
}

// ValidateStartCommand 验证启动命令
func ValidateStartCommand(startCmd string) error {
	if strings.TrimSpace(startCmd) == "" {
		return ErrEmptyStartCommand
	}

	return nil
}

// ValidateDeployParams 验证部署参数
func ValidateDeployParams(service, path, packageURL, start string) error {
	if err := ValidateServiceName(service); err != nil {
		return err
	}

	if err := ValidatePath(path); err != nil {
		return err
	}

	if err := ValidateURL(packageURL); err != nil {
		return err
	}

	if err := ValidateStartCommand(start); err != nil {
		return err
	}

	return nil
}

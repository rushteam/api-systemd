package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"api-systemd/internal/pkg/hooks"
)

// 统一的 systemd 模板，支持简单和复杂配置
const systemdTpl = `[Unit]
Description={{.Description}}
{{- if .After}}
After={{join .After " "}}
{{- end}}
{{- if .Before}}
Before={{join .Before " "}}
{{- end}}
{{- if .Requires}}
Requires={{join .Requires " "}}
{{- end}}
{{- if .Wants}}
Wants={{join .Wants " "}}
{{- end}}

[Service]
Type=simple
{{- if .User}}
User={{.User}}
{{- end}}
{{- if .Group}}
Group={{.Group}}
{{- end}}
WorkingDirectory={{.WorkingDirectory}}
{{- if .Environment}}
{{- range $key, $value := .Environment}}
Environment="{{$key}}={{$value}}"
{{- end}}
{{- end}}
{{- if .PreStartHooks}}
{{- range .PreStartHooks}}
ExecStartPre={{.}}
{{- end}}
{{- end}}
ExecStart={{.ExecStart}}
{{- if .PostStartHooks}}
{{- range .PostStartHooks}}
ExecStartPost={{.}}
{{- end}}
{{- end}}
{{- if .PreStopHooks}}
{{- range .PreStopHooks}}
ExecStopPre={{.}}
{{- end}}
{{- end}}
{{- if .PostStopHooks}}
{{- range .PostStopHooks}}
ExecStopPost={{.}}
{{- end}}
{{- end}}
Restart={{.RestartPolicy}}
{{- if .RestartDelaySec}}
RestartSec={{.RestartDelaySecValue}}
{{- end}}
{{- if .StartLimitBurst}}
StartLimitBurst={{.StartLimitBurst}}
{{- end}}
{{- if .MemoryLimit}}
MemoryMax={{.MemoryLimit}}
{{- end}}
{{- if .CPUQuota}}
CPUQuota={{.CPUQuota}}
{{- end}}
{{- if .TasksMax}}
TasksMax={{.TasksMax}}
{{- end}}

[Install]
WantedBy=multi-user.target
`

// SystemdConfig 统一的 systemd 配置结构
type SystemdConfig struct {
	*hooks.ServiceConfig
	PreStartHooks        []string
	PostStartHooks       []string
	PreStopHooks         []string
	PostStopHooks        []string
	RestartDelaySecValue int
}

// NewSystemdConfig 创建 systemd 配置
// 如果传入完整的 ServiceConfig，则使用增强配置
// 如果传入简单参数，则创建基本配置
func NewSystemdConfig(serviceName, workingDirectory, startCmd string, serviceConfig ...*hooks.ServiceConfig) *SystemdConfig {
	var config *hooks.ServiceConfig

	if len(serviceConfig) > 0 && serviceConfig[0] != nil {
		// 使用传入的增强配置
		config = serviceConfig[0]
	} else {
		// 创建基本配置
		config = &hooks.ServiceConfig{
			ServiceName:      serviceName,
			Description:      fmt.Sprintf("%s Service", serviceName),
			WorkingDirectory: workingDirectory,
			ExecStart:        filepath.Join(workingDirectory, startCmd),
			RestartPolicy:    "always",
		}
	}

	systemdConfig := &SystemdConfig{
		ServiceConfig:        config,
		RestartDelaySecValue: int(config.RestartDelaySec.Seconds()),
	}

	// 从钩子中提取 systemd 原生命令
	for _, hook := range config.Hooks {
		if hook.Command == "" {
			continue
		}

		switch hook.Type {
		case hooks.HookPreStart:
			systemdConfig.PreStartHooks = append(systemdConfig.PreStartHooks, hook.Command)
		case hooks.HookPostStart:
			systemdConfig.PostStartHooks = append(systemdConfig.PostStartHooks, hook.Command)
		case hooks.HookPreStop:
			systemdConfig.PreStopHooks = append(systemdConfig.PreStopHooks, hook.Command)
		case hooks.HookPostStop:
			systemdConfig.PostStopHooks = append(systemdConfig.PostStopHooks, hook.Command)
		}
	}

	return systemdConfig
}

// WriteFile 写入 systemd 配置文件
func (sc *SystemdConfig) WriteFile(filename string) error {
	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	tmpl, err := template.New("systemd").Funcs(funcMap).Parse(systemdTpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, sc)
}

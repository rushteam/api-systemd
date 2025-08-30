package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"api-systemd/internal/pkg/hooks"
)

// Define the enhanced systemd template
const enhancedSystemdTpl = `[Unit]
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

// Struct to hold the template data (legacy)
type SystemdConfig struct {
	ServiceName      string
	WorkingDirectory string
	ExecStart        string
}

// EnhancedSystemdConfig 增强的systemd配置
type EnhancedSystemdConfig struct {
	*hooks.ServiceConfig
	PreStartHooks        []string
	PostStartHooks       []string
	PreStopHooks         []string
	PostStopHooks        []string
	RestartDelaySecValue int
}

func NewSystemdConfig(serviceName, workingDirectory, startCmd string) *SystemdConfig {
	return &SystemdConfig{
		ServiceName:      serviceName,
		WorkingDirectory: workingDirectory,
		ExecStart:        filepath.Join(workingDirectory, startCmd),
	}
}

// NewEnhancedSystemdConfig 创建增强的systemd配置
func NewEnhancedSystemdConfig(config *hooks.ServiceConfig) *EnhancedSystemdConfig {
	enhanced := &EnhancedSystemdConfig{
		ServiceConfig:        config,
		RestartDelaySecValue: int(config.RestartDelaySec.Seconds()),
	}

	// 从钩子中提取systemd原生命令
	for _, hook := range config.Hooks {
		if hook.Command == "" {
			continue
		}

		switch hook.Type {
		case hooks.HookPreStart:
			enhanced.PreStartHooks = append(enhanced.PreStartHooks, hook.Command)
		case hooks.HookPostStart:
			enhanced.PostStartHooks = append(enhanced.PostStartHooks, hook.Command)
		case hooks.HookPreStop:
			enhanced.PreStopHooks = append(enhanced.PreStopHooks, hook.Command)
		case hooks.HookPostStop:
			enhanced.PostStopHooks = append(enhanced.PostStopHooks, hook.Command)
		}
	}

	return enhanced
}

func (s *SystemdConfig) WriteFile(filename string) error {
	tmpl, err := template.New("systemd").Parse(systemdTpl)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, s)
}

// WriteFile 写入增强的systemd配置文件
func (esc *EnhancedSystemdConfig) WriteFile(filename string) error {
	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	tmpl, err := template.New("systemd").Funcs(funcMap).Parse(enhancedSystemdTpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, esc)
}

// 保持向后兼容的简单模板
const systemdTpl = `[Unit]
Description={{.ServiceName}} Service

[Service]
WorkingDirectory={{.WorkingDirectory}}
ExecStart={{.ExecStart}}
Restart=always

[Install]
WantedBy=multi-user.target
`

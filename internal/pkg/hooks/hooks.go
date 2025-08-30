package hooks

import (
	"context"
	"time"
)

// HookType 钩子类型
type HookType string

const (
	HookPreStart    HookType = "pre_start"    // 启动前
	HookPostStart   HookType = "post_start"   // 启动后
	HookPreStop     HookType = "pre_stop"     // 停止前
	HookPostStop    HookType = "post_stop"    // 停止后
	HookPreRestart  HookType = "pre_restart"  // 重启前
	HookPostRestart HookType = "post_restart" // 重启后
	HookOnFailure   HookType = "on_failure"   // 失败时
	HookOnSuccess   HookType = "on_success"   // 成功时
)

// Hook 钩子配置
type Hook struct {
	Type        HookType               `json:"type"`
	Name        string                 `json:"name"`
	Command     string                 `json:"command,omitempty"`      // 执行命令
	Script      string                 `json:"script,omitempty"`       // 执行脚本路径
	CallbackURL string                 `json:"callback_url,omitempty"` // 回调URL
	Headers     map[string]string      `json:"headers,omitempty"`      // HTTP头
	Payload     map[string]interface{} `json:"payload,omitempty"`      // 载荷数据
	Timeout     time.Duration          `json:"timeout"`                // 超时时间
	Retry       int                    `json:"retry"`                  // 重试次数
	Enabled     bool                   `json:"enabled"`                // 是否启用
	Async       bool                   `json:"async"`                  // 是否异步执行
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	OTEL     *OTELConfig     `json:"otel,omitempty"`
	Callback *CallbackConfig `json:"callback,omitempty"`
	Webhook  *WebhookConfig  `json:"webhook,omitempty"`
}

// OTELConfig OpenTelemetry配置
type OTELConfig struct {
	Enabled     bool              `json:"enabled"`
	Endpoint    string            `json:"endpoint"`
	Headers     map[string]string `json:"headers"`
	ServiceName string            `json:"service_name"`
	Attributes  map[string]string `json:"attributes"`
}

// CallbackConfig 回调配置
type CallbackConfig struct {
	Enabled bool              `json:"enabled"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Timeout time.Duration     `json:"timeout"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	Enabled bool              `json:"enabled"`
	URL     string            `json:"url"`
	Secret  string            `json:"secret"`
	Headers map[string]string `json:"headers"`
}

// ServiceConfig 增强的服务配置
type ServiceConfig struct {
	// 基本配置
	ServiceName      string            `json:"service_name"`
	Description      string            `json:"description"`
	WorkingDirectory string            `json:"working_directory"`
	ExecStart        string            `json:"exec_start"`
	User             string            `json:"user,omitempty"`
	Group            string            `json:"group,omitempty"`
	Environment      map[string]string `json:"environment,omitempty"`

	// 重启策略
	RestartPolicy   string        `json:"restart_policy"` // no, always, on-failure
	RestartDelaySec time.Duration `json:"restart_delay_sec"`
	StartLimitBurst int           `json:"start_limit_burst"`

	// 资源限制
	MemoryLimit string `json:"memory_limit,omitempty"` // 如: "1G"
	CPUQuota    string `json:"cpu_quota,omitempty"`    // 如: "50%"
	TasksMax    int    `json:"tasks_max,omitempty"`

	// 依赖关系
	After    []string `json:"after,omitempty"`
	Before   []string `json:"before,omitempty"`
	Requires []string `json:"requires,omitempty"`
	Wants    []string `json:"wants,omitempty"`

	// 生命周期钩子
	Hooks []Hook `json:"hooks"`

	// 通知配置
	Notifications NotificationConfig `json:"notifications"`

	// 健康检查
	HealthCheck *HealthCheckConfig `json:"health_check,omitempty"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	Command          string        `json:"command"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	StartPeriod      time.Duration `json:"start_period"`
	Retries          int           `json:"retries"`
	SuccessThreshold int           `json:"success_threshold"`
}

// HookEvent 钩子事件
type HookEvent struct {
	ServiceName string                 `json:"service_name"`
	HookType    HookType               `json:"hook_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      string                 `json:"status"` // success, failure, timeout
	Duration    time.Duration          `json:"duration"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// HookExecutorInterface 钩子执行器接口
type HookExecutorInterface interface {
	ExecuteHook(ctx context.Context, hook Hook, serviceName string, metadata map[string]interface{}) *HookEvent
	ExecuteHooks(ctx context.Context, hooks []Hook, hookType HookType, serviceName string, metadata map[string]interface{}) []*HookEvent
}

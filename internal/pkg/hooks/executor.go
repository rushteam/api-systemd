package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"api-systemd/internal/pkg/logger"
)

// HookExecutor 钩子执行器
type HookExecutor struct {
	client *http.Client
}

// NewHookExecutor 创建钩子执行器
func NewHookExecutor() *HookExecutor {
	return &HookExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteHook 执行钩子
func (he *HookExecutor) ExecuteHook(ctx context.Context, hook Hook, serviceName string, metadata map[string]interface{}) *HookEvent {
	event := &HookEvent{
		ServiceName: serviceName,
		HookType:    hook.Type,
		Timestamp:   time.Now(),
		Metadata:    metadata,
	}

	start := time.Now()
	defer func() {
		event.Duration = time.Since(start)
	}()

	// 设置超时上下文
	if hook.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, hook.Timeout)
		defer cancel()
	}

	// 执行重试逻辑
	maxRetries := hook.Retry
	if maxRetries <= 0 {
		maxRetries = 1
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			logger.Info(ctx, "Retrying hook execution", "service", serviceName, "hook", hook.Name, "attempt", i+1)
			time.Sleep(time.Second * time.Duration(i)) // 指数退避
		}

		// 根据钩子类型执行不同操作
		if hook.Command != "" {
			lastErr = he.executeCommand(ctx, hook, event)
		} else if hook.Script != "" {
			lastErr = he.executeScript(ctx, hook, event)
		} else if hook.CallbackURL != "" {
			lastErr = he.executeCallback(ctx, hook, event, metadata)
		}

		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		event.Status = "failure"
		event.Error = lastErr.Error()
	}

	return event
}

// executeCommand 执行命令
func (he *HookExecutor) executeCommand(ctx context.Context, hook Hook, event *HookEvent) error {
	// 替换变量
	command := strings.ReplaceAll(hook.Command, "${SERVICE_NAME}", event.ServiceName)
	command = strings.ReplaceAll(command, "${HOOK_TYPE}", string(event.HookType))

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()

	event.Output = string(output)
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	event.Status = "success"
	return nil
}

// executeScript 执行脚本
func (he *HookExecutor) executeScript(ctx context.Context, hook Hook, event *HookEvent) error {
	cmd := exec.CommandContext(ctx, hook.Script, event.ServiceName, string(event.HookType))
	output, err := cmd.CombinedOutput()

	event.Output = string(output)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	event.Status = "success"
	return nil
}

// executeCallback 执行回调
func (he *HookExecutor) executeCallback(ctx context.Context, hook Hook, event *HookEvent, metadata map[string]interface{}) error {
	payload := map[string]interface{}{
		"service_name": event.ServiceName,
		"hook_type":    string(event.HookType),
		"timestamp":    event.Timestamp,
		"metadata":     metadata,
	}

	// 合并自定义载荷
	for k, v := range hook.Payload {
		payload[k] = v
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hook.CallbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range hook.Headers {
		req.Header.Set(k, v)
	}

	resp, err := he.client.Do(req)
	if err != nil {
		return fmt.Errorf("callback request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		event.Status = "success"
		return nil
	}

	return fmt.Errorf("callback returned status %d", resp.StatusCode)
}

// ExecuteHooks 执行多个钩子
func (he *HookExecutor) ExecuteHooks(ctx context.Context, hooks []Hook, hookType HookType, serviceName string, metadata map[string]interface{}) []*HookEvent {
	syncHooks := make(chan Hook)
	syncEvents := make(chan *HookEvent)
	syncHookCount := 0
	go func() {
		// 处理同步执行的hook
		for hook := range syncHooks {
			event := he.ExecuteHook(ctx, hook, serviceName, metadata)
			logger.Info(ctx, "Hook executed",
				"service", serviceName,
				"hook_type", string(hookType),
				"hook_name", hook.Name,
				"status", event.Status,
				"duration", event.Duration)
			syncEvents <- event
		}
	}()

	for _, hook := range hooks {
		if !hook.Enabled || hook.Type != hookType {
			continue
		}
		if hook.Async {
			// 异步执行
			go func(h Hook) {
				event := he.ExecuteHook(ctx, h, serviceName, metadata)
				logger.Info(ctx, "Hook executed asynchronously",
					"service", serviceName,
					"hook_type", string(hookType),
					"hook_name", h.Name,
					"status", event.Status,
					"duration", event.Duration)
			}(hook)
		} else {
			// 同步执行
			syncHookCount++
			syncHooks <- hook
		}
	}
	close(syncHooks)

	var events []*HookEvent
	for i := 0; i < syncHookCount; i++ {
		event := <-syncEvents
		events = append(events, event)
	}
	return events
}

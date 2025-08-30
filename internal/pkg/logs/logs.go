package logs

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Level     string `json:"level"`
}

// GetServiceLogs 获取服务日志
func GetServiceLogs(ctx context.Context, serviceName string, lines int) ([]LogEntry, error) {
	var args []string
	args = append(args, "-u", serviceName)

	if lines > 0 {
		args = append(args, "-n", strconv.Itoa(lines))
	}

	args = append(args, "--no-pager", "--output=json")

	cmd := exec.CommandContext(ctx, "journalctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	return parseJournalOutput(string(output))
}

// GetServiceLogsFollow 实时跟踪服务日志
func GetServiceLogsFollow(ctx context.Context, serviceName string) (<-chan LogEntry, <-chan error) {
	logChan := make(chan LogEntry, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(logChan)
		defer close(errChan)

		cmd := exec.CommandContext(ctx, "journalctl", "-u", serviceName, "-f", "--no-pager", "--output=json")

		if err := cmd.Start(); err != nil {
			errChan <- err
			return
		}

		// 这里可以添加实时日志解析逻辑
		// 简化版本，实际使用时需要更复杂的解析

		if err := cmd.Wait(); err != nil {
			errChan <- err
		}
	}()

	return logChan, errChan
}

// parseJournalOutput 解析journalctl JSON输出
func parseJournalOutput(output string) ([]LogEntry, error) {
	var entries []LogEntry

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 简化的解析，实际应该使用JSON解析
		entry := LogEntry{
			Timestamp: "unknown",
			Message:   line,
			Level:     "info",
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

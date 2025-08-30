package telemetry

import (
	"context"
	"fmt"

	"api-systemd/internal/pkg/hooks"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// OTELReporter OTEL上报器
type OTELReporter struct {
	tracer  oteltrace.Tracer
	config  hooks.OTELConfig
	enabled bool
}

// NewOTELReporter 创建OTEL上报器
func NewOTELReporter(config hooks.OTELConfig) (*OTELReporter, error) {
	if !config.Enabled {
		return &OTELReporter{enabled: false}, nil
	}

	// 创建OTLP HTTP导出器
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(config.Endpoint),
		otlptracehttp.WithHeaders(config.Headers),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// 创建资源
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(config.ServiceName),
		semconv.ServiceVersionKey.String("1.0.0"),
	)

	// 添加自定义属性
	attrs := make([]attribute.KeyValue, 0, len(config.Attributes))
	for k, v := range config.Attributes {
		attrs = append(attrs, attribute.String(k, v))
	}
	res, _ = resource.Merge(res, resource.NewWithAttributes(semconv.SchemaURL, attrs...))

	// 创建trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("api-systemd")

	return &OTELReporter{
		tracer:  tracer,
		config:  config,
		enabled: true,
	}, nil
}

// ReportServiceEvent 上报服务事件
func (or *OTELReporter) ReportServiceEvent(ctx context.Context, serviceName, eventType string, metadata map[string]interface{}) {
	if !or.enabled {
		return
	}

	_, span := or.tracer.Start(ctx, fmt.Sprintf("service.%s", eventType))
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("event.type", eventType),
		attribute.String("component", "systemd"),
		attribute.String("system", "api-systemd"),
	)

	// 添加元数据属性
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			span.SetAttributes(attribute.String(fmt.Sprintf("metadata.%s", k), str))
		} else if num, ok := v.(int); ok {
			span.SetAttributes(attribute.Int(fmt.Sprintf("metadata.%s", k), num))
		} else if b, ok := v.(bool); ok {
			span.SetAttributes(attribute.Bool(fmt.Sprintf("metadata.%s", k), b))
		}
	}
}

// ReportHookExecution 上报钩子执行
func (or *OTELReporter) ReportHookExecution(ctx context.Context, hookEvent *hooks.HookEvent) {
	if !or.enabled {
		return
	}

	_, span := or.tracer.Start(ctx, fmt.Sprintf("hook.%s", string(hookEvent.HookType)))
	defer span.End()

	span.SetAttributes(
		attribute.String("service.name", hookEvent.ServiceName),
		attribute.String("hook.type", string(hookEvent.HookType)),
		attribute.String("hook.status", hookEvent.Status),
		attribute.Int64("hook.duration_ms", hookEvent.Duration.Milliseconds()),
	)

	if hookEvent.Error != "" {
		span.SetAttributes(
			attribute.String("hook.error", hookEvent.Error),
			attribute.Bool("hook.failed", true),
		)
	}

	if hookEvent.Output != "" {
		span.SetAttributes(attribute.String("hook.output", hookEvent.Output))
	}

	// 添加元数据
	for k, v := range hookEvent.Metadata {
		if str, ok := v.(string); ok {
			span.SetAttributes(attribute.String(fmt.Sprintf("hook.metadata.%s", k), str))
		}
	}
}

// Close 关闭OTEL上报器
func (or *OTELReporter) Close(ctx context.Context) error {
	if !or.enabled {
		return nil
	}

	// 获取TracerProvider并关闭
	if tp, ok := otel.GetTracerProvider().(*trace.TracerProvider); ok {
		return tp.Shutdown(ctx)
	}

	return nil
}

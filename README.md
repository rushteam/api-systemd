# API-Systemd 服务管理系统

一个现代化的 systemd 服务管理 API，提供完整的服务生命周期管理和增强配置功能。

## 🚀 特性

### 核心功能
- **服务部署**: 自动下载、解压、配置和启动服务
- **生命周期管理**: 启动、停止、重启、移除服务
- **状态监控**: 获取服务状态和日志
- **配置管理**: 动态创建和删除 systemd 配置

### 增强功能
- **生命周期钩子**: 支持 pre/post 启动、停止、重启钩子
- **多种钩子类型**: 命令执行、脚本运行、HTTP 回调
- **通知集成**: OTEL 上报、Webhook 通知
- **高级配置**: 资源限制、环境变量、依赖管理
- **并发安全**: 内置读写锁保护

### 系统特性
- **D-Bus 集成**: 直接与 systemd 通信，无需 shell 调用
- **结构化日志**: 使用 slog 提供详细的操作日志
- **中间件支持**: 恢复、日志、CORS 中间件
- **优雅关闭**: 支持信号处理和优雅停机
- **健康检查**: 内置系统健康状态检查

## 📡 API 接口

### 服务管理
```
POST   /services/deploy    # 部署服务
GET    /services/start     # 启动服务
GET    /services/stop      # 停止服务
GET    /services/restart   # 重启服务
GET    /services/remove    # 移除服务
GET    /services/status    # 获取服务状态
GET    /services/logs      # 获取服务日志
```

### 配置管理
```
POST   /configs/create     # 创建配置文件
DELETE /configs/delete     # 删除配置文件
```

### 系统监控
```
GET    /health            # 健康检查
```

## 🛠️ 部署请求格式

### 基础部署
```json
{
  "service": "my-app",
  "path": "/opt/services",
  "package_url": "https://example.com/app.tar.gz",
  "start_command": "app"
}
```

### 增强部署
```json
{
  "service": "my-app",
  "path": "/opt/services", 
  "package_url": "https://example.com/app.tar.gz",
  "start_command": "app",
  "config": {
    "description": "My Application Service",
    "user": "appuser",
    "environment": {
      "NODE_ENV": "production"
    },
    "restart_policy": "always",
    "memory_limit": "1G",
    "cpu_quota": "50%"
  },
  "hooks": [
    {
      "type": "pre_start",
      "name": "database-check",
      "command": "curl -f http://db:5432/health",
      "timeout": "30s",
      "enabled": true
    },
    {
      "type": "post_start", 
      "name": "notify-slack",
      "callback_url": "https://hooks.slack.com/...",
      "async": true,
      "enabled": true
    }
  ],
  "notifications": {
    "otel": {
      "enabled": true,
      "endpoint": "http://jaeger:14268/api/traces",
      "service_name": "api-systemd"
    },
    "callback": {
      "enabled": true,
      "url": "https://api.example.com/webhooks/deployment",
      "method": "POST"
    }
  }
}
```

## 🏗️ 架构设计

### 模块结构
```
internal/
├── app/           # HTTP 处理层
├── service/       # 业务逻辑层  
├── pkg/
│   ├── hooks/     # 钩子系统
│   ├── telemetry/ # OTEL 集成
│   ├── systemd/   # D-Bus 接口
│   ├── logger/    # 结构化日志
│   ├── validator/ # 参数验证
│   ├── config/    # 配置管理
│   ├── logs/      # 日志获取
│   └── middleware/# HTTP 中间件
└── middleware/    # 中间件实现
```

### 核心组件

#### 钩子系统
- **执行器**: 支持命令、脚本、HTTP 回调
- **事件类型**: pre/post start/stop/restart, on success/failure
- **执行模式**: 同步/异步执行
- **重试机制**: 可配置重试次数和策略

#### 通知系统
- **OTEL 集成**: 分布式追踪和指标上报
- **Webhook 支持**: HTTP 回调通知
- **事件上报**: 服务状态变更、钩子执行结果

#### D-Bus 接口
- **直接通信**: 与 systemd 直接交互
- **高效操作**: 避免 shell 调用开销
- **完整功能**: 支持所有 systemctl 操作

## ⚙️ 配置

### 环境变量
```bash
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
LOG_LEVEL=info
```

### 配置文件示例
参考 `config.example.env` 文件。

## 🚀 快速开始

### 构建
```bash
go build -o api-systemd
```

### 运行
```bash
./api-systemd
```

### 健康检查
```bash
curl http://localhost:8080/health
```

### 部署服务
```bash
curl -X POST http://localhost:8080/services/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "service": "test-app",
    "path": "/opt/services",
    "package_url": "https://example.com/app.tar.gz", 
    "start_command": "app"
  }'
```

## 🔧 开发

### 依赖管理
```bash
go mod tidy
go mod download
```

### 代码检查
```bash
golangci-lint run
```

## 📋 系统要求

- Go 1.22+
- Linux 系统 (systemd)
- 足够的权限操作 systemd 服务

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

Apache License
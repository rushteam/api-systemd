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
- **Chi 路由框架**: 高性能、轻量级的 HTTP 路由器
- **RESTful API**: 支持路径参数和查询参数的灵活路由
- **强制认证**: 自动生成临时密钥或使用配置的API Key
- **Bearer Token 认证**: 安全的API访问控制
- **工作空间管理**: 自动管理服务文件和日志目录
- **中间件生态**: 请求ID、认证、恢复、日志、CORS、超时、压缩等
- **结构化日志**: 使用 slog 提供详细的操作日志
- **优雅关闭**: 支持信号处理和优雅停机
- **健康检查**: 内置系统健康状态检查
- **性能分析**: 内置 pprof 调试工具

## 📡 RESTful API 接口

### 服务管理
```
GET    /services                          # 获取服务列表
POST   /services/deploy                   # 部署新服务
GET    /services/{serviceName}/status     # 获取服务状态
GET    /services/{serviceName}/logs       # 获取服务日志 (?lines=100)
POST   /services/{serviceName}/start      # 启动服务
POST   /services/{serviceName}/stop       # 停止服务
POST   /services/{serviceName}/restart    # 重启服务
DELETE /services/{serviceName}            # 删除服务
```

### 配置管理
```
POST   /configs/                         # 创建配置文件
DELETE /configs/{serviceName}            # 删除指定服务的配置文件
```

### 系统监控
```
GET    /health            # 健康检查
GET    /ping              # 简单连通性测试
GET    /debug/            # 性能分析工具 (开发环境)
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

### 方式一：作为 systemd 服务安装（推荐）

#### 1. 构建和安装
```bash
# 构建项目
make build-local

# 安装到 systemd（需要 root 权限）
make install
```

#### 2. 服务管理
```bash
# 查看服务状态
systemctl status api-systemd

# 启动/停止/重启服务
systemctl start api-systemd
systemctl stop api-systemd
systemctl restart api-systemd

# 查看服务日志
journalctl -u api-systemd -f

# 或使用便捷命令
api-systemd-ctl status
api-systemd-ctl logs
api-systemd-ctl health
```

#### 3. 配置文件
服务配置文件位于：`/etc/api-systemd/config.env`
```bash
# 服务器配置
SERVER_PORT=:8080
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s
SHUTDOWN_TIMEOUT=10s

# 安全配置
# API_KEY=your-secret-api-key-change-this-in-production  # 如果不设置，启动时会生成临时密钥

# 日志配置
LOG_LEVEL=info
LOG_FORMAT=json

# 工作空间配置
WORK_DIR=/opt/api-systemd  # 工作目录根路径
```

#### 4. 卸载服务
```bash
make uninstall
```

### 方式二：直接运行

#### 构建
```bash
make build-local
# 或
go build -o api-systemd
```

#### 运行
```bash
./api-systemd
```

**首次启动时，如果未设置 `API_KEY`，系统会自动生成临时密钥并在控制台显示：**
```
🔑 API_KEY 未设置，已生成临时密钥:
   API_KEY: tmp-a1b2c3d4e5f6...
   请使用此密钥进行API认证: Authorization: Bearer tmp-a1b2c3d4e5f6...
   建议在生产环境中设置固定的 API_KEY 环境变量
```

### 方式三：Docker 部署

#### 使用 Docker
```bash
# 构建镜像
make docker

# 运行容器
docker run -d \
  --name api-systemd \
  -p 8080:8080 \
  -v /var/log/api-systemd:/var/log/api-systemd \
  api-systemd:latest
```

#### 使用 Docker Compose
```bash
# 启动所有服务（包括监控）
docker-compose up -d

# 仅启动 API 服务
docker-compose up -d api-systemd
```

### 健康检查
```bash
curl http://localhost:8080/health
# 或
make health
```

### API 使用示例

> **注意**: 所有API请求都需要包含 Bearer Token（强制认证）

#### 部署服务
```bash
curl -X POST http://localhost:8080/services/deploy \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-secret-api-key" \
  -d '{
    "service": "test-app",
    "path": "/opt/services",
    "package_url": "https://example.com/app.tar.gz", 
    "start_command": "app"
  }'
```

#### 管理服务
```bash
# 获取服务列表
curl http://localhost:8080/services \
  -H "Authorization: Bearer your-secret-api-key"

# 启动服务
curl -X POST http://localhost:8080/services/test-app/start \
  -H "Authorization: Bearer your-secret-api-key"

# 获取服务状态
curl http://localhost:8080/services/test-app/status \
  -H "Authorization: Bearer your-secret-api-key"

# 获取服务日志（最近100行）
curl http://localhost:8080/services/test-app/logs?lines=100 \
  -H "Authorization: Bearer your-secret-api-key"

# 重启服务
curl -X POST http://localhost:8080/services/test-app/restart \
  -H "Authorization: Bearer your-secret-api-key"

# 停止服务
curl -X POST http://localhost:8080/services/test-app/stop \
  -H "Authorization: Bearer your-secret-api-key"

# 删除服务
curl -X DELETE http://localhost:8080/services/test-app \
  -H "Authorization: Bearer your-secret-api-key"
```

#### 配置管理
```bash
# 创建配置
curl -X POST http://localhost:8080/configs/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-secret-api-key" \
  -d '{
    "service": "test-app",
    "config": "[Unit]\nDescription=Test App\n[Service]\nExecStart=/opt/test-app/app\n[Install]\nWantedBy=multi-user.target"
  }'

# 删除配置
curl -X DELETE http://localhost:8080/configs/test-app \
  -H "Authorization: Bearer your-secret-api-key"
```

#### 无需认证的接口
```bash
# 健康检查（无需认证）
curl http://localhost:8080/health

# 连通性测试（无需认证）
curl http://localhost:8080/ping
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

## 🔐 安全特性

### systemd 安全配置
- **用户隔离**: 运行在专用的 `api-systemd` 用户下
- **权限限制**: 使用最小权限原则
- **文件系统保护**: 只读系统文件，受限的写入路径
- **进程隔离**: 私有临时目录和进程命名空间

### 服务安全
- **输入验证**: 严格的参数验证
- **资源限制**: 可配置的内存和CPU限制
- **日志审计**: 详细的操作日志记录
- **权限检查**: systemd 操作权限验证

## 📁 目录结构

### systemd 安装后的目录结构
```
/opt/api-systemd/              # 主安装目录
├── api-systemd                # 主程序
├── manage.sh                  # 管理脚本
├── services/                  # 服务文件目录
│   ├── my-app/               # 服务名称目录
│   │   ├── app               # 应用程序文件
│   │   └── config.json       # 应用配置文件
│   └── worker/               # 另一个服务
│       └── worker            # 工作进程文件
└── logs/                     # 日志目录
    ├── my-app/               # 服务日志目录
    └── worker/               # 工作进程日志目录

/etc/api-systemd/              # 配置目录
└── config.env                 # 主配置文件

/var/log/api-systemd/          # 系统日志目录

/etc/systemd/system/           # systemd 配置
└── api-systemd.service        # 服务定义文件

/usr/local/bin/                # 全局命令
└── api-systemd-ctl            # 管理命令链接
```

### 配置文件说明
- **主配置**: `/etc/api-systemd/config.env` - 服务运行配置
- **systemd配置**: `/etc/systemd/system/api-systemd.service` - 服务定义
- **日志轮转**: `/etc/logrotate.d/api-systemd` - 日志管理

## 🛠️ 管理命令

### Make 命令
```bash
make help          # 显示所有可用命令
make build          # 构建二进制文件
make install        # 安装到 systemd
make uninstall      # 从 systemd 卸载
make status         # 查看服务状态
make logs           # 查看服务日志
make health         # 执行健康检查
make restart        # 重启服务
```

### systemctl 命令
```bash
systemctl start api-systemd     # 启动服务
systemctl stop api-systemd      # 停止服务
systemctl restart api-systemd   # 重启服务
systemctl status api-systemd    # 查看状态
systemctl enable api-systemd    # 开机自启
systemctl disable api-systemd   # 禁用自启
```

### 便捷管理命令
```bash
api-systemd-ctl start       # 启动服务
api-systemd-ctl stop        # 停止服务
api-systemd-ctl restart     # 重启服务
api-systemd-ctl status      # 查看状态
api-systemd-ctl logs        # 查看日志
api-systemd-ctl health      # 健康检查
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

Apache License
# API-Systemd Makefile

# 变量定义
BINARY_NAME=api-systemd
VERSION?=latest
BUILD_DIR=build
DOCKER_IMAGE=api-systemd
SERVICE_NAME=api-systemd

# Go 相关变量
GOOS?=linux
GOARCH?=amd64
CGO_ENABLED?=0

# 构建标志
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

.PHONY: help build clean test install uninstall docker run stop status logs health

# 默认目标
all: build

# 显示帮助信息
help:
	@echo "API-Systemd Makefile"
	@echo ""
	@echo "可用命令:"
	@echo "  build      - 构建二进制文件"
	@echo "  clean      - 清理构建文件"
	@echo "  test       - 运行测试"
	@echo "  install    - 安装到 systemd"
	@echo "  uninstall  - 从 systemd 卸载"
	@echo "  docker     - 构建 Docker 镜像"
	@echo "  run        - 直接运行程序"
	@echo "  stop       - 停止服务"
	@echo "  status     - 查看服务状态"
	@echo "  logs       - 查看服务日志"
	@echo "  health     - 健康检查"

# 构建二进制文件
build:
	@echo "构建 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 构建当前平台版本
build-local:
	@echo "构建本地版本..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "构建完成: $(BINARY_NAME)"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "清理完成"

# 运行测试
test:
	@echo "运行测试..."
	go test -v ./...

# 代码检查
lint:
	@echo "运行代码检查..."
	golangci-lint run

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod download
	go mod tidy

# 安装到 systemd
install: build-local
	@echo "安装 $(SERVICE_NAME) 到 systemd..."
	@chmod +x scripts/install.sh
	@sudo ./scripts/install.sh

# 从 systemd 卸载
uninstall:
	@echo "从 systemd 卸载 $(SERVICE_NAME)..."
	@chmod +x scripts/uninstall.sh
	@sudo ./scripts/uninstall.sh

# 构建 Docker 镜像
docker:
	@echo "构建 Docker 镜像..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker build -t $(DOCKER_IMAGE):latest .

# 直接运行程序
run: build-local
	@echo "启动 $(BINARY_NAME)..."
	./$(BINARY_NAME)

# 停止服务
stop:
	@echo "停止 $(SERVICE_NAME) 服务..."
	@sudo systemctl stop $(SERVICE_NAME) || echo "服务未运行"

# 查看服务状态
status:
	@echo "查看 $(SERVICE_NAME) 服务状态..."
	@systemctl status $(SERVICE_NAME) --no-pager -l || echo "服务未安装"

# 查看服务日志
logs:
	@echo "查看 $(SERVICE_NAME) 服务日志..."
	@journalctl -u $(SERVICE_NAME) -f

# 健康检查
health:
	@echo "执行健康检查..."
	@curl -s http://localhost:8080/health | jq . || echo "健康检查失败"

# 重启服务
restart:
	@echo "重启 $(SERVICE_NAME) 服务..."
	@sudo systemctl restart $(SERVICE_NAME)

# 启用服务
enable:
	@echo "启用 $(SERVICE_NAME) 服务..."
	@sudo systemctl enable $(SERVICE_NAME)

# 禁用服务
disable:
	@echo "禁用 $(SERVICE_NAME) 服务..."
	@sudo systemctl disable $(SERVICE_NAME)

# 重新加载配置
reload:
	@echo "重新加载 $(SERVICE_NAME) 配置..."
	@sudo systemctl reload $(SERVICE_NAME)

# 发布准备
release: clean test lint build
	@echo "发布准备完成"

# 开发模式
dev: build-local
	@echo "开发模式启动..."
	@./$(BINARY_NAME) &
	@echo "服务已在后台启动，PID: $$!"

# 查看版本
version:
	@echo "Version: $(VERSION)"

# 创建发布包
package: build
	@echo "创建发布包..."
	@mkdir -p $(BUILD_DIR)/package
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/package/
	@cp -r scripts $(BUILD_DIR)/package/
	@cp -r systemd $(BUILD_DIR)/package/
	@cp README.md $(BUILD_DIR)/package/
	@cp LICENSE $(BUILD_DIR)/package/
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz package/
	@echo "发布包创建完成: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"

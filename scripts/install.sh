#!/bin/bash

# API-Systemd 安装脚本
# 用于将 api-systemd 服务安装到 systemd 中

set -e

# 配置变量
SERVICE_NAME="api-systemd"
SERVICE_USER="api-systemd"
SERVICE_GROUP="api-systemd"
INSTALL_DIR="/opt/api-systemd"
CONFIG_DIR="/etc/api-systemd"
LOG_DIR="/var/log/api-systemd"
BINARY_NAME="api-systemd"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查是否以 root 权限运行
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本需要 root 权限运行"
        exit 1
    fi
}

# 检查系统是否支持 systemd
check_systemd() {
    if ! command -v systemctl &> /dev/null; then
        log_error "系统不支持 systemd"
        exit 1
    fi
    
    if ! systemctl --version &> /dev/null; then
        log_error "systemd 服务不可用"
        exit 1
    fi
    
    log_info "systemd 检查通过"
}

# 创建系统用户和组
create_user() {
    if ! getent group "$SERVICE_GROUP" > /dev/null 2>&1; then
        log_info "创建组: $SERVICE_GROUP"
        groupadd --system "$SERVICE_GROUP"
    else
        log_info "组 $SERVICE_GROUP 已存在"
    fi

    if ! getent passwd "$SERVICE_USER" > /dev/null 2>&1; then
        log_info "创建用户: $SERVICE_USER"
        useradd --system --gid "$SERVICE_GROUP" \
                --home-dir "$INSTALL_DIR" \
                --shell /bin/false \
                --comment "API-Systemd Service User" \
                "$SERVICE_USER"
    else
        log_info "用户 $SERVICE_USER 已存在"
    fi
}

# 创建目录结构
create_directories() {
    log_info "创建目录结构"
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    
    # 设置目录权限
    chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"
    chown -R "$SERVICE_USER:$SERVICE_GROUP" "$CONFIG_DIR"
    chown -R "$SERVICE_USER:$SERVICE_GROUP" "$LOG_DIR"
    
    chmod 755 "$INSTALL_DIR"
    chmod 755 "$CONFIG_DIR"
    chmod 755 "$LOG_DIR"
}

# 安装二进制文件
install_binary() {
    if [[ ! -f "./$BINARY_NAME" ]]; then
        log_error "找不到二进制文件: ./$BINARY_NAME"
        log_info "请先运行: go build -o $BINARY_NAME"
        exit 1
    fi
    
    log_info "安装二进制文件到 $INSTALL_DIR"
    cp "./$BINARY_NAME" "$INSTALL_DIR/"
    chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/$BINARY_NAME"
    chmod 755 "$INSTALL_DIR/$BINARY_NAME"
}

# 安装配置文件
install_config() {
    log_info "安装配置文件"
    
    # 创建主配置文件
    cat > "$CONFIG_DIR/config.env" << EOF
# API-Systemd 配置文件
SERVER_PORT=:8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_IDLE_TIMEOUT=120s
LOG_LEVEL=info
EOF

    chown "$SERVICE_USER:$SERVICE_GROUP" "$CONFIG_DIR/config.env"
    chmod 644 "$CONFIG_DIR/config.env"
    
    log_info "配置文件已创建: $CONFIG_DIR/config.env"
}

# 创建 systemd 服务文件
create_systemd_service() {
    log_info "创建 systemd 服务文件"
    
    cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=API-Systemd Service Management API
Documentation=https://github.com/rushteam/api-systemd
After=network.target
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME
ExecReload=/bin/kill -HUP \$MAINPID
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30
Restart=always
RestartSec=5
StartLimitBurst=3
StartLimitInterval=60

# 环境配置
EnvironmentFile=-$CONFIG_DIR/config.env

# 安全配置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$LOG_DIR $CONFIG_DIR
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

# 资源限制
LimitNOFILE=65536
LimitNPROC=4096

# 日志配置
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

[Install]
WantedBy=multi-user.target
EOF

    log_info "systemd 服务文件已创建: /etc/systemd/system/$SERVICE_NAME.service"
}

# 创建日志轮转配置
create_logrotate() {
    log_info "创建日志轮转配置"
    
    cat > "/etc/logrotate.d/$SERVICE_NAME" << EOF
$LOG_DIR/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 $SERVICE_USER $SERVICE_GROUP
    postrotate
        systemctl reload $SERVICE_NAME > /dev/null 2>&1 || true
    endscript
}
EOF
}

# 启用并启动服务
enable_service() {
    log_info "重新加载 systemd 配置"
    systemctl daemon-reload
    
    log_info "启用 $SERVICE_NAME 服务"
    systemctl enable "$SERVICE_NAME"
    
    log_info "启动 $SERVICE_NAME 服务"
    systemctl start "$SERVICE_NAME"
    
    # 等待服务启动
    sleep 2
    
    # 检查服务状态
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "✅ $SERVICE_NAME 服务启动成功"
        systemctl status "$SERVICE_NAME" --no-pager -l
    else
        log_error "❌ $SERVICE_NAME 服务启动失败"
        systemctl status "$SERVICE_NAME" --no-pager -l
        exit 1
    fi
}

# 创建管理脚本
create_management_scripts() {
    log_info "创建管理脚本"
    
    # 创建服务管理脚本
    cat > "$INSTALL_DIR/manage.sh" << 'EOF'
#!/bin/bash

SERVICE_NAME="api-systemd"

case "$1" in
    start)
        echo "启动 $SERVICE_NAME 服务..."
        sudo systemctl start "$SERVICE_NAME"
        ;;
    stop)
        echo "停止 $SERVICE_NAME 服务..."
        sudo systemctl stop "$SERVICE_NAME"
        ;;
    restart)
        echo "重启 $SERVICE_NAME 服务..."
        sudo systemctl restart "$SERVICE_NAME"
        ;;
    status)
        systemctl status "$SERVICE_NAME" --no-pager -l
        ;;
    logs)
        journalctl -u "$SERVICE_NAME" -f
        ;;
    health)
        curl -s http://localhost:8080/health | jq . || echo "健康检查失败"
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status|logs|health}"
        exit 1
        ;;
esac
EOF

    chmod 755 "$INSTALL_DIR/manage.sh"
    chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/manage.sh"
    
    # 创建全局命令链接
    ln -sf "$INSTALL_DIR/manage.sh" "/usr/local/bin/api-systemd-ctl"
    
    log_info "管理脚本已创建: $INSTALL_DIR/manage.sh"
    log_info "全局命令: api-systemd-ctl {start|stop|restart|status|logs|health}"
}

# 显示安装信息
show_info() {
    log_info "🎉 API-Systemd 安装完成！"
    echo
    echo "📋 安装信息:"
    echo "  - 服务名称: $SERVICE_NAME"
    echo "  - 安装目录: $INSTALL_DIR"
    echo "  - 配置目录: $CONFIG_DIR"
    echo "  - 日志目录: $LOG_DIR"
    echo "  - 服务用户: $SERVICE_USER"
    echo "  - 监听端口: 8080"
    echo
    echo "🔧 管理命令:"
    echo "  - systemctl start $SERVICE_NAME     # 启动服务"
    echo "  - systemctl stop $SERVICE_NAME      # 停止服务"
    echo "  - systemctl restart $SERVICE_NAME   # 重启服务"
    echo "  - systemctl status $SERVICE_NAME    # 查看状态"
    echo "  - journalctl -u $SERVICE_NAME -f    # 查看日志"
    echo "  - api-systemd-ctl health            # 健康检查"
    echo
    echo "🌐 API 访问:"
    echo "  - 健康检查: http://localhost:8080/health"
    echo "  - API 文档: 参考 README.md"
    echo
    echo "📝 配置文件: $CONFIG_DIR/config.env"
}

# 主安装流程
main() {
    log_info "开始安装 API-Systemd 服务"
    
    check_root
    check_systemd
    create_user
    create_directories
    install_binary
    install_config
    create_systemd_service
    create_logrotate
    create_management_scripts
    enable_service
    
    show_info
}

# 执行安装
main "$@"

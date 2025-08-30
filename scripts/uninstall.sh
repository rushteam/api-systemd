#!/bin/bash

# API-Systemd 卸载脚本

set -e

# 配置变量
SERVICE_NAME="api-systemd"
SERVICE_USER="api-systemd"
SERVICE_GROUP="api-systemd"
INSTALL_DIR="/opt/api-systemd"
CONFIG_DIR="/etc/api-systemd"
LOG_DIR="/var/log/api-systemd"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

# 停止并禁用服务
stop_service() {
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "停止 $SERVICE_NAME 服务"
        systemctl stop "$SERVICE_NAME"
    fi
    
    if systemctl is-enabled --quiet "$SERVICE_NAME"; then
        log_info "禁用 $SERVICE_NAME 服务"
        systemctl disable "$SERVICE_NAME"
    fi
}

# 删除 systemd 服务文件
remove_systemd_service() {
    if [[ -f "/etc/systemd/system/$SERVICE_NAME.service" ]]; then
        log_info "删除 systemd 服务文件"
        rm -f "/etc/systemd/system/$SERVICE_NAME.service"
        systemctl daemon-reload
    fi
}

# 删除文件和目录
remove_files() {
    log_info "删除安装文件"
    
    # 删除安装目录
    if [[ -d "$INSTALL_DIR" ]]; then
        rm -rf "$INSTALL_DIR"
    fi
    
    # 删除配置目录
    if [[ -d "$CONFIG_DIR" ]]; then
        rm -rf "$CONFIG_DIR"
    fi
    
    # 删除日志目录
    if [[ -d "$LOG_DIR" ]]; then
        rm -rf "$LOG_DIR"
    fi
    
    # 删除日志轮转配置
    if [[ -f "/etc/logrotate.d/$SERVICE_NAME" ]]; then
        rm -f "/etc/logrotate.d/$SERVICE_NAME"
    fi
    
    # 删除全局命令链接
    if [[ -L "/usr/local/bin/api-systemd-ctl" ]]; then
        rm -f "/usr/local/bin/api-systemd-ctl"
    fi
}

# 删除用户和组
remove_user() {
    if getent passwd "$SERVICE_USER" > /dev/null 2>&1; then
        log_info "删除用户: $SERVICE_USER"
        userdel "$SERVICE_USER"
    fi
    
    if getent group "$SERVICE_GROUP" > /dev/null 2>&1; then
        log_info "删除组: $SERVICE_GROUP"
        groupdel "$SERVICE_GROUP"
    fi
}

# 主卸载流程
main() {
    log_info "开始卸载 API-Systemd 服务"
    
    check_root
    stop_service
    remove_systemd_service
    remove_files
    remove_user
    
    log_info "🎉 API-Systemd 卸载完成！"
}

# 执行卸载
main "$@"

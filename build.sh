#!/bin/bash

# API-Systemd 构建脚本

echo "🔧 正在更新依赖..."
go mod tidy

echo "📦 正在构建应用..."
go build -o api-systemd

if [ $? -eq 0 ]; then
    echo "✅ 构建成功！"
    echo "🚀 运行命令: ./api-systemd"
else
    echo "❌ 构建失败，请检查错误信息"
    exit 1
fi

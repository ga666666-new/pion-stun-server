#!/bin/bash

# Go 1.24 环境设置脚本
echo "=== 设置 Go 1.24 环境 ==="

# 检查Go 1.24是否已安装
if [ ! -d "/opt/homebrew/opt/go@1.24" ]; then
    echo "Go 1.24 未安装，正在安装..."
    brew install go@1.24
else
    echo "Go 1.24 已安装"
fi

# 设置PATH
export PATH="/opt/homebrew/opt/go@1.24/bin:$PATH"

# 验证版本
echo "当前Go版本："
go version

# 更新项目依赖
echo "更新项目依赖..."
go mod tidy

# 编译项目
echo "编译项目..."
go build -o bin/server ./cmd/server

echo "=== Go 1.24 环境设置完成 ==="
echo ""
echo "使用方法："
echo "1. 运行此脚本设置环境：./scripts/setup-go1.24.sh"
echo "2. 或者手动设置PATH：export PATH=\"/opt/homebrew/opt/go@1.24/bin:\$PATH\""
echo "3. 将此行添加到 ~/.zshrc 或 ~/.bash_profile 以永久设置"
echo ""
echo "Docker构建："
echo "docker build -t pion-stun-server:go1.24 ." 
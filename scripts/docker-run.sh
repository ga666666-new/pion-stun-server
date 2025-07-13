#!/bin/bash

# Docker运行脚本
set -e

echo "=== PION STUN/TURN Server Docker 运行脚本 ==="

# 检查镜像是否存在
if ! docker images | grep -q pion-stun-server; then
    echo "错误：镜像 pion-stun-server:latest 不存在"
    echo "请先运行构建脚本：./scripts/docker-build.sh"
    exit 1
fi

# 停止并删除现有容器（如果存在）
echo "1. 清理现有容器..."
docker stop pion-stun-server 2>/dev/null || true
docker rm pion-stun-server 2>/dev/null || true

# 运行新容器
echo "2. 启动新容器..."
docker run -d \
    --name pion-stun-server \
    -p 3478:3478/udp \
    -p 3479:3479/udp \
    -p 3479:3479/tcp \
    -p 8080:8080/tcp \
    --restart unless-stopped \
    pion-stun-server:latest

echo "3. 容器启动完成！"

# 显示容器状态
echo "4. 容器状态："
docker ps | grep pion-stun-server

echo ""
echo "=== 运行完成 ==="
echo "查看日志："
echo "  docker logs -f pion-stun-server"
echo ""
echo "健康检查："
echo "  curl http://localhost:8080/health"
echo ""
echo "停止容器："
echo "  docker stop pion-stun-server" 
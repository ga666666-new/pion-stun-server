#!/bin/bash

# Docker构建脚本
set -e

echo "=== PION STUN/TURN Server Docker 构建脚本 ==="

# 构建镜像
echo "1. 构建Docker镜像..."
docker build -t pion-stun-server:latest .

echo "2. 镜像构建完成！"

# 显示镜像信息
echo "3. 镜像信息："
docker images | grep pion-stun-server

echo ""
echo "=== 构建完成 ==="
echo "运行容器："
echo "  docker run -p 3478:3478/udp -p 3479:3479/udp -p 3479:3479/tcp -p 8080:8080/tcp pion-stun-server:latest"
echo ""
echo "或使用 docker-compose："
echo "  docker-compose up" 
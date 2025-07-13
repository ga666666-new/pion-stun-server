#!/bin/bash

# 调试权限错误的启动脚本
# 使用特殊的调试配置，遇到权限错误时会立即终止程序并输出详细调试信息

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== PION STUN/TURN 服务器权限调试模式 ===${NC}"
echo -e "${YELLOW}此模式会在遇到权限错误时立即终止程序并输出详细调试信息${NC}"
echo

# 检查是否存在调试配置文件
if [ ! -f "configs/config.debug.yaml" ]; then
    echo -e "${RED}错误：调试配置文件 configs/config.debug.yaml 不存在${NC}"
    exit 1
fi

# 检查是否已编译服务器
if [ ! -f "bin/server" ]; then
    echo -e "${YELLOW}服务器未编译，正在编译...${NC}"
    go build -o bin/server ./cmd/server
    echo -e "${GREEN}编译完成${NC}"
fi

echo -e "${BLUE}调试配置信息：${NC}"
echo -e "  - 配置文件: configs/config.debug.yaml"
echo -e "  - 日志级别: trace (最详细)"
echo -e "  - 权限错误终止: 启用"
echo -e "  - 公网IP: 223.254.128.13"
echo

echo -e "${YELLOW}启动服务器...${NC}"
echo -e "${RED}注意：遇到权限错误时程序会立即终止！${NC}"
echo

# 启动服务器，使用调试配置
exec ./bin/server --config configs/config.debug.yaml 
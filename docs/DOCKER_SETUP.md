# Docker 部署指南

## 概述

本文档描述如何使用Docker部署PION STUN/TURN服务器。

## 快速开始

### 1. 使用脚本部署

#### 构建镜像
```bash
./scripts/docker-build.sh
```

#### 运行容器
```bash
./scripts/docker-run.sh
```

### 2. 使用docker-compose部署

#### 启动所有服务
```bash
docker-compose up -d
```

#### 查看日志
```bash
docker-compose logs -f pion-stun-server
```

#### 停止服务
```bash
docker-compose down
```

## 配置说明

### Dockerfile优化

新的Dockerfile具有以下特性：

1. **配置文件打包**：配置文件现在直接打包到镜像中的 `/app/config/` 目录
2. **自动配置**：如果不存在 `config.yaml`，会自动从 `config.example.yaml` 复制
3. **启动参数**：容器启动时自动指定配置文件路径 `config/config.yaml`
4. **安全性**：使用非root用户运行应用程序

### 端口映射

- `3478/udp`: STUN服务端口
- `3479/udp`: TURN UDP端口
- `3479/tcp`: TURN TCP端口
- `8080/tcp`: 健康检查端口

### 环境变量

可以通过环境变量覆盖配置：

```bash
docker run -e MONGODB_URI="mongodb://user:pass@host:port/db" \
           -e LOGGING_LEVEL="debug" \
           pion-stun-server:latest
```

## 自定义配置

### 方法1：挂载配置文件

如果需要自定义配置，可以挂载外部配置文件：

```bash
docker run -v ./my-config.yaml:/app/config/config.yaml:ro \
           -p 3478:3478/udp \
           -p 3479:3479/udp \
           -p 3479:3479/tcp \
           -p 8080:8080/tcp \
           pion-stun-server:latest
```

### 方法2：在docker-compose中挂载

取消注释 `docker-compose.yml` 中的volumes部分：

```yaml
volumes:
  - ./configs/config.yaml:/app/config/config.yaml:ro
```

### 方法3：构建自定义镜像

1. 修改 `configs/config.yaml`
2. 重新构建镜像：
   ```bash
   ./scripts/docker-build.sh
   ```

## 服务组件

### STUN/TURN服务器
- 主要的STUN/TURN服务
- 支持UDP和TCP连接
- 包含增强的日志功能

### MongoDB
- 用户认证数据库
- 自动初始化脚本
- 持久化数据存储

### MongoDB Express (可选)
- Web界面数据库管理工具
- 访问地址：http://localhost:8081
- 用户名/密码：admin/admin

## 健康检查

容器包含内置的健康检查：

```bash
# 检查服务状态
curl http://localhost:8080/health

# 查看容器健康状态
docker ps
```

## 日志查看

### 查看实时日志
```bash
# 单容器
docker logs -f pion-stun-server

# docker-compose
docker-compose logs -f pion-stun-server
```

### 查看增强日志

新版本包含详细的客户端追踪日志：
- 会话开始/结束
- 认证过程
- 权限检查
- 数据传输状态

## 故障排除

### 常见问题

1. **端口冲突**
   ```bash
   # 检查端口占用
   lsof -i :3478
   lsof -i :3479
   ```

2. **MongoDB连接失败**
   ```bash
   # 检查MongoDB状态
   docker-compose logs mongodb
   ```

3. **配置文件问题**
   ```bash
   # 检查配置文件
   docker exec pion-stun-server cat /app/config/config.yaml
   ```

### 调试模式

启用调试日志：

```bash
# 环境变量方式
docker run -e LOGGING_LEVEL=debug pion-stun-server:latest

# 或在docker-compose.yml中设置
environment:
  - LOGGING_LEVEL=debug
```

## 生产部署建议

1. **安全配置**
   - 修改默认密码
   - 使用强密码
   - 限制网络访问

2. **资源限制**
   ```yaml
   deploy:
     resources:
       limits:
         memory: 512M
         cpus: '0.5'
   ```

3. **数据备份**
   ```bash
   # 备份MongoDB数据
   docker exec pion-stun-mongodb mongodump --out /backup
   ```

4. **监控**
   - 使用健康检查端点
   - 监控容器状态
   - 分析日志输出 
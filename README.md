# STUN/TURN 服务器 - 基于 MongoDB 认证

基于 Go 语言和 Pion WebRTC 库实现的高性能 STUN/TURN 服务器，支持 MongoDB 用户认证和灵活的数据库架构配置。

## 功能特性

- **STUN 服务器**: 符合 RFC 5389 标准的 NAT 穿透发现服务
- **TURN 服务器**: 符合 RFC 5766 标准的媒体中继服务，支持 WebRTC 应用
- **MongoDB 认证**: 灵活的用户认证系统，支持自定义数据库架构
- **配置管理**: 支持环境变量和配置文件的灵活配置
- **健康监控**: 提供 HTTP 健康检查端点
- **Docker 支持**: 完整的容器化部署方案
- **全面测试**: 包含单元测试、集成测试和协议测试

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   STUN 客户端   │    │   TURN 客户端   │    │    健康检查     │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │ UDP:3478             │ UDP:3479             │ HTTP:8080
          │                      │                      │
┌─────────▼──────────────────────▼──────────────────────▼───────┐
│                    STUN/TURN 服务器                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐   │
│  │ STUN 服务器 │  │ TURN 服务器 │  │    健康检查处理器   │   │
│  └─────────────┘  └─────────────┘  └─────────────────────┘   │
│                           │                                  │
│  ┌─────────────────────────▼─────────────────────────────┐   │
│  │              MongoDB 认证模块                        │   │
│  └─────────────────────────┬─────────────────────────────┘   │
└────────────────────────────┼─────────────────────────────────┘
                             │
                   ┌─────────▼─────────┐
                   │     MongoDB       │
                   │    (用户存储)     │
                   └───────────────────┘
```

### 项目结构

```
├── cmd/server/           # 应用程序入口
├── internal/
│   ├── config/          # 配置管理
│   ├── auth/            # MongoDB 认证
│   ├── server/          # STUN/TURN 服务器实现
│   └── health/          # 健康检查处理器
├── pkg/
│   └── models/          # 数据模型
├── tests/               # 测试文件
├── configs/             # 配置文件
└── docker/              # Docker 相关文件
```

## 快速开始

### 环境要求

- Go 1.21+
- MongoDB 4.4+
- Docker (可选)

### 安装步骤

1. 克隆仓库:
```bash
git clone https://github.com/ga666666-new/pion-stun-server.git
cd pion-stun-server
```

2. 安装依赖:
```bash
go mod tidy
```

3. 启动 MongoDB (使用 Docker):
```bash
docker-compose up -d mongodb
```

4. 配置服务器:
```bash
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml 设置您的配置
```

5. 运行服务器:
```bash
go run cmd/server/main.go
```

## 配置说明

服务器支持通过环境变量或 YAML 配置文件进行配置。

### 环境变量

- `STUN_PORT`: STUN 服务器端口 (默认: 3478)
- `TURN_PORT`: TURN 服务器端口 (默认: 3479)
- `HEALTH_PORT`: 健康检查 HTTP 端口 (默认: 8080)
- `MONGO_URI`: MongoDB 连接 URI
- `MONGO_DATABASE`: MongoDB 数据库名
- `MONGO_COLLECTION`: MongoDB 集合名
- `MONGO_USERNAME_FIELD`: 用户名字段名 (默认: "username")
- `MONGO_PASSWORD_FIELD`: 密码字段名 (默认: "password")

### 配置文件示例

```yaml
stun:
  address: "0.0.0.0"
  port: 3478

turn:
  address: "0.0.0.0"
  port: 3479
  realm: "example.com"
  public_ip: "YOUR_PUBLIC_IP"

mongodb:
  uri: "mongodb://localhost:27017"
  database: "stun_server"
  collection: "users"
  fields:
    username: "username"
    password: "password"
    enabled: "enabled"

health:
  address: "0.0.0.0"
  port: 8080

logging:
  level: "info"
```

## MongoDB 认证配置

服务器支持灵活的 MongoDB 认证，可自定义数据库架构:

### 用户文档示例
```javascript
// MongoDB 中的用户文档示例
{
  "_id": ObjectId("..."),
  "username": "user1",
  "password": "$2a$10$...", // bcrypt 哈希密码
  "enabled": true,
  "created_at": ISODate("..."),
  "updated_at": ISODate("...")
}
```

### 自定义字段映射
您可以通过配置自定义字段名称:

```yaml
mongodb:
  fields:
    username: "user_name"      # 自定义用户名字段
    password: "user_pass"      # 自定义密码字段
    enabled: "is_active"       # 自定义启用状态字段
```

## API 端点

### 健康检查端点

- `GET /health` - 基础健康检查
- `GET /ready` - 就绪状态检查 (检查 MongoDB 连接)
- `GET /metrics` - 服务器指标和统计信息
- `GET /sessions` - 活跃的 TURN 会话

### 响应示例

```json
{
  "status": "healthy",
  "timestamp": "2023-12-01T10:00:00Z",
  "services": {
    "stun": "running",
    "turn": "running", 
    "mongodb": "connected"
  }
}
```

## 测试

### 运行所有测试
```bash
make test
```

### 运行带覆盖率的测试
```bash
go test -cover ./...
```

### 运行集成测试 (需要 MongoDB)
```bash
go test -tags=integration ./tests/...
```

### 运行 STUN 协议测试
```bash
go test ./tests -v -run TestSTUN
```

## Docker 部署

### 使用 Docker Compose

```bash
docker-compose up -d
```

### 构建 Docker 镜像

```bash
docker build -t pion-stun-server .
```

### 检查服务状态

```bash
# 检查健康状态
curl http://localhost:8080/health

# 查看日志
docker-compose logs -f stun-turn-server
```

## 性能调优

- 调整 `GOMAXPROCS` 以优化 CPU 利用率
- 配置 MongoDB 连接池设置
- 调整网络缓冲区大小以提高吞吐量
- 设置适当的 TURN 会话连接限制

## 安全考虑

- 使用强密码并启用 MongoDB 认证
- 为 STUN/TURN 端口配置防火墙规则
- 在生产环境中使用 TLS 连接 MongoDB
- 定期轮换认证凭据
- 实施认证尝试的速率限制

## 监控和运维

### 指标收集

服务器在 `/metrics` 端点暴露指标:

- 活跃的 STUN/TURN 会话数
- 请求/响应速率
- 认证成功/失败率
- MongoDB 连接状态

### 日志管理

支持结构化日志，可配置日志级别:

```yaml
logging:
  level: "info"        # debug, info, warn, error
  format: "json"       # json, text
  output: "stdout"     # stdout, 文件路径
```

## 故障排除

### 常见问题

1. **端口已被占用**:
   ```bash
   netstat -tulpn | grep :3478
   ```

2. **MongoDB 连接失败**:
   - 检查 MongoDB 服务状态
   - 验证连接字符串和凭据
   - 检查网络连接

3. **STUN/TURN 不工作**:
   - 验证防火墙规则
   - 检查 TURN 的公网 IP 配置
   - 使用 STUN/TURN 客户端工具测试

### 调试模式

启用调试日志:

```bash
export LOG_LEVEL=debug
./pion-stun-server
```

## 开发指南

### 添加新的认证方法

1. 实现 `Authenticator` 接口:

```go
type Authenticator interface {
    Authenticate(ctx context.Context, username, password string) (*models.User, error)
    GetUser(ctx context.Context, username string) (*models.User, error)
    CreateUser(ctx context.Context, user *models.User) error
    UpdateUser(ctx context.Context, user *models.User) error
    DeleteUser(ctx context.Context, username string) error
    Close() error
}
```

2. 在服务器初始化中注册新的认证器

### 扩展 STUN/TURN 功能

服务器使用 Pion WebRTC 库实现 STUN/TURN。要添加新功能:

1. 在 `internal/config/` 中扩展服务器配置
2. 在 `internal/server/` 中实现新的处理器
3. 在 `tests/` 中添加相应的测试

## 贡献指南

1. Fork 仓库
2. 创建功能分支
3. 进行更改
4. 为新功能添加测试
5. 确保所有测试通过
6. 提交 Pull Request

## 许可证

MIT 许可证 - 详见 LICENSE 文件。

## 致谢

- [Pion WebRTC](https://github.com/pion/webrtc) - Go 语言 WebRTC 实现
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver) - 官方 MongoDB 驱动
- [Viper](https://github.com/spf13/viper) - 配置管理
- [Logrus](https://github.com/sirupsen/logrus) - 结构化日志
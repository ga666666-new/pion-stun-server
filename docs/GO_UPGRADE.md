# Go 版本升级指南

## 概述

本项目已升级到 Go 1.24，以获得更好的性能和安全性。

## 升级步骤

### 1. 安装 Go 1.24

#### 使用 Homebrew (macOS)
```bash
brew install go@1.24
```

#### 使用官方安装包
访问 [Go 官方下载页面](https://golang.org/dl/) 下载 Go 1.24

### 2. 设置环境

#### 临时设置
```bash
export PATH="/opt/homebrew/opt/go@1.24/bin:$PATH"
```

#### 永久设置
将以下行添加到 `~/.zshrc` 或 `~/.bash_profile`：
```bash
export PATH="/opt/homebrew/opt/go@1.24/bin:$PATH"
```

### 3. 使用自动化脚本

我们提供了一个自动化脚本来设置 Go 1.24 环境：

```bash
./scripts/setup-go1.24.sh
```

这个脚本会：
- 检查并安装 Go 1.24
- 设置正确的 PATH
- 更新项目依赖
- 编译项目

### 4. 验证升级

```bash
go version
# 应该显示: go version go1.24.x darwin/arm64
```

## 项目文件更新

### go.mod
```go
module github.com/ga666666-new/pion-stun-server

go 1.24
```

### Dockerfile
```dockerfile
FROM golang:1.24-alpine AS builder
```

## 新特性

Go 1.24 带来的改进：

1. **性能提升**：
   - 更快的编译速度
   - 更好的内存管理
   - 优化的垃圾回收

2. **安全性增强**：
   - 更新的加密库
   - 更好的安全补丁

3. **开发体验**：
   - 更好的错误信息
   - 改进的工具链

## 兼容性

- 所有现有代码都与 Go 1.24 兼容
- 依赖包已更新到支持 Go 1.24 的版本
- Docker 镜像使用 Go 1.24 构建

## 故障排除

### 常见问题

1. **"toolchain not available" 错误**
   - 确保已正确安装 Go 1.24
   - 检查 PATH 设置

2. **依赖包兼容性问题**
   - 运行 `go mod tidy` 更新依赖
   - 检查是否有过时的包

3. **Docker 构建失败**
   - 确保 Dockerfile 使用正确的 Go 版本
   - 清理 Docker 缓存：`docker system prune`

### 回滚到旧版本

如果需要回滚到 Go 1.21：

1. 修改 `go.mod`：
   ```go
   go 1.21
   ```

2. 修改 `Dockerfile`：
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   ```

3. 重新安装 Go 1.21：
   ```bash
   brew install go@1.21
   export PATH="/opt/homebrew/opt/go@1.21/bin:$PATH"
   ```

## 支持

如果遇到升级问题，请：

1. 检查 [Go 1.24 发布说明](https://golang.org/doc/go1.24)
2. 查看项目 Issues
3. 提交新的 Issue 描述问题 
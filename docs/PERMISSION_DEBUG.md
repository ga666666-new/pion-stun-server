# TURN 权限错误调试指南

## 概述

当 TURN 服务器出现 "No Permission or Channel exists" 错误时，这通常意味着客户端没有正确建立 TURN 权限。为了更好地调试这类问题，我们添加了一个特殊的调试功能，可以在遇到权限错误时立即终止程序并输出详细的调试信息。

## 问题背景

在 WebRTC 中使用 TURN 服务器时，标准流程如下：

1. **分配请求 (Allocate Request)**: 客户端向 TURN 服务器请求分配一个中继地址
2. **权限创建 (CreatePermission)**: 客户端必须明确告诉 TURN 服务器允许哪些远程地址发送数据
3. **数据传输**: 只有获得权限的远程地址才能通过 TURN 服务器进行数据中继

当出现 "No Permission or Channel exists" 错误时，说明第2步没有正确执行。

## 调试功能

### 配置选项

在 `configs/config.yaml` 中添加了新的配置选项：

```yaml
server:
  turn:
    # 调试选项：遇到权限错误时终止程序并输出详细调试信息
    terminate_on_permission_error: true
```

### 调试配置文件

我们提供了一个专门用于调试的配置文件 `configs/config.debug.yaml`：

- 启用了权限错误终止功能
- 设置日志级别为 `trace`（最详细）
- 使用了您的公网 IP 地址

### 使用方法

#### 方法1：使用调试脚本（推荐）

```bash
./scripts/debug-permission-errors.sh
```

#### 方法2：手动启动

```bash
./bin/server --config configs/config.debug.yaml
```

### 调试信息输出

当遇到权限错误时，程序会输出以下详细信息：

1. **错误消息**: 具体的权限错误信息
2. **活跃会话数量**: 当前所有活跃的 TURN 会话
3. **每个会话的详细信息**:
   - 客户端地址和用户名
   - 会话持续时间
   - 总步骤数
   - 分配、权限、通道数量
   - 最后活动时间

4. **会话步骤历史**: 每个客户端的完整操作历史
5. **分配信息**: 所有已创建的 TURN 分配
6. **权限信息**: 所有已创建的权限
7. **通道信息**: 所有已创建的通道

## 常见问题分析

### 1. 客户端没有发送 CreatePermission 请求

**症状**: 会话步骤历史中没有 "PERMISSION_CHECK" 或 "PERMISSION_CREATED" 步骤

**原因**: WebRTC 客户端实现有问题，没有正确发送 CreatePermission 请求

**解决方案**: 检查客户端代码，确保 WebRTC 实现正确

### 2. 权限创建时机错误

**症状**: 有 CreatePermission 请求，但时机不对

**原因**: 客户端在错误的时机发送了权限请求

**解决方案**: 确保在数据传输之前创建权限

### 3. 权限目标地址错误

**症状**: 创建了权限，但目标地址不匹配

**原因**: CreatePermission 请求中的目标地址与实际数据传输的目标地址不匹配

**解决方案**: 检查 ICE 候选者和权限目标地址的一致性

## 示例调试流程

1. **启动调试模式**:
   ```bash
   ./scripts/debug-permission-errors.sh
   ```

2. **运行客户端测试**: 打开浏览器访问 `demo/p2p-html-demo/app.html` 和 `device.html`

3. **观察日志**: 查看详细的会话追踪日志

4. **触发权限错误**: 当出现权限错误时，程序会立即终止并输出完整的调试信息

5. **分析调试信息**: 根据输出的调试信息分析问题原因

## 注意事项

- 调试模式会在遇到权限错误时立即终止程序，这是正常行为
- 调试模式会产生大量日志，建议只在调试时使用
- 生产环境请务必关闭 `terminate_on_permission_error` 选项
- 调试信息包含敏感的网络信息，请谨慎处理

## 相关文件

- `configs/config.debug.yaml` - 调试配置文件
- `scripts/debug-permission-errors.sh` - 调试启动脚本
- `internal/server/turn.go` - TURN 服务器实现
- `internal/config/config.go` - 配置结构定义

## 更多帮助

如果调试信息仍然无法帮助您定位问题，请：

1. 保存完整的调试输出
2. 记录客户端的操作步骤
3. 检查网络配置和防火墙设置
4. 验证 TURN 服务器的公网 IP 配置 
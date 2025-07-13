# P2P通信配置重构说明

## 重构目标
确保APP端严格使用服务端API下发的RTC配置，设备端严格使用MQTT下发的RTC配置。

## 配置流程

### APP端配置流程
1. **调用服务端API**: `POST /api/v1/signal/session`
2. **解析API响应**: 从响应中提取 `rtcConfiguration` 字段
3. **应用配置**: 为TURN服务器添加APP端认证信息 (`username: 'APP', credential: 'APP'`)
4. **创建PeerConnection**: 使用解析后的配置创建WebRTC连接

### 设备端配置流程
1. **订阅MQTT主题**: `dl/{productKey}/{deviceSn}/device/service/sub`
2. **接收P2P配置**: 通过MQTT接收 `P2P_SIGNAL_CONFIG_SERVICE` 消息
3. **解析配置**: 从消息中提取 `rtcConfiguration` 字段
4. **应用配置**: 为TURN服务器添加设备端认证信息 (`username: 'DEVICE', credential: 'DEVICE'`)
5. **创建PeerConnection**: 使用解析后的配置创建WebRTC连接

## 关键改进

### 移除硬编码配置
- 移除了APP端和设备端的硬编码RTC配置
- 配置完全依赖服务端下发

### 增强日志记录
- 详细记录配置解析过程
- 明确标识配置来源（服务端API vs MQTT）
- 记录TURN服务器认证信息添加过程

### 错误处理
- 配置解析失败时使用备用配置
- 在创建PeerConnection前验证配置是否已获取

### ICE候选者优化
- **批量收集**: 收集所有ICE候选者后一次性发送，而不是逐个发送
- **减少MQTT消息**: 大幅减少ICE候选者相关的MQTT消息数量
- **提高效率**: 减少网络开销和信令延迟

## 配置示例

### 服务端API响应格式
```json
{
  "code": 0,
  "data": {
    "sessionId": "session_123",
    "rtcConfiguration": {
      "iceServers": [
        {
          "urls": "stun:223.254.128.13:3478"
        },
        {
          "urls": "turn:223.254.128.13:3479"
        }
      ],
      "iceTransportPolicy": "all",
      "iceCandidatePoolSize": 10,
      "bundlePolicy": "max-bundle",
      "rtcpMuxPolicy": "require"
    }
  }
}
```

### MQTT配置消息格式
```json
{
  "cmd": "P2P_SIGNAL_CONFIG_SERVICE",
  "sessionId": "session_123",
  "memberId": "188815492",
  "rtcConfiguration": {
    "iceServers": [
      {
        "urls": "stun:223.254.128.13:3478"
      },
      {
        "urls": "turn:223.254.128.13:3479"
      }
    ],
    "iceTransportPolicy": "all",
    "iceCandidatePoolSize": 0,
    "bundlePolicy": "max-bundle",
    "rtcpMuxPolicy": "require"
  }
}
```

### ICE候选者批量消息格式
```json
{
  "cmd": "WEB_RTC",
  "ts": 1752046984589,
  "msgId": "app_1752046984589",
  "data": {
    "type": "candidates",
    "sessionId": "session_123",
    "from": "188815492",
    "to": "AF070135F064641AG",
    "candidates": [
      {
        "candidate": "candidate:0 1 UDP 2122252543 192.168.99.57 50610 typ host",
        "sdpMLineIndex": 0,
        "sdpMid": "0",
        "usernameFragment": "04806d97"
      },
      {
        "candidate": "candidate:1 1 UDP 1686052863 119.123.179.137 50610 typ srflx raddr 192.168.99.57 rport 50610",
        "sdpMLineIndex": 0,
        "sdpMid": "0",
        "usernameFragment": "04806d97"
      }
    ]
  }
}
```

## 认证信息
- **APP端**: `username: 'APP', credential: 'APP'`
- **设备端**: `username: 'DEVICE', credential: 'DEVICE'`

## 备用配置
当服务端配置获取失败时，使用以下备用配置：
- STUN服务器: `stun:223.254.128.13:3478`
- 基本WebRTC参数设置

## 性能优化
- **ICE候选者批量处理**: 减少MQTT消息数量，提高连接建立效率
- **配置来源明确**: 确保配置来源清晰，便于调试和维护
- **错误处理完善**: 提供备用配置和详细的错误日志

## 音频可视化功能

### 设备端音频可视化
设备端支持实时音频可视化，将接收到的APP音频绘制成动态波形图：

#### 功能特性
- **实时频谱分析**: 显示音频的频率分布
- **动态波形图**: 绿色波浪线显示音频振幅变化
- **音量指示器**: 实时显示音量级别（0-100%）
- **颜色渐变**: 根据频率和音量动态变化颜色
- **暂停/恢复**: 支持暂停和恢复可视化显示

#### 可视化效果
- **频谱柱状图**: 底部显示频率分布，颜色从绿色到黄色渐变
- **中心波浪线**: 中间显示动态波形，随音频变化而波动
- **音量条**: 底部显示当前音量，颜色从绿色到红色渐变
- **实时更新**: 60fps的流畅动画效果

#### 技术实现
- 使用Web Audio API的AnalyserNode进行音频分析
- Canvas 2D绘图实现可视化效果
- requestAnimationFrame实现流畅动画
- 支持暂停/恢复控制

#### 使用说明
1. 设备端启动后自动初始化音频可视化
2. 当接收到APP的音频流时，自动开始可视化
3. 点击"暂停可视化"按钮可暂停显示
4. 点击"恢复可视化"按钮可恢复显示
5. 点击"测试可视化"按钮可测试音频可视化效果（需要麦克风权限）
6. 设备停止时自动停止可视化

#### 调试功能
- **测试按钮**: 可以测试音频可视化是否正常工作
- **模拟数据测试**: 当麦克风权限被拒绝时，自动使用模拟数据进行测试
- **调试信息**: 画布左上角显示数据长度、平均音量和音量级别
- **错误提示**: 当音频分析器未初始化时显示错误信息
- **详细日志**: 控制台和日志区域显示详细的调试信息

#### 故障排除
1. **没有图像显示**: 
   - 检查浏览器是否支持Web Audio API
   - 点击"测试可视化"按钮测试功能
   - 查看日志区域的错误信息
   
2. **音频上下文被暂停**:
   - 需要用户交互才能启动音频上下文
   - 点击页面任意位置或按钮即可恢复
   - 系统会自动尝试恢复音频上下文
   
3. **麦克风权限被拒绝**:
   - 系统会自动使用模拟数据进行测试
   - 模拟数据会显示动态的波形效果
   - 可以验证可视化功能是否正常工作
   
4. **没有音频数据**:
   - 确保APP端正在发送音频
   - 检查P2P连接是否正常建立
   - 查看音量指示器是否显示变化 
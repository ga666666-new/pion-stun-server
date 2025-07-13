package com.dl.cloud.mall;

import com.alibaba.fastjson.JSON;
import com.alibaba.fastjson.JSONObject;
import org.eclipse.paho.client.mqttv3.*;
import org.eclipse.paho.client.mqttv3.persist.MemoryPersistence;

import java.util.Scanner;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

/**
 * 设备端模拟器 - 模拟IoT设备的P2P通信行为
 *
 * 通过真实MQTT连接演示P2P直接通信流程
 *
 * 主要功能：
 * 1. 真实MQTT连接
 * 2. 订阅设备服务配置topic: dl/PLAF204/AF070135F064641AG/device/service/sub
 * 3. 接收P2P配置并发送设备就绪状态
 * 4. 通过定向topic进行WebRTC信令交换: dl/PRODUCT_KEY/DEVICE_SN/p2p/SESSION_ID/signal
 * 5. 维护P2P连接和心跳
 */
public class DeviceSimulator implements MqttCallback {

    private static final String MQTT_BROKER = "tcp://192.168.10.120:1883";
    private static final String CLIENT_ID_PREFIX = "DEVICE_";
    private static final String USERNAME = "DEVICE"; // 设备端认证
    private static final String PASSWORD = "DEVICE"; // 设备端认证

    // 模拟数据
    private static final String PRODUCT_KEY = "PLAF204";
    private static final String DEVICE_SN = "AF070135F064641AG";

    // MQTT客户端
    private MqttClient mqttClient;
    private boolean connected = false;
    private String currentSessionId;
    private String currentAppId;
    private boolean isReady = false;
    private boolean isConnected = false;
    private ScheduledExecutorService scheduler;
    private ScheduledFuture<?> heartbeatTask;

    // 监控字段
    private long sendReadyTs = 0;
    private long offerReceiveTs = 0;
    private long sendAnswerTs = 0;
    private long connectedSendTs = 0;

    // P2P配置字段
    private String[] stunServers;
    private String[] turnServers;
    private String rtcConfiguration;
    private String extraConfig;

    // 日志工具方法
    private void logInfo(String module, String message) {
        System.out.println(String.format("[%s] %s", module, message));
    }

    private void logSuccess(String module, String message) {
        System.out.println(String.format("[%s] ✓ %s", module, message));
    }

    private void logError(String module, String message) {
        System.err.println(String.format("[%s] ✗ %s", module, message));
    }

    private void logWarning(String module, String message) {
        System.out.println(String.format("[%s] ⚠ %s", module, message));
    }

    private void logStep(String step) {
        System.out.println(String.format("\n=== %s ===", step));
    }

    private void logSeparator() {
        System.out.println("─".repeat(60));
    }

    private void logMqttReceive(String topic, String payload) {
        System.out.println(String.format("[MQTT-IN] Topic: %s", topic));
        System.out.println(String.format("[MQTT-IN] Message: %s", formatJson(payload)));
    }

    private void logMqttSend(String topic, String payload) {
        System.out.println(String.format("[MQTT-OUT] Topic: %s", topic));
        System.out.println(String.format("[MQTT-OUT] Message: %s", formatJson(payload)));
    }

    private void logSignaling(String type, String direction, String details) {
        System.out.println(String.format("[SIGNAL-%s] %s: %s", direction, type, details));
    }

    private void logMonitor(String metric, String value) {
        System.out.println(String.format("[MONITOR] %s: %s", metric, value));
    }

    public static void main(String[] args) {
        DeviceSimulator simulator = new DeviceSimulator();
        try {
            simulator.start();
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    public void start() throws Exception {
        logStep("设备模拟器启动");
        logInfo("CONFIG", "Product Key: " + PRODUCT_KEY);
        logInfo("CONFIG", "Device SN: " + DEVICE_SN);
        logInfo("CONFIG", "MQTT Broker: " + MQTT_BROKER);
        logSeparator();

        // 1. 模拟连接MQTT
        connectMqtt();

        // 2. 模拟订阅相关topic
        subscribeTopics();

        // 3. 模拟设备初始化完成
        initializeDevice();

        // 4. 等待P2P配置
        waitForP2PConfiguration();

        // 5. 处理用户输入
        handleUserInput();
    }

    private void connectMqtt() throws Exception {
        String clientId = "DEVICE";
        logStep("MQTT连接初始化");
        logInfo("MQTT", "正在连接到服务器: " + MQTT_BROKER);
        logInfo("MQTT", "客户端ID: " + clientId);

        try {
            // 创建MQTT客户端
            mqttClient = new MqttClient(MQTT_BROKER, clientId, new MemoryPersistence());

            // 设置连接选项
            MqttConnectOptions options = new MqttConnectOptions();
            options.setCleanSession(true);
            options.setConnectionTimeout(10);
            options.setKeepAliveInterval(30);

            if (!USERNAME.isEmpty()) {
                options.setUserName(USERNAME);
                options.setPassword(PASSWORD.toCharArray());
                logInfo("MQTT", "使用认证连接");
            }

            // 设置回调
            mqttClient.setCallback(this);

            // 连接
            mqttClient.connect(options);
            connected = true;

            logSuccess("MQTT", "连接成功");
            logSeparator();
        } catch (MqttException e) {
            logError("MQTT", "连接失败: " + e.getMessage());
            throw e;
        }
    }

    private void subscribeTopics() throws Exception {
        logStep("订阅MQTT主题");
        try {
            // 订阅设备服务配置topic（接收P2P配置）
            String deviceServiceTopic = String.format("dl/%s/%s/device/service/sub", PRODUCT_KEY, DEVICE_SN);
            mqttClient.subscribe(deviceServiceTopic, 1);
            logSuccess("MQTT", "已订阅设备服务配置: " + deviceServiceTopic);

            // 注意：不再订阅广播信令topic，只订阅定向信令topic
            logInfo("MQTT", "等待P2P配置后订阅定向信令topic");
            logSuccess("MQTT", "基础topic订阅完成");
            logSeparator();
        } catch (MqttException e) {
            logError("MQTT", "订阅topic失败: " + e.getMessage());
            throw e;
        }
    }

    private void initializeDevice() {
        logStep("设备初始化");
        logSuccess("DEVICE", "设备初始化完成");
        logInfo("DEVICE", "等待P2P配置下发...");

        // 启动定时任务调度器
        scheduler = Executors.newScheduledThreadPool(2);
        logSeparator();
    }

    private void waitForP2PConfiguration() {
        logStep("等待P2P配置");
        logInfo("WAIT", "等待cloud-p2p-signalling下发P2P配置...");
        logInfo("WAIT", "提示：请先启动AppSimulator创建P2P会话");
        logInfo("WAIT", "AppSimulator调用API成功后，cloud-p2p-signalling会自动下发配置到设备");
        logInfo("WAIT", "或手动输入 'ready <sessionId>' 命令模拟接收到配置");
        logInfo("WAIT", "例如: ready session_12345");
        logSeparator();
    }

    // MQTT回调方法
    @Override
    public void connectionLost(Throwable cause) {
        logError("MQTT", "连接丢失: " + cause.getMessage());
        connected = false;
        // 可以在这里实现重连逻辑
    }

    @Override
    public void messageArrived(String topic, MqttMessage message) throws Exception {
        String payload = new String(message.getPayload());
        logMqttReceive(topic, payload);
        handleMqttMessage(topic, payload);
    }

    @Override
    public void deliveryComplete(IMqttDeliveryToken token) {
        // 消息发送完成回调
    }

    private void handleP2PConfiguration(JSONObject config) {
        logStep("收到P2P配置");
        logInfo("CONFIG", "配置内容: " + config.toJSONString());

        // 检查是否为P2P配置命令
        String cmd = config.getString("cmd");
        if (!"P2P_SIGNAL_CONFIG_SERVICE".equals(cmd)) {
            logWarning("CONFIG", "非P2P配置命令，忽略");
            return;
        }

        // 提取会话信息
        currentSessionId = config.getString("sessionId");
        currentAppId = String.valueOf(config.getLong("memberId")); // memberId作为APP ID

        logInfo("SESSION", "会话ID: " + currentSessionId);
        logInfo("SESSION", "APP ID: " + currentAppId);

        // 解析P2P配置
        if (config.containsKey("stunServers")) {
            stunServers = config.getJSONArray("stunServers").toArray(new String[0]);
            logInfo("CONFIG", "STUN服务器: " + String.join(", ", stunServers));
        }
        if (config.containsKey("turnServers")) {
            turnServers = config.getJSONArray("turnServers").toArray(new String[0]);
            logInfo("CONFIG", "TURN服务器: " + String.join(", ", turnServers));
            logInfo("CONFIG", "TURN认证: DEVICE/DEVICE (与MQTT保持一致)");
        }
        
        rtcConfiguration = config.getString("rtcConfiguration");
        extraConfig = config.getString("extraConfig");
        logSuccess("WEBRTC", "WebRTC配置初始化完成");

        // 订阅设备端信令topic（接收APP发送的消息）
        try {
            String deviceSignalTopic = String.format("dl/%s/%s/device/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN,
                    currentSessionId);
            mqttClient.subscribe(deviceSignalTopic, 1);
            logSuccess("MQTT", "已订阅设备信令: " + deviceSignalTopic);
        } catch (MqttException e) {
            logError("MQTT", "订阅设备信令失败: " + e.getMessage());
            return;
        }

        // 发送Ready消息
        sendReady();

        logSeparator();
    }

    private void sendReady() {
        try {
            logStep("发送Ready消息");

            sendReadyTs = System.currentTimeMillis();
            JSONObject readyMessage = new JSONObject();
            readyMessage.put("cmd", "WEB_RTC");
            readyMessage.put("ts", System.currentTimeMillis());
            readyMessage.put("msgId", System.currentTimeMillis());

            JSONObject readyData = new JSONObject();
            readyData.put("type", "ready");
            readyData.put("sessionId", currentSessionId);
            readyData.put("from", DEVICE_SN);
            readyData.put("to", currentAppId);
            readyData.put("capabilities", createDeviceCapabilities());

            readyMessage.put("data", readyData);

            // 设备发送到APP端订阅的topic
            String topic = String.format("dl/%s/%s/app/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN, currentSessionId);
            publishMessage(topic, readyMessage.toJSONString());

            isReady = true;
            logSignaling("READY", "OUT", "设备就绪消息已发送（定向）");
            logInfo("TOPIC", "使用定向topic: " + topic);
            logMonitor("Ready消息延迟(App->Device)", "0ms"); // 发送端无需延迟
            logSuccess("DEVICE", "设备已就绪，等待APP发起信令交换");

            // 只接收真实APP发送的消息，不自动发送模拟数据

            logSeparator();
        } catch (Exception e) {
            logError("SIGNAL", "发送Ready消息失败: " + e.getMessage());
        }
    }

    private JSONObject createDeviceCapabilities() {
        JSONObject capabilities = new JSONObject();
        capabilities.put("video", true);
        capabilities.put("audio", true);
        capabilities.put("maxResolution", "1920x1080");
        capabilities.put("codecs", "H264,H265");
        return capabilities;
    }

    private void handleMqttMessage(String topic, String payload) {
        try {
            JSONObject message = JSON.parseObject(payload);

            // 判断消息类型
            if (topic.contains("/device/service/sub")) {
                // 设备服务配置消息
                handleP2PConfiguration(message);
                return;
            }

            // 检查是否为P2P数据通道消息
            if (topic.contains("/p2p/") && topic.contains("/data")) {
                handleP2PDataMessage(message);
                return;
            }

            // 支持两种结构：旧结构(type) 和新结构(cmd)
            String cmd = message.getString("cmd");
            if (cmd == null || !cmd.equals("WEB_RTC")) {
                // 非WebRTC消息或无效结构
                logWarning("SIGNAL", "非WEB_RTC消息类型: " + (cmd != null ? cmd : "未知"));
                return;
            }

            JSONObject data = message.getJSONObject("data");
            if (data == null) {
                logWarning("SIGNAL", "消息数据为空");
                return;
            }

            // 信令消息
            String type = data.getString("type");
            String from = data.getString("from");
            String to = data.getString("to");
            String sessionId = data.getString("sessionId");

            if (type == null) {
                logWarning("SIGNAL", "消息类型为空，忽略");
                return;
            }

            logInfo("SIGNAL", String.format("收到消息类型: %s, 来源: %s", type, from));

            switch (type) {
                case "offer":
                    handleOffer(message);
                    break;
                case "candidate":
                    handleIceCandidate(message);
                    break;
                case "connected":
                    handleConnected(message);
                    break;
                case "bye":
                    handleBye(message);
                    break;
                case "keepalive":
                    handleMQTTKeepAlive(message);
                    break;
                default:
                    logWarning("SIGNAL", "未知消息类型: " + type);
            }
        } catch (Exception e) {
            logError("SIGNAL", "处理MQTT消息失败: " + e.getMessage());
            e.printStackTrace();
        }
    }

    /**
     * 处理P2P数据通道消息
     */
    private void handleP2PDataMessage(JSONObject message) {
        String cmd = message.getString("cmd");
        if ("P2P_KEEPALIVE".equals(cmd)) {
            handleP2PKeepAlive(message);
        } else {
            logInfo("P2P-DATA", "收到P2P数据消息: " + cmd);
        }
    }

    /**
     * 处理P2P通道保活消息
     */
    private void handleP2PKeepAlive(JSONObject message) {
        logSignaling("P2P-KEEPALIVE", "IN", "收到P2P通道保活消息");
        
        String fromId = message.getString("from");
        String sessionId = message.getString("sessionId");
        Integer sequence = message.getInteger("sequence");
        String channel = message.getString("channel");
        
        logInfo("P2P-KEEPALIVE", "来源: " + fromId + ", 会话: " + sessionId);
        logInfo("P2P-KEEPALIVE", "序号: " + sequence + ", 通道: " + channel);

        long now = System.currentTimeMillis();
        long msgTs = message.getLongValue("ts");
        if (msgTs > 0) {
            logMonitor("P2P保活消息延迟", (now - msgTs) + "ms");
        }

        // 响应P2P保活
        respondToP2PKeepAlive(message);
    }

    /**
     * 处理MQTT通道保活消息
     */
    private void handleMQTTKeepAlive(JSONObject message) {
        logSignaling("MQTT-KEEPALIVE", "IN", "收到MQTT通道保活消息");
        JSONObject data = message.getJSONObject("data");
        
        String fromId = null;
        String sessionId = null;
        Integer sequence = null;
        String channel = null;
        
        if (data != null) {
            fromId = data.getString("from");
            sessionId = data.getString("sessionId");
            sequence = data.getInteger("sequence");
            channel = data.getString("channel");
        }
        
        logInfo("MQTT-KEEPALIVE", "来源: " + fromId + ", 会话: " + sessionId);
        logInfo("MQTT-KEEPALIVE", "序号: " + sequence + ", 通道: " + channel);

        long now = System.currentTimeMillis();
        long msgTs = message.getLongValue("ts");
        if (msgTs > 0) {
            logMonitor("MQTT保活消息延迟", (now - msgTs) + "ms");
        }

        // 响应MQTT保活
        respondToMQTTKeepAlive(message);
    }

    /**
     * 响应P2P保活消息
     */
    private void respondToP2PKeepAlive(JSONObject originalMessage) {
        try {
            JSONObject response = new JSONObject();
            response.put("cmd", "P2P_KEEPALIVE_RESPONSE");
            response.put("ts", System.currentTimeMillis());
            response.put("msgId", "p2p_keepalive_resp_" + System.currentTimeMillis());
            response.put("sessionId", originalMessage.getString("sessionId"));
            response.put("from", DEVICE_SN);
            response.put("to", originalMessage.getString("from"));
            response.put("originalSequence", originalMessage.getInteger("sequence"));
            response.put("channel", "P2P");
            response.put("status", "alive");

            // 通过P2P数据通道响应
            String p2pTopic = String.format("dl/%s/%s/p2p/%s/data", PRODUCT_KEY, DEVICE_SN, currentSessionId);
            publishMqttMessage(p2pTopic, response.toJSONString());
            
            logInfo("P2P-KEEPALIVE", "P2P保活响应已发送");
            
        } catch (Exception e) {
            logError("P2P-KEEPALIVE", "发送P2P保活响应失败: " + e.getMessage());
        }
    }

    /**
     * 响应MQTT保活消息
     */
    private void respondToMQTTKeepAlive(JSONObject originalMessage) {
        try {
            JSONObject response = new JSONObject();
            response.put("cmd", "WEB_RTC");
            response.put("ts", System.currentTimeMillis());
            response.put("msgId", "mqtt_keepalive_resp_" + System.currentTimeMillis());
            
            JSONObject data = new JSONObject();
            JSONObject originalData = originalMessage.getJSONObject("data");
            data.put("sessionId", originalData.getString("sessionId"));
            data.put("type", "keepalive_response");
            data.put("from", DEVICE_SN);
            data.put("to", originalData.getString("from"));
            data.put("originalSequence", originalData.getInteger("sequence"));
            data.put("channel", "MQTT");
            data.put("status", "alive");
            
            response.put("data", data);

            // 通过MQTT信令通道响应
            String mqttTopic = String.format("dl/%s/%s/app/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN, currentSessionId);
            publishMqttMessage(mqttTopic, response.toJSONString());
            
            logInfo("MQTT-KEEPALIVE", "MQTT保活响应已发送");
            
        } catch (Exception e) {
            logError("MQTT-KEEPALIVE", "发送MQTT保活响应失败: " + e.getMessage());
        }
    }

    private void handleOffer(JSONObject message) {
        try {
            logSignaling("OFFER", "IN", "收到APP的Offer信令");

            long now = System.currentTimeMillis();
            long msgTs = message.getLongValue("ts");
            if (msgTs > 0) {
                logMonitor("Offer消息延迟", (now - msgTs) + "ms");
            }
            JSONObject data = message.getJSONObject("data");
            if (data != null) {
                String fromId = data.getString("from");
                if (currentAppId == null) {
                    currentAppId = fromId;
                    logInfo("SESSION", "设置APP ID: " + currentAppId);
                }

                String sdp = data.getString("sdp");
                if (sdp != null && sdp.length() > 50) {
                    logInfo("WEBRTC", "SDP: " + sdp.substring(0, 50) + "...");
                }
            }

            // 只记录收到Offer，不自动发送Answer
            logInfo("WEBRTC", "收到Offer，等待手动操作或真实设备响应");
            logInfo("HINT", "可以在手动模式下使用 'offer' 命令来模拟设备响应");

            offerReceiveTs = System.currentTimeMillis();
            if (sendReadyTs > 0) {
                logMonitor("Ready->Offer耗时", (offerReceiveTs - sendReadyTs) + "ms");
            }

            logSeparator();

        } catch (Exception e) {
            logError("SIGNAL", "处理Offer失败: " + e.getMessage());
        }
    }

    private void sendAnswer() throws Exception {
        logStep("发送Answer信令");

        sendAnswerTs = System.currentTimeMillis();
        JSONObject answerMessage = new JSONObject();
        answerMessage.put("cmd", "WEB_RTC");
        answerMessage.put("ts", System.currentTimeMillis());
        answerMessage.put("msgId", System.currentTimeMillis());

        JSONObject answerData = new JSONObject();
        answerData.put("type", "answer");
        answerData.put("sdp", "v=0\\r\\no=- 9876543210 9876543210 IN IP4 192.168.1.200\\r\\ns=-\\r\\n...");
        answerData.put("from", DEVICE_SN);
        answerData.put("to", currentAppId);
        answerData.put("sessionId", currentSessionId);
        answerData.put("candidate", "");

        answerMessage.put("data", answerData);

        // 设备发送到APP端订阅的topic
        String topic = String.format("dl/%s/%s/app/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN, currentSessionId);
        publishMessage(topic, answerMessage.toJSONString());
        logSignaling("ANSWER", "OUT", "Answer信令已发送到APP");
        logMonitor("Answer发送耗时", (System.currentTimeMillis() - sendAnswerTs) + "ms");
        logSeparator();
    }

    private void sendIceCandidate() {
        try {
            logStep("发送ICE候选");

            JSONObject candidateMessage = new JSONObject();
            candidateMessage.put("cmd", "WEB_RTC");
            candidateMessage.put("ts", System.currentTimeMillis());
            candidateMessage.put("msgId", System.currentTimeMillis());

            JSONObject candidateData = new JSONObject();
            candidateData.put("type", "candidate");
            candidateData.put("candidate", "candidate:1 1 UDP 2013266431 192.168.1.200 12345 typ host");
            candidateData.put("from", DEVICE_SN);
            candidateData.put("to", currentAppId);
            candidateData.put("sessionId", currentSessionId);
            candidateData.put("sdpMid", "0");
            candidateData.put("sdpMLineIndex", 0);

            candidateMessage.put("data", candidateData);

            // 设备发送到APP端订阅的topic
            String topic = String.format("dl/%s/%s/app/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN, currentSessionId);
            publishMessage(topic, candidateMessage.toJSONString());
            logSignaling("ICE", "OUT", "ICE候选已发送到APP");
            logSeparator();
        } catch (Exception e) {
            logError("SIGNAL", "发送ICE候选失败: " + e.getMessage());
        }
    }

    private void handleIceCandidate(JSONObject message) {
        logSignaling("ICE", "IN", "收到APP的ICE候选");
        JSONObject data = message.getJSONObject("data");
        if (data != null) {
            logInfo("WEBRTC", "Candidate: " + data.getString("candidate"));
            logInfo("WEBRTC", "SDP MID: " + data.getString("sdpMid"));
        }

        logInfo("WEBRTC", "处理ICE候选...");
        logInfo("WEBRTC", "实际场景: 添加到WebRTC连接中");

        // 模拟ICE候选处理成功后自动发送连接成功消息
        try {
            logInfo("WEBRTC", "ICE连接建立成功，发送连接状态");
            sendConnected();
        } catch (Exception e) {
            logError("WEBRTC", "发送连接状态失败: " + e.getMessage());
        }

        logSeparator();
    }

    private void handleConnected(JSONObject message) {
        JSONObject data = message.getJSONObject("data");
        String fromId = null;
        if (data != null) {
            fromId = data.getString("from");
        }

        logSignaling("CONNECTED", "IN", "收到连接建立消息");
        logInfo("DEVICE", "来源: " + (fromId != null ? fromId : "unknown"));

        // 检查是否来自当前APP或者是手动触发的连接
        if ((currentAppId != null && currentAppId.equals(fromId)) ||
                (currentAppId == null && fromId == null)) {
            isConnected = true;
            logSignaling("CONNECTED", "IN", "P2P连接已建立");
            logSuccess("P2P", "开始音视频流传输...");
            logInfo("P2P", "实际场景: WebRTC DataChannel或MediaStream传输");

            // 启动心跳
            startHeartbeat();
            logSeparator();
        } else {
            logWarning("DEVICE", "连接消息来源不匹配，当前APP ID: " + currentAppId + ", 消息来源: " + fromId);
            logSeparator();
        }
    }

    private void handleBye(JSONObject message) {
        JSONObject data = message.getJSONObject("data");
        String fromId = null;
        String reason = null;

        if (data != null) {
            fromId = data.getString("from");
            reason = data.getString("reason");
        }

        logSignaling("BYE", "IN", "收到断开信号 from " + (fromId != null ? fromId : "unknown"));
        if (reason != null) {
            logInfo("DEVICE", "断开原因: " + reason);
        }

        // 清理连接状态
        isConnected = false;
        stopHeartbeat();

        // 如果是APP发起的断开，设备也响应
        if (currentAppId != null && currentAppId.equals(fromId)) {
            sendBye("响应APP断开请求");
        }
        logSeparator();
    }

    private void sendConnected() throws Exception {
        logStep("发送连接状态");

        connectedSendTs = System.currentTimeMillis();
        JSONObject connectedMessage = new JSONObject();
        connectedMessage.put("cmd", "WEB_RTC");
        connectedMessage.put("ts", System.currentTimeMillis());
        connectedMessage.put("msgId", System.currentTimeMillis());

        JSONObject connectedData = new JSONObject();
        connectedData.put("type", "connected");
        connectedData.put("sessionId", currentSessionId);
        connectedData.put("from", DEVICE_SN);
        connectedData.put("to", currentAppId);

        connectedMessage.put("data", connectedData);

        // 设备发送到APP端订阅的topic
        String topic = String.format("dl/%s/%s/app/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN, currentSessionId);
        publishMessage(topic, connectedMessage.toJSONString());
        logSignaling("CONNECTED", "OUT", "Connected状态已发送到APP");
        logInfo("TOPIC", "使用定向topic: " + topic);
        logMonitor("Connected发送耗时", (System.currentTimeMillis() - connectedSendTs) + "ms");

        // 直接处理连接建立，因为是设备自己发送的
        isConnected = true;
        startHeartbeat();
        logSuccess("P2P", "连接已建立，开始心跳");
        logSeparator();
    }

    private void sendBye(String reason) {
        try {
            logStep("发送断开消息");

            JSONObject byeMessage = new JSONObject();
            byeMessage.put("cmd", "WEB_RTC");
            byeMessage.put("ts", System.currentTimeMillis());
            byeMessage.put("msgId", System.currentTimeMillis());

            JSONObject byeData = new JSONObject();
            byeData.put("type", "bye");
            byeData.put("sessionId", currentSessionId);
            byeData.put("from", DEVICE_SN);
            byeData.put("to", currentAppId);
            byeData.put("reason", reason);

            byeMessage.put("data", byeData);

            // 设备发送到APP端订阅的topic
            String topic = String.format("dl/%s/%s/app/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN, currentSessionId);
            publishMessage(topic, byeMessage.toJSONString());
            logSignaling("BYE", "OUT", "Bye消息已发送到APP");
            logInfo("TOPIC", "使用定向topic: " + topic);

            isConnected = false;
            stopHeartbeat();
            logSeparator();
        } catch (Exception e) {
            logError("SIGNAL", "发送Bye消息失败: " + e.getMessage());
        }
    }

    private void startHeartbeat() {
        // 先停止现有的心跳任务
        if (heartbeatTask != null && !heartbeatTask.isCancelled()) {
            heartbeatTask.cancel(false);
        }

        if (scheduler != null) {
            heartbeatTask = scheduler.scheduleAtFixedRate(() -> {
                if (isConnected) {
                    try {
                        sendHeartbeat();
                    } catch (Exception e) {
                        logError("HEARTBEAT", "发送心跳失败: " + e.getMessage());
                    }
                }
            }, 30, 30, TimeUnit.SECONDS);
            logSuccess("HEARTBEAT", "心跳启动 (30秒间隔)");
        }
    }

    private void stopHeartbeat() {
        // 取消心跳任务
        if (heartbeatTask != null && !heartbeatTask.isCancelled()) {
            heartbeatTask.cancel(false);
            heartbeatTask = null;
            logSuccess("HEARTBEAT", "心跳任务已取消");
        }
    }

    private void sendHeartbeat() throws Exception {
        JSONObject heartbeatMessage = new JSONObject();
        heartbeatMessage.put("cmd", "WEB_RTC");
        heartbeatMessage.put("ts", System.currentTimeMillis());
        heartbeatMessage.put("msgId", System.currentTimeMillis());

        JSONObject heartbeatData = new JSONObject();
        heartbeatData.put("type", "keepalive");
        heartbeatData.put("sessionId", currentSessionId);
        heartbeatData.put("from", DEVICE_SN);
        heartbeatData.put("to", currentAppId);

        heartbeatMessage.put("data", heartbeatData);

        // 设备发送到APP端订阅的topic
        String topic = String.format("dl/%s/%s/app/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN, currentSessionId);
        publishMessage(topic, heartbeatMessage.toJSONString());
        logSignaling("KEEPALIVE", "OUT", "心跳已发送到APP");
    }

    private void publishMessage(String topic, String payload) throws Exception {
        if (!connected || mqttClient == null || !mqttClient.isConnected()) {
            logError("MQTT", "未连接，无法发送消息");
            return;
        }

        try {
            MqttMessage message = new MqttMessage(payload.getBytes());
            message.setQos(1);
            mqttClient.publish(topic, message);

            logMqttSend(topic, payload);
        } catch (MqttException e) {
            logError("MQTT", "发布消息失败: " + e.getMessage());
            throw e;
        }
    }

    private void handleUserInput() {
        Scanner scanner = new Scanner(System.in);
        logStep("设备消息接收器");
        logInfo("MODE", "设备现在只接收APP发送的真实消息");
        logInfo("MODE", "已删除所有自动发送的模拟数据功能");
        logInfo("HELP", "可用命令:");
        logInfo("HELP", "  status    - 查看当前状态");
        logInfo("HELP", "  manual    - 切换到手动模式");
        logInfo("HELP", "  quit      - 退出程序");
        logSeparator();

        while (true) {
            System.out.print("DEVICE-RECEIVER> ");
            String input = scanner.nextLine().trim();

            try {
                String command = input.split(" ")[0];
                switch (command) {
                    case "status":
                        showStatus();
                        break;
                    case "manual":
                        enterManualMode(scanner);
                        break;
                    case "quit":
                        logInfo("SYSTEM", "程序退出");
                        if (scheduler != null) {
                            scheduler.shutdown();
                        }
                        return;
                    case "":
                        break;
                    default:
                        logWarning("INPUT", "未知命令: " + input);
                        logInfo("HELP", "输入 'status' 查看状态，'manual' 进入手动模式，'quit' 退出");
                }
            } catch (Exception e) {
                logError("COMMAND", "执行命令失败: " + e.getMessage());
            }
        }
    }

    private void showStatus() {
        logStep("设备状态");
        logInfo("SESSION", "会话ID: " + (currentSessionId != null ? currentSessionId : "未设置"));
        logInfo("SESSION", "APP ID: " + (currentAppId != null ? currentAppId : "未设置"));
        logInfo("DEVICE", "就绪状态: " + (isReady ? "已就绪" : "未就绪"));
        logInfo("P2P", "连接状态: " + (isConnected ? "已连接" : "未连接"));
        logInfo("MQTT", "MQTT连接: " + (connected ? "已连接" : "未连接"));
        logSeparator();
    }

    private void enterManualMode(Scanner scanner) {
        logStep("切换到手动模式");
        logInfo("MANUAL", "进入手动控制模式");
        logInfo("HELP", "手动模式命令:");
        logInfo("HELP", "  ready [sessionId] - 发送就绪消息");
        logInfo("HELP", "  offer     - 提示（已删除自动模拟功能）");
        logInfo("HELP", "  candidate - 发送ICE候选");
        logInfo("HELP", "  connected - 发送连接状态");
        logInfo("HELP", "  bye       - 发送断开消息");
        logInfo("HELP", "  auto      - 返回接收模式");
        logInfo("HELP", "  quit      - 退出程序");
        logSeparator();

        while (true) {
            System.out.print("DEVICE-MANUAL> ");
            String input = scanner.nextLine().trim();

            try {
                String command = input.split(" ")[0];
                switch (command) {
                    case "ready":
                        String[] parts = input.split(" ", 2);
                        if (parts.length > 1) {
                            currentSessionId = parts[1];
                            currentAppId = "app_188815492";
                            logInfo("SESSION", "使用指定会话ID: " + currentSessionId);
                        } else if (currentSessionId == null) {
                            currentSessionId = "manual_session_" + System.currentTimeMillis();
                            currentAppId = "manual_app";
                            logInfo("SESSION", "生成会话ID: " + currentSessionId);
                        }
                        sendReady();
                        break;
                    case "offer":
                        logInfo("MANUAL", "已删除自动模拟功能，只接收真实APP发送的Offer");
                        break;
                    case "candidate":
                        sendIceCandidate();
                        break;
                    case "connected":
                        sendConnected();
                        break;
                    case "bye":
                        sendBye("用户手动断开");
                        break;
                    case "auto":
                        logInfo("MODE", "返回接收模式");
                        return; // 返回接收模式
                    case "quit":
                        logInfo("SYSTEM", "程序退出");
                        if (scheduler != null) {
                            scheduler.shutdown();
                        }
                        System.exit(0);
                        return;
                    case "":
                        break;
                    default:
                        logWarning("INPUT", "未知命令: " + input);
                }
            } catch (Exception e) {
                logError("COMMAND", "执行命令失败: " + e.getMessage());
            }
        }
    }

    // 删除了所有自动模拟方法 - 只接收真实APP发送的消息

    /**
     * 格式化JSON显示
     */
    private String formatJson(String json) {
        try {
            JSONObject jsonObject = JSON.parseObject(json);
            return JSON.toJSONString(jsonObject, true);
        } catch (Exception e) {
            return json;
        }
    }

    // 删除了所有mock方法 - 只接收真实APP发送的消息
}

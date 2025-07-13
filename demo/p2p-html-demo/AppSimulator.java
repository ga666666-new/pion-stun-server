package com.dl.cloud.mall;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.HashMap;
import java.util.Map;
import java.util.Scanner;

import cn.hutool.json.JSONUtil;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.JsonNode;
import org.eclipse.paho.client.mqttv3.*;
import org.eclipse.paho.client.mqttv3.persist.MemoryPersistence;
import com.alibaba.fastjson.JSON;
import com.alibaba.fastjson.JSONObject;

/**
 * APP端P2P信令模拟器
 * 模拟APP端的P2P通信流程，调用真实的API接口并通过MQTT进行信令交换
 */
public class AppSimulator implements MqttCallback {

    // 真实数据配置
    private static final String PRODUCT_KEY = "PLAF204";
    private static final String DEVICE_SN = "AF070135F064641AG";
    private static final String APP_ID = "188815492";
    private static String token = "f6465b2bb4b048e1b159f2a7ff9d17df"; // 改为可变，支持动态更新

    // 服务配置
    private static final String P2P_API_BASE_URL = "http://localhost:8067/api/v1/signal";
    private static final String MQTT_BROKER_URL = "tcp://192.168.10.120:1883";
    private static final String MQTT_CLIENT_ID = "APP";
    private static final String MQTT_USERNAME = "APP"; // APP端认证
    private static final String MQTT_PASSWORD = "APP"; // APP端认证

    // HTTP客户端
    private static final HttpClient httpClient = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(10))
            .build();

    private static final ObjectMapper objectMapper = new ObjectMapper();

    // MQTT客户端
    private MqttClient mqttClient;

    // 状态
    private boolean connected = false;
    private String currentSessionId = null;
    private String deviceReadyStatus = "waiting";
    private boolean autoMode = false;
    private boolean sessionCreated = false;
    private long sessionCreateTs = 0;
    private long sendOfferTs = 0;
    private long sendCandidateTs = 0;
    private long readyReceiveTs = 0;
    private long answerReceiveTs = 0;

    // P2P配置字段
    private String[] stunServers;
    private String[] turnServers;
    private String iceTransportPolicy;
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

    private void logHttpRequest(String method, String url, int statusCode) {
        System.out.println(String.format("[HTTP] %s %s -> %d", method, url, statusCode));
    }

    private void logSignaling(String type, String direction, String details) {
        System.out.println(String.format("[SIGNAL-%s] %s: %s", direction, type, details));
    }

    private void logMonitor(String metric, String value) {
        System.out.println(String.format("[MONITOR] %s: %s", metric, value));
    }

    public static void main(String[] args) {
        AppSimulator simulator = new AppSimulator();
        simulator.startSimulation();
    }

    public void startSimulation() {
        logStep("APP端P2P信令模拟器启动");
        logInfo("CONFIG", "产品密钥: " + PRODUCT_KEY);
        logInfo("CONFIG", "设备序列号: " + DEVICE_SN);
        logInfo("CONFIG", "APP ID: " + APP_ID);
        logInfo("CONFIG", "MQTT服务器: " + MQTT_BROKER_URL);
        logSeparator();

        // 模拟MQTT连接
        connectMqtt();

        Scanner scanner = new Scanner(System.in);

        while (true) {
            System.out.println("\n请选择操作:");
            System.out.println("1. 创建P2P会话");
            System.out.println("2. 查询会话状态");
            System.out.println("3. 发送Offer信令");
            System.out.println("4. 发送ICE候选");
            System.out.println("5. 发送心跳");
            System.out.println("6. 结束会话");
            System.out.println("7. 自动模式 - 基于P2P配置建立真实连接");
            System.out.println("0. 退出");
            System.out.print("请输入选择: ");

            String input = scanner.nextLine();

            try {
                switch (input) {
                    case "1":
                        createP2PSession();
                        break;
                    case "2":
                        checkSessionStatus();
                        break;
                    case "3":
                        sendOffer();
                        break;
                    case "4":
                        sendICECandidate();
                        break;
                    case "5":
                        sendHeartbeat();
                        break;
                    case "6":
                        endSession();
                        break;
                    case "7":
                        startAutoMode();
                        break;
                    case "0":
                        logInfo("SYSTEM", "退出模拟器");
                        if (mqttClient != null && mqttClient.isConnected()) {
                            mqttClient.disconnect();
                        }
                        return;
                    default:
                        logWarning("INPUT", "无效选择，请重新输入");
                }
            } catch (Exception e) {
                logError("OPERATION", "操作失败: " + e.getMessage());
                e.printStackTrace();
            }
        }
    }

    /**
     * 连接MQTT服务器
     */
    private void connectMqtt() {
        logStep("MQTT连接初始化");
        logInfo("MQTT", "正在连接到服务器: " + MQTT_BROKER_URL);
        logInfo("MQTT", "客户端ID: " + MQTT_CLIENT_ID);

        try {
            // 创建MQTT客户端
            mqttClient = new MqttClient(MQTT_BROKER_URL, MQTT_CLIENT_ID, new MemoryPersistence());

            // 设置连接选项
            MqttConnectOptions options = new MqttConnectOptions();
            options.setCleanSession(true);
            options.setConnectionTimeout(10);
            options.setKeepAliveInterval(30);

            if (!MQTT_USERNAME.isEmpty()) {
                options.setUserName(MQTT_USERNAME);
                options.setPassword(MQTT_PASSWORD.toCharArray());
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
            e.printStackTrace();
        }
    }

    // MQTT回调方法
    @Override
    public void connectionLost(Throwable cause) {
        logError("MQTT", "连接丢失: " + cause.getMessage());
        connected = false;
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

    /**
     * 处理接收到的MQTT消息
     */
    private void handleMqttMessage(String topic, String payload) {
        try {
            JSONObject message = JSON.parseObject(payload);

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

            String type = data.getString("type");
            String from = data.getString("from");
            String to = data.getString("to");
            String sessionId = data.getString("sessionId");

            if (type == null) {
                logWarning("SIGNAL", "消息类型为空，忽略消息");
                return;
            }

            // 过滤来自APP自己的消息（MQTT会回显自己发送的消息）
            if (APP_ID.equals(from)) {
                logInfo("SIGNAL", String.format("忽略来自自己的消息: %s", type));
                return;
            }

            logInfo("SIGNAL", String.format("收到消息类型: %s, 来源: %s", type, from));

            switch (type) {
                case "ready":
                    handleDeviceReady(message);
                    break;
                case "answer":
                    handleDeviceAnswer(message);
                    break;
                case "candidate":
                    handleDeviceICE(message);
                    break;
                case "connected":
                    handleDeviceConnected(message);
                    break;
                case "bye":
                    handleDeviceBye(message);
                    break;
                case "keepalive":
                    handleDeviceKeepAlive(message);
                    break;
                default:
                    logWarning("SIGNAL", "未知消息类型: " + type);
            }
        } catch (Exception e) {
            logError("SIGNAL", "处理MQTT消息失败: " + e.getMessage());
        }
    }

    /**
     * 创建P2P会话
     */
    private void createP2PSession() throws IOException, InterruptedException {
        logStep("创建P2P会话");

        // 构建请求数据
        Map req = new HashMap();
        req.put("deviceSn", DEVICE_SN);
        String requestBody = JSONUtil.toJsonStr(req);

        logInfo("HTTP", "准备发送创建会话请求");
        logInfo("HTTP", "目标设备: " + DEVICE_SN);

        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(P2P_API_BASE_URL + "/session"))
                .header("Content-Type", "application/json")
                .header("Authorization", "Bearer mock_token_" + APP_ID)
                .header("source", "IOS")
                .header("token", token)
                .POST(HttpRequest.BodyPublishers.ofString(requestBody))
                .timeout(Duration.ofSeconds(30))
                .build();

        HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());

        logHttpRequest("POST", P2P_API_BASE_URL + "/session", response.statusCode());
        logInfo("HTTP", "响应内容: " + response.body());

        if (response.statusCode() == 200) {
            // 解析响应获取sessionId
            JsonNode responseJson = objectMapper.readTree(response.body());
            if (responseJson.has("data") && responseJson.get("data").has("sessionId")) {
                currentSessionId = responseJson.get("data").get("sessionId").asText();
                sessionCreateTs = System.currentTimeMillis();
                logSuccess("SESSION", "会话创建成功");
                logInfo("SESSION", "会话ID: " + currentSessionId);

                // 订阅APP端信令topic（接收设备发送的消息）
                try {
                    String appSignalTopic = String.format("dl/%s/%s/app/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN,
                            currentSessionId);
                    mqttClient.subscribe(appSignalTopic, 1);
                    logSuccess("MQTT", "已订阅APP信令topic: " + appSignalTopic);
                } catch (MqttException e) {
                    logError("MQTT", "订阅APP信令失败: " + e.getMessage());
                }

                // 等待设备就绪
                logStep("等待设备就绪");
                logInfo("WAIT", "等待设备接收P2P配置并发送Ready消息...");
                deviceReadyStatus = "waiting";
            } else {
                logError("HTTP", "响应格式异常，未找到sessionId");
            }
        } else {
            logError("HTTP", "会话创建失败，状态码: " + response.statusCode());
        }
        logSeparator();
    }

    /**
     * 处理设备Ready消息
     */
    private void handleDeviceReady(JSONObject message) {
        logSignaling("READY", "IN", "设备已就绪");
        JSONObject data = message.getJSONObject("data");
        String sessionId = null;
        if (data != null) {
            sessionId = data.getString("sessionId");
        }

        if (currentSessionId != null && currentSessionId.equals(sessionId)) {
            deviceReadyStatus = "device_ready";
            logSuccess("DEVICE", "设备已就绪，可以开始信令交换");
            readyReceiveTs = System.currentTimeMillis();

            if (data != null) {
                JSONObject capabilities = data.getJSONObject("capabilities");
                if (capabilities != null) {
                    logInfo("DEVICE", "设备能力: " + capabilities.toJSONString());
                }
            }

            // 如果是自动模式，获取P2P配置并开始真实连接流程
            if (autoMode && !sessionCreated) {
                sessionCreated = true;
                logInfo("AUTO", "检测到自动模式，获取P2P配置并开始真实连接流程");
                fetchP2PConfigAndConnect();
            }
        } else {
            logWarning("DEVICE", "会话ID不匹配，忽略Ready消息");
        }
        logSeparator();
    }

    /**
     * 处理设备Answer消息
     */
    private void handleDeviceAnswer(JSONObject message) {
        logSignaling("ANSWER", "IN", "收到设备Answer");
        JSONObject data = message.getJSONObject("data");
        String sessionId = null;
        if (data != null) {
            sessionId = data.getString("sessionId");
        }

        if (currentSessionId != null && currentSessionId.equals(sessionId)) {
            if (data != null) {
                String sdp = data.getString("sdp");
                if (sdp != null && sdp.length() > 50) {
                    logInfo("WEBRTC", "SDP: " + sdp.substring(0, 50) + "...");
                }
            }
            answerReceiveTs = System.currentTimeMillis();
            logMonitor("Offer->Answer耗时", (answerReceiveTs - sendOfferTs) + "ms");
            logSuccess("WEBRTC", "Answer协商完成");
        } else {
            logWarning("DEVICE", "会话ID不匹配，忽略Answer消息");
        }
        logSeparator();
    }

    /**
     * 处理设备ICE候选
     */
    private void handleDeviceICE(JSONObject message) {
        logSignaling("ICE", "IN", "收到设备ICE候选");
        JSONObject data = message.getJSONObject("data");
        String sessionId = null;
        if (data != null) {
            sessionId = data.getString("sessionId");
        }

        if (currentSessionId != null && currentSessionId.equals(sessionId)) {
            if (data != null) {
                logInfo("WEBRTC", "Candidate: " + data.getString("candidate"));
                long now = System.currentTimeMillis();
                long msgTs = message.getLongValue("ts");
                if (msgTs > 0) {
                    logMonitor("ICE消息延迟", (now - msgTs) + "ms");
                }
            }
        } else {
            logWarning("DEVICE", "会话ID不匹配，忽略ICE候选");
        }
        logSeparator();
    }

    /**
     * 处理设备连接状态
     */
    private void handleDeviceConnected(JSONObject message) {
        logSignaling("CONNECTED", "IN", "设备连接已建立");
        logSuccess("P2P", "连接建立成功，可以开始数据传输");
        long connectedTs = System.currentTimeMillis();
        logMonitor("会话到Connected耗时", (connectedTs - sessionCreateTs) + "ms");
        logSeparator();
    }

    /**
     * 处理设备断开消息
     */
    private void handleDeviceBye(JSONObject message) {
        logSignaling("BYE", "IN", "收到设备断开消息");
        JSONObject data = message.getJSONObject("data");
        String reason = null;
        if (data != null) {
            reason = data.getString("reason");
        }
        if (reason != null) {
            logInfo("DEVICE", "断开原因: " + reason);
        }
        logSeparator();
    }

    /**
     * 处理设备心跳
     */
    private void handleDeviceKeepAlive(JSONObject message) {
        logSignaling("KEEPALIVE", "IN", "收到设备心跳");
        long now = System.currentTimeMillis();
        long msgTs = message.getLongValue("ts");
        if (msgTs > 0) {
            logMonitor("KEEPALIVE消息延迟", (now - msgTs) + "ms");
        }
    }

    /**
     * 查询会话状态
     */
    private void checkSessionStatus() throws IOException, InterruptedException {
        if (currentSessionId == null) {
            logError("SESSION", "当前没有活跃会话");
            return;
        }

        logStep("查询会话状态");
        logInfo("SESSION", "查询会话ID: " + currentSessionId);

        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(P2P_API_BASE_URL + "/session/" + currentSessionId + "/status"))
                .header("Authorization", "Bearer mock_token_" + APP_ID)
                .header("X-User-Id", APP_ID)
                .header("source", "IOS")
                .header("token", token)
                .GET()
                .timeout(Duration.ofSeconds(10))
                .build();

        HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());

        logHttpRequest("GET", P2P_API_BASE_URL + "/session/" + currentSessionId + "/status", response.statusCode());
        logInfo("HTTP", "响应内容: " + response.body());
        logSeparator();
    }

    /**
     * 发送Offer信令
     */
    private void sendOffer() {
        if (!deviceReadyStatus.equals("device_ready")) {
            logError("SIGNAL", "设备尚未就绪，无法发送Offer");
            return;
        }

        logStep("发送Offer信令");

        String offerSdp = "v=0\r\no=- 123456789 2 IN IP4 127.0.0.1\r\ns=-\r\n..."; // 模拟SDP
        sendOfferTs = System.currentTimeMillis();
        // APP发送到设备端订阅的topic
        String targetTopic = String.format("dl/%s/%s/device/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN,
                currentSessionId);

        Map<String, Object> body = new HashMap<>();

        body.put("cmd", "WEB_RTC");
        body.put("ts", System.currentTimeMillis());
        body.put("msgId", System.currentTimeMillis());
        body.put("data", Map.of(
                "sessionId", currentSessionId,
                "type", "offer",
                "sdp", offerSdp,
                "from", APP_ID,
                "to", DEVICE_SN,
                "candidate", ""));

        String message = JSON.toJSONString(body);
        publishMqttMessage(targetTopic, message);
        logSignaling("OFFER", "OUT", "Offer信令已发送到设备");
        logSeparator();
    }

    /**
     * 发送ICE候选
     */
    private void sendICECandidate() {
        if (currentSessionId == null) {
            logError("SIGNAL", "当前没有活跃会话");
            return;
        }

        logStep("发送ICE候选");

        String candidate = "candidate:1 1 UDP 2113667326 192.168.1.100 54400 typ host";
        sendCandidateTs = System.currentTimeMillis();
        // APP发送到设备端订阅的topic
        String targetTopic = String.format("dl/%s/%s/device/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN,
                currentSessionId);

        Map<String, Object> body = new HashMap<>();

        body.put("cmd", "WEB_RTC");
        body.put("ts", System.currentTimeMillis());
        body.put("msgId", System.currentTimeMillis());
        body.put("data", Map.of(
                "sessionId", currentSessionId,
                "type", "candidate",
                "from", APP_ID,
                "to", DEVICE_SN,
                "candidate", candidate));

        String message = JSON.toJSONString(body);

        publishMqttMessage(targetTopic, message);
        logSignaling("ICE", "OUT", "ICE候选已发送到设备");
        logSeparator();
    }

    /**
     * 发送心跳
     */
    private void sendHeartbeat() {
        if (currentSessionId == null) {
            logError("SIGNAL", "当前没有活跃会话");
            return;
        }

        logStep("发送心跳");

        // APP发送到设备端订阅的topic
        String deviceTopic = String.format("dl/%s/%s/device/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN,
                currentSessionId);

        Map<String, Object> body = new HashMap<>();

        body.put("cmd", "WEB_RTC");
        body.put("ts", System.currentTimeMillis());
        body.put("msgId", System.currentTimeMillis());
        body.put("data", Map.of(
                "sessionId", currentSessionId,
                "type", "keepalive",
                "from", APP_ID,
                "to", DEVICE_SN));

        String message = JSON.toJSONString(body);

        publishMqttMessage(deviceTopic, message);
        logSignaling("KEEPALIVE", "OUT", "心跳消息已发送");
        logSeparator();
    }

    /**
     * 结束会话
     */
    private void endSession() {
        if (currentSessionId == null) {
            logError("SIGNAL", "当前没有活跃会话");
            return;
        }

        logStep("结束会话");

        // APP发送到设备端订阅的topic
        String deviceTopic = String.format("dl/%s/%s/device/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN,
                currentSessionId);

        Map<String, Object> body = new HashMap<>();

        body.put("cmd", "WEB_RTC");
        body.put("ts", System.currentTimeMillis());
        body.put("msgId", System.currentTimeMillis());
        body.put("data", Map.of(
                "sessionId", currentSessionId,
                "type", "bye",
                "from", APP_ID,
                "to", DEVICE_SN));

        String message = JSON.toJSONString(body);
        publishMqttMessage(deviceTopic, message);

        currentSessionId = null;
        deviceReadyStatus = "waiting";
        sessionCreated = false;

        if (autoMode) {
            logSuccess("AUTO", "自动模式会话已结束");
            autoMode = false;
        }

        logSignaling("BYE", "OUT", "会话已结束");
        logSeparator();
    }

    /**
     * 发布MQTT消息
     */
    private void publishMqttMessage(String topic, String message) {
        if (!connected || mqttClient == null) {
            logError("MQTT", "未连接到MQTT服务器");
            return;
        }

        try {
            MqttMessage mqttMessage = new MqttMessage(message.getBytes());
            mqttMessage.setQos(1);
            mqttClient.publish(topic, mqttMessage);

            logMqttSend(topic, message);
        } catch (MqttException e) {
            logError("MQTT", "发送消息失败: " + e.getMessage());
        }
    }

    /**
     * 启动自动模式 - 基于P2P配置建立真实连接
     */
    private void startAutoMode() throws IOException, InterruptedException {
        logStep("启动自动模式 - 基于P2P配置建立真实连接");
        logInfo("AUTO", "自动模式将调用接口创建会话，获取P2P配置并建立真实连接");
        autoMode = true;
        sessionCreated = false;

        // 1. 调用接口创建P2P会话（这会触发服务端发送配置给设备）
        createP2PSession();

        if (currentSessionId != null) {
            logInfo("AUTO", "会话创建成功，服务端将自动发送P2P配置给设备");
            logInfo("AUTO", "等待设备收到配置并发送Ready信号...");
            logInfo("AUTO", "设备Ready后将基于真实配置建立P2P连接");
        } else {
            logError("AUTO", "会话创建失败，退出自动模式");
            autoMode = false;
        }
        logSeparator();
    }

    /**
     * 自动执行信令流程
     */
    private void executeAutoSignalingFlow() {
        if (!autoMode || currentSessionId == null) {
            return;
        }

        logStep("自动执行信令流程");

        // 创建一个新线程来执行自动流程，避免阻塞MQTT消息处理
        new Thread(() -> {
            try {
                // 1. 发送Offer
                logInfo("AUTO", "步骤1: 发送Offer信令");
                sendOffer();

                // 2. 发送ICE候选
                logInfo("AUTO", "步骤2: 发送ICE候选");
                sendICECandidate();

                // 3. 发送多个心跳
                for (int i = 1; i <= 3; i++) {
                    logInfo("AUTO", "步骤3." + i + ": 发送心跳 (" + i + "/3)");
                    sendHeartbeat();
                    Thread.sleep(3000); // 每3秒发送一次心跳
                }

                // 4. 等待一段时间模拟通话
                logInfo("AUTO", "步骤4: 模拟通话进行中...");
                Thread.sleep(5000); // 等待5秒

                // 5. 结束会话
                logInfo("AUTO", "步骤5: 结束会话");
                endSession();

                logSuccess("AUTO", "自动信令流程执行完成");
                autoMode = false;

            } catch (InterruptedException e) {
                logError("AUTO", "自动流程被中断: " + e.getMessage());
                autoMode = false;
            } catch (Exception e) {
                logError("AUTO", "自动流程执行失败: " + e.getMessage());
                autoMode = false;
            }
        }).start();
    }

    /**
     * 格式化JSON显示
     */
    private String formatJson(String json) {
        try {
            JsonNode node = objectMapper.readTree(json);
            return objectMapper.writerWithDefaultPrettyPrinter().writeValueAsString(node);
        } catch (Exception e) {
            return json;
        }
    }

    /**
     * 获取P2P配置并建立连接
     */
    private void fetchP2PConfigAndConnect() {
        new Thread(() -> {
            try {
                logInfo("AUTO", "获取P2P配置...");

                // 调用接口获取P2P配置
                HttpRequest request = HttpRequest.newBuilder()
                        .uri(URI.create(P2P_API_BASE_URL + "/session/" + currentSessionId + "/config"))
                        .header("Authorization", "Bearer mock_token_" + APP_ID)
                        .header("X-User-Id", APP_ID)
                        .header("source", "IOS")
                        .header("token", token)
                        .GET()
                        .timeout(Duration.ofSeconds(10))
                        .build();

                HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());

                logHttpRequest("GET", P2P_API_BASE_URL + "/session/" + currentSessionId + "/config", response.statusCode());
                logInfo("HTTP", "响应内容: " + response.body());

                if (response.statusCode() == 200) {
                    // 解析P2P配置
                    JsonNode responseJson = objectMapper.readTree(response.body());
                    if (responseJson.has("data")) {
                        JsonNode configData = responseJson.get("data");

                        // 解析STUN/TURN服务器
                        if (configData.has("stunServers")) {
                            JsonNode stunArray = configData.get("stunServers");
                            stunServers = new String[stunArray.size()];
                            for (int i = 0; i < stunArray.size(); i++) {
                                stunServers[i] = stunArray.get(i).asText();
                            }
                            logInfo("CONFIG", "STUN服务器: " + String.join(", ", stunServers));
                        }

                        if (configData.has("turnServers")) {
                            JsonNode turnArray = configData.get("turnServers");
                            turnServers = new String[turnArray.size()];
                            for (int i = 0; i < turnArray.size(); i++) {
                                turnServers[i] = turnArray.get(i).asText();
                            }
                            logInfo("CONFIG", "TURN服务器: " + String.join(", ", turnServers));
                            logInfo("CONFIG", "TURN认证: APP/APP (与MQTT保持一致)");
                        }

                        // 解析其他配置
                        if (configData.has("rtcConfiguration")) {
                            rtcConfiguration = configData.get("rtcConfiguration").asText();
                        }
                        if (configData.has("extraConfig")) {
                            extraConfig = configData.get("extraConfig").asText();
                        }

                        logSuccess("CONFIG", "P2P配置获取成功");

                        // 开始真实P2P连接流程
                        executeRealP2PFlow();

                    } else {
                        logError("CONFIG", "响应格式异常，未找到配置数据");
                        // 如果接口没有返回配置，使用提供的示例配置
                        useExampleConfig();
                    }
                } else {
                    logError("CONFIG", "获取P2P配置失败，状态码: " + response.statusCode());
                    // 如果接口调用失败，使用提供的示例配置
                    useExampleConfig();
                }

            } catch (Exception e) {
                logError("CONFIG", "获取P2P配置异常: " + e.getMessage());
                // 如果出现异常，使用提供的示例配置
                useExampleConfig();
            }
        }).start();
    }

    /**
     * 使用示例配置（当接口调用失败时的备用方案）
     */
    private void useExampleConfig() {
        logInfo("CONFIG", "使用示例P2P配置作为备用方案");

        stunServers = new String[]{"stun:223.254.128.13:3478"};
        turnServers = new String[]{"turn:223.254.128.13:3479"};
        rtcConfiguration = "{\"iceServers\": [{\"urls\": \"stun:223.254.128.13:3478\"},{\"urls\": \"turn:223.254.128.13:3479\"}],\"iceCandidatePoolSize\": 10,\"bundlePolicy\": \"max-bundle\",\"rtcpMuxPolicy\": \"require\"}";
        extraConfig = "{\"video\": true,\"audio\": true,\"timeout\": 30000,\"retryAttempts\": 3,\"keepAliveInterval\": 10000}";

        logInfo("CONFIG", "STUN服务器: " + String.join(", ", stunServers));
        logInfo("CONFIG", "TURN服务器: " + String.join(", ", turnServers));
        logInfo("CONFIG", "TURN认证: APP/APP (与MQTT保持一致)");
        logSuccess("CONFIG", "示例P2P配置设置完成");

        // 开始真实P2P连接流程
        executeRealP2PFlow();
    }

    /**
     * 执行真实P2P连接流程
     */
    private void executeRealP2PFlow() {
        new Thread(() -> {
            try {
                logInfo("P2P", "开始真实P2P连接流程");

                // 1. 发送Offer（使用真实配置）
                logInfo("P2P", "1. 发送Offer信令（基于WebRTC配置）");
                sendOffer();
                Thread.sleep(1000);

                // 2. 发送ICE候选（基于STUN/TURN配置）
                logInfo("P2P", "2. 发送ICE候选（使用配置的ICE服务器）");
                sendICECandidate();
                Thread.sleep(2000); // 等待ICE连接建立

                // 3. 模拟P2P连接建立成功
                logSuccess("P2P", "P2P连接建立成功！");
                logInfo("P2P", "连接状态: CONNECTED");
                if (stunServers != null && turnServers != null) {
                    logInfo("P2P", "使用的ICE服务器: " + String.join(", ", stunServers) + ", " + String.join(", ", turnServers));
                }

                // 4. 启动双通道保活机制
                logInfo("P2P", "4. 启动双通道保活机制（P2P + MQTT）");
                startDualChannelKeepAlive();

            } catch (Exception e) {
                logError("P2P", "真实P2P连接流程执行失败: " + e.getMessage());
            }
        }).start();
    }

    /**
     * 启动双通道保活机制
     */
    private void startDualChannelKeepAlive() {
        if (extraConfig == null) {
            logWarning("KEEPALIVE", "缺少配置信息，使用默认保活间隔");
            startDualChannelKeepAliveWithInterval(10000); // 默认10秒
            return;
        }

        try {
            JSONObject extra = JSON.parseObject(extraConfig);
            int keepAliveInterval = extra.getInteger("keepAliveInterval");
            logInfo("KEEPALIVE", "保活间隔: " + keepAliveInterval + "ms");

            startDualChannelKeepAliveWithInterval(keepAliveInterval);

        } catch (Exception e) {
            logError("KEEPALIVE", "解析保活配置失败: " + e.getMessage());
            startDualChannelKeepAliveWithInterval(10000); // 使用默认间隔
        }
    }

    /**
     * 使用指定间隔启动双通道保活
     */
    private void startDualChannelKeepAliveWithInterval(int intervalMs) {
        logStep("启动双通道保活机制");
        logInfo("KEEPALIVE", "P2P通道 + MQTT通道同时发送保活消息");
        logInfo("KEEPALIVE", "保活间隔: " + intervalMs + "ms");

        new Thread(() -> {
            int keepAliveCount = 0;

            while (currentSessionId != null && keepAliveCount < 10) { // 发送10次保活消息
                try {
                    keepAliveCount++;
                    long timestamp = System.currentTimeMillis();

                    logInfo("KEEPALIVE", "=== 第" + keepAliveCount + "次双通道保活 ===");

                    // 1. 通过P2P通道发送保活
                    sendP2PKeepAlive(timestamp, keepAliveCount);

                    // 2. 通过MQTT通道发送保活
                    sendMQTTKeepAlive(timestamp, keepAliveCount);

                    logSuccess("KEEPALIVE", "双通道保活发送完成 (" + keepAliveCount + "/10)");

                    // 等待下一次保活
                    Thread.sleep(intervalMs);

                } catch (InterruptedException e) {
                    logWarning("KEEPALIVE", "保活线程被中断");
                    Thread.currentThread().interrupt();
                    break;
                } catch (Exception e) {
                    logError("KEEPALIVE", "发送保活消息失败: " + e.getMessage());
                }
            }

            logInfo("KEEPALIVE", "双通道保活机制结束");

        }).start();
    }

    /**
     * 通过P2P通道发送保活消息
     */
    private void sendP2PKeepAlive(long timestamp, int sequence) {
        try {
            // 构建P2P保活消息
            JSONObject p2pKeepAlive = new JSONObject();
            p2pKeepAlive.put("cmd", "P2P_KEEPALIVE");
            p2pKeepAlive.put("ts", timestamp);
            p2pKeepAlive.put("msgId", "p2p_keepalive_" + timestamp);
            p2pKeepAlive.put("sessionId", currentSessionId);
            p2pKeepAlive.put("from", APP_ID);
            p2pKeepAlive.put("to", DEVICE_SN);
            p2pKeepAlive.put("sequence", sequence);
            p2pKeepAlive.put("channel", "P2P");
            p2pKeepAlive.put("type", "keepalive");

            // 通过P2P数据通道发送（这里模拟P2P通道发送）
            String p2pTopic = String.format("dl/%s/%s/p2p/%s/data", PRODUCT_KEY, DEVICE_SN, currentSessionId);
            publishMqttMessage(p2pTopic, p2pKeepAlive.toJSONString());

            logInfo("P2P-KEEPALIVE", "P2P通道保活已发送 (序号: " + sequence + ")");

        } catch (Exception e) {
            logError("P2P-KEEPALIVE", "P2P通道保活发送失败: " + e.getMessage());
        }
    }

    /**
     * 通过MQTT通道发送保活消息
     */
    private void sendMQTTKeepAlive(long timestamp, int sequence) {
        try {
            // 构建MQTT保活消息
            JSONObject mqttKeepAlive = new JSONObject();
            mqttKeepAlive.put("cmd", "WEB_RTC");
            mqttKeepAlive.put("ts", timestamp);
            mqttKeepAlive.put("msgId", "mqtt_keepalive_" + timestamp);

            JSONObject data = new JSONObject();
            data.put("sessionId", currentSessionId);
            data.put("type", "keepalive");
            data.put("from", APP_ID);
            data.put("to", DEVICE_SN);
            data.put("sequence", sequence);
            data.put("channel", "MQTT");
            data.put("status", "connected");

            mqttKeepAlive.put("data", data);

            // 通过MQTT信令通道发送
            String mqttTopic = String.format("dl/%s/%s/device/%s/p2p/signal/sub", PRODUCT_KEY, DEVICE_SN, currentSessionId);
            publishMqttMessage(mqttTopic, mqttKeepAlive.toJSONString());

            logInfo("MQTT-KEEPALIVE", "MQTT通道保活已发送 (序号: " + sequence + ")");

        } catch (Exception e) {
            logError("MQTT-KEEPALIVE", "MQTT通道保活发送失败: " + e.getMessage());
        }
    }
}

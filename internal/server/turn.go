package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	pionlogger "github.com/pion/logging"
	"github.com/pion/stun"
	"github.com/pion/turn/v4"
	"github.com/sirupsen/logrus"

	"github.com/ga666666-new/pion-stun-server/internal/auth"
	"github.com/ga666666-new/pion-stun-server/internal/config"
	"github.com/ga666666-new/pion-stun-server/pkg/models"
)

// turnLeveledLogger is a simple adapter to use logrus with pion's logger interface
type turnLeveledLogger struct {
	entry                      *logrus.Entry
	sessionTracker            *clientSessionTracker
	terminateOnPermissionError bool
}

func (l *turnLeveledLogger) Trace(msg string) {
	l.entry.Trace(msg)
}
func (l *turnLeveledLogger) Tracef(format string, args ...interface{}) {
	l.entry.Tracef(format, args...)
}
func (l *turnLeveledLogger) Debug(msg string) {
	l.entry.Debug(msg)
}
func (l *turnLeveledLogger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}
func (l *turnLeveledLogger) Info(msg string) {
	l.entry.Info(msg)
}
func (l *turnLeveledLogger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}
func (l *turnLeveledLogger) Warn(msg string) {
	l.entry.Warn(msg)
}
func (l *turnLeveledLogger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}
func (l *turnLeveledLogger) Error(msg string) {
	l.entry.Error(msg)
	l.checkPermissionError(msg)
}
func (l *turnLeveledLogger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
	msg := fmt.Sprintf(format, args...)
	l.checkPermissionError(msg)
}

// checkPermissionError 检查是否为权限错误，如果是且配置了终止选项，则终止程序
func (l *turnLeveledLogger) checkPermissionError(msg string) {
	if !l.terminateOnPermissionError {
		return
	}
	
	// 检查是否包含权限错误信息
	if strings.Contains(msg, "No Permission or Channel exists") {
		l.entry.Error("=== 检测到权限错误，程序即将终止以便调试 ===")
		
		// 输出详细的调试信息
		l.dumpDebugInfo(msg)
		
		// 终止程序
		l.entry.Fatal("=== 程序因权限错误终止 ===")
		os.Exit(1)
	}
}

// dumpDebugInfo 输出详细的调试信息
func (l *turnLeveledLogger) dumpDebugInfo(errorMsg string) {
	l.entry.Error("=== 权限错误详细信息 ===")
	l.entry.WithField("error_message", errorMsg).Error("错误消息")
	
	// 输出所有活跃会话的详细信息
	if l.sessionTracker != nil {
		l.sessionTracker.mutex.RLock()
		defer l.sessionTracker.mutex.RUnlock()
		
		l.entry.WithField("total_sessions", len(l.sessionTracker.sessions)).Error("当前活跃会话数量")
		
		for clientAddr, session := range l.sessionTracker.sessions {
			l.entry.WithFields(logrus.Fields{
				"client_addr":      clientAddr,
				"username":         session.Username,
				"session_duration": time.Since(session.StartTime).String(),
				"total_steps":      len(session.Steps),
				"allocations":      len(session.Allocations),
				"permissions":      len(session.Permissions),
				"channels":         len(session.Channels),
				"last_activity":    time.Since(session.LastActivity).String(),
			}).Error("会话详细信息")
			
			// 输出所有步骤
			l.entry.WithField("client_addr", clientAddr).Error("=== 会话步骤历史 ===")
			for i, step := range session.Steps {
				l.entry.WithFields(logrus.Fields{
					"client_addr": clientAddr,
					"step_number": i + 1,
					"step_detail": step,
				}).Error("步骤详情")
			}
			
			// 输出所有分配
			l.entry.WithField("client_addr", clientAddr).Error("=== 分配信息 ===")
			for relayAddr, allocation := range session.Allocations {
				l.entry.WithFields(logrus.Fields{
					"client_addr": clientAddr,
					"relay_addr":  relayAddr,
					"created_at":  allocation.CreatedAt.Format("15:04:05.000"),
					"age":         time.Since(allocation.CreatedAt).String(),
				}).Error("分配详情")
			}
			
			// 输出所有权限
			l.entry.WithField("client_addr", clientAddr).Error("=== 权限信息 ===")
			for peerAddr, permission := range session.Permissions {
				l.entry.WithFields(logrus.Fields{
					"client_addr": clientAddr,
					"peer_addr":   peerAddr,
					"created_at":  permission.CreatedAt.Format("15:04:05.000"),
					"age":         time.Since(permission.CreatedAt).String(),
				}).Error("权限详情")
			}
			
			// 输出所有通道
			l.entry.WithField("client_addr", clientAddr).Error("=== 通道信息 ===")
			for peerAddr, channel := range session.Channels {
				l.entry.WithFields(logrus.Fields{
					"client_addr":     clientAddr,
					"peer_addr":       peerAddr,
					"channel_number":  channel.ChannelNumber,
					"created_at":      channel.CreatedAt.Format("15:04:05.000"),
					"age":             time.Since(channel.CreatedAt).String(),
				}).Error("通道详情")
			}
		}
	}
	
	l.entry.Error("=== 调试信息输出完成 ===")
}

// turnLoggerFactory creates new loggers for the pion/turn library
type turnLoggerFactory struct {
	logger                     *logrus.Logger
	sessionTracker            *clientSessionTracker
	terminateOnPermissionError bool
}

func (f *turnLoggerFactory) NewLogger(scope string) pionlogger.LeveledLogger {
	return &turnLeveledLogger{
		entry:                      f.logger.WithField("scope", scope),
		sessionTracker:            f.sessionTracker,
		terminateOnPermissionError: f.terminateOnPermissionError,
	}
}

// 客户端会话追踪器
type clientSessionTracker struct {
	logger        *logrus.Logger
	sessions      map[string]*clientSession
	mutex         sync.RWMutex
}

type clientSession struct {
	ClientAddr    string
	Username      string
	StartTime     time.Time
	LastActivity  time.Time
	Steps         []string
	Allocations   map[string]*allocationInfo
	Permissions   map[string]*permissionInfo
	Channels      map[string]*channelInfo
}

type allocationInfo struct {
	RelayAddr     string
	CreatedAt     time.Time
	LastActivity  time.Time
}

type permissionInfo struct {
	PeerAddr      string
	CreatedAt     time.Time
	LastActivity  time.Time
}

type channelInfo struct {
	PeerAddr      string
	ChannelNumber uint16
	CreatedAt     time.Time
	LastActivity  time.Time
}

func newClientSessionTracker(logger *logrus.Logger) *clientSessionTracker {
	return &clientSessionTracker{
		logger:   logger,
		sessions: make(map[string]*clientSession),
	}
}

func (t *clientSessionTracker) trackStep(clientAddr, username, step string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	session, exists := t.sessions[clientAddr]
	if !exists {
		session = &clientSession{
			ClientAddr:   clientAddr,
			Username:     username,
			StartTime:    time.Now(),
			LastActivity: time.Now(),
			Steps:        []string{},
			Allocations:  make(map[string]*allocationInfo),
			Permissions:  make(map[string]*permissionInfo),
			Channels:     make(map[string]*channelInfo),
		}
		t.sessions[clientAddr] = session
		t.logger.WithFields(logrus.Fields{
			"client_addr": clientAddr,
			"username":    username,
			"action":      "NEW_SESSION",
		}).Info("=== 新客户端会话开始 ===")
	}
	
	session.LastActivity = time.Now()
	session.Steps = append(session.Steps, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05.000"), step))
	
	t.logger.WithFields(logrus.Fields{
		"client_addr":    clientAddr,
		"username":       username,
		"step":           step,
		"total_steps":    len(session.Steps),
		"session_duration": time.Since(session.StartTime).String(),
	}).Info("=== 客户端步骤追踪 ===")
}

func (t *clientSessionTracker) trackAllocation(clientAddr, relayAddr string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if session, exists := t.sessions[clientAddr]; exists {
		session.Allocations[relayAddr] = &allocationInfo{
			RelayAddr:    relayAddr,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
		t.logger.WithFields(logrus.Fields{
			"client_addr": clientAddr,
			"relay_addr":  relayAddr,
			"action":      "ALLOCATION_CREATED",
		}).Info("=== 分配创建 ===")
	}
}

func (t *clientSessionTracker) trackPermission(clientAddr, peerAddr string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if session, exists := t.sessions[clientAddr]; exists {
		session.Permissions[peerAddr] = &permissionInfo{
			PeerAddr:     peerAddr,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
		t.logger.WithFields(logrus.Fields{
			"client_addr": clientAddr,
			"peer_addr":   peerAddr,
			"action":      "PERMISSION_CREATED",
		}).Info("=== 权限创建 ===")
	}
}

func (t *clientSessionTracker) getSessionInfo(clientAddr string) *clientSession {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if session, exists := t.sessions[clientAddr]; exists {
		return session
	}
	return nil
}

func (t *clientSessionTracker) logSessionSummary(clientAddr string) {
	session := t.getSessionInfo(clientAddr)
	if session == nil {
		return
	}
	
	t.logger.WithFields(logrus.Fields{
		"client_addr":      clientAddr,
		"username":         session.Username,
		"session_duration": time.Since(session.StartTime).String(),
		"total_steps":      len(session.Steps),
		"allocations":      len(session.Allocations),
		"permissions":      len(session.Permissions),
		"channels":         len(session.Channels),
	}).Info("=== 会话摘要 ===")
	
	// 打印详细步骤
	for i, step := range session.Steps {
		t.logger.WithFields(logrus.Fields{
			"client_addr": clientAddr,
			"step_number": i + 1,
			"step_detail": step,
		}).Info("步骤详情")
	}
}



// 添加一个全局权限管理器，用于自动创建权限
type autoPermissionManager struct {
	logger *logrus.Logger
	permissions map[string]bool
	mutex sync.RWMutex
}

func newAutoPermissionManager(logger *logrus.Logger) *autoPermissionManager {
	return &autoPermissionManager{
		logger: logger,
		permissions: make(map[string]bool),
	}
}

func (m *autoPermissionManager) ensurePermission(allocationAddr, peerAddr string) {
	key := fmt.Sprintf("%s->%s", allocationAddr, peerAddr)
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.permissions[key] {
		m.permissions[key] = true
		m.logger.WithFields(logrus.Fields{
			"allocation": allocationAddr,
			"peer":       peerAddr,
			"action":     "AutoCreatePermission",
		}).Warn("=== AUTO-CREATED PERMISSION (NON-STANDARD) ===")
	}
}

// TURN请求类型追踪
type turnRequestTracker struct {
	logger  *logrus.Logger
	tracker *clientSessionTracker
}

func newTurnRequestTracker(logger *logrus.Logger, tracker *clientSessionTracker) *turnRequestTracker {
	return &turnRequestTracker{
		logger:  logger,
		tracker: tracker,
	}
}

// TURNServer represents a TURN server
type TURNServer struct {
	config        *config.TURNConfig
	auth          *auth.MongoAuthenticator
	server        *turn.Server
	logger        *logrus.Logger
	sessions      map[string]*models.SessionInfo
	sessionsMutex sync.RWMutex
	stopChan      chan struct{}
	// 添加自动权限管理器
	permManager   *autoPermissionManager
	// 添加客户端会话追踪器
	sessionTracker *clientSessionTracker
}

// NewTURNServer creates a new TURN server
func NewTURNServer(cfg *config.TURNConfig, authenticator *auth.MongoAuthenticator, logger *logrus.Logger) *TURNServer {
	return &TURNServer{
		config:         cfg,
		auth:           authenticator,
		logger:         logger,
		sessions:       make(map[string]*models.SessionInfo),
		stopChan:       make(chan struct{}),
		permManager:    newAutoPermissionManager(logger),
		sessionTracker: newClientSessionTracker(logger),
	}
}

// 创建自定义的TURN服务器配置，增加请求拦截
func (t *TURNServer) createEnhancedServerConfig(relayAddressGenerator turn.RelayAddressGenerator, udpListener net.PacketConn, tcpListener net.Listener) turn.ServerConfig {
	loggerFactory := &turnLoggerFactory{
		logger:                     t.logger,
		sessionTracker:            t.sessionTracker,
		terminateOnPermissionError: t.config.TerminateOnPermissionError,
	}
	
	// 创建自定义的权限处理器，增加更多调试信息
	permissionHandler := func(clientAddr net.Addr, peerIP net.IP) bool {
		clientAddrStr := clientAddr.String()
		peerIPStr := peerIP.String()
		
		t.logger.WithFields(logrus.Fields{
			"client_addr": clientAddrStr,
			"peer_ip":     peerIPStr,
			"action":      "PERMISSION_CHECK",
			"timestamp":   time.Now().Format("15:04:05.000"),
		}).Info("=== 权限检查被调用 ===")
		
		// 追踪权限检查步骤
		t.sessionTracker.trackStep(clientAddrStr, "", "PERMISSION_CHECK: "+peerIPStr)
		
		// 检查会话信息
		session := t.sessionTracker.getSessionInfo(clientAddrStr)
		if session != nil {
			t.logger.WithFields(logrus.Fields{
				"client_addr":    clientAddrStr,
				"username":       session.Username,
				"existing_perms": len(session.Permissions),
				"allocations":    len(session.Allocations),
				"session_age":    time.Since(session.StartTime).String(),
			}).Info("=== 现有会话信息 ===")
			
			// 检查是否已有此peer的权限
			if _, exists := session.Permissions[peerIPStr]; exists {
				t.logger.WithFields(logrus.Fields{
					"client_addr": clientAddrStr,
					"peer_ip":     peerIPStr,
					"result":      "ALLOWED",
					"reason":      "existing_permission",
				}).Info("=== 权限已存在，允许 ===")
				return true
			}
		}
		
		// 自动授予权限并记录
		t.sessionTracker.trackPermission(clientAddrStr, peerIPStr)
		
		t.logger.WithFields(logrus.Fields{
			"client_addr": clientAddrStr,
			"peer_ip":     peerIPStr,
			"result":      "ALLOWED",
			"reason":      "auto_grant_debug_mode",
		}).Info("=== 权限自动授予（调试模式）===")
		
		return true
	}
	
	return turn.ServerConfig{
		Realm:         t.config.Realm,
		AuthHandler:   t.enhancedAuthHandler,
		LoggerFactory: loggerFactory,
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn:            udpListener,
				RelayAddressGenerator: relayAddressGenerator,
				PermissionHandler:     permissionHandler,
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener:              tcpListener,
				RelayAddressGenerator: relayAddressGenerator,
				PermissionHandler:     permissionHandler,
			},
		},
	}
}

// 增强的认证处理器，包含更多调试信息
func (t *TURNServer) enhancedAuthHandler(username, realm string, srcAddr net.Addr) (key []byte, ok bool) {
	clientAddrStr := srcAddr.String()
	
	logger := t.logger.WithFields(logrus.Fields{
		"username":   username,
		"realm":      realm,
		"client":     clientAddrStr,
		"timestamp":  time.Now().Format("15:04:05.000"),
	})
	
	logger.Info("=== TURN 认证请求接收 ===")
	t.sessionTracker.trackStep(clientAddrStr, username, "AUTHENTICATION_REQUEST")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	storedKey, user, err := t.auth.GetTURNAuthKey(ctx, username)
	if err != nil {
		logger.WithError(err).Error("=== 认证失败 ===")
		t.sessionTracker.trackStep(clientAddrStr, username, "AUTHENTICATION_FAILED: "+err.Error())
		return nil, false
	}
	
	// The stored key is hex-encoded, decode it for pion/turn
	decodedKey, err := hex.DecodeString(storedKey)
	if err != nil {
		logger.WithError(err).Error("=== 密钥解码失败 ===")
		t.sessionTracker.trackStep(clientAddrStr, username, "KEY_DECODE_FAILED: "+err.Error())
		return nil, false
	}
	
	// Check user quota
	if user.Quota != nil && user.Quota.CurrentSessions >= user.Quota.MaxSessions {
		logger.WithFields(logrus.Fields{
			"current_sessions": user.Quota.CurrentSessions,
			"max_sessions":     user.Quota.MaxSessions,
		}).Warn("=== 用户配额超限 ===")
		t.sessionTracker.trackStep(clientAddrStr, username, "QUOTA_EXCEEDED")
		return nil, false
	}
	
	logger.WithFields(logrus.Fields{
		"username":     user.Username,
		"key_length":   len(decodedKey),
		"quota_current": func() int { if user.Quota != nil { return user.Quota.CurrentSessions } else { return 0 } }(),
		"quota_max":    func() int { if user.Quota != nil { return user.Quota.MaxSessions } else { return -1 } }(),
	}).Info("=== TURN 认证成功 ===")
	
	t.sessionTracker.trackStep(clientAddrStr, username, "AUTHENTICATION_SUCCESS")
	
	// Create session info
	sessionID := fmt.Sprintf("%s-%d", srcAddr.String(), time.Now().Unix())
	session := &models.SessionInfo{
		ID:         sessionID,
		Username:   user.Username,
		ClientAddr: srcAddr.String(),
		StartTime:  time.Now(),
		LastActive: time.Now(),
	}
	
	t.sessionsMutex.Lock()
	t.sessions[sessionID] = session
	t.sessionsMutex.Unlock()
	
	logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"username":   user.Username,
		"client":     clientAddrStr,
	}).Info("=== TURN 会话创建 ===")
	t.sessionTracker.trackStep(clientAddrStr, username, "SESSION_CREATED: "+sessionID)
	
	return decodedKey, true
}

// sessionCleanup periodically cleans up inactive sessions
func (t *TURNServer) sessionCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.cleanupInactiveSessions()
		}
	}
}

// cleanupInactiveSessions removes inactive sessions
func (t *TURNServer) cleanupInactiveSessions() {
	t.sessionsMutex.Lock()
	defer t.sessionsMutex.Unlock()
	
	now := time.Now()
	for sessionID, session := range t.sessions {
		if now.Sub(session.LastActive) > 5*time.Minute {
			delete(t.sessions, sessionID)
			t.logger.WithField("session_id", sessionID).Info("=== 清理非活跃会话 ===")
			
			// 打印会话摘要
			t.sessionTracker.logSessionSummary(session.ClientAddr)
		}
	}
}

// 定期打印会话摘要
func (t *TURNServer) periodicSessionSummary() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.printActiveSessions()
		}
	}
}

// 打印活跃会话信息
func (t *TURNServer) printActiveSessions() {
	t.sessionTracker.mutex.RLock()
	defer t.sessionTracker.mutex.RUnlock()
	
	if len(t.sessionTracker.sessions) == 0 {
		t.logger.Info("=== 当前无活跃TURN会话 ===")
		return
	}
	
	t.logger.WithField("active_sessions", len(t.sessionTracker.sessions)).Info("=== 活跃会话摘要 ===")
	
	for clientAddr, session := range t.sessionTracker.sessions {
		t.logger.WithFields(logrus.Fields{
			"client_addr":      clientAddr,
			"username":         session.Username,
			"session_duration": time.Since(session.StartTime).String(),
			"total_steps":      len(session.Steps),
			"allocations":      len(session.Allocations),
			"permissions":      len(session.Permissions),
			"channels":         len(session.Channels),
			"last_activity":    time.Since(session.LastActivity).String(),
		}).Info("会话状态")
		
		// 打印最近的步骤
		if len(session.Steps) > 0 {
			recentSteps := session.Steps
			if len(recentSteps) > 3 {
				recentSteps = recentSteps[len(recentSteps)-3:]
			}
			for _, step := range recentSteps {
				t.logger.WithFields(logrus.Fields{
					"client_addr": clientAddr,
					"recent_step": step,
				}).Info("最近步骤")
			}
		}
	}
}

// GetSessions returns current active sessions
func (t *TURNServer) GetSessions() []*models.SessionInfo {
	t.sessionsMutex.RLock()
	defer t.sessionsMutex.RUnlock()
	
	sessions := make([]*models.SessionInfo, 0, len(t.sessions))
	for _, session := range t.sessions {
		sessions = append(sessions, session)
	}
	
	return sessions
}

// GetStats returns TURN server statistics
func (t *TURNServer) GetStats() map[string]interface{} {
	t.sessionsMutex.RLock()
	sessionCount := len(t.sessions)
	t.sessionsMutex.RUnlock()
	
	return map[string]interface{}{
		"status":          "running",
		"address":         fmt.Sprintf("%s:%d", t.config.Address, t.config.Port),
		"realm":           t.config.Realm,
		"active_sessions": sessionCount,
	}
}

func discoverPublicIP(logger *logrus.Logger) (net.IP, error) {
	// We use a public STUN server to discover our public IP address.
	// Google's STUN server is a good choice.
	c, err := net.Dial("udp4", "stun.l.google.com:19302")
	if err != nil {
		return nil, fmt.Errorf("failed to dial STUN server: %w", err)
	}
	defer c.Close()

	// Create a STUN client
	client, err := stun.NewClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create STUN client: %w", err)
	}
	defer client.Close()

	// Build a binding request
	message, err := stun.Build(stun.BindingRequest, stun.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to build STUN request: %w", err)
	}

	var xorMappedAddr stun.XORMappedAddress
	var discoveryErr error
	var wg sync.WaitGroup
	wg.Add(1)

	// Callback function to handle the STUN response
	callback := func(res stun.Event) {
		defer wg.Done()
		if res.Error != nil {
			discoveryErr = res.Error
			return
		}
		// Get the XOR-MAPPED-ADDRESS attribute from the response
		if err := xorMappedAddr.GetFrom(res.Message); err != nil {
			discoveryErr = fmt.Errorf("failed to get XOR-MAPPED-ADDRESS: %w", err)
			return
		}
	}

	// Send the request and wait for the response
	if err := client.Do(message, callback); err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %w", err)
	}

	wg.Wait()

	if discoveryErr != nil {
		return nil, discoveryErr
	}

	return xorMappedAddr.IP, nil
}

// Start starts the TURN server
func (t *TURNServer) Start() error {
	addr := fmt.Sprintf("%s:%d", t.config.Address, t.config.Port)

	var relayAddress net.IP
	if t.config.PublicIP != "" {
		relayAddress = net.ParseIP(t.config.PublicIP)
		if relayAddress == nil {
			return fmt.Errorf("invalid public_ip address: %s", t.config.PublicIP)
		}
		t.logger.WithField("ip", relayAddress.String()).Info("Using configured public IP")
	} else {
		t.logger.Info("Public IP not configured, attempting to discover using STUN")
		discoveredIP, err := discoverPublicIP(t.logger)
		if err != nil {
			t.logger.WithError(err).Warn("Failed to discover public IP, falling back to 127.0.0.1. TURN will likely not work externally.")
			relayAddress = net.ParseIP("127.0.0.1")
		} else {
			relayAddress = discoveredIP
			t.logger.WithField("ip", relayAddress.String()).Info("Discovered public IP")
		}
	}

	// 创建 relay address generator
	relayAddressGenerator := &turn.RelayAddressGeneratorStatic{
		RelayAddress: relayAddress,
		Address:      "0.0.0.0",
	}

	// UDP 监听
	udpListener, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP %s: %w", addr, err)
	}

	// TCP 监听
	tcpListener, err := net.Listen("tcp4", addr)
	if err != nil {
		udpListener.Close()
		return fmt.Errorf("failed to listen on TCP %s: %w", addr, err)
	}

	// 使用增强的服务器配置
	serverConfig := t.createEnhancedServerConfig(relayAddressGenerator, udpListener, tcpListener)

	t.logger.WithFields(logrus.Fields{
		"realm":           t.config.Realm,
		"relay_address":   relayAddress.String(),
		"udp_listener":    addr,
		"tcp_listener":    addr,
		"permission_mode": "enhanced_debug_tracking",
	}).Info("TURN server configuration")

	// Create TURN server
	server, err := turn.NewServer(serverConfig)
	if err != nil {
		udpListener.Close()
		tcpListener.Close()
		return fmt.Errorf("failed to create TURN server: %w", err)
	}

	t.server = server
	t.logger.WithField("address", addr).Info("TURN server started")
	
	// 添加启动后的调试信息
	t.logger.Info("=== TURN Server Enhanced Debug Info ===")
	t.logger.Info("1. Server is listening for TURN requests")
	t.logger.Info("2. Enhanced PermissionHandler with detailed session tracking")
	t.logger.Info("3. Expected TURN flow:")
	t.logger.Info("   a) AllocateRequest -> Authentication -> Allocation created")
	t.logger.Info("   b) CreatePermission -> Permission granted")
	t.logger.Info("   c) SendIndication -> Data relay")
	t.logger.Info("4. All client steps, permissions, and data flows will be tracked")
	t.logger.Info("5. 'No Permission' errors indicate missing CreatePermission step")
	t.logger.Info("=== End Enhanced Debug Info ===")
	
	go t.sessionCleanup()
	go t.periodicSessionSummary()
	return nil
}

// Stop stops the TURN server
func (t *TURNServer) Stop() error {
	close(t.stopChan)
	
	if t.server != nil {
		if err := t.server.Close(); err != nil {
			return fmt.Errorf("failed to close TURN server: %w", err)
		}
	}
	
	t.logger.Info("TURN server stopped")
	return nil
}
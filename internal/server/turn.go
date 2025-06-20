package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/turn/v2"
	"github.com/sirupsen/logrus"

	"github.com/ga666666-new/pion-stun-server/internal/auth"
	"github.com/ga666666-new/pion-stun-server/internal/config"
	"github.com/ga666666-new/pion-stun-server/pkg/models"
)

// TURNServer represents a TURN server
type TURNServer struct {
	config        *config.TURNConfig
	auth          *auth.MongoAuthenticator
	server        *turn.Server
	logger        *logrus.Logger
	sessions      map[string]*models.SessionInfo
	sessionsMutex sync.RWMutex
	stopChan      chan struct{}
}

// NewTURNServer creates a new TURN server
func NewTURNServer(cfg *config.TURNConfig, authenticator *auth.MongoAuthenticator, logger *logrus.Logger) *TURNServer {
	return &TURNServer{
		config:   cfg,
		auth:     authenticator,
		logger:   logger,
		sessions: make(map[string]*models.SessionInfo),
		stopChan: make(chan struct{}),
	}
}

// Start starts the TURN server
func (t *TURNServer) Start() error {
	addr := fmt.Sprintf("%s:%d", t.config.Address, t.config.Port)
	
	// Create relay address generator
	relayAddressGenerator := &turn.RelayAddressGeneratorStatic{
		RelayAddress: net.ParseIP("127.0.0.1"), // Use localhost for testing
		Address:      "0.0.0.0",
	}
	
	if t.config.PublicIP != "" {
		if publicIP := net.ParseIP(t.config.PublicIP); publicIP != nil {
			relayAddressGenerator.RelayAddress = publicIP
		}
	}
	
	// Listen on UDP
	udpListener, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP %s: %w", addr, err)
	}
	
	// Listen on TCP
	tcpListener, err := net.Listen("tcp4", addr)
	if err != nil {
		udpListener.Close()
		return fmt.Errorf("failed to listen on TCP %s: %w", addr, err)
	}
	
	// Create TURN server configuration
	serverConfig := turn.ServerConfig{
		Realm:       t.config.Realm,
		AuthHandler: t.handleAuth,
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn:            udpListener,
				RelayAddressGenerator: relayAddressGenerator,
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener:              tcpListener,
				RelayAddressGenerator: relayAddressGenerator,
			},
		},
	}
	
	// Create TURN server
	server, err := turn.NewServer(serverConfig)
	if err != nil {
		udpListener.Close()
		tcpListener.Close()
		return fmt.Errorf("failed to create TURN server: %w", err)
	}
	
	t.server = server
	
	t.logger.WithField("address", addr).Info("TURN server started")
	
	// Start session cleanup routine
	go t.sessionCleanup()
	
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

// handleAuth handles TURN authentication
func (t *TURNServer) handleAuth(username, realm string, srcAddr net.Addr) (key []byte, ok bool) {
	logger := t.logger.WithFields(logrus.Fields{
		"username": username,
		"realm":    realm,
		"client":   srcAddr.String(),
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	storedKey, user, err := t.auth.GetTURNAuthKey(ctx, username)
	if err != nil {
		logger.WithError(err).Debug("Authentication failed")
		return nil, false
	}
	
	// The stored key is hex-encoded, decode it for pion/turn
	decodedKey, err := hex.DecodeString(storedKey)
	if err != nil {
		logger.WithError(err).Error("Failed to decode stored TURN key")
		return nil, false
	}
	
	// Check user quota
	if user.Quota != nil && user.Quota.CurrentSessions >= user.Quota.MaxSessions {
		logger.Debug("User quota exceeded")
		return nil, false
	}
	
	logger.Debug("Authentication successful")
	
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
			t.logger.WithField("session_id", sessionID).Debug("Cleaned up inactive session")
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
package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/stun"
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

	// Create relay address generator
	relayAddressGenerator := &turn.RelayAddressGeneratorStatic{
		RelayAddress: relayAddress,
		Address:      "0.0.0.0",
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
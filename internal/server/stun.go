package server

import (
	"fmt"
	"net"

	"github.com/pion/stun"
	"github.com/sirupsen/logrus"

	"github.com/ga666666-new/pion-stun-server/internal/config"
)

// STUNServer represents a STUN server
type STUNServer struct {
	config   *config.STUNConfig
	conn     net.PacketConn
	logger   *logrus.Logger
	stopChan chan struct{}
}

// NewSTUNServer creates a new STUN server
func NewSTUNServer(cfg *config.STUNConfig, logger *logrus.Logger) *STUNServer {
	return &STUNServer{
		config:   cfg,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start starts the STUN server
func (s *STUNServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Address, s.config.Port)
	
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	
	s.conn = conn
	s.logger.WithField("address", addr).Info("STUN server started")

	go s.handlePackets()
	
	return nil
}

// Stop stops the STUN server
func (s *STUNServer) Stop() error {
	close(s.stopChan)
	
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}
	
	s.logger.Info("STUN server stopped")
	return nil
}

// handlePackets handles incoming STUN packets
func (s *STUNServer) handlePackets() {
	for {
		select {
		case <-s.stopChan:
			return
		default:
			buffer := make([]byte, 1500) // Create new buffer for each packet
			n, addr, err := s.conn.ReadFrom(buffer)
			if err != nil {
				s.logger.WithError(err).Error("Failed to read packet")
				continue
			}
			
			// Make a copy of the data for the goroutine
			data := make([]byte, n)
			copy(data, buffer[:n])
			go s.handlePacket(data, addr)
		}
	}
}

// handlePacket processes a single STUN packet
func (s *STUNServer) handlePacket(data []byte, addr net.Addr) {
	logger := s.logger.WithField("client", addr.String())
	
	// Parse STUN message
	msg := &stun.Message{
		Raw: data,
	}
	if err := msg.Decode(); err != nil {
		logger.WithError(err).Debug("Failed to parse STUN message")
		return
	}
	
	logger.WithFields(logrus.Fields{
		"type":   msg.Type.String(),
		"length": msg.Length,
	}).Debug("Received STUN message")
	
	// Handle different STUN message types
	switch msg.Type.Method {
	case stun.MethodBinding:
		s.handleBindingRequest(msg, addr)
	default:
		logger.WithField("method", msg.Type.Method).Debug("Unsupported STUN method")
	}
}

// handleBindingRequest handles STUN binding requests
func (s *STUNServer) handleBindingRequest(msg *stun.Message, addr net.Addr) {
	logger := s.logger.WithField("client", addr.String())
	
	// Create response message
	response := &stun.Message{
		Type:          stun.NewType(stun.MethodBinding, stun.ClassSuccessResponse),
		TransactionID: msg.TransactionID,
	}
	
	// Add XOR-MAPPED-ADDRESS attribute
	xorAddr := &stun.XORMappedAddress{}
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		xorAddr.IP = udpAddr.IP
		xorAddr.Port = udpAddr.Port
	}
	
	if err := xorAddr.AddTo(response); err != nil {
		logger.WithError(err).Error("Failed to add XOR-MAPPED-ADDRESS")
		return
	}
	
	// Add SOFTWARE attribute
	software := stun.NewSoftware("pion-stun-server/1.0")
	if err := software.AddTo(response); err != nil {
		logger.WithError(err).Error("Failed to add SOFTWARE attribute")
		return
	}
	
	// Send response
	response.Encode()
	responseData := response.Raw
	if len(responseData) == 0 {
		logger.Error("Failed to encode response")
		return
	}
	
	if _, err := s.conn.WriteTo(responseData, addr); err != nil {
		logger.WithError(err).Error("Failed to send response")
		return
	}
	
	logger.WithFields(logrus.Fields{
		"mapped_ip":   xorAddr.IP.String(),
		"mapped_port": xorAddr.Port,
	}).Debug("Sent binding response")
}

// GetStats returns STUN server statistics
func (s *STUNServer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":  "running",
		"address": fmt.Sprintf("%s:%d", s.config.Address, s.config.Port),
	}
}
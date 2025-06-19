package tests

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/pion/stun"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ga666666-new/pion-stun-server/internal/config"
	"github.com/ga666666-new/pion-stun-server/internal/server"
)

func TestSTUNServer(t *testing.T) {
	// Create test configuration
	cfg := &config.STUNConfig{
		Port:    19302, // Use standard STUN port for testing
		Address: "127.0.0.1",
	}

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Create STUN server
	stunServer := server.NewSTUNServer(cfg, logger)

	// Start server
	err := stunServer.Start()
	require.NoError(t, err)
	defer stunServer.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Use the configured address
	serverAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)

	t.Run("BindingRequest", func(t *testing.T) {
		// Create UDP connection to server
		conn, err := net.Dial("udp", serverAddr)
		require.NoError(t, err)
		defer conn.Close()

		// Create STUN binding request
		msg := &stun.Message{
			Type:          stun.NewType(stun.MethodBinding, stun.ClassRequest),
			TransactionID: stun.NewTransactionID(),
		}

		// Add SOFTWARE attribute
		software := stun.NewSoftware("test-client")
		err = software.AddTo(msg)
		require.NoError(t, err)

		// Encode and send message
		msg.Encode()
		data := msg.Raw

		_, err = conn.Write(data)
		require.NoError(t, err)

		// Read response
		buffer := make([]byte, 1500)
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := conn.Read(buffer)
		require.NoError(t, err)

		// Parse response
		response := &stun.Message{
			Raw: buffer[:n],
		}
		err = response.Decode()
		require.NoError(t, err)

		// Verify response
		assert.Equal(t, stun.NewType(stun.MethodBinding, stun.ClassSuccessResponse), response.Type)
		assert.Equal(t, msg.TransactionID, response.TransactionID)

		// Check for XOR-MAPPED-ADDRESS attribute
		var xorAddr stun.XORMappedAddress
		err = xorAddr.GetFrom(response)
		assert.NoError(t, err)
		assert.NotNil(t, xorAddr.IP)
		assert.NotZero(t, xorAddr.Port)
	})

	t.Run("InvalidMessage", func(t *testing.T) {
		// Create UDP connection to server
		conn, err := net.Dial("udp", serverAddr)
		require.NoError(t, err)
		defer conn.Close()

		// Send invalid data
		invalidData := []byte{0x00, 0x01, 0x02, 0x03}
		_, err = conn.Write(invalidData)
		require.NoError(t, err)

		// Server should not crash and should not send a response
		buffer := make([]byte, 1500)
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, err = conn.Read(buffer)
		// Should timeout since server doesn't respond to invalid messages
		assert.Error(t, err)
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := stunServer.GetStats()
		assert.Equal(t, "running", stats["status"])
		assert.Contains(t, stats["address"], "127.0.0.1:")
	})
}

func TestSTUNServerMultipleClients(t *testing.T) {
	// Create test configuration
	cfg := &config.STUNConfig{
		Port:    19303, // Use different port for second test
		Address: "127.0.0.1",
	}

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create and start STUN server
	stunServer := server.NewSTUNServer(cfg, logger)
	err := stunServer.Start()
	require.NoError(t, err)
	defer stunServer.Stop()

	time.Sleep(100 * time.Millisecond)

	// Use the configured address
	serverAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)

	// Test multiple concurrent clients
	numClients := 10
	done := make(chan bool, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			defer func() { done <- true }()

			// Create connection
			conn, err := net.Dial("udp", serverAddr)
			if err != nil {
				t.Errorf("Client %d: failed to connect: %v", clientID, err)
				return
			}
			defer conn.Close()

			// Send binding request
			msg := &stun.Message{
				Type:          stun.NewType(stun.MethodBinding, stun.ClassRequest),
				TransactionID: stun.NewTransactionID(),
			}

			msg.Encode()
			data := msg.Raw

			_, err = conn.Write(data)
			if err != nil {
				t.Errorf("Client %d: failed to send message: %v", clientID, err)
				return
			}

			// Read response
			buffer := make([]byte, 1500)
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := conn.Read(buffer)
			if err != nil {
				t.Errorf("Client %d: failed to read response: %v", clientID, err)
				return
			}

			// Parse response
			response := &stun.Message{
				Raw: buffer[:n],
			}
			err = response.Decode()
			if err != nil {
				t.Errorf("Client %d: failed to parse response: %v", clientID, err)
				return
			}

			// Verify response
			if response.Type != stun.NewType(stun.MethodBinding, stun.ClassSuccessResponse) {
				t.Errorf("Client %d: unexpected response type", clientID)
				return
			}

			if response.TransactionID != msg.TransactionID {
				t.Errorf("Client %d: transaction ID mismatch", clientID)
				return
			}
		}(i)
	}

	// Wait for all clients to complete
	for i := 0; i < numClients; i++ {
		select {
		case <-done:
			// Client completed
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for clients to complete")
		}
	}
}
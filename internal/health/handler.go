package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ga666666-new/pion-stun-server/internal/auth"
	"github.com/ga666666-new/pion-stun-server/internal/config"
	"github.com/ga666666-new/pion-stun-server/internal/server"
	"github.com/ga666666-new/pion-stun-server/pkg/models"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	config      *config.Config
	auth        *auth.MongoAuthenticator
	stunServer  *server.STUNServer
	turnServer  *server.TURNServer
	logger      *logrus.Logger
	startTime   time.Time
	httpServer  *http.Server
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(
	cfg *config.Config,
	auth *auth.MongoAuthenticator,
	stunServer *server.STUNServer,
	turnServer *server.TURNServer,
	logger *logrus.Logger,
) *HealthHandler {
	return &HealthHandler{
		config:     cfg,
		auth:       auth,
		stunServer: stunServer,
		turnServer: turnServer,
		logger:     logger,
		startTime:  time.Now(),
	}
}

// Start starts the health check HTTP server
func (h *HealthHandler) Start() error {
	mux := http.NewServeMux()
	
	// Health endpoints
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/ready", h.handleReady)
	mux.HandleFunc("/metrics", h.handleMetrics)
	mux.HandleFunc("/sessions", h.handleSessions)
	
	addr := fmt.Sprintf("%s:%d", h.config.Server.Health.Address, h.config.Server.Health.Port)
	
	h.httpServer = &http.Server{
		Addr:         addr,
		Handler:      h.corsMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	h.logger.WithField("address", addr).Info("Health check server started")
	
	go func() {
		if err := h.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.WithError(err).Error("Health server error")
		}
	}()
	
	return nil
}

// Stop stops the health check HTTP server
func (h *HealthHandler) Stop() error {
	if h.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := h.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown health server: %w", err)
		}
	}
	
	h.logger.Info("Health check server stopped")
	return nil
}

// handleHealth handles health check requests
func (h *HealthHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := &models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    time.Since(h.startTime),
		Services:  make(map[string]string),
	}
	
	// Check MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := h.checkMongoDB(ctx); err != nil {
		status.Status = "unhealthy"
		status.Services["mongodb"] = "unhealthy: " + err.Error()
	} else {
		status.Services["mongodb"] = "healthy"
	}
	
	// Check STUN server
	if h.stunServer != nil {
		status.Services["stun"] = "healthy"
	} else {
		status.Services["stun"] = "not running"
	}
	
	// Check TURN server
	if h.turnServer != nil {
		status.Services["turn"] = "healthy"
	} else {
		status.Services["turn"] = "not running"
	}
	
	// Add metrics
	status.Metrics = h.getMetrics()
	
	// Set response status code
	statusCode := http.StatusOK
	if status.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	h.writeJSONResponse(w, statusCode, status)
}

// handleReady handles readiness check requests
func (h *HealthHandler) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if all services are ready
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	ready := true
	services := make(map[string]string)
	
	// Check MongoDB
	if err := h.checkMongoDB(ctx); err != nil {
		ready = false
		services["mongodb"] = "not ready: " + err.Error()
	} else {
		services["mongodb"] = "ready"
	}
	
	// Check servers
	if h.stunServer == nil {
		ready = false
		services["stun"] = "not ready"
	} else {
		services["stun"] = "ready"
	}
	
	if h.turnServer == nil {
		ready = false
		services["turn"] = "not ready"
	} else {
		services["turn"] = "ready"
	}
	
	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now(),
		"services":  services,
	}
	
	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}
	
	h.writeJSONResponse(w, statusCode, response)
}

// handleMetrics handles metrics requests
func (h *HealthHandler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.getMetrics()
	h.writeJSONResponse(w, http.StatusOK, metrics)
}

// handleSessions handles active sessions requests
func (h *HealthHandler) handleSessions(w http.ResponseWriter, r *http.Request) {
	var sessions []*models.SessionInfo
	
	if h.turnServer != nil {
		sessions = h.turnServer.GetSessions()
	}
	
	response := map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
		"timestamp": time.Now(),
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

// checkMongoDB checks MongoDB connection health
func (h *HealthHandler) checkMongoDB(ctx context.Context) error {
	// Try to perform a simple operation to check connectivity
	// This is a simplified check - you might want to implement a proper ping
	_, err := h.auth.ListUsers(ctx, 0, 1)
	return err
}

// getMetrics returns server metrics
func (h *HealthHandler) getMetrics() *models.ServerMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	metrics := &models.ServerMetrics{
		GoroutineCount: runtime.NumGoroutine(),
		MemoryUsage:    float64(m.Alloc) / 1024 / 1024, // MB
	}
	
	// Add TURN session metrics
	if h.turnServer != nil {
		sessions := h.turnServer.GetSessions()
		metrics.ActiveSessions = len(sessions)
		
		// Calculate total bytes transferred
		var totalBytes int64
		for _, session := range sessions {
			totalBytes += session.BytesSent + session.BytesRecv
		}
		metrics.BytesTransferred = totalBytes
	}
	
	return metrics
}

// corsMiddleware adds CORS headers
func (h *HealthHandler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// writeJSONResponse writes a JSON response
func (h *HealthHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}
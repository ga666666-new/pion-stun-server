package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the authentication system
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Username  string             `bson:"username" json:"username"`
	Password  string             `bson:"password" json:"-"` // Never expose password in JSON
	Salt      string             `bson:"salt,omitempty" json:"-"`
	Enabled   bool               `bson:"enabled" json:"enabled"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	LastLogin *time.Time         `bson:"last_login,omitempty" json:"last_login,omitempty"`
	
	// Additional fields for TURN server
	Quota     *UserQuota         `bson:"quota,omitempty" json:"quota,omitempty"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// UserQuota represents usage quotas for a user
type UserQuota struct {
	MaxSessions     int   `bson:"max_sessions" json:"max_sessions"`
	MaxBandwidth    int64 `bson:"max_bandwidth" json:"max_bandwidth"` // bytes per second
	MaxDuration     int   `bson:"max_duration" json:"max_duration"`   // seconds
	CurrentSessions int   `bson:"current_sessions" json:"current_sessions"`
	UsedBandwidth   int64 `bson:"used_bandwidth" json:"used_bandwidth"`
	ResetAt         time.Time `bson:"reset_at" json:"reset_at"`
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Realm    string `json:"realm,omitempty"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Success   bool      `json:"success"`
	User      *User     `json:"user,omitempty"`
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// SessionInfo represents an active TURN session
type SessionInfo struct {
	ID          string    `bson:"_id" json:"id"`
	Username    string    `bson:"username" json:"username"`
	ClientAddr  string    `bson:"client_addr" json:"client_addr"`
	RelayAddr   string    `bson:"relay_addr" json:"relay_addr"`
	StartTime   time.Time `bson:"start_time" json:"start_time"`
	LastActive  time.Time `bson:"last_active" json:"last_active"`
	BytesSent   int64     `bson:"bytes_sent" json:"bytes_sent"`
	BytesRecv   int64     `bson:"bytes_recv" json:"bytes_recv"`
	PacketsSent int64     `bson:"packets_sent" json:"packets_sent"`
	PacketsRecv int64     `bson:"packets_recv" json:"packets_recv"`
}

// HealthStatus represents the health status of the server
type HealthStatus struct {
	Status      string            `json:"status"`
	Timestamp   time.Time         `json:"timestamp"`
	Version     string            `json:"version"`
	Uptime      time.Duration     `json:"uptime"`
	Services    map[string]string `json:"services"`
	Metrics     *ServerMetrics    `json:"metrics,omitempty"`
}

// ServerMetrics represents server performance metrics
type ServerMetrics struct {
	ActiveSessions    int     `json:"active_sessions"`
	TotalSessions     int64   `json:"total_sessions"`
	TotalUsers        int64   `json:"total_users"`
	BytesTransferred  int64   `json:"bytes_transferred"`
	PacketsTransferred int64  `json:"packets_transferred"`
	CPUUsage          float64 `json:"cpu_usage"`
	MemoryUsage       float64 `json:"memory_usage"`
	GoroutineCount    int     `json:"goroutine_count"`
}
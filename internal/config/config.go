package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	MongoDB  MongoDBConfig  `mapstructure:"mongodb"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Security SecurityConfig `mapstructure:"security"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	STUN   STUNConfig   `mapstructure:"stun"`
	TURN   TURNConfig   `mapstructure:"turn"`
	Health HealthConfig `mapstructure:"health"`
}

// STUNConfig holds STUN server configuration
type STUNConfig struct {
	Port    int    `mapstructure:"port"`
	Address string `mapstructure:"address"`
}

// TURNConfig holds TURN server configuration
type TURNConfig struct {
	Port         int      `mapstructure:"port"`
	Address      string   `mapstructure:"address"`
	Realm        string   `mapstructure:"realm"`
	PublicIP     string   `mapstructure:"public_ip"`
	RelayRanges  []string `mapstructure:"relay_ranges"`
	MaxLifetime  int      `mapstructure:"max_lifetime"`
	DefaultTTL   int      `mapstructure:"default_ttl"`
	// 调试选项：遇到权限错误时终止程序
	TerminateOnPermissionError bool `mapstructure:"terminate_on_permission_error"`
}

// HealthConfig holds health check configuration
type HealthConfig struct {
	Port    int    `mapstructure:"port"`
	Address string `mapstructure:"address"`
	Path    string `mapstructure:"path"`
}

// MongoDBConfig holds MongoDB connection and authentication configuration
type MongoDBConfig struct {
	URI        string            `mapstructure:"uri"`
	Database   string            `mapstructure:"database"`
	Collection string            `mapstructure:"collection"`
	Fields     MongoDBFields     `mapstructure:"fields"`
	Options    MongoDBOptions    `mapstructure:"options"`
}

// MongoDBFields defines customizable field names for user authentication
type MongoDBFields struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Enabled  string `mapstructure:"enabled"`
	Salt     string `mapstructure:"salt"`
}

// MongoDBOptions holds MongoDB connection options
type MongoDBOptions struct {
	MaxPoolSize     int `mapstructure:"max_pool_size"`
	MinPoolSize     int `mapstructure:"min_pool_size"`
	ConnectTimeout  int `mapstructure:"connect_timeout"`
	ServerSelection int `mapstructure:"server_selection_timeout"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	PasswordHashCost int    `mapstructure:"password_hash_cost"`
	SecretKey        string `mapstructure:"secret_key"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}

	// Set default values
	setDefaults()

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Configuration file not found - provide helpful error message
			if configPath != "" {
				return nil, fmt.Errorf("configuration file not found: %s\n\nPlease ensure the file exists and is readable", configPath)
			}
			return nil, fmt.Errorf("configuration file not found\n\n"+
				"Please create a configuration file at one of these locations:\n"+
				"  - ./configs/config.yaml\n"+
				"  - ./config.yaml\n\n"+
				"You can:\n"+
				"  1. Copy the example: cp configs/config.example.yaml configs/config.yaml\n"+
				"  2. Use the development template: cp configs/config.dev.yaml configs/config.yaml\n"+
				"  3. Specify a custom path: go run cmd/server/main.go -config /path/to/config.yaml\n\n"+
				"IMPORTANT: For Docker Compose MongoDB, ensure your config.yaml contains:\n"+
				"  mongodb:\n"+
				"    uri: \"mongodb://admin:password@localhost:27017/stun_turn?authSource=admin\"")
		}
		// Check if it's a file not found error when config path is specified
		if configPath != "" && strings.Contains(err.Error(), "no such file or directory") {
			return nil, fmt.Errorf("configuration file not found: %s\n\n"+
				"Please ensure the file exists and is readable.\n\n"+
				"You can:\n"+
				"  1. Create the file at the specified path\n"+
				"  2. Use an existing config: go run cmd/server/main.go -config configs/config.dev.yaml\n"+
				"  3. Use the default location: cp configs/config.dev.yaml configs/config.yaml\n\n"+
				"IMPORTANT: For Docker Compose MongoDB, ensure your config.yaml contains:\n"+
				"  mongodb:\n"+
				"    uri: \"mongodb://admin:password@localhost:27017/stun_turn?authSource=admin\"", configPath)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.stun.port", 3478)
	viper.SetDefault("server.stun.address", "0.0.0.0")
	viper.SetDefault("server.turn.port", 3479)
	viper.SetDefault("server.turn.address", "0.0.0.0")
	viper.SetDefault("server.turn.realm", "pion-stun-turn")
	viper.SetDefault("server.turn.relay_ranges", []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	viper.SetDefault("server.turn.max_lifetime", 3600)
	viper.SetDefault("server.turn.default_ttl", 600)
	viper.SetDefault("server.turn.terminate_on_permission_error", false)
	viper.SetDefault("server.health.port", 8080)
	viper.SetDefault("server.health.address", "0.0.0.0")
	viper.SetDefault("server.health.path", "/health")

	// MongoDB defaults
	viper.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongodb.database", "stun_turn")
	viper.SetDefault("mongodb.collection", "users")
	viper.SetDefault("mongodb.fields.username", "username")
	viper.SetDefault("mongodb.fields.password", "password")
	viper.SetDefault("mongodb.fields.enabled", "enabled")
	viper.SetDefault("mongodb.fields.salt", "salt")
	viper.SetDefault("mongodb.options.max_pool_size", 10)
	viper.SetDefault("mongodb.options.min_pool_size", 1)
	viper.SetDefault("mongodb.options.connect_timeout", 10)
	viper.SetDefault("mongodb.options.server_selection_timeout", 5)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")

	// Security defaults
	viper.SetDefault("security.password_hash_cost", 12)
}

// validate validates the configuration
func validate(config *Config) error {
	if config.MongoDB.URI == "" {
		return fmt.Errorf("mongodb.uri is required")
	}
	if config.MongoDB.Database == "" {
		return fmt.Errorf("mongodb.database is required")
	}
	if config.MongoDB.Collection == "" {
		return fmt.Errorf("mongodb.collection is required")
	}
	if config.MongoDB.Fields.Username == "" {
		return fmt.Errorf("mongodb.fields.username is required")
	}
	if config.MongoDB.Fields.Password == "" {
		return fmt.Errorf("mongodb.fields.password is required")
	}
	if config.Server.STUN.Port <= 0 || config.Server.STUN.Port > 65535 {
		return fmt.Errorf("invalid STUN port: %d", config.Server.STUN.Port)
	}
	if config.Server.TURN.Port <= 0 || config.Server.TURN.Port > 65535 {
		return fmt.Errorf("invalid TURN port: %d", config.Server.TURN.Port)
	}
	if config.Server.Health.Port <= 0 || config.Server.Health.Port > 65535 {
		return fmt.Errorf("invalid health port: %d", config.Server.Health.Port)
	}
	return nil
}
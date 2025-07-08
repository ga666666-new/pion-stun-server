package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ga666666-new/pion-stun-server/internal/auth"
	"github.com/ga666666-new/pion-stun-server/internal/config"
	"github.com/ga666666-new/pion-stun-server/internal/health"
	"github.com/ga666666-new/pion-stun-server/internal/server"
)

var (
	configPath = flag.String("config", "configs/config.dev.yaml", "Path to configuration file")
	version    = "1.0.0"
	buildTime  = "unknown"
	gitCommit  = "unknown"
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.WithFields(logrus.Fields{
		"version":    version,
		"build_time": buildTime,
		"git_commit": gitCommit,
	}).Info("Starting pion-stun-server")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level from configuration
	if level, err := logrus.ParseLevel(cfg.Logging.Level); err == nil {
		logger.SetLevel(level)
	}

	// Set log format from configuration
	if cfg.Logging.Format == "text" {
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	logger.WithField("config", cfg).Debug("Configuration loaded")

	// Initialize MongoDB authenticator
	authenticator, err := auth.NewMongoAuthenticator(&cfg.MongoDB)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize MongoDB authenticator")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := authenticator.Close(ctx); err != nil {
			logger.WithError(err).Error("Failed to close MongoDB connection")
		}
	}()

	logger.Info("MongoDB authenticator initialized")

	// Initialize STUN server
	stunServer := server.NewSTUNServer(&cfg.Server.STUN, logger)
	if err := stunServer.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start STUN server")
	}
	defer func() {
		if err := stunServer.Stop(); err != nil {
			logger.WithError(err).Error("Failed to stop STUN server")
		}
	}()

	// Initialize TURN server
	turnServer := server.NewTURNServer(&cfg.Server.TURN, authenticator, logger)
	if err := turnServer.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start TURN server")
	}
	defer func() {
		if err := turnServer.Stop(); err != nil {
			logger.WithError(err).Error("Failed to stop TURN server")
		}
	}()

	// Initialize health check handler
	healthHandler := health.NewHealthHandler(cfg, authenticator, stunServer, turnServer, logger)
	if err := healthHandler.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start health check server")
	}
	defer func() {
		if err := healthHandler.Stop(); err != nil {
			logger.WithError(err).Error("Failed to stop health check server")
		}
	}()

	logger.Info("All servers started successfully")

	// Print server information
	printServerInfo(cfg, logger)

	// Wait for shutdown signal
	waitForShutdown(logger)

	logger.Info("Server shutdown complete")
}

// printServerInfo prints server startup information
func printServerInfo(cfg *config.Config, logger *logrus.Logger) {
	logger.Info("=== Server Information ===")
	logger.WithFields(logrus.Fields{
		"address": fmt.Sprintf("%s:%d", cfg.Server.STUN.Address, cfg.Server.STUN.Port),
		"type":    "STUN",
	}).Info("Server listening")

	logger.WithFields(logrus.Fields{
		"address": fmt.Sprintf("%s:%d", cfg.Server.TURN.Address, cfg.Server.TURN.Port),
		"realm":   cfg.Server.TURN.Realm,
		"type":    "TURN",
	}).Info("Server listening")

	logger.WithFields(logrus.Fields{
		"address": fmt.Sprintf("%s:%d", cfg.Server.Health.Address, cfg.Server.Health.Port),
		"type":    "Health Check",
	}).Info("Server listening")

	logger.WithFields(logrus.Fields{
		"database":   cfg.MongoDB.Database,
		"collection": cfg.MongoDB.Collection,
	}).Info("MongoDB configuration")

	logger.Info("=== Ready to serve ===")
}

// waitForShutdown waits for shutdown signals
func waitForShutdown(logger *logrus.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.WithField("signal", sig.String()).Info("Received shutdown signal")
}
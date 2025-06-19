#!/bin/bash

# Configuration file checker for pion-stun-server
# This script helps users verify their configuration setup

set -e

echo "🔍 Checking pion-stun-server configuration..."
echo

# Check if config file exists
CONFIG_FILE=""
if [ -f "configs/config.yaml" ]; then
    CONFIG_FILE="configs/config.yaml"
    echo "✅ Found configuration file: configs/config.yaml"
elif [ -f "config.yaml" ]; then
    CONFIG_FILE="config.yaml"
    echo "✅ Found configuration file: config.yaml"
else
    echo "❌ Configuration file not found!"
    echo
    echo "Please create a configuration file at one of these locations:"
    echo "  - configs/config.yaml (recommended)"
    echo "  - config.yaml"
    echo
    echo "You can:"
    echo "  1. Copy the example: cp configs/config.example.yaml configs/config.yaml"
    echo "  2. Use the development template: cp configs/config.dev.yaml configs/config.yaml"
    echo
    echo "For Docker Compose MongoDB, ensure your config.yaml contains:"
    echo "  mongodb:"
    echo "    uri: \"mongodb://admin:password@localhost:27017/stun_turn?authSource=admin\""
    exit 1
fi

# Check MongoDB URI in config file
echo
echo "🔍 Checking MongoDB configuration..."

if grep -q "mongodb://.*@.*authSource=admin" "$CONFIG_FILE"; then
    echo "✅ MongoDB URI appears to have authentication configured"
elif grep -q "mongodb://localhost:27017" "$CONFIG_FILE"; then
    echo "⚠️  WARNING: MongoDB URI may be missing authentication"
    echo "   For Docker Compose, you need: mongodb://admin:password@localhost:27017/stun_turn?authSource=admin"
else
    echo "❓ Could not verify MongoDB URI format"
fi

# Check if Docker Compose MongoDB is running
echo
echo "🔍 Checking Docker Compose services..."

if command -v docker-compose >/dev/null 2>&1; then
    if docker-compose ps | grep -q "mongodb.*Up"; then
        echo "✅ MongoDB container is running"
    else
        echo "⚠️  MongoDB container is not running"
        echo "   Start it with: docker-compose up -d mongodb"
    fi
else
    echo "❓ docker-compose not found, skipping container check"
fi

# Check if MongoDB is accessible
echo
echo "🔍 Testing MongoDB connectivity..."

if command -v mongosh >/dev/null 2>&1; then
    if mongosh "mongodb://admin:password@localhost:27017/admin" --eval "db.runCommand('ping')" >/dev/null 2>&1; then
        echo "✅ MongoDB is accessible with authentication"
    else
        echo "❌ Cannot connect to MongoDB with authentication"
        echo "   Ensure MongoDB is running and credentials are correct"
    fi
elif command -v mongo >/dev/null 2>&1; then
    if mongo "mongodb://admin:password@localhost:27017/admin" --eval "db.runCommand('ping')" >/dev/null 2>&1; then
        echo "✅ MongoDB is accessible with authentication"
    else
        echo "❌ Cannot connect to MongoDB with authentication"
        echo "   Ensure MongoDB is running and credentials are correct"
    fi
else
    echo "❓ MongoDB client not found, skipping connectivity test"
fi

echo
echo "🎉 Configuration check complete!"
echo
echo "To start the server:"
echo "  go run cmd/server/main.go"
echo
echo "Or with a specific config file:"
echo "  go run cmd/server/main.go -config $CONFIG_FILE"
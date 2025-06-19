# Pion STUN/TURN Server Makefile

# Variables
BINARY_NAME=pion-stun-server
VERSION?=1.0.0
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags="-w -s -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Docker parameters
DOCKER_IMAGE=pion-stun-server
DOCKER_TAG?=latest

.PHONY: all build clean test test-coverage test-integration deps fmt vet lint docker docker-build docker-run help

# Default target
all: deps fmt vet test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/server

# Build for current platform
build-local:
	@echo "Building $(BINARY_NAME) for local platform..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/server

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests (requires MongoDB)
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./tests/...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Security scan (requires gosec)
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

# Stop Docker Compose services
docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

# View Docker Compose logs
docker-logs:
	docker-compose logs -f

# Create sample configuration
config:
	@echo "Creating sample configuration..."
	@if [ ! -f configs/config.yaml ]; then \
		cp configs/config.example.yaml configs/config.yaml; \
		echo "Configuration created at configs/config.yaml"; \
	else \
		echo "Configuration already exists at configs/config.yaml"; \
	fi

# Run the server locally
run: build-local config
	@echo "Starting server..."
	./$(BINARY_NAME) -config configs/config.yaml

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Create user in MongoDB (requires running MongoDB)
create-user:
	@echo "Creating test user..."
	@read -p "Enter username: " username; \
	read -s -p "Enter password: " password; \
	echo ""; \
	docker exec -it pion-stun-mongodb mongosh stun_turn --eval "db.users.insertOne({username: '$$username', password: '$$password', enabled: true, created_at: new Date(), updated_at: new Date()})"

# MongoDB shell
mongo-shell:
	@echo "Opening MongoDB shell..."
	docker exec -it pion-stun-mongodb mongosh stun_turn

# Show server status
status:
	@echo "Checking server status..."
	@curl -s http://localhost:8080/health | jq . || echo "Server not responding or jq not installed"

# Show active sessions
sessions:
	@echo "Checking active sessions..."
	@curl -s http://localhost:8080/sessions | jq . || echo "Server not responding or jq not installed"

# Show metrics
metrics:
	@echo "Checking server metrics..."
	@curl -s http://localhost:8080/metrics | jq . || echo "Server not responding or jq not installed"

# Release build (cross-platform)
release:
	@echo "Building release binaries..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/server
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/server
	@echo "Release binaries created in dist/"

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary for Linux"
	@echo "  build-local    - Build the binary for current platform"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  fmt            - Format code"
	@echo "  vet            - Vet code"
	@echo "  lint           - Lint code"
	@echo "  security       - Run security scan"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker Compose services"
	@echo "  docker-logs    - View Docker Compose logs"
	@echo "  config         - Create sample configuration"
	@echo "  run            - Run the server locally"
	@echo "  install-tools  - Install development tools"
	@echo "  create-user    - Create user in MongoDB"
	@echo "  mongo-shell    - Open MongoDB shell"
	@echo "  status         - Check server status"
	@echo "  sessions       - Check active sessions"
	@echo "  metrics        - Check server metrics"
	@echo "  release        - Build release binaries"
	@echo "  help           - Show this help"
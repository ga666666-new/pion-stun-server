# Pion STUN/TURN Server with MongoDB Authentication

A high-performance STUN/TURN server implementation using Go and Pion WebRTC library, with MongoDB-based authentication support.

## Features

- **STUN Server**: NAT traversal discovery service
- **TURN Server**: Media relay service for WebRTC applications
- **MongoDB Authentication**: Flexible user authentication with customizable database schema
- **Configuration Management**: Environment variables and configuration file support
- **Health Monitoring**: HTTP health check endpoints
- **Docker Support**: Containerized deployment
- **Comprehensive Testing**: Unit and integration tests

## Architecture

```
├── cmd/server/           # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── auth/            # MongoDB authentication
│   ├── server/          # STUN/TURN server implementation
│   └── health/          # Health check handlers
├── pkg/
│   └── models/          # Data models
├── tests/               # Test files
├── configs/             # Configuration files
└── docker/              # Docker related files
```

## Quick Start

### Prerequisites

- Go 1.21+
- MongoDB 4.4+
- Docker (optional)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/ga666666-new/pion-stun-server.git
cd pion-stun-server
```

2. Install dependencies:
```bash
go mod tidy
```

3. Start MongoDB (using Docker):
```bash
docker-compose up -d mongodb
```

4. Configure the server:
```bash
cp configs/config.example.yaml configs/config.yaml
# Edit configs/config.yaml with your settings
```

5. Run the server:
```bash
go run cmd/server/main.go
```

## Configuration

The server can be configured using environment variables or a YAML configuration file.

### Environment Variables

- `STUN_PORT`: STUN server port (default: 3478)
- `TURN_PORT`: TURN server port (default: 3479)
- `HEALTH_PORT`: Health check HTTP port (default: 8080)
- `MONGO_URI`: MongoDB connection URI
- `MONGO_DATABASE`: MongoDB database name
- `MONGO_COLLECTION`: MongoDB collection name
- `MONGO_USERNAME_FIELD`: Username field name (default: "username")
- `MONGO_PASSWORD_FIELD`: Password field name (default: "password")

### Configuration File

See `configs/config.example.yaml` for a complete configuration example.

## MongoDB Authentication

The server supports flexible MongoDB authentication with customizable schema:

```javascript
// Example user document in MongoDB
{
  "_id": ObjectId("..."),
  "username": "user1",
  "password": "$2a$10$...", // bcrypt hashed password
  "enabled": true,
  "created_at": ISODate("..."),
  "updated_at": ISODate("...")
}
```

You can customize the field names through configuration:

```yaml
mongodb:
  uri: "mongodb://localhost:27017"
  database: "stun_turn"
  collection: "users"
  fields:
    username: "username"
    password: "password"
    enabled: "enabled"
```

## API Endpoints

### Health Check

- `GET /health` - Server health status
- `GET /ready` - Readiness check

### Metrics (Optional)

- `GET /metrics` - Prometheus metrics (if enabled)

## Testing

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

Run integration tests:
```bash
go test -tags=integration ./tests/...
```

## Docker Deployment

### Using Docker Compose

```bash
docker-compose up -d
```

### Building Docker Image

```bash
docker build -t pion-stun-server .
```

## Performance Tuning

- Adjust `GOMAXPROCS` for CPU utilization
- Configure MongoDB connection pool settings
- Tune network buffer sizes for high throughput

## Security Considerations

- Use strong passwords and enable MongoDB authentication
- Configure firewall rules for STUN/TURN ports
- Use TLS for MongoDB connections in production
- Regularly rotate authentication credentials

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
# PION STUN/TURN Server

A high-performance STUN/TURN server implementation using the Pion WebRTC library.

## Features

- **STUN Server**: RFC 5389 compliant STUN server
- **TURN Server**: RFC 5766 compliant TURN server with authentication
- **MongoDB Authentication**: User management with MongoDB backend
- **Health Check**: HTTP health check endpoint
- **User Management**: CLI tool for user management
- **Enhanced Logging**: Detailed client session tracking and debugging (NEW!)

## Enhanced Logging Features (NEW!)

The server now includes comprehensive logging capabilities for debugging TURN protocol issues:

### Key Features
- **Client Session Tracking**: Complete tracking of each client's journey
- **Step-by-Step Logging**: Detailed logs for authentication, allocation, permission, and data transfer
- **Automatic Permission Debugging**: Enhanced permission handler with auto-grant in debug mode
- **Periodic Session Summaries**: Regular reports of active sessions
- **Detailed Error Diagnostics**: Comprehensive error analysis and troubleshooting

### Quick Start with Enhanced Logging

1. **Build the server**:
   ```bash
   make build
   ```

2. **Start with enhanced logging**:
   ```bash
   ./scripts/test-enhanced-logging.sh
   ```

3. **Monitor logs for detailed client tracking**:
   - `=== 新客户端会话开始 ===` - New client session
   - `=== TURN 认证请求接收 ===` - Authentication request
   - `=== 权限检查被调用 ===` - Permission check
   - `=== 活跃会话摘要 ===` - Active session summary

### Debugging "No Permission" Errors

When you see:
```
No Permission or Channel exists for [peer] on allocation [relay]
```

The enhanced logging will show:
1. Whether permission checks were triggered
2. Existing session information
3. Auto-granted permissions (in debug mode)
4. Complete client step history

For detailed information, see [Enhanced Logging Documentation](docs/ENHANCED_LOGGING.md).

## Quick Start

### Prerequisites

- Go 1.21 or higher
- MongoDB 4.4 or higher
- Docker (optional)

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/ga666666-new/pion-stun-server.git
   cd pion-stun-server
   ```

2. **Build the project**:
   ```bash
   make build
   ```

3. **Set up MongoDB**:
   ```bash
   # Using Docker
   docker-compose up -d mongodb
   
   # Or install MongoDB locally
   # Follow MongoDB installation guide for your OS
   ```

4. **Configure the server**:
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   # Edit configs/config.yaml with your settings
   ```

5. **Create a user**:
   ```bash
   ./bin/usermgr create --username testuser --password testpass
   ```

6. **Start the server**:
   ```bash
   ./bin/server
   ```

### Docker Deployment

```bash
# Build and start all services
docker-compose up -d

# Check logs
docker-compose logs -f stun-server
```

## Configuration

The server uses YAML configuration files. See `configs/config.example.yaml` for all available options.

### Key Configuration Sections

- **Server**: Basic server settings (address, port)
- **STUN**: STUN server configuration
- **TURN**: TURN server configuration (realm, public IP)
- **MongoDB**: Database connection settings
- **Logging**: Log level and format settings

## User Management

Use the `usermgr` CLI tool to manage users:

```bash
# Create a user
./bin/usermgr create --username john --password secret123

# List users
./bin/usermgr list

# Delete a user
./bin/usermgr delete --username john

# Update user quota
./bin/usermgr update --username john --max-sessions 10
```

## API Endpoints

- `GET /health` - Health check endpoint
- `GET /stats` - Server statistics
- `GET /sessions` - Active TURN sessions

## Testing

### Unit Tests
```bash
make test
```

### Integration Tests
```bash
make test-integration
```

### Manual Testing
```bash
# Test STUN functionality
stun-client stun://localhost:3478

# Test TURN functionality (requires authentication)
turn-client turn://testuser:testpass@localhost:3479
```

## Monitoring

The server provides several monitoring endpoints:

- `/health` - Basic health status
- `/stats` - Detailed server statistics
- `/sessions` - Active TURN sessions with user information

## Performance Tuning

### MongoDB Optimization
- Use appropriate indexes for user queries
- Configure connection pooling
- Monitor query performance

### Server Optimization
- Adjust log levels for production
- Configure appropriate timeouts
- Monitor memory usage with session tracking

## Troubleshooting

### Common Issues

1. **"No Permission" errors**: 
   - Check enhanced logs for permission tracking
   - Verify client implements CreatePermission correctly
   - Review debug mode auto-grant logs

2. **Authentication failures**:
   - Verify user credentials in MongoDB
   - Check TURN realm configuration
   - Review authentication logs

3. **Connection issues**:
   - Verify firewall settings
   - Check public IP configuration
   - Test with STUN discovery

### Debug Mode

The server includes a debug mode with automatic permission granting:
- All permission checks are logged in detail
- Permissions are automatically granted for debugging
- Complete client session history is maintained

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Pion WebRTC](https://github.com/pion/webrtc) for the excellent WebRTC implementation
- [MongoDB](https://www.mongodb.com/) for the database backend
- [Logrus](https://github.com/sirupsen/logrus) for structured logging
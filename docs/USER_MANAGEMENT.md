# User Management for TURN Authentication

This document describes how to manage users for TURN server authentication.

## Overview

The TURN server requires user authentication to allocate relay addresses. Users are stored in MongoDB with bcrypt-hashed passwords.

## User Management CLI Tool

The user management tool is located at `cmd/usermgr/main.go` and provides CRUD operations for user management.

### Prerequisites

1. MongoDB server running (via Docker Compose or standalone)
2. Go 1.21+ installed
3. Proper MongoDB connection configuration

### Usage

#### List all users
```bash
go run cmd/usermgr/main.go list
```

#### Add a new user
```bash
go run cmd/usermgr/main.go add <username> <password>
```

Example:
```bash
go run cmd/usermgr/main.go add testuser1 password123
go run cmd/usermgr/main.go add testuser2 password456
```

#### Delete a user
```bash
go run cmd/usermgr/main.go delete <username>
```

Example:
```bash
go run cmd/usermgr/main.go delete testuser1
```

#### Update user password
```bash
go run cmd/usermgr/main.go update <username> <new_password>
```

Example:
```bash
go run cmd/usermgr/main.go update testuser1 newpassword123
```

## MongoDB Configuration

The tool uses the same MongoDB configuration as the TURN server:

- **Connection String**: `mongodb://admin:password@localhost:27017/stun_turn?authSource=admin`
- **Database**: `stun_turn`
- **Collection**: `users`

## User Schema

Users are stored with the following structure:

```json
{
  "_id": "ObjectId",
  "username": "string (unique)",
  "password": "string (bcrypt hash)",
  "enabled": true,
  "created_at": "ISODate",
  "updated_at": "ISODate"
}
```

## Security Features

- **Password Hashing**: All passwords are hashed using bcrypt with cost factor 12
- **Unique Usernames**: Username uniqueness is enforced at the database level
- **User Status**: Users can be enabled/disabled (currently all users are enabled by default)

## Testing TURN Authentication

After adding users, you can test TURN authentication using TURN client tools or the provided test scripts.

### Example Test Users

The following test users have been created for development:

- **Username**: `testuser1`, **Password**: `password123`
- **Username**: `testuser2`, **Password**: `password456`

## Troubleshooting

### Common Issues

1. **MongoDB Connection Failed**
   - Ensure MongoDB is running
   - Check connection string in configuration
   - Verify authentication credentials

2. **User Already Exists**
   - Usernames must be unique
   - Use the `update` command to change existing user passwords

3. **Authentication Failed in TURN Server**
   - Verify user exists: `go run cmd/usermgr/main.go list`
   - Check username/password spelling
   - Ensure user is enabled

### Logs

The user management tool provides detailed logging for all operations. Check the output for specific error messages.

## Integration with TURN Server

The TURN server automatically uses the MongoDB user database for authentication. No additional configuration is required once users are added to the database.

The authentication flow:
1. TURN client sends allocation request with username/password
2. TURN server queries MongoDB for the user
3. Server verifies password using bcrypt
4. If authentication succeeds, allocation is granted

## Production Considerations

1. **Strong Passwords**: Use strong passwords for production users
2. **User Rotation**: Regularly rotate user credentials
3. **Monitoring**: Monitor authentication attempts and failures
4. **Backup**: Regularly backup the user database
5. **Access Control**: Limit access to the user management tool
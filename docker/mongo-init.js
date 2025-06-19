// MongoDB initialization script for STUN/TURN server

// Switch to the stun_turn database
db = db.getSiblingDB('stun_turn');

// Create users collection with validation schema
db.createCollection('users', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['username', 'password', 'enabled', 'created_at', 'updated_at'],
      properties: {
        username: {
          bsonType: 'string',
          description: 'Username must be a string and is required'
        },
        password: {
          bsonType: 'string',
          description: 'Password must be a string and is required'
        },
        enabled: {
          bsonType: 'bool',
          description: 'Enabled must be a boolean and is required'
        },
        salt: {
          bsonType: 'string',
          description: 'Salt must be a string if provided'
        },
        created_at: {
          bsonType: 'date',
          description: 'Created at must be a date and is required'
        },
        updated_at: {
          bsonType: 'date',
          description: 'Updated at must be a date and is required'
        },
        last_login: {
          bsonType: 'date',
          description: 'Last login must be a date if provided'
        },
        quota: {
          bsonType: 'object',
          properties: {
            max_sessions: {
              bsonType: 'int',
              minimum: 0
            },
            max_bandwidth: {
              bsonType: 'long',
              minimum: 0
            },
            max_duration: {
              bsonType: 'int',
              minimum: 0
            },
            current_sessions: {
              bsonType: 'int',
              minimum: 0
            },
            used_bandwidth: {
              bsonType: 'long',
              minimum: 0
            },
            reset_at: {
              bsonType: 'date'
            }
          }
        },
        metadata: {
          bsonType: 'object',
          description: 'Additional metadata as key-value pairs'
        }
      }
    }
  }
});

// Create unique index on username
db.users.createIndex({ username: 1 }, { unique: true });

// Create index on enabled field for faster queries
db.users.createIndex({ enabled: 1 });

// Create index on created_at for sorting
db.users.createIndex({ created_at: -1 });

// Insert sample users (passwords are bcrypt hashed)
// Password for all sample users is "password123"
db.users.insertMany([
  {
    username: 'testuser1',
    password: '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/RK.s5uIoO', // password123
    enabled: true,
    created_at: new Date(),
    updated_at: new Date(),
    quota: {
      max_sessions: 5,
      max_bandwidth: 1048576, // 1MB/s
      max_duration: 3600,     // 1 hour
      current_sessions: 0,
      used_bandwidth: 0,
      reset_at: new Date(Date.now() + 24 * 60 * 60 * 1000) // 24 hours from now
    },
    metadata: {
      plan: 'basic',
      region: 'us-east-1'
    }
  },
  {
    username: 'testuser2',
    password: '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/RK.s5uIoO', // password123
    enabled: true,
    created_at: new Date(),
    updated_at: new Date(),
    quota: {
      max_sessions: 10,
      max_bandwidth: 5242880, // 5MB/s
      max_duration: 7200,     // 2 hours
      current_sessions: 0,
      used_bandwidth: 0,
      reset_at: new Date(Date.now() + 24 * 60 * 60 * 1000)
    },
    metadata: {
      plan: 'premium',
      region: 'us-west-2'
    }
  },
  {
    username: 'disableduser',
    password: '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/RK.s5uIoO', // password123
    enabled: false,
    created_at: new Date(),
    updated_at: new Date(),
    metadata: {
      plan: 'basic',
      disabled_reason: 'account_suspended'
    }
  }
]);

// Create sessions collection for tracking active TURN sessions
db.createCollection('sessions', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['username', 'client_addr', 'start_time', 'last_active'],
      properties: {
        username: {
          bsonType: 'string',
          description: 'Username must be a string and is required'
        },
        client_addr: {
          bsonType: 'string',
          description: 'Client address must be a string and is required'
        },
        relay_addr: {
          bsonType: 'string',
          description: 'Relay address must be a string if provided'
        },
        start_time: {
          bsonType: 'date',
          description: 'Start time must be a date and is required'
        },
        last_active: {
          bsonType: 'date',
          description: 'Last active must be a date and is required'
        },
        bytes_sent: {
          bsonType: 'long',
          minimum: 0
        },
        bytes_recv: {
          bsonType: 'long',
          minimum: 0
        },
        packets_sent: {
          bsonType: 'long',
          minimum: 0
        },
        packets_recv: {
          bsonType: 'long',
          minimum: 0
        }
      }
    }
  }
});

// Create indexes for sessions collection
db.sessions.createIndex({ username: 1 });
db.sessions.createIndex({ start_time: -1 });
db.sessions.createIndex({ last_active: 1 }); // For cleanup queries

// Create TTL index to automatically remove old sessions (after 1 hour of inactivity)
db.sessions.createIndex({ last_active: 1 }, { expireAfterSeconds: 3600 });

print('MongoDB initialization completed successfully!');
print('Sample users created:');
print('- testuser1 (enabled, basic plan)');
print('- testuser2 (enabled, premium plan)');
print('- disableduser (disabled)');
print('Password for all users: password123');
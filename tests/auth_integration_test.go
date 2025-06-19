//go:build integration

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ga666666-new/pion-stun-server/internal/auth"
	"github.com/ga666666-new/pion-stun-server/internal/config"
	"github.com/ga666666-new/pion-stun-server/pkg/models"
)

// Note: These tests require a running MongoDB instance
// Use build tag 'integration' to run these tests: go test -tags=integration

func TestMongoAuthenticator(t *testing.T) {
	// Setup test configuration
	cfg := &config.MongoDBConfig{
		URI:        "mongodb://localhost:27017",
		Database:   "test_stun_server",
		Collection: "test_users",
		Fields: config.MongoDBFields{
			Username: "username",
			Password: "password",
			Enabled:  "enabled",
		},
	}

	// Create authenticator
	authenticator, err := auth.NewMongoAuthenticator(cfg)
	require.NoError(t, err)
	defer authenticator.Close()

	ctx := context.Background()

	// Test user creation
	testUser := &models.User{
		Username: "testuser",
		Password: "testpass",
		Enabled:  true,
		Quota: &models.UserQuota{
			MaxSessions:     10,
			CurrentSessions: 0,
		},
	}

	err = authenticator.CreateUser(ctx, testUser)
	require.NoError(t, err)

	// Test authentication with correct credentials
	user, err := authenticator.Authenticate(ctx, "testuser", "testpass")
	require.NoError(t, err)
	assert.Equal(t, "testuser", user.Username)
	assert.True(t, user.Enabled)

	// Test authentication with wrong password
	_, err = authenticator.Authenticate(ctx, "testuser", "wrongpass")
	assert.Error(t, err)

	// Test authentication with non-existent user
	_, err = authenticator.Authenticate(ctx, "nonexistent", "password")
	assert.Error(t, err)

	// Test user update
	testUser.Enabled = false
	err = authenticator.UpdateUser(ctx, testUser)
	require.NoError(t, err)

	// Test authentication with disabled user
	_, err = authenticator.Authenticate(ctx, "testuser", "testpass")
	assert.Error(t, err)

	// Test user deletion
	err = authenticator.DeleteUser(ctx, "testuser")
	require.NoError(t, err)

	// Verify user is deleted
	_, err = authenticator.GetUser(ctx, "testuser")
	assert.Error(t, err)
}

func TestMongoAuthenticatorCustomFields(t *testing.T) {
	// Setup test configuration with custom field names
	cfg := &config.MongoDBConfig{
		URI:        "mongodb://localhost:27017",
		Database:   "test_stun_server",
		Collection: "custom_users",
		Fields: config.MongoDBFields{
			Username: "user_name",
			Password: "user_pass",
			Enabled:  "is_active",
		},
	}

	// Create authenticator
	authenticator, err := auth.NewMongoAuthenticator(cfg)
	require.NoError(t, err)
	defer authenticator.Close()

	ctx := context.Background()

	// Test user creation with custom fields
	testUser := &models.User{
		Username: "customuser",
		Password: "custompass",
		Enabled:  true,
	}

	err = authenticator.CreateUser(ctx, testUser)
	require.NoError(t, err)

	// Test authentication
	user, err := authenticator.Authenticate(ctx, "customuser", "custompass")
	require.NoError(t, err)
	assert.Equal(t, "customuser", user.Username)

	// Cleanup
	err = authenticator.DeleteUser(ctx, "customuser")
	require.NoError(t, err)
}
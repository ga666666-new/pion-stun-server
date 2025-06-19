package auth

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/ga666666-new/pion-stun-server/internal/config"
	"github.com/ga666666-new/pion-stun-server/pkg/models"
)

// MongoAuthenticator implements authentication using MongoDB
type MongoAuthenticator struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	config     *config.MongoDBConfig
}

// NewMongoAuthenticator creates a new MongoDB authenticator
func NewMongoAuthenticator(cfg *config.MongoDBConfig) (*MongoAuthenticator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Options.ConnectTimeout)*time.Second)
	defer cancel()

	// Configure MongoDB client options
	clientOptions := options.Client().ApplyURI(cfg.URI)
	clientOptions.SetMaxPoolSize(uint64(cfg.Options.MaxPoolSize))
	clientOptions.SetMinPoolSize(uint64(cfg.Options.MinPoolSize))
	clientOptions.SetServerSelectionTimeout(time.Duration(cfg.Options.ServerSelection) * time.Second)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.Database)
	collection := database.Collection(cfg.Collection)

	auth := &MongoAuthenticator{
		client:     client,
		database:   database,
		collection: collection,
		config:     cfg,
	}

	// Create indexes
	if err := auth.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return auth, nil
}

// Authenticate verifies user credentials
func (m *MongoAuthenticator) Authenticate(ctx context.Context, username, password string) (*models.User, error) {
	// Build query using configured field names
	filter := bson.M{
		m.config.Fields.Username: username,
	}

	// Add enabled field check if configured
	if m.config.Fields.Enabled != "" {
		filter[m.config.Fields.Enabled] = true
	}

	var result bson.M
	err := m.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Extract password from result
	storedPassword, ok := result[m.config.Fields.Password].(string)
	if !ok {
		return nil, fmt.Errorf("invalid password field type")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	// Convert result to User model
	user, err := m.resultToUser(result)
	if err != nil {
		return nil, fmt.Errorf("failed to convert user data: %w", err)
	}

	// Update last login time
	go m.updateLastLogin(context.Background(), user.ID)

	return user, nil
}

// CreateUser creates a new user
func (m *MongoAuthenticator) CreateUser(ctx context.Context, user *models.User, plainPassword string) error {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Build document using configured field names
	doc := bson.M{
		m.config.Fields.Username: user.Username,
		m.config.Fields.Password: string(hashedPassword),
		"created_at":             time.Now(),
		"updated_at":             time.Now(),
	}

	// Add enabled field if configured
	if m.config.Fields.Enabled != "" {
		doc[m.config.Fields.Enabled] = user.Enabled
	}

	// Add salt field if configured and provided
	if m.config.Fields.Salt != "" && user.Salt != "" {
		doc[m.config.Fields.Salt] = user.Salt
	}

	// Add additional fields
	if user.Quota != nil {
		doc["quota"] = user.Quota
	}
	if user.Metadata != nil {
		doc["metadata"] = user.Metadata
	}

	result, err := m.collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// UpdateUser updates an existing user
func (m *MongoAuthenticator) UpdateUser(ctx context.Context, user *models.User) error {
	filter := bson.M{"_id": user.ID}
	
	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// Update configurable fields
	if user.Username != "" {
		update["$set"].(bson.M)[m.config.Fields.Username] = user.Username
	}
	if m.config.Fields.Enabled != "" {
		update["$set"].(bson.M)[m.config.Fields.Enabled] = user.Enabled
	}
	if user.Quota != nil {
		update["$set"].(bson.M)["quota"] = user.Quota
	}
	if user.Metadata != nil {
		update["$set"].(bson.M)["metadata"] = user.Metadata
	}

	_, err := m.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdatePassword updates user password
func (m *MongoAuthenticator) UpdatePassword(ctx context.Context, userID primitive.ObjectID, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			m.config.Fields.Password: string(hashedPassword),
			"updated_at":             time.Now(),
		},
	}

	_, err = m.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// DeleteUser deletes a user
func (m *MongoAuthenticator) DeleteUser(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"_id": userID}
	_, err := m.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// GetUser retrieves a user by ID
func (m *MongoAuthenticator) GetUser(ctx context.Context, userID primitive.ObjectID) (*models.User, error) {
	filter := bson.M{"_id": userID}
	
	var result bson.M
	err := m.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return m.resultToUser(result)
}

// ListUsers retrieves all users with pagination
func (m *MongoAuthenticator) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	opts := options.Find()
	opts.SetSkip(int64(offset))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := m.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}

		user, err := m.resultToUser(result)
		if err != nil {
			continue // Skip invalid users
		}
		users = append(users, user)
	}

	return users, nil
}

// Close closes the MongoDB connection
func (m *MongoAuthenticator) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// createIndexes creates necessary database indexes
func (m *MongoAuthenticator) createIndexes(ctx context.Context) error {
	// Create unique index on username field
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: m.config.Fields.Username, Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := m.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create username index: %w", err)
	}

	// Create index on enabled field if configured
	if m.config.Fields.Enabled != "" {
		indexModel = mongo.IndexModel{
			Keys: bson.D{{Key: m.config.Fields.Enabled, Value: 1}},
		}
		_, err = m.collection.Indexes().CreateOne(ctx, indexModel)
		if err != nil {
			return fmt.Errorf("failed to create enabled index: %w", err)
		}
	}

	return nil
}

// updateLastLogin updates the user's last login time
func (m *MongoAuthenticator) updateLastLogin(ctx context.Context, userID primitive.ObjectID) {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"last_login": time.Now(),
		},
	}
	m.collection.UpdateOne(ctx, filter, update)
}

// resultToUser converts MongoDB result to User model
func (m *MongoAuthenticator) resultToUser(result bson.M) (*models.User, error) {
	user := &models.User{}

	// Extract ID
	if id, ok := result["_id"].(primitive.ObjectID); ok {
		user.ID = id
	}

	// Extract username using configured field name
	if username, ok := result[m.config.Fields.Username].(string); ok {
		user.Username = username
	}

	// Extract enabled status using configured field name
	if m.config.Fields.Enabled != "" {
		if enabled, ok := result[m.config.Fields.Enabled].(bool); ok {
			user.Enabled = enabled
		}
	} else {
		user.Enabled = true // Default to enabled if field not configured
	}

	// Extract timestamps
	if createdAt, ok := result["created_at"].(primitive.DateTime); ok {
		user.CreatedAt = createdAt.Time()
	}
	if updatedAt, ok := result["updated_at"].(primitive.DateTime); ok {
		user.UpdatedAt = updatedAt.Time()
	}
	if lastLogin, ok := result["last_login"].(primitive.DateTime); ok {
		t := lastLogin.Time()
		user.LastLogin = &t
	}

	// Extract quota if present
	if quotaData, ok := result["quota"].(bson.M); ok {
		quota := &models.UserQuota{}
		if maxSessions, ok := quotaData["max_sessions"].(int32); ok {
			quota.MaxSessions = int(maxSessions)
		}
		if maxBandwidth, ok := quotaData["max_bandwidth"].(int64); ok {
			quota.MaxBandwidth = maxBandwidth
		}
		if maxDuration, ok := quotaData["max_duration"].(int32); ok {
			quota.MaxDuration = int(maxDuration)
		}
		if currentSessions, ok := quotaData["current_sessions"].(int32); ok {
			quota.CurrentSessions = int(currentSessions)
		}
		if usedBandwidth, ok := quotaData["used_bandwidth"].(int64); ok {
			quota.UsedBandwidth = usedBandwidth
		}
		if resetAt, ok := quotaData["reset_at"].(primitive.DateTime); ok {
			quota.ResetAt = resetAt.Time()
		}
		user.Quota = quota
	}

	// Extract metadata if present
	if metadata, ok := result["metadata"].(bson.M); ok {
		user.Metadata = make(map[string]interface{})
		for k, v := range metadata {
			user.Metadata[k] = v
		}
	}

	return user, nil
}
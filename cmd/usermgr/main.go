package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/ga666666-new/pion-stun-server/internal/config"
)

func main() {
	var (
		configPath = flag.String("config", "configs/config.yaml", "Path to configuration file")
		action     = flag.String("action", "", "Action to perform: add, delete, list, update")
		username   = flag.String("username", "", "Username")
		password   = flag.String("password", "", "Password")
		enabled    = flag.Bool("enabled", true, "Enable user")
	)
	flag.Parse()

	if *action == "" {
		fmt.Println("User Management Tool for Pion STUN/TURN Server")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  Add user:    go run cmd/usermgr/main.go -action add -username testuser1 -password password123")
		fmt.Println("  List users:  go run cmd/usermgr/main.go -action list")
		fmt.Println("  Delete user: go run cmd/usermgr/main.go -action delete -username testuser1")
		fmt.Println("  Update user: go run cmd/usermgr/main.go -action update -username testuser1 -password newpass")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -config      Path to configuration file (default: configs/config.yaml)")
		fmt.Println("  -action      Action to perform: add, delete, list, update")
		fmt.Println("  -username    Username")
		fmt.Println("  -password    Password")
		fmt.Println("  -enabled     Enable user (default: true)")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	db := client.Database(cfg.MongoDB.Database)
	collection := db.Collection(cfg.MongoDB.Collection)

	switch *action {
	case "add":
		if *username == "" || *password == "" {
			log.Fatal("Username and password are required for add action")
		}
		err = addUser(ctx, collection, cfg, *username, *password, *enabled)
	case "delete":
		if *username == "" {
			log.Fatal("Username is required for delete action")
		}
		err = deleteUser(ctx, collection, cfg, *username)
	case "list":
		err = listUsers(ctx, collection, cfg)
	case "update":
		if *username == "" {
			log.Fatal("Username is required for update action")
		}
		err = updateUser(ctx, collection, cfg, *username, *password, *enabled)
	default:
		log.Fatalf("Unknown action: %s", *action)
	}

	if err != nil {
		log.Fatalf("Operation failed: %v", err)
	}
}

func addUser(ctx context.Context, collection *mongo.Collection, cfg *config.Config, username, password string, enabled bool) error {
	// Check if user already exists
	filter := bson.M{cfg.MongoDB.Fields.Username: username}
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("user '%s' already exists", username)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user document
	now := time.Now()
	user := bson.M{
		cfg.MongoDB.Fields.Username: username,
		cfg.MongoDB.Fields.Password: string(hashedPassword),
		cfg.MongoDB.Fields.Enabled:  enabled,
		"created_at":                now,
		"updated_at":                now,
	}

	// Insert user
	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	fmt.Printf("User '%s' added successfully with ID: %s\n", username, result.InsertedID)
	return nil
}

func deleteUser(ctx context.Context, collection *mongo.Collection, cfg *config.Config, username string) error {
	filter := bson.M{cfg.MongoDB.Fields.Username: username}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user '%s' not found", username)
	}

	fmt.Printf("User '%s' deleted successfully\n", username)
	return nil
}

func listUsers(ctx context.Context, collection *mongo.Collection, cfg *config.Config) error {
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer cursor.Close(ctx)

	fmt.Println("Users:")
	fmt.Println("------")
	count := 0
	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return fmt.Errorf("failed to decode user: %w", err)
		}

		username := result[cfg.MongoDB.Fields.Username]
		enabled := result[cfg.MongoDB.Fields.Enabled]
		createdAt := result["created_at"]
		
		fmt.Printf("Username: %s, Enabled: %v, Created: %v\n", username, enabled, createdAt)
		count++
	}

	if count == 0 {
		fmt.Println("No users found")
	} else {
		fmt.Printf("\nTotal users: %d\n", count)
	}

	return cursor.Err()
}

func updateUser(ctx context.Context, collection *mongo.Collection, cfg *config.Config, username, password string, enabled bool) error {
	filter := bson.M{cfg.MongoDB.Fields.Username: username}
	
	update := bson.M{
		"$set": bson.M{
			cfg.MongoDB.Fields.Enabled: enabled,
			"updated_at":               time.Now(),
		},
	}

	// Update password if provided
	if password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		update["$set"].(bson.M)[cfg.MongoDB.Fields.Password] = string(hashedPassword)
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user '%s' not found", username)
	}

	fmt.Printf("User '%s' updated successfully\n", username)
	return nil
}
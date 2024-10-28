package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectToMongoDB(host, port string) (*mongo.Database, error) {
	// Construct the connection URI for replica set
	uri := fmt.Sprintf("mongodb://%s-0.%s:%s,%s-1.%s:%s,%s-2.%s:%s/?replicaSet=rs0&connectTimeoutMS=20000&serverSelectionTimeoutMS=20000",
		host, host, port,
		host, host, port,
		host, host, port)

	// Set client options
	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(20 * time.Second).
		SetConnectTimeout(20 * time.Second).
		SetDirect(false)

	// Create context with longer timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to MongoDB with retries
	var client *mongo.Client
	var err error
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		client, err = mongo.Connect(ctx, clientOptions)
		if err == nil {
			// Check the connection
			if err = client.Ping(ctx, nil); err == nil {
				return client.Database("product_service"), nil
			}
		}

		log.Printf("Failed to connect to MongoDB (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %v", maxRetries, err)
}

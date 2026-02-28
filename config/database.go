package config

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB() *mongo.Client {

	log.Println("Attempting to connect to MongoDB...")

	// Read from environment variable
	mongoURI := os.Getenv("MONGO_URI")

	// Fallback for local development
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	log.Println("Using Mongo URI:", mongoURI)

	clientOptions := options.Client().ApplyURI(mongoURI)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("MongoDB is not reachable: %v", err)
	}

	log.Println("Successfully connected to MongoDB!")
	return client
}

var Client *mongo.Client = ConnectDB()

func OpenCollection(collectionName string) *mongo.Collection {

	if Client == nil {
		log.Fatal("MongoDB client is not initialized.")
	}

	return Client.Database("usersdb").Collection(collectionName)
}

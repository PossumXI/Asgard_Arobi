package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
)

func main() {
	log.Println("ASGARD Database Verification & Migration Tool")

	cfg, err := db.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Testing PostgreSQL connection...")
	pgDB, err := db.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	defer pgDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pgDB.Health(ctx); err != nil {
		log.Fatalf("PostgreSQL health check failed: %v", err)
	}
	log.Println("✓ PostgreSQL connection successful")

	log.Println("Testing MongoDB connection...")
	mongoDB, err := db.NewMongoDB(cfg)
	if err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	defer mongoDB.Close(ctx)

	if err := mongoDB.Health(ctx); err != nil {
		log.Fatalf("MongoDB health check failed: %v", err)
	}
	log.Println("✓ MongoDB connection successful")

	log.Println("Verifying PostgreSQL schema...")
	tables := []string{"users", "satellites", "hunoids", "missions", "alerts", "threats", "subscriptions", "audit_logs", "ethical_decisions"}
	for _, table := range tables {
		var exists bool
		query := fmt.Sprintf("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '%s')", table)
		if err := pgDB.QueryRowContext(ctx, query).Scan(&exists); err != nil {
			log.Fatalf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			log.Fatalf("Table %s does not exist", table)
		}
		log.Printf("✓ Table '%s' exists", table)
	}

	log.Println("Verifying MongoDB collections...")
	collections := []string{
		"satellite_telemetry",
		"hunoid_telemetry",
		"network_flows",
		"security_events",
		"vla_inferences",
		"router_training_episodes",
	}
	dbCollections, err := mongoDB.Database().ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to list collections: %v", err)
	}

	collectionMap := make(map[string]bool)
	for _, col := range dbCollections {
		collectionMap[col] = true
	}

	for _, col := range collections {
		if !collectionMap[col] {
			log.Fatalf("Collection %s does not exist", col)
		}
		log.Printf("✓ Collection '%s' exists", col)
	}

	log.Println("\n=== DATABASE VERIFICATION COMPLETE ===")
	log.Println("All connections successful")
	log.Println("All schemas verified")
	os.Exit(0)
}

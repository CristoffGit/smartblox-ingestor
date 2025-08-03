package main

import (
	"context"
	"log"
	"os/signal"
	"smartblox-ingestor/client"
	"smartblox-ingestor/config"
	"smartblox-ingestor/persistence"
	"syscall"
	"time"
)

const (
	pollInterval    = 2 * time.Second
	persistInterval = 10
)

var mongoURI = config.GetEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")

func main() {
	// Initialize context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize api
	apiClient := client.APIClient(client.NewSmartBloxClient())

	log.Printf("Connecting to MongoDB at %s...", mongoURI)
	mongoStore, err := persistence.NewMongoStore(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		log.Println("Disconnecting from MongoDB...")
		if err := mongoStore.Close(context.Background()); err != nil {
			log.Printf("ERROR: Failed to disconnect from MongoDB cleanly: %v", err)
		}
	}()
	log.Println("MongoDB connection successful.")

	metrics, err := mongoStore.Load(ctx)
	if err != nil {
		log.Fatalf("FATAL: could not load metrics from MongoDB: %v", err)
	}
	log.Printf("Service starting. Last processed round: %d", metrics.LastProcessedRound)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Iterate
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentRound := metrics.LastProcessedRound + 1
			block, err := apiClient.GetBlock(ctx, currentRound)

			if err != nil {
				log.Printf("WARN: Failed to get block %d: %v", 1, err)
				continue
			}

			// Persist metrics periodically
			if metrics.LastProcessedRound > 0 && metrics.LastProcessedRound%uint64(persistInterval) == 0 {
				if err := mongoStore.Save(ctx, metrics); err != nil {
					log.Printf("ERROR: Failed to persist metrics: %v", err)
				} else {
					log.Printf("Metrics successfully persisted at round %d", metrics.LastProcessedRound)
				}
			}

			log.Printf("INFO: Processing new block for round %d", block.Round)

		}
	}

}

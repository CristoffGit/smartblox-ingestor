package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"smartblox-ingestor/client"
	"smartblox-ingestor/config"
	"smartblox-ingestor/persistence"
	"smartblox-ingestor/processor"
	"smartblox-ingestor/types"
	"syscall"
	"time"
)

const (
	pollInterval    = 5 * time.Second
	persistInterval = 10
)

var mongoURI = config.GetEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")

func main() {
	// Initialize context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize api
	apiClient := client.NewSmartBloxClient()

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

	coreProcessor := processor.NewProcessor(mongoStore)

	metrics, err := mongoStore.Load(ctx)
	if err != nil {
		log.Fatalf("FATAL: could not load metrics from MongoDB: %v", err)
	}
	log.Printf("Service starting. Last processed round: %d", metrics.LastProcessedRound)

	// Go through missed blocks
	if err := backfill(ctx, apiClient, coreProcessor, metrics); err != nil {
		log.Printf("ERROR: Backfill process failed: %v", err)
		if err := mongoStore.Save(ctx, metrics); err != nil {
			log.Fatalf("FATAL: Failed to save metrics on exit: %v", err)
		}
		os.Exit(1)
	}

	run(ctx, apiClient, coreProcessor, mongoStore, metrics)

	log.Println("INFO: Shutdown signal received. Saving final state...")
	if err := mongoStore.Save(context.Background(), metrics); err != nil { // Use a background context for final save
		log.Fatalf("FATAL: Failed to save metrics on exit: %v", err)
	}
	log.Println("INFO: State saved. Exiting.")

}

// Catches up on any blocks missed since the last run
func backfill(ctx context.Context, c client.APIClient, p *processor.Processor, m *types.Metrics) error {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return err
	}

	latestRound := status.LastRound
	log.Printf("Current network round is %d. Starting backfill...", latestRound)

	for round := m.LastProcessedRound + 1; round <= latestRound; round++ {
		select {
		case <-ctx.Done():
			log.Println("Shutdown requested during backfill.")
			return ctx.Err()
		default:
			log.Printf("Backfilling round %d", round)
			block, err := c.GetBlock(ctx, round)
			if err != nil {
				log.Printf("WARN: Could not fetch block %d during backfill: %v. Skipping.", round, err)
				continue
			}
			// Pass context to the processor
			p.ProcessBlock(ctx, block, m)
		}
	}
	log.Println("Backfill complete.")
	return nil
}

// Run main loop to get and process new blocks
func run(ctx context.Context, c client.APIClient, p *processor.Processor, ms persistence.MetricsStore, m *types.Metrics) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentRound := m.LastProcessedRound + 1
			block, err := c.GetBlock(ctx, currentRound)

			if err != nil {
				if err == client.ErrBlockNotFound {
					continue
				}
				log.Printf("WARN: Failed to get block %d: %v", currentRound, err)
				continue
			}

			log.Printf("Processing new block for round %d", block.Round)
			p.ProcessBlock(ctx, block, m)

			// Persist metrics periodically
			if m.LastProcessedRound > 0 && m.LastProcessedRound%uint64(persistInterval) == 0 {
				if err := ms.Save(ctx, m); err != nil {
					log.Printf("ERROR: Failed to persist metrics: %v", err)
				} else {
					log.Printf("Metrics successfully persisted at round %d", m.LastProcessedRound)
				}
			}
		}
	}
}

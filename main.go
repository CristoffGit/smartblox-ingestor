package main

import (
	"context"
	"log"
	"os/signal"
	"smartblox-ingestor/client"
	"syscall"
	"time"
)

const pollInterval = 2 * time.Second

func main() {
	// Initialize context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize api
	apiClient := client.APIClient(client.NewSmartBloxClient())

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Iterate
	for i := range 10 {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			block, err := apiClient.GetBlock(ctx, uint64(i))

			if err != nil {
				log.Printf("WARN: Failed to get block %d: %v", 1, err)
				continue
			}

			log.Printf("INFO: Processing new block for round %d", block.Round)

		}
	}

}

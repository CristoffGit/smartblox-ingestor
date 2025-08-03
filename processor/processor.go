package processor

import (
	"context"
	"log"

	"smartblox-ingestor/config"
	"smartblox-ingestor/persistence"
	"smartblox-ingestor/types"
)

var txferType = config.GetEnvOrDefault("TX_TYPE", "txfer")

// Handle business logic of processing blockchain data
type Processor struct {
	txLogger persistence.TransactionLogger
}

// Create new processor
func NewProcessor(txLogger persistence.TransactionLogger) *Processor {
	return &Processor{txLogger: txLogger}
}

// Iterates through transactions in a block
func (p *Processor) ProcessBlock(ctx context.Context, block *types.Block, metrics *types.Metrics) {
	for _, tx := range block.Txs {
		// Ignore none txfer transactions
		if tx.Tx.Type != txferType {
			continue
		}

		// Write data to persistence layer
		if err := p.txLogger.Log(ctx, tx); err != nil {
			log.Printf("ERROR: Failed to log transaction %s: %v", tx.Sig, err)
			continue
		}

		// Update running metrics
		amount := tx.Tx.Amount
		metrics.TxnCount++
		metrics.TotalAmount += amount

		if metrics.TxnCount == 1 {
			// First run
			metrics.MinAmount = types.AmountRecord{Amount: amount, Round: block.Round}
			metrics.MaxAmount = types.AmountRecord{Amount: amount, Round: block.Round}
		} else {
			if amount < metrics.MinAmount.Amount {
				metrics.MinAmount = types.AmountRecord{Amount: amount, Round: block.Round}
			}
			if amount > metrics.MaxAmount.Amount {
				metrics.MaxAmount = types.AmountRecord{Amount: amount, Round: block.Round}
			}
		}
	}
	metrics.LastProcessedRound = block.Round
}

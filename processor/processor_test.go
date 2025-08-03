package processor

import (
	"context"
	"errors"
	"smartblox-ingestor/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementation of the TransactionLogger interface
type mockTxLogger struct {
	loggedTxs   []types.Transaction
	errToReturn error
}

// Log records the transaction that was passed in
func (m *mockTxLogger) Log(ctx context.Context, tx types.Transaction) error {
	if m.errToReturn != nil {
		return m.errToReturn
	}
	m.loggedTxs = append(m.loggedTxs, tx)
	return nil
}

//Run tests

func TestProcessBlock(t *testing.T) {
	// Set the transaction type we are testing against.
	// This mirrors the main package's behavior.
	txferType = "txfer"
	ctx := context.Background()

	t.Run("should process valid transactions and update metrics", func(t *testing.T) {
		// Arrange
		mockLogger := &mockTxLogger{}
		processor := NewProcessor(mockLogger)

		initialMetrics := &types.Metrics{}
		block := &types.Block{
			Round: 100,
			Txs: []types.Transaction{
				{Sig: "sig1", Tx: types.TxDetail{Type: "txfer", Amount: 100}},
				{Sig: "sig2", Tx: types.TxDetail{Type: "other", Amount: 50}}, // Should be ignored
				{Sig: "sig3", Tx: types.TxDetail{Type: "txfer", Amount: 25}},
				{Sig: "sig4", Tx: types.TxDetail{Type: "txfer", Amount: 200}},
			},
		}

		// Act
		processor.ProcessBlock(ctx, block, initialMetrics)

		// Assert
		// Check that the metrics were updated correctly.
		assert.Equal(t, uint64(100), initialMetrics.LastProcessedRound)
		assert.Equal(t, uint64(3), initialMetrics.TxnCount)
		assert.Equal(t, uint64(325), initialMetrics.TotalAmount) // 100 + 25 + 200

		// Check min/max amounts
		assert.Equal(t, uint64(25), initialMetrics.MinAmount.Amount)
		assert.Equal(t, uint64(100), initialMetrics.MinAmount.Round)
		assert.Equal(t, uint64(200), initialMetrics.MaxAmount.Amount)
		assert.Equal(t, uint64(100), initialMetrics.MaxAmount.Round)

		// Check that only the 'txfer' transactions were logged.
		require.Len(t, mockLogger.loggedTxs, 3)
		assert.Equal(t, "sig1", mockLogger.loggedTxs[0].Sig)
		assert.Equal(t, "sig3", mockLogger.loggedTxs[1].Sig)
		assert.Equal(t, "sig4", mockLogger.loggedTxs[2].Sig)
	})

	t.Run("should handle first transaction correctly for min/max", func(t *testing.T) {
		// Arrange
		mockLogger := &mockTxLogger{}
		processor := NewProcessor(mockLogger)

		// Start with empty metrics
		initialMetrics := &types.Metrics{}
		block := &types.Block{
			Round: 50,
			Txs: []types.Transaction{
				{Sig: "sig1", Tx: types.TxDetail{Type: "txfer", Amount: 500}},
			},
		}

		// Act
		processor.ProcessBlock(ctx, block, initialMetrics)

		// Assert
		// On the first transaction, min and max should be the same.
		assert.Equal(t, uint64(1), initialMetrics.TxnCount)
		assert.Equal(t, uint64(500), initialMetrics.MinAmount.Amount)
		assert.Equal(t, uint64(50), initialMetrics.MinAmount.Round)
		assert.Equal(t, uint64(500), initialMetrics.MaxAmount.Amount)
		assert.Equal(t, uint64(50), initialMetrics.MaxAmount.Round)
	})

	t.Run("should handle an empty block", func(t *testing.T) {
		// Arrange
		mockLogger := &mockTxLogger{}
		processor := NewProcessor(mockLogger)

		initialMetrics := &types.Metrics{TxnCount: 5, TotalAmount: 1000}
		block := &types.Block{
			Round: 200,
			Txs:   []types.Transaction{}, // No transactions
		}

		// Act
		processor.ProcessBlock(ctx, block, initialMetrics)

		// Assert
		// Metrics should be unchanged, except for the last processed round.
		assert.Equal(t, uint64(200), initialMetrics.LastProcessedRound)
		assert.Equal(t, uint64(5), initialMetrics.TxnCount)
		assert.Equal(t, uint64(1000), initialMetrics.TotalAmount)
		assert.Empty(t, mockLogger.loggedTxs, "Logger should not be called for an empty block")
	})

	t.Run("should skip updating metrics if logging fails", func(t *testing.T) {
		// Arrange
		mockLogger := &mockTxLogger{
			errToReturn: errors.New("database is down"),
		}
		processor := NewProcessor(mockLogger)

		initialMetrics := &types.Metrics{}
		block := &types.Block{
			Round: 300,
			Txs: []types.Transaction{
				{Sig: "sig1", Tx: types.TxDetail{Type: "txfer", Amount: 100}},
			},
		}

		// Act
		processor.ProcessBlock(ctx, block, initialMetrics)

		// Assert
		// If logging fails, the transaction should be skipped entirely.
		assert.Equal(t, uint64(300), initialMetrics.LastProcessedRound)
		assert.Equal(t, uint64(0), initialMetrics.TxnCount, "TxnCount should not be incremented on failure")
		assert.Equal(t, uint64(0), initialMetrics.TotalAmount, "TotalAmount should not be updated on failure")
		assert.Empty(t, mockLogger.loggedTxs, "No transactions should be successfully logged")
	})
}

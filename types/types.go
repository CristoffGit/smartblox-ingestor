package types

// Represents the response from /api/status
type Status struct {
	LastRound uint64 `json:"last-round"`
}

// Represents the response from /api/block/{round}
type Block struct {
	Round uint64        `json:"round"`
	Txs   []Transaction `json:"transactions"`
}

// Contains the core information about a transaction
type TxDetail struct {
	Type      string `json:"type"`
	Sender    uint64 `json:"sender"`
	Recipient uint64 `json:"receipient"`
	Amount    uint64 `json:"amount"`
}

// Represents a transaction
type Transaction struct {
	Sig string   `json:"sig"`
	Tx  TxDetail `json:"tx"`
}

// Store value of a round
type AmountRecord struct {
	Amount uint64 `json:"amount"`
	Round  uint64 `json:"round"`
}

// Holding metrics
type Metrics struct {
	LastProcessedRound uint64 `json:"last_processed_round`
	TxnCount           uint64 `json:"txn_count"`
	TotalAmount        uint64 `json:"total_amount"`
	MinAmount          uint64 `json:"min_amount"`
	MaxAmount          uint64 `json:"max_amount"`
}

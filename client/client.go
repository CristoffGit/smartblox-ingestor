package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"smartblox-ingestor/types"

	"github.com/Metrika-Inc/blckmock"
)

var ErrBlockNotFound = errors.New("block not found")

// Communicating with the SmartBlox node API
type APIClient interface {
	GetStatus(ctx context.Context) (*types.Status, error)
	GetBlock(ctx context.Context, round uint64) (*types.Block, error)
}

type smartBloxClient struct{}

// Create a new client for the API
func NewSmartBloxClient() APIClient {
	return &smartBloxClient{}
}

// Fetches the latest round (status) from the API
func (s *smartBloxClient) GetStatus(ctx context.Context) (*types.Status, error) {
	res, err := blckmock.GetStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get status from /api/status: %w", err)
	}

	var status types.Status
	if err := json.Unmarshal(res, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status response: %w", err)
	}

	return &status, nil
}

// Fetches a block from the API
func (s *smartBloxClient) GetBlock(ctx context.Context, round uint64) (*types.Block, error) {
	res, err := blckmock.GetBlock(int64(round))
	if err != nil {
		if err.Error() == "not found" {
			return nil, ErrBlockNotFound
		}
		return nil, fmt.Errorf("failed to get block from /api/blocks/%d: %w", round, err)
	}

	var block types.Block
	if err := json.Unmarshal(res, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to unmarshal block response: %w", err)
	}

	return &block, nil

}

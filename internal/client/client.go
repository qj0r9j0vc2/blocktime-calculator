package client

import (
	"context"
	"fmt"
	"time"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/qj0r9j0vc2/blocktime-calculator/pkg/types"
)

// BlockchainClient interface for blockchain interactions
type BlockchainClient interface {
	GetLatestBlockHeight(ctx context.Context) (int64, error)
	GetBlockByHeight(ctx context.Context, height int64) (*types.BlockInfo, error)
	GetBlockRange(ctx context.Context, startHeight, endHeight int64) ([]*types.BlockInfo, error)
	Close() error
}

// CosmosSDKClient implements BlockchainClient for Cosmos SDK blockchain
type CosmosSDKClient struct {
	config *types.ChainConfig
	client *rpchttp.HTTP
}

// NewCosmosSDKClient creates a new Cosmos SDK blockchain client
func NewCosmosSDKClient(config *types.ChainConfig) (*CosmosSDKClient, error) {
	if config.RPCEndpoint == "" {
		return nil, fmt.Errorf("RPC endpoint is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	// Create CometBFT HTTP client
	client, err := rpchttp.New(config.RPCEndpoint, "/websocket")
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}

	return &CosmosSDKClient{
		config: config,
		client: client,
	}, nil
}

// GetLatestBlockHeight gets the latest block height
func (c *CosmosSDKClient) GetLatestBlockHeight(ctx context.Context) (int64, error) {
	status, err := c.client.Status(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get status: %w", err)
	}

	return status.SyncInfo.LatestBlockHeight, nil
}

// GetBlockByHeight gets block information by height
func (c *CosmosSDKClient) GetBlockByHeight(ctx context.Context, height int64) (*types.BlockInfo, error) {
	blockResult, err := c.client.Block(ctx, &height)
	if err != nil {
		return nil, fmt.Errorf("failed to get block at height %d: %w", height, err)
	}

	if blockResult == nil || blockResult.Block == nil {
		return nil, fmt.Errorf("nil block result at height %d", height)
	}

	return &types.BlockInfo{
		Height:   blockResult.Block.Height,
		Time:     blockResult.Block.Time,
		Hash:     blockResult.Block.LastBlockID.Hash.String(),
		Proposer: blockResult.Block.ProposerAddress.String(),
		TxCount:  len(blockResult.Block.Txs),
	}, nil
}

// GetBlockRange gets a range of blocks
func (c *CosmosSDKClient) GetBlockRange(ctx context.Context, startHeight, endHeight int64) ([]*types.BlockInfo, error) {
	if startHeight > endHeight {
		return nil, fmt.Errorf("invalid range: start height %d > end height %d", startHeight, endHeight)
	}

	blocks := make([]*types.BlockInfo, 0, endHeight-startHeight+1)

	// Fetch blocks in parallel with controlled concurrency
	const maxConcurrent = 10
	semaphore := make(chan struct{}, maxConcurrent)
	blockChan := make(chan *tmtypes.Block, endHeight-startHeight+1)
	errChan := make(chan error, 1)

	// Fetch all blocks first
	for height := startHeight; height <= endHeight; height++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		semaphore <- struct{}{}
		go func(h int64) {
			defer func() { <-semaphore }()

			blockResult, err := c.client.Block(ctx, &h)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("failed to get block %d: %w", h, err):
				default:
				}
				return
			}

			if blockResult != nil && blockResult.Block != nil {
				blockChan <- blockResult.Block
			}
		}(height)
	}

	// Wait for all goroutines to complete
	for i := 0; i < maxConcurrent; i++ {
		semaphore <- struct{}{}
	}

	close(blockChan)
	close(errChan)

	// Check for errors
	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	default:
	}

	// Collect and convert blocks
	tmBlocks := make([]*tmtypes.Block, 0)
	for block := range blockChan {
		tmBlocks = append(tmBlocks, block)
	}

	// Sort blocks by height
	for i := 0; i < len(tmBlocks)-1; i++ {
		for j := i + 1; j < len(tmBlocks); j++ {
			if tmBlocks[i].Height > tmBlocks[j].Height {
				tmBlocks[i], tmBlocks[j] = tmBlocks[j], tmBlocks[i]
			}
		}
	}

	// Convert to BlockInfo and calculate block times
	for i, block := range tmBlocks {
		blockInfo := &types.BlockInfo{
			Height:   block.Height,
			Time:     block.Time,
			Hash:     block.LastBlockID.Hash.String(),
			Proposer: block.ProposerAddress.String(),
			TxCount:  len(block.Txs),
		}

		// Calculate block time if not the first block
		if i > 0 {
			blockInfo.BlockTime = block.Time.Sub(tmBlocks[i-1].Time).Seconds()
		}

		blocks = append(blocks, blockInfo)
	}

	return blocks, nil
}

// Close closes the client
func (c *CosmosSDKClient) Close() error {
	if c.client != nil {
		return c.client.Stop()
	}
	return nil
}

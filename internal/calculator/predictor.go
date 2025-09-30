package calculator

import (
	"context"
	"fmt"
	"time"

	"github.com/qj0r9j0vc2/blocktime-calculator/internal/client"
	"github.com/qj0r9j0vc2/blocktime-calculator/pkg/types"
)

// BlockPredictor predicts when a target block will be created
type BlockPredictor struct {
	client     client.BlockchainClient
	calculator *BlockTimeCalculator
}

// NewBlockPredictor creates a new block predictor
func NewBlockPredictor(client client.BlockchainClient, calculator *BlockTimeCalculator) *BlockPredictor {
	return &BlockPredictor{
		client:     client,
		calculator: calculator,
	}
}

// PredictBlockTime predicts when a target block will be created
func (p *BlockPredictor) PredictBlockTime(ctx context.Context, targetHeight int64) (*BlockPrediction, error) {
	// Get current block height
	currentHeight, err := p.client.GetLatestBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current height: %w", err)
	}

	// If target is in the past or current
	if targetHeight <= currentHeight {
		// Get the actual block
		block, err := p.client.GetBlockByHeight(ctx, targetHeight)
		if err != nil {
			return nil, fmt.Errorf("failed to get block %d: %w", targetHeight, err)
		}
		return &BlockPrediction{
			TargetHeight:  targetHeight,
			CurrentHeight: currentHeight,
			BlocksLeft:    0,
			IsComplete:    true,
			ActualTime:    &block.Time,
		}, nil
	}

	// Calculate block time statistics
	stats, err := p.calculator.CalculateStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate block time stats: %w", err)
	}

	// Calculate blocks left
	blocksLeft := targetHeight - currentHeight

	// Get current block to use as reference time
	currentBlock, err := p.client.GetBlockByHeight(ctx, currentHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}

	// Calculate time predictions using different estimates
	now := time.Now()
	blockAge := now.Sub(currentBlock.Time)

	// Typical (median) prediction
	typicalSeconds := float64(blocksLeft) * stats.EstimatedRange.Typical
	typicalTime := now.Add(time.Duration(typicalSeconds * float64(time.Second)))

	// Optimistic (lower bound) prediction
	optimisticSeconds := float64(blocksLeft) * stats.EstimatedRange.Lower
	optimisticTime := now.Add(time.Duration(optimisticSeconds * float64(time.Second)))

	// Pessimistic (upper bound) prediction
	pessimisticSeconds := float64(blocksLeft) * stats.EstimatedRange.Upper
	pessimisticTime := now.Add(time.Duration(pessimisticSeconds * float64(time.Second)))

	// Calculate time ranges
	typicalDuration := time.Duration(typicalSeconds * float64(time.Second))
	minDuration := time.Duration(optimisticSeconds * float64(time.Second))
	maxDuration := time.Duration(pessimisticSeconds * float64(time.Second))

	return &BlockPrediction{
		TargetHeight:    targetHeight,
		CurrentHeight:   currentHeight,
		BlocksLeft:      blocksLeft,
		CurrentTime:     now,
		CurrentBlockAge: blockAge,
		EstimatedTime:   typicalTime,
		OptimisticTime:  optimisticTime,
		PessimisticTime: pessimisticTime,
		Duration: DurationEstimate{
			Typical: typicalDuration,
			Min:     minDuration,
			Max:     maxDuration,
		},
		BlockTimeStats:  stats,
		ConfidenceLevel: stats.ConfidenceLevel,
		IsComplete:      false,
	}, nil
}

// PredictNextBlocks predicts when the next N blocks will be created
func (p *BlockPredictor) PredictNextBlocks(ctx context.Context, count int) (*MultiBlockPrediction, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	// Get current height
	currentHeight, err := p.client.GetLatestBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current height: %w", err)
	}

	// Calculate stats
	stats, err := p.calculator.CalculateStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate stats: %w", err)
	}

	// Get current block
	currentBlock, err := p.client.GetBlockByHeight(ctx, currentHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}

	now := time.Now()
	blockAge := now.Sub(currentBlock.Time)

	predictions := make([]BlockMilestone, count)

	for i := 0; i < count; i++ {
		blocksAhead := int64(i + 1)
		height := currentHeight + blocksAhead

		// Calculate time for this block
		typicalSeconds := float64(blocksAhead) * stats.EstimatedRange.Typical
		estimatedTime := now.Add(time.Duration(typicalSeconds * float64(time.Second)))

		predictions[i] = BlockMilestone{
			Height:        height,
			BlocksFromNow: blocksAhead,
			EstimatedTime: estimatedTime,
			Duration:      time.Duration(typicalSeconds * float64(time.Second)),
		}
	}

	return &MultiBlockPrediction{
		CurrentHeight:   currentHeight,
		CurrentTime:     now,
		CurrentBlockAge: blockAge,
		Predictions:     predictions,
		BlockTimeStats:  stats,
	}, nil
}

// BlockPrediction represents a prediction for when a block will be created
type BlockPrediction struct {
	TargetHeight    int64                 `json:"target_height"`
	CurrentHeight   int64                 `json:"current_height"`
	BlocksLeft      int64                 `json:"blocks_left"`
	CurrentTime     time.Time             `json:"current_time"`
	CurrentBlockAge time.Duration         `json:"current_block_age"`
	EstimatedTime   time.Time             `json:"estimated_time"`
	OptimisticTime  time.Time             `json:"optimistic_time"`
	PessimisticTime time.Time             `json:"pessimistic_time"`
	Duration        DurationEstimate      `json:"duration"`
	BlockTimeStats  *types.BlockTimeStats `json:"block_time_stats,omitempty"`
	ConfidenceLevel float64               `json:"confidence_level"`
	IsComplete      bool                  `json:"is_complete"`
	ActualTime      *time.Time            `json:"actual_time,omitempty"`
}

// DurationEstimate represents estimated duration ranges
type DurationEstimate struct {
	Typical time.Duration `json:"typical"`
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
}

// BlockMilestone represents a future block milestone
type BlockMilestone struct {
	Height        int64         `json:"height"`
	BlocksFromNow int64         `json:"blocks_from_now"`
	EstimatedTime time.Time     `json:"estimated_time"`
	Duration      time.Duration `json:"duration"`
}

// MultiBlockPrediction represents predictions for multiple blocks
type MultiBlockPrediction struct {
	CurrentHeight   int64                 `json:"current_height"`
	CurrentTime     time.Time             `json:"current_time"`
	CurrentBlockAge time.Duration         `json:"current_block_age"`
	Predictions     []BlockMilestone      `json:"predictions"`
	BlockTimeStats  *types.BlockTimeStats `json:"block_time_stats,omitempty"`
}

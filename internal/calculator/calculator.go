package calculator

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/qj0r9j0vc2/blocktime-calculator/internal/client"
	"github.com/qj0r9j0vc2/blocktime-calculator/pkg/types"
)

// BlockTimeCalculator calculates block time statistics
type BlockTimeCalculator struct {
	client client.BlockchainClient
	config *types.CalculatorConfig
}

// NewBlockTimeCalculator creates a new calculator instance
func NewBlockTimeCalculator(client client.BlockchainClient, config *types.CalculatorConfig) (*BlockTimeCalculator, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.SampleSize <= 0 {
		config.SampleSize = 100
	}

	if config.MinSampleSize <= 0 {
		config.MinSampleSize = 30
	}

	if config.OutlierThreshold <= 0 {
		config.OutlierThreshold = 1.5 // 1.5 * IQR for outlier detection
	}

	if config.ConfidenceLevel <= 0 || config.ConfidenceLevel >= 1 {
		config.ConfidenceLevel = 0.95
	}

	if config.TrimPercent < 0 || config.TrimPercent >= 0.5 {
		config.TrimPercent = 0.05 // Trim 5% from each end
	}

	return &BlockTimeCalculator{
		client: client,
		config: config,
	}, nil
}

// DefaultConfig returns default calculator configuration
func DefaultConfig() *types.CalculatorConfig {
	return &types.CalculatorConfig{
		SampleSize:        100,
		OutlierThreshold:  1.5,
		ConfidenceLevel:   0.95,
		MinSampleSize:     30,
		TrimPercent:       0.05,
		UseMedianAbsolute: true,
	}
}

// CalculateStats calculates block time statistics for the latest blocks
func (c *BlockTimeCalculator) CalculateStats(ctx context.Context) (*types.BlockTimeStats, error) {
	// Get latest block height
	latestHeight, err := c.client.GetLatestBlockHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest height: %w", err)
	}

	// Calculate start height
	startHeight := latestHeight - int64(c.config.SampleSize) + 1
	if startHeight < 1 {
		startHeight = 1
	}

	return c.CalculateStatsForRange(ctx, startHeight, latestHeight)
}

// CalculateStatsForRange calculates block time statistics for a specific range
func (c *BlockTimeCalculator) CalculateStatsForRange(ctx context.Context, startHeight, endHeight int64) (*types.BlockTimeStats, error) {
	// Validate range
	if startHeight > endHeight {
		return nil, fmt.Errorf("invalid range: start %d > end %d", startHeight, endHeight)
	}

	sampleSize := int(endHeight - startHeight + 1)
	if sampleSize < c.config.MinSampleSize {
		return nil, fmt.Errorf("insufficient sample size: %d < minimum %d", sampleSize, c.config.MinSampleSize)
	}

	// Fetch blocks
	blocks, err := c.client.GetBlockRange(ctx, startHeight, endHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get block range: %w", err)
	}

	// Calculate block times
	blockTimes := make([]float64, 0, len(blocks)-1)
	for i := 1; i < len(blocks); i++ {
		timeDiff := blocks[i].Time.Sub(blocks[i-1].Time).Seconds()
		if timeDiff > 0 { // Filter out negative or zero times
			blockTimes = append(blockTimes, timeDiff)
		}
	}

	if len(blockTimes) < c.config.MinSampleSize {
		return nil, fmt.Errorf("insufficient valid block times: %d < minimum %d", len(blockTimes), c.config.MinSampleSize)
	}

	// Remove outliers
	cleanedTimes, outlierCount := c.removeOutliers(blockTimes)

	// Calculate statistics
	stats := c.calculateStatistics(cleanedTimes)

	// Fill in additional information
	stats.SampleSize = len(blockTimes)
	stats.StartHeight = startHeight
	stats.EndHeight = endHeight
	stats.StartTime = blocks[0].Time
	stats.EndTime = blocks[len(blocks)-1].Time
	stats.OutlierCount = outlierCount
	stats.ConfidenceLevel = c.config.ConfidenceLevel

	// Calculate estimated range
	stats.EstimatedRange = c.calculateRange(cleanedTimes, stats)

	return stats, nil
}

// removeOutliers removes outliers using IQR method or MAD method
func (c *BlockTimeCalculator) removeOutliers(times []float64) ([]float64, int) {
	if len(times) <= 3 {
		return times, 0
	}

	// Sort times
	sorted := make([]float64, len(times))
	copy(sorted, times)
	sort.Float64s(sorted)

	var cleaned []float64
	var outlierCount int

	if c.config.UseMedianAbsolute {
		// Use Median Absolute Deviation (MAD) method
		median := percentile(sorted, 0.5)

		// Calculate MAD
		deviations := make([]float64, len(sorted))
		for i, v := range sorted {
			deviations[i] = math.Abs(v - median)
		}
		sort.Float64s(deviations)
		mad := percentile(deviations, 0.5)

		// Modified Z-score threshold (usually 3.5 for MAD)
		threshold := 3.5
		if mad == 0 {
			// If MAD is 0, use IQR method as fallback
			return c.removeOutliersIQR(sorted)
		}

		for _, v := range times {
			modifiedZScore := 0.6745 * (v - median) / mad
			if math.Abs(modifiedZScore) <= threshold {
				cleaned = append(cleaned, v)
			} else {
				outlierCount++
			}
		}
	} else {
		// Use IQR method
		return c.removeOutliersIQR(sorted)
	}

	// Apply trimming if configured
	if c.config.TrimPercent > 0 && len(cleaned) > 10 {
		trimCount := int(float64(len(cleaned)) * c.config.TrimPercent)
		if trimCount > 0 {
			sort.Float64s(cleaned)
			cleaned = cleaned[trimCount : len(cleaned)-trimCount]
			outlierCount += trimCount * 2
		}
	}

	return cleaned, outlierCount
}

// removeOutliersIQR removes outliers using Interquartile Range method
func (c *BlockTimeCalculator) removeOutliersIQR(sorted []float64) ([]float64, int) {
	q1 := percentile(sorted, 0.25)
	q3 := percentile(sorted, 0.75)
	iqr := q3 - q1

	lowerBound := q1 - c.config.OutlierThreshold*iqr
	upperBound := q3 + c.config.OutlierThreshold*iqr

	cleaned := make([]float64, 0, len(sorted))
	outlierCount := 0

	for _, v := range sorted {
		if v >= lowerBound && v <= upperBound {
			cleaned = append(cleaned, v)
		} else {
			outlierCount++
		}
	}

	return cleaned, outlierCount
}

// calculateStatistics calculates basic statistics
func (c *BlockTimeCalculator) calculateStatistics(times []float64) *types.BlockTimeStats {
	if len(times) == 0 {
		return &types.BlockTimeStats{}
	}

	sorted := make([]float64, len(times))
	copy(sorted, times)
	sort.Float64s(sorted)

	stats := &types.BlockTimeStats{
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		Median: percentile(sorted, 0.5),
		P25:    percentile(sorted, 0.25),
		P75:    percentile(sorted, 0.75),
		P95:    percentile(sorted, 0.95),
		P99:    percentile(sorted, 0.99),
	}

	// Calculate mean
	sum := 0.0
	for _, v := range times {
		sum += v
	}
	stats.Mean = sum / float64(len(times))

	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, v := range times {
		diff := v - stats.Mean
		sumSquaredDiff += diff * diff
	}
	stats.StdDev = math.Sqrt(sumSquaredDiff / float64(len(times)))

	return stats
}

// calculateRange calculates the estimated range for block times
func (c *BlockTimeCalculator) calculateRange(times []float64, stats *types.BlockTimeStats) types.Range {
	if len(times) == 0 {
		return types.Range{}
	}

	// Use robust statistics for range estimation
	// Lower bound: max(P25 - 0.5*IQR, Min)
	// Upper bound: min(P75 + 0.5*IQR, P95)
	iqr := stats.P75 - stats.P25

	lower := math.Max(stats.P25-0.5*iqr, stats.Min)
	upper := math.Min(stats.P75+0.5*iqr, stats.P95)

	// Typical value is the median for robustness
	typical := stats.Median

	// Adjust based on confidence level
	if c.config.ConfidenceLevel < 0.95 {
		// Narrow the range for lower confidence
		factor := c.config.ConfidenceLevel / 0.95
		rangeWidth := upper - lower
		adjustment := rangeWidth * (1 - factor) / 2
		lower += adjustment
		upper -= adjustment
	}

	return types.Range{
		Lower:   math.Max(lower, 0), // Block time can't be negative
		Upper:   upper,
		Typical: typical,
	}
}

// percentile calculates the percentile value
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}

	index := p * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// AnalyzeProposerPatterns analyzes block proposer patterns
func (c *BlockTimeCalculator) AnalyzeProposerPatterns(ctx context.Context, blocks []*types.BlockInfo) map[string]*types.BlockTimeStats {
	proposerBlocks := make(map[string][]float64)

	// Group block times by proposer
	for i := 1; i < len(blocks); i++ {
		if blocks[i].BlockTime > 0 {
			proposer := blocks[i].Proposer
			proposerBlocks[proposer] = append(proposerBlocks[proposer], blocks[i].BlockTime)
		}
	}

	// Calculate statistics per proposer
	proposerStats := make(map[string]*types.BlockTimeStats)
	for proposer, times := range proposerBlocks {
		if len(times) >= 5 { // Need at least 5 blocks for meaningful stats
			cleaned, outlierCount := c.removeOutliers(times)
			stats := c.calculateStatistics(cleaned)
			stats.OutlierCount = outlierCount
			stats.SampleSize = len(times)
			proposerStats[proposer] = stats
		}
	}

	return proposerStats
}

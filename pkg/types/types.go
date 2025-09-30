package types

import (
	"time"
)

// BlockInfo represents block information
type BlockInfo struct {
	Height    int64     `json:"height"`
	Time      time.Time `json:"time"`
	Hash      string    `json:"hash"`
	Proposer  string    `json:"proposer"`
	TxCount   int       `json:"tx_count"`
	BlockTime float64   `json:"block_time"` // seconds between this and previous block
}

// BlockTimeStats represents statistical analysis of block times
type BlockTimeStats struct {
	SampleSize       int       `json:"sample_size"`
	StartHeight      int64     `json:"start_height"`
	EndHeight        int64     `json:"end_height"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	Mean             float64   `json:"mean"`
	Median           float64   `json:"median"`
	StdDev           float64   `json:"std_dev"`
	Min              float64   `json:"min"`
	Max              float64   `json:"max"`
	P25              float64   `json:"p25"`  // 25th percentile
	P75              float64   `json:"p75"`  // 75th percentile
	P95              float64   `json:"p95"`  // 95th percentile
	P99              float64   `json:"p99"`  // 99th percentile
	OutlierCount     int       `json:"outlier_count"`
	EstimatedRange   Range     `json:"estimated_range"`
	ConfidenceLevel  float64   `json:"confidence_level"`
}

// Range represents an estimated range for block times
type Range struct {
	Lower  float64 `json:"lower"`
	Upper  float64 `json:"upper"`
	Typical float64 `json:"typical"` // Most common block time
}

// ChainConfig represents blockchain connection configuration
type ChainConfig struct {
	RPCEndpoint    string        `json:"rpc_endpoint" mapstructure:"rpc_endpoint"`
	GRPCEndpoint   string        `json:"grpc_endpoint" mapstructure:"grpc_endpoint"`
	ChainID        string        `json:"chain_id" mapstructure:"chain_id"`
	Timeout        time.Duration `json:"timeout" mapstructure:"timeout"`
	MaxRetries     int           `json:"max_retries" mapstructure:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay" mapstructure:"retry_delay"`
}

// CalculatorConfig represents calculator configuration
type CalculatorConfig struct {
	SampleSize          int     `json:"sample_size" mapstructure:"sample_size"`                     // Number of blocks to analyze
	OutlierThreshold    float64 `json:"outlier_threshold" mapstructure:"outlier_threshold"`         // IQR multiplier for outlier detection
	ConfidenceLevel     float64 `json:"confidence_level" mapstructure:"confidence_level"`           // Confidence level for range estimation (e.g., 0.95)
	MinSampleSize       int     `json:"min_sample_size" mapstructure:"min_sample_size"`             // Minimum blocks required for analysis
	TrimPercent         float64 `json:"trim_percent" mapstructure:"trim_percent"`                   // Percentage of extremes to trim (e.g., 0.05 for 5%)
	UseMedianAbsolute   bool    `json:"use_median_absolute" mapstructure:"use_median_absolute"`     // Use MAD instead of standard deviation
}
package config

import (
	"fmt"
	"time"

	"github.com/qj0r9j0vc2/blocktime-calculator/pkg/types"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Chain      types.ChainConfig      `json:"chain" mapstructure:"chain"`
	Calculator types.CalculatorConfig `json:"calculator" mapstructure:"calculator"`
	Output     OutputConfig           `json:"output" mapstructure:"output"`
}

// OutputConfig represents output formatting configuration
type OutputConfig struct {
	Format      string `json:"format" mapstructure:"format"`             // json, text, table
	Verbose     bool   `json:"verbose" mapstructure:"verbose"`           // Show extended statistics
	PrettyPrint bool   `json:"pretty_print" mapstructure:"pretty_print"` // Pretty print JSON
	SaveToFile  string `json:"save_to_file" mapstructure:"save_to_file"` // Save output to file
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Chain: types.ChainConfig{
			RPCEndpoint:  "http://localhost:26657",
			GRPCEndpoint: "localhost:9090",
			ChainID:      "cosmoshub-4",
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			RetryDelay:   time.Second,
		},
		Calculator: types.CalculatorConfig{
			SampleSize:        100,
			OutlierThreshold:  1.5,
			ConfidenceLevel:   0.95,
			MinSampleSize:     30,
			TrimPercent:       0.05,
			UseMedianAbsolute: true,
		},
		Output: OutputConfig{
			Format:      "text",
			Verbose:     false,
			PrettyPrint: true,
			SaveToFile:  "",
		},
	}
}

// BuildConfig builds configuration from viper settings
func BuildConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Load from config file first if exists - only unmarshal the nested structures
	if viper.IsSet("chain") {
		if err := viper.UnmarshalKey("chain", &cfg.Chain); err != nil {
			return nil, fmt.Errorf("failed to unmarshal chain config: %w", err)
		}
	}
	if viper.IsSet("calculator") {
		if err := viper.UnmarshalKey("calculator", &cfg.Calculator); err != nil {
			return nil, fmt.Errorf("failed to unmarshal calculator config: %w", err)
		}
	}
	// Only unmarshal output if it's a map structure (from config file)
	if viper.IsSet("output.format") || viper.IsSet("output.verbose") {
		if err := viper.UnmarshalKey("output", &cfg.Output); err != nil {
			// Ignore error - use defaults
			_ = err
		}
	}

	// Override with CLI flags and viper settings
	// Chain configuration
	if viper.IsSet("rpc") {
		cfg.Chain.RPCEndpoint = viper.GetString("rpc")
	}
	if viper.IsSet("grpc") {
		cfg.Chain.GRPCEndpoint = viper.GetString("grpc")
	}
	if viper.IsSet("chain-id") {
		cfg.Chain.ChainID = viper.GetString("chain-id")
	}
	if viper.IsSet("timeout") {
		cfg.Chain.Timeout = viper.GetDuration("timeout")
	}
	if viper.IsSet("max-retries") {
		cfg.Chain.MaxRetries = viper.GetInt("max-retries")
	}
	if viper.IsSet("retry-delay") {
		cfg.Chain.RetryDelay = viper.GetDuration("retry-delay")
	}

	// Calculator configuration
	if viper.IsSet("sample-size") {
		cfg.Calculator.SampleSize = viper.GetInt("sample-size")
	}
	if viper.IsSet("outlier-threshold") {
		cfg.Calculator.OutlierThreshold = viper.GetFloat64("outlier-threshold")
	}
	if viper.IsSet("confidence") {
		cfg.Calculator.ConfidenceLevel = viper.GetFloat64("confidence")
	}
	if viper.IsSet("min-sample-size") {
		cfg.Calculator.MinSampleSize = viper.GetInt("min-sample-size")
	}
	if viper.IsSet("trim-percent") {
		cfg.Calculator.TrimPercent = viper.GetFloat64("trim-percent")
	}
	if viper.IsSet("use-mad") {
		cfg.Calculator.UseMedianAbsolute = viper.GetBool("use-mad")
	}

	// Output configuration
	// Check for output format from CLI flag first, then from config file
	if viper.IsSet("output") && viper.GetString("output") != "" {
		cfg.Output.Format = viper.GetString("output")
	} else if viper.IsSet("output.format") {
		cfg.Output.Format = viper.GetString("output.format")
	}
	if viper.IsSet("verbose") {
		cfg.Output.Verbose = viper.GetBool("verbose")
	}
	if viper.IsSet("pretty-print") {
		cfg.Output.PrettyPrint = viper.GetBool("pretty-print")
	}
	if viper.IsSet("save-to-file") {
		cfg.Output.SaveToFile = viper.GetString("save-to-file")
	}

	// Validate configuration
	if err := ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	// Validate chain config
	if cfg.Chain.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if cfg.Chain.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative")
	}
	if cfg.Chain.RetryDelay < 0 {
		return fmt.Errorf("retry delay must be non-negative")
	}

	// Validate calculator config
	if cfg.Calculator.SampleSize <= 0 {
		return fmt.Errorf("sample size must be positive")
	}
	if cfg.Calculator.MinSampleSize <= 0 {
		return fmt.Errorf("min sample size must be positive")
	}
	if cfg.Calculator.MinSampleSize > cfg.Calculator.SampleSize {
		return fmt.Errorf("min sample size cannot be greater than sample size")
	}
	if cfg.Calculator.OutlierThreshold <= 0 {
		return fmt.Errorf("outlier threshold must be positive")
	}
	if cfg.Calculator.ConfidenceLevel <= 0 || cfg.Calculator.ConfidenceLevel >= 1 {
		return fmt.Errorf("confidence level must be between 0 and 1")
	}
	if cfg.Calculator.TrimPercent < 0 || cfg.Calculator.TrimPercent >= 0.5 {
		return fmt.Errorf("trim percent must be between 0 and 0.5")
	}

	// Validate output config
	validFormats := map[string]bool{
		"json":  true,
		"text":  true,
		"table": true,
	}
	if !validFormats[cfg.Output.Format] {
		return fmt.Errorf("invalid output format: %s (must be json, text, or table)", cfg.Output.Format)
	}

	return nil
}

// LoadFromFile loads configuration from a file
func LoadFromFile(path string) (*Config, error) {
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return BuildConfig()
}

// SaveToFile saves configuration to a file
func SaveToFile(cfg *Config, path string) error {
	viper.Set("chain", cfg.Chain)
	viper.Set("calculator", cfg.Calculator)
	viper.Set("output", cfg.Output)

	if err := viper.WriteConfigAs(path); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

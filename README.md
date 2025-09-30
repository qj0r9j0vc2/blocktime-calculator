# Blockchain Block Time Calculator

A sophisticated block time calculator for Cosmos SDK blockchain that analyzes block production patterns and provides statistical insights with outlier removal and confidence-based range estimation.

## Features

- **Advanced Statistical Analysis**: Calculates mean, median, standard deviation, and percentiles
- **Outlier Detection**: Two methods available:
  - IQR (Interquartile Range) method
  - MAD (Median Absolute Deviation) method for robust outlier detection
- **Range Estimation**: Provides confidence-based block time range predictions
- **Block Time Prediction**: Predicts when target blocks will be created
- **Proposer Analysis**: Analyzes block time patterns per validator/proposer
- **Flexible Configuration**: Supports both CLI flags and configuration files
- **Multiple Output Formats**: JSON, text, and table formats

## Installation

```bash
go get github.com/qj0r9j0vc2/blocktime-calculator
go build -o blocktime-calculator cmd/main.go
```

## Usage

### Basic Usage

Calculate block time statistics for the latest 100 blocks:

```bash
./blocktime-calculator calculate --rpc http://localhost:26657
```

### Specify Block Range

Analyze a specific range of blocks:

```bash
./blocktime-calculator calculate --rpc http://localhost:26657 --start-height 1000 --end-height 2000
```

### Analyze Proposer Patterns

Analyze block time patterns by proposer:

```bash
./blocktime-calculator analyze --rpc http://localhost:26657 --sample-size 500
```

### Predict Block Creation Time

Predict when a specific block will be created:

```bash
# Predict when block 1000000 will be created
./blocktime-calculator predict 1000000 --rpc http://localhost:26657

# Predict with specific height flag
./blocktime-calculator predict --height 1000000 --rpc http://localhost:26657

# Predict next 10 blocks
./blocktime-calculator predict --next 10 --rpc http://localhost:26657

# Verbose output with statistics
./blocktime-calculator predict 1000000 --verbose --rpc http://localhost:26657
```

### Configuration File

Generate a default configuration file:

```bash
./blocktime-calculator config config.yaml
```

Use with configuration file:

```bash
./blocktime-calculator calculate --config config.yaml
```

## Command Line Options

### Global Flags
- `--config`: Path to configuration file
- `--rpc`: RPC endpoint URL
- `--chain-id`: Chain ID (default: "cosmoshub-4")
- `--timeout`: Request timeout (default: 30s)

### Calculate Command Flags
- `--sample-size`: Number of blocks to analyze (default: 100)
- `--start-height`: Start height (0 for latest - sample-size)
- `--end-height`: End height (0 for latest)
- `--outlier-threshold`: IQR multiplier for outlier detection (default: 1.5)
- `--confidence`: Confidence level for range estimation (default: 0.95)
- `--trim-percent`: Percentage of extremes to trim (default: 0.05)
- `--use-mad`: Use Median Absolute Deviation for outlier detection (default: true)
- `--output`: Output format (json, text, table) (default: "json")
- `--verbose`: Show extended statistics

### Analyze Command Flags
- `--sample-size`: Number of blocks to analyze (default: 500)
- `--min-blocks`: Minimum blocks per proposer to include (default: 10)
- `--output`: Output format (json, text, table) (default: "table")

### Predict Command Flags
- `--height`: Target block height to predict
- `--next`: Predict next N blocks
- `--sample-size`: Number of blocks to analyze for statistics (default: 100)
- `--output`: Output format (json, text, table) (default: "text")
- `--verbose`: Show detailed statistics

## Configuration File Example

```yaml
chain:
  rpc_endpoint: "http://localhost:26657"
  grpc_endpoint: "localhost:9090"
  chain_id: "cosmoshub-4"
  timeout: 30s
  max_retries: 3
  retry_delay: 1s

calculator:
  sample_size: 100
  outlier_threshold: 1.5
  confidence_level: 0.95
  min_sample_size: 30
  trim_percent: 0.05
  use_median_absolute: true

output:
  format: "text"
  verbose: false
  pretty_print: true
```

## Output Examples

### Text Format

```
Block Time Statistics
=====================
Sample Size: 100 blocks
Height Range: 1900 - 2000
Time Range: 2024-01-01T10:00:00Z - 2024-01-01T10:10:00Z

Statistics (seconds):
  Mean: 6.12
  Median: 6.00
  Std Dev: 0.85
  Min: 5.02
  Max: 8.15

Estimated Block Time Range (95% confidence):
  Lower Bound: 5.50 seconds
  Upper Bound: 6.75 seconds
  Typical: 6.00 seconds
```

### Table Format

```
Metric               | Value
---------------------|----------------
Sample Size          | 100 blocks
Mean                 | 6.12 s
Median               | 6.00 s
Std Dev              | 0.85 s
Range                | 5.02 - 8.15 s
Outliers Removed     | 5
---------------------|----------------
Estimated Range      | 5.50 - 6.75 s
Typical Block Time   | 6.00 s
Confidence Level     | 95%
```

### JSON Format

```json
{
  "sample_size": 100,
  "start_height": 1900,
  "end_height": 2000,
  "mean": 6.12,
  "median": 6.00,
  "std_dev": 0.85,
  "min": 5.02,
  "max": 8.15,
  "p25": 5.75,
  "p75": 6.45,
  "p95": 7.20,
  "p99": 7.95,
  "outlier_count": 5,
  "estimated_range": {
    "lower": 5.50,
    "upper": 6.75,
    "typical": 6.00
  },
  "confidence_level": 0.95
}
```

### Block Prediction Output

```
Block Time Prediction
=====================
Target Block: 1000000
Current Block: 999500
Blocks Remaining: 500

Estimated Arrival Time:
  Typical: 2024-01-01T12:04:00Z (in 4m 10s)
  Optimistic: 2024-01-01T12:03:45Z (in 3m 55s)
  Pessimistic: 2024-01-01T12:04:15Z (in 4m 25s)
```

## Outlier Detection Methods

### IQR Method
Uses the Interquartile Range to identify outliers:
- Lower bound: Q1 - (threshold × IQR)
- Upper bound: Q3 + (threshold × IQR)
- Default threshold: 1.5

### MAD Method (Recommended)
Uses Median Absolute Deviation for robust outlier detection:
- More resistant to extreme outliers
- Better for non-normal distributions
- Automatically falls back to IQR if MAD is 0

## Architecture

The calculator uses a modular architecture:

- **Client Module**: Handles blockchain RPC communication with retry logic
- **Calculator Module**: Implements statistical analysis and outlier detection
- **Config Module**: Manages configuration and validation
- **CLI Module**: Provides command-line interface using Cobra

## Dependencies

- Cosmos SDK v0.50.10
- Cobra for CLI
- Viper for configuration management
- gRPC for blockchain communication

## License

Apache 2.0

## Contributing

Contributions are welcome! Please submit pull requests or open issues for any improvements or bug fixes.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/qj0r9j0vc2/blocktime-calculator/internal/calculator"
	"github.com/qj0r9j0vc2/blocktime-calculator/internal/client"
	"github.com/qj0r9j0vc2/blocktime-calculator/internal/config"
	"github.com/qj0r9j0vc2/blocktime-calculator/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "blocktime-calculator",
		Short: "Cosmos SDK blockchain block time calculator",
		Long: `A sophisticated block time calculator for Cosmos SDK blockchain
that analyzes block production patterns and provides statistical insights
with outlier removal and confidence-based range estimation.`,
		Version: "1.0.0",
	}

	calculateCmd = &cobra.Command{
		Use:   "calculate",
		Short: "Calculate block time statistics",
		Long:  `Calculate block time statistics for the latest blocks or a specific range`,
		RunE:  runCalculate,
	}

	analyzeCmd = &cobra.Command{
		Use:   "analyze",
		Short: "Analyze proposer patterns",
		Long:  `Analyze block time patterns per validator/proposer`,
		RunE:  runAnalyze,
	}

	predictCmd = &cobra.Command{
		Use:   "predict [target-height]",
		Short: "Predict when a target block will be created",
		Long:  `Predict when a target block will be created based on current block time statistics`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runPredict,
	}

	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Generate default configuration",
		Long:  `Generate a default configuration file`,
		RunE:  runConfig,
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().String("rpc", "", "RPC endpoint URL")
	rootCmd.PersistentFlags().String("chain-id", "cosmoshub-4", "Chain ID")
	rootCmd.PersistentFlags().Duration("timeout", 30*time.Second, "Request timeout")

	// Calculate command flags
	calculateCmd.Flags().Int("sample-size", 100, "Number of blocks to analyze")
	calculateCmd.Flags().Int64("start-height", 0, "Start height (0 for latest - sample-size)")
	calculateCmd.Flags().Int64("end-height", 0, "End height (0 for latest)")
	calculateCmd.Flags().Float64("outlier-threshold", 1.5, "IQR multiplier for outlier detection")
	calculateCmd.Flags().Float64("confidence", 0.95, "Confidence level for range estimation")
	calculateCmd.Flags().Float64("trim-percent", 0.05, "Percentage of extremes to trim")
	calculateCmd.Flags().Bool("use-mad", true, "Use Median Absolute Deviation for outlier detection")
	calculateCmd.Flags().String("output", "json", "Output format (json, text, table)")
	calculateCmd.Flags().Bool("verbose", false, "Verbose output")

	// Analyze command flags
	analyzeCmd.Flags().Int("sample-size", 500, "Number of blocks to analyze")
	analyzeCmd.Flags().Int("min-blocks", 10, "Minimum blocks per proposer to include")
	analyzeCmd.Flags().String("output", "table", "Output format (json, text, table)")

	// Predict command flags
	predictCmd.Flags().Int64("height", 0, "Target block height to predict")
	predictCmd.Flags().Int("next", 0, "Predict next N blocks")
	predictCmd.Flags().Int("sample-size", 100, "Number of blocks to analyze for statistics")
	predictCmd.Flags().String("output", "text", "Output format (json, text, table)")
	predictCmd.Flags().Bool("verbose", false, "Show detailed statistics")

	// Bind flags to viper
	viper.BindPFlags(rootCmd.PersistentFlags())
	viper.BindPFlags(calculateCmd.Flags())
	viper.BindPFlags(analyzeCmd.Flags())
	viper.BindPFlags(predictCmd.Flags())

	// Add commands
	rootCmd.AddCommand(calculateCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(configCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("json")
	}

	viper.SetEnvPrefix("BLOCKTIME")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// Config file found - silently use it
		_ = viper.ConfigFileUsed()
	}
}

func runCalculate(cmd *cobra.Command, args []string) error {
	// Build configuration
	cfg, err := config.BuildConfig()
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	// Validate RPC endpoint
	if cfg.Chain.RPCEndpoint == "" {
		return fmt.Errorf("RPC endpoint is required (use --rpc flag or config file)")
	}

	// Create client
	blockClient, err := client.NewCosmosSDKClient(&cfg.Chain)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer blockClient.Close()

	// Create calculator
	calc, err := calculator.NewBlockTimeCalculator(blockClient, &cfg.Calculator)
	if err != nil {
		return fmt.Errorf("failed to create calculator: %w", err)
	}

	// Get height range
	ctx := context.Background()
	startHeight := viper.GetInt64("start-height")
	endHeight := viper.GetInt64("end-height")

	var stats *types.BlockTimeStats
	if startHeight > 0 && endHeight > 0 {
		// Use specified range
		stats, err = calc.CalculateStatsForRange(ctx, startHeight, endHeight)
	} else {
		// Use latest blocks
		stats, err = calc.CalculateStats(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to calculate statistics: %w", err)
	}

	// Output results - check if flag was explicitly set
	outputFormat := cfg.Output.Format
	if cmd.Flags().Changed("output") {
		outputFormat, _ = cmd.Flags().GetString("output")
	}
	if outputFormat == "" {
		outputFormat = "json"
	}

	verbose := cfg.Output.Verbose
	if cmd.Flags().Changed("verbose") {
		verbose = viper.GetBool("verbose")
	}

	return outputStats(stats, outputFormat, verbose)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Build configuration
	cfg, err := config.BuildConfig()
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	// Validate RPC endpoint
	if cfg.Chain.RPCEndpoint == "" {
		return fmt.Errorf("RPC endpoint is required (use --rpc flag or config file)")
	}

	// Create client
	blockClient, err := client.NewCosmosSDKClient(&cfg.Chain)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer blockClient.Close()

	// Create calculator
	calc, err := calculator.NewBlockTimeCalculator(blockClient, &cfg.Calculator)
	if err != nil {
		return fmt.Errorf("failed to create calculator: %w", err)
	}

	// Get blocks
	ctx := context.Background()
	latestHeight, err := blockClient.GetLatestBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest height: %w", err)
	}

	sampleSize := viper.GetInt("sample-size")
	startHeight := latestHeight - int64(sampleSize) + 1
	if startHeight < 1 {
		startHeight = 1
	}

	blocks, err := blockClient.GetBlockRange(ctx, startHeight, latestHeight)
	if err != nil {
		return fmt.Errorf("failed to get blocks: %w", err)
	}

	// Analyze proposer patterns
	proposerStats := calc.AnalyzeProposerPatterns(ctx, blocks)

	// Output results
	outputFormat := viper.GetString("output")
	return outputProposerStats(proposerStats, outputFormat)
}

func runPredict(cmd *cobra.Command, args []string) error {
	// Build configuration
	cfg, err := config.BuildConfig()
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	// Validate RPC endpoint
	if cfg.Chain.RPCEndpoint == "" {
		return fmt.Errorf("RPC endpoint is required (use --rpc flag or config file)")
	}

	// Create client
	blockClient, err := client.NewCosmosSDKClient(&cfg.Chain)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer blockClient.Close()

	// Create calculator
	calc, err := calculator.NewBlockTimeCalculator(blockClient, &cfg.Calculator)
	if err != nil {
		return fmt.Errorf("failed to create calculator: %w", err)
	}

	// Create predictor
	predictor := calculator.NewBlockPredictor(blockClient, calc)

	ctx := context.Background()

	// Check what type of prediction to do
	nextBlocks := viper.GetInt("next")
	targetHeight := viper.GetInt64("height")

	// If target height provided as argument
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &targetHeight); err != nil {
			return fmt.Errorf("invalid target height: %s", args[0])
		}
	}

	outputFormat := cfg.Output.Format
	if cmd.Flags().Changed("output") {
		outputFormat, _ = cmd.Flags().GetString("output")
	}
	if outputFormat == "" {
		outputFormat = "text"
	}

	verbose := cfg.Output.Verbose
	if cmd.Flags().Changed("verbose") {
		verbose = viper.GetBool("verbose")
	}

	// Predict next N blocks
	if nextBlocks > 0 {
		prediction, err := predictor.PredictNextBlocks(ctx, nextBlocks)
		if err != nil {
			return fmt.Errorf("failed to predict next blocks: %w", err)
		}
		return outputMultiBlockPrediction(prediction, outputFormat, verbose)
	}

	// Predict specific target height
	if targetHeight > 0 {
		prediction, err := predictor.PredictBlockTime(ctx, targetHeight)
		if err != nil {
			return fmt.Errorf("failed to predict block time: %w", err)
		}
		return outputPrediction(prediction, outputFormat, verbose)
	}

	// Default: predict next 10 blocks
	prediction, err := predictor.PredictNextBlocks(ctx, 10)
	if err != nil {
		return fmt.Errorf("failed to predict next blocks: %w", err)
	}
	return outputMultiBlockPrediction(prediction, outputFormat, verbose)
}

func runConfig(cmd *cobra.Command, args []string) error {
	defaultConfig := config.DefaultConfig()

	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	filename := "config.json"
	if len(args) > 0 {
		filename = args[0]
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Default configuration written to %s\n", filename)
	return nil
}

func outputStats(stats *types.BlockTimeStats, format string, verbose bool) error {
	// Trim any whitespace from format
	format = strings.TrimSpace(format)

	switch format {
	case "json":
		data, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "text":
		fmt.Println("Block Time Statistics")
		fmt.Println("=====================")
		fmt.Printf("Sample Size: %d blocks\n", stats.SampleSize)
		fmt.Printf("Height Range: %d - %d\n", stats.StartHeight, stats.EndHeight)
		fmt.Printf("Time Range: %s - %s\n", stats.StartTime.Format(time.RFC3339), stats.EndTime.Format(time.RFC3339))
		fmt.Println("\nStatistics (seconds):")
		fmt.Printf("  Mean: %.2f\n", stats.Mean)
		fmt.Printf("  Median: %.2f\n", stats.Median)
		fmt.Printf("  Std Dev: %.2f\n", stats.StdDev)
		fmt.Printf("  Min: %.2f\n", stats.Min)
		fmt.Printf("  Max: %.2f\n", stats.Max)

		if verbose {
			fmt.Println("\nPercentiles:")
			fmt.Printf("  P25: %.2f\n", stats.P25)
			fmt.Printf("  P75: %.2f\n", stats.P75)
			fmt.Printf("  P95: %.2f\n", stats.P95)
			fmt.Printf("  P99: %.2f\n", stats.P99)
			fmt.Printf("\nOutliers Removed: %d\n", stats.OutlierCount)
		}

		fmt.Printf("\nEstimated Block Time Range (%.0f%% confidence):\n", stats.ConfidenceLevel*100)
		fmt.Printf("  Lower Bound: %.2f seconds\n", stats.EstimatedRange.Lower)
		fmt.Printf("  Upper Bound: %.2f seconds\n", stats.EstimatedRange.Upper)
		fmt.Printf("  Typical: %.2f seconds\n", stats.EstimatedRange.Typical)

	case "table":
		fmt.Printf("%-20s | %-15s\n", "Metric", "Value")
		fmt.Println("---------------------|----------------")
		fmt.Printf("%-20s | %d blocks\n", "Sample Size", stats.SampleSize)
		fmt.Printf("%-20s | %.2f s\n", "Mean", stats.Mean)
		fmt.Printf("%-20s | %.2f s\n", "Median", stats.Median)
		fmt.Printf("%-20s | %.2f s\n", "Std Dev", stats.StdDev)
		fmt.Printf("%-20s | %.2f - %.2f s\n", "Range", stats.Min, stats.Max)
		fmt.Printf("%-20s | %d\n", "Outliers Removed", stats.OutlierCount)
		fmt.Println("---------------------|----------------")
		fmt.Printf("%-20s | %.2f - %.2f s\n", "Estimated Range", stats.EstimatedRange.Lower, stats.EstimatedRange.Upper)
		fmt.Printf("%-20s | %.2f s\n", "Typical Block Time", stats.EstimatedRange.Typical)
		fmt.Printf("%-20s | %.0f%%\n", "Confidence Level", stats.ConfidenceLevel*100)

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func outputProposerStats(proposerStats map[string]*types.BlockTimeStats, format string) error {
	if len(proposerStats) == 0 {
		fmt.Println("No proposer statistics available")
		return nil
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(proposerStats, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "table", "text":
		fmt.Printf("%-40s | %-10s | %-10s | %-10s | %-10s\n", "Proposer", "Blocks", "Mean (s)", "Median (s)", "Std Dev")
		fmt.Println("------------------------------------------|------------|------------|------------|------------")

		for proposer, stats := range proposerStats {
			displayProposer := proposer
			if len(displayProposer) > 38 {
				displayProposer = displayProposer[:35] + "..."
			}
			fmt.Printf("%-40s | %10d | %10.2f | %10.2f | %10.2f\n",
				displayProposer, stats.SampleSize, stats.Mean, stats.Median, stats.StdDev)
		}

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func outputPrediction(pred *calculator.BlockPrediction, format string, verbose bool) error {
	format = strings.TrimSpace(format)

	switch format {
	case "json":
		data, err := json.MarshalIndent(pred, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "text":
		fmt.Println("Block Time Prediction")
		fmt.Println("=====================")

		if pred.IsComplete {
			fmt.Printf("Block %d already exists\n", pred.TargetHeight)
			fmt.Printf("Created at: %s\n", pred.ActualTime.Format(time.RFC3339))
			return nil
		}

		fmt.Printf("Target Block: %d\n", pred.TargetHeight)
		fmt.Printf("Current Block: %d\n", pred.CurrentHeight)
		fmt.Printf("Blocks Remaining: %d\n", pred.BlocksLeft)
		fmt.Printf("Current Time: %s\n", pred.CurrentTime.Format(time.RFC3339))

		fmt.Println("\nEstimated Arrival Time:")
		fmt.Printf("  Typical: %s (in %s)\n",
			pred.EstimatedTime.Format(time.RFC3339),
			formatDuration(pred.Duration.Typical))
		fmt.Printf("  Optimistic: %s (in %s)\n",
			pred.OptimisticTime.Format(time.RFC3339),
			formatDuration(pred.Duration.Min))
		fmt.Printf("  Pessimistic: %s (in %s)\n",
			pred.PessimisticTime.Format(time.RFC3339),
			formatDuration(pred.Duration.Max))

		if verbose && pred.BlockTimeStats != nil {
			fmt.Printf("\nBlock Time Statistics:\n")
			fmt.Printf("  Mean: %.2f seconds\n", pred.BlockTimeStats.Mean)
			fmt.Printf("  Median: %.2f seconds\n", pred.BlockTimeStats.Median)
			fmt.Printf("  Confidence: %.0f%%\n", pred.ConfidenceLevel*100)
		}

	case "table":
		fmt.Printf("%-20s | %-30s\n", "Metric", "Value")
		fmt.Println("---------------------|-------------------------------")
		fmt.Printf("%-20s | %d\n", "Target Block", pred.TargetHeight)
		fmt.Printf("%-20s | %d\n", "Current Block", pred.CurrentHeight)
		fmt.Printf("%-20s | %d\n", "Blocks Remaining", pred.BlocksLeft)
		fmt.Println("---------------------|-------------------------------")
		fmt.Printf("%-20s | %s\n", "Estimated Time", pred.EstimatedTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("%-20s | %s\n", "Time from Now", formatDuration(pred.Duration.Typical))
		fmt.Printf("%-20s | %s - %s\n", "Range",
			formatDuration(pred.Duration.Min),
			formatDuration(pred.Duration.Max))

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func outputMultiBlockPrediction(pred *calculator.MultiBlockPrediction, format string, verbose bool) error {
	format = strings.TrimSpace(format)

	switch format {
	case "json":
		data, err := json.MarshalIndent(pred, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "text":
		fmt.Println("Next Blocks Prediction")
		fmt.Println("======================")
		fmt.Printf("Current Block: %d\n", pred.CurrentHeight)
		fmt.Printf("Current Time: %s\n", pred.CurrentTime.Format(time.RFC3339))
		fmt.Println("\nUpcoming Blocks:")

		for _, p := range pred.Predictions {
			fmt.Printf("  Block %d: %s (in %s)\n",
				p.Height,
				p.EstimatedTime.Format("15:04:05"),
				formatDuration(p.Duration))
		}

		if verbose && pred.BlockTimeStats != nil {
			fmt.Printf("\nBased on block time: %.2fs (±%.2fs)\n",
				pred.BlockTimeStats.Median,
				pred.BlockTimeStats.StdDev)
		}

	case "table":
		fmt.Printf("%-10s | %-20s | %-15s\n", "Block", "Estimated Time", "Duration")
		fmt.Println("-----------|----------------------|----------------")

		for _, p := range pred.Predictions {
			fmt.Printf("%-10d | %-20s | %-15s\n",
				p.Height,
				p.EstimatedTime.Format("2006-01-02 15:04:05"),
				formatDuration(p.Duration))
		}

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", mins, secs)
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%d hours", hours)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/llm-d-incubation/llm-d-edge/pkg/config"
	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"github.com/llm-d-incubation/llm-d-edge/tools/compare-models/pkg/compare"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Import platform-specific engines (reused from edge-router)
	_ "github.com/llm-d-incubation/llm-d-edge/internal/macos"
	_ "github.com/llm-d-incubation/llm-d-edge/internal/stub"
)

var (
	messagesJSON string
	configPath   string
	remoteModel  string
	outputPath   string
	verbose      bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "compare-models",
		Short: "Compare inference performance across local and remote models",
		Long: `compare-models is a tool for benchmarking and comparing LLM inference performance.
It runs the same query against all local models (from edge-router config) and a specified
remote model, collecting comprehensive performance metrics.`,
		RunE: runComparison,
	}

	rootCmd.Flags().StringVarP(&messagesJSON, "messages", "m", "", "JSON array of messages (required)")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to edge-router config file (required)")
	rootCmd.Flags().StringVarP(&remoteModel, "remote-model", "r", "", "Remote model name (required)")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output JSON file (optional, defaults to stdout)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	rootCmd.MarkFlagRequired("messages")
	rootCmd.MarkFlagRequired("config")
	rootCmd.MarkFlagRequired("remote-model")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runComparison(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger, err := initLogger(verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("compare-models starting",
		zap.String("config", configPath),
		zap.String("remote_model", remoteModel),
	)

	// Parse messages JSON
	var messages []engine.Message
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return fmt.Errorf("failed to parse messages JSON: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("messages array cannot be empty")
	}

	logger.Info("parsed messages", zap.Int("count", len(messages)))

	// Load configuration (reused from edge-router)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger.Info("configuration loaded",
		zap.String("platform", cfg.Edge.Platform),
		zap.Int("local_models", len(cfg.Edge.Models.Local)),
		zap.String("remote_cluster", cfg.Edge.Models.Remote.ClusterURL),
	)

	// Validate that we have models to compare
	if len(cfg.Edge.Models.Local) == 0 {
		logger.Warn("no local models configured")
	}

	if cfg.Edge.Models.Remote.ClusterURL == "" {
		return fmt.Errorf("remote cluster URL not configured")
	}

	// Create inference request
	req := &engine.InferenceRequest{
		Messages:    messages,
		MaxTokens:   100, // Default, can be made configurable
		Temperature: 0.7, // Default, can be made configurable
	}

	// Create comparator
	comparator := compare.NewComparator(cfg, logger)

	// Run comparison
	ctx := context.Background()
	result, err := comparator.RunComparison(ctx, req, remoteModel)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}

	// Format output as JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	// Write output
	if outputPath != "" {
		if err := os.WriteFile(outputPath, output, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		logger.Info("results written to file", zap.String("path", outputPath))
	} else {
		fmt.Println(string(output))
	}

	// Print summary to stderr for visibility
	fmt.Fprintf(os.Stderr, "\n=== Comparison Summary ===\n")
	fmt.Fprintf(os.Stderr, "Total Models: %d\n", result.Summary.TotalModels)
	fmt.Fprintf(os.Stderr, "Successful: %d\n", result.Summary.Successful)
	fmt.Fprintf(os.Stderr, "Failed: %d\n", result.Summary.Failed)
	if result.Summary.Successful > 0 {
		fmt.Fprintf(os.Stderr, "Average Latency: %.2f ms\n", result.Summary.AverageLatencyMs)
		fmt.Fprintf(os.Stderr, "Fastest Model: %s\n", result.Summary.FastestModel)
		fmt.Fprintf(os.Stderr, "Slowest Model: %s\n", result.Summary.SlowestModel)
		fmt.Fprintf(os.Stderr, "Highest TPS: %s\n", result.Summary.HighestTPSModel)
	}
	fmt.Fprintf(os.Stderr, "=========================\n")

	return nil
}

// initLogger initializes the logger (reused pattern from edge-router)
func initLogger(verbose bool) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	if verbose {
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	config := zap.Config{
		Level:            zapLevel,
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stderr"}, // Log to stderr to keep stdout clean for JSON
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

// Made with Bob
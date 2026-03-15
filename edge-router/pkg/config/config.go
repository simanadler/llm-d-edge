package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

// Config represents the complete edge router configuration
type Config struct {
	Edge EdgeConfig `mapstructure:"edge" yaml:"edge"`
}

// EdgeConfig contains edge-specific configuration
type EdgeConfig struct {
	Platform         string                 `mapstructure:"platform" yaml:"platform"`
	Device           DeviceConfig           `mapstructure:"device" yaml:"device"`
	Routing          RoutingConfig          `mapstructure:"routing" yaml:"routing"`
	Models           ModelsConfig           `mapstructure:"models" yaml:"models"`
	RoutingRules     []RoutingRule          `mapstructure:"routing_rules" yaml:"routing_rules"`
	PlatformOverrides map[string]interface{} `mapstructure:"platform_overrides,omitempty" yaml:"platform_overrides,omitempty"`
}

// DeviceConfig contains device-specific configuration
type DeviceConfig struct {
	Type    string `mapstructure:"type" yaml:"type"`       // auto, apple-silicon-m4, nvidia-rtx-4090, etc.
	Memory  string `mapstructure:"memory" yaml:"memory"`   // auto, 16GB, 32GB, etc.
	Storage string `mapstructure:"storage" yaml:"storage"` // auto, 256GB, 512GB, etc.
}

// RoutingConfig contains routing policy configuration
type RoutingConfig struct {
	Policy   RoutingPolicy `mapstructure:"policy" yaml:"policy"`     // local-first, remote-first, hybrid, cost-optimized, mobile-optimized
	Fallback string        `mapstructure:"fallback" yaml:"fallback"` // remote, local, fail
}

// RoutingPolicy defines the routing strategy
type RoutingPolicy string

const (
	PolicyLocalFirst      RoutingPolicy = "local-first"
	PolicyRemoteFirst     RoutingPolicy = "remote-first"
	PolicyHybrid          RoutingPolicy = "hybrid"
	PolicyCostOptimized   RoutingPolicy = "cost-optimized"
	PolicyLatencyOptimized RoutingPolicy = "latency-optimized"
	PolicyMobileOptimized RoutingPolicy = "mobile-optimized"
)

// ModelsConfig contains model configuration
type ModelsConfig struct {
	Local  []LocalModelConfig  `mapstructure:"local" yaml:"local"`
	Remote RemoteClusterConfig `mapstructure:"remote" yaml:"remote"`
}

// LocalModelConfig defines a local model
type LocalModelConfig struct {
	Name         string `mapstructure:"name" yaml:"name"`
	Format       string `mapstructure:"format" yaml:"format"`             // auto, mlx, gguf, pytorch, coreml
	Quantization string `mapstructure:"quantization" yaml:"quantization"` // 4bit, 8bit, none
	Priority     int    `mapstructure:"priority" yaml:"priority"`         // Higher priority models loaded first
	Path         string `mapstructure:"path,omitempty" yaml:"path,omitempty"`
}

// RemoteClusterConfig defines remote cluster configuration
type RemoteClusterConfig struct {
	ClusterURL string            `mapstructure:"cluster_url" yaml:"cluster_url"`
	AuthToken  string            `mapstructure:"auth_token" yaml:"auth_token"`
	Timeout    int               `mapstructure:"timeout,omitempty" yaml:"timeout,omitempty"` // seconds
	Headers    map[string]string `mapstructure:"headers,omitempty" yaml:"headers"`           // Custom headers like RITS_API_KEY, etc
}

// RoutingRule defines a routing rule
type RoutingRule struct {
	Condition string `mapstructure:"condition" yaml:"condition"` // Expression to evaluate
	Action    string `mapstructure:"action" yaml:"action"`       // route_local, route_remote, route_local_or_fail
}

// LoadConfig loads configuration from file and environment
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config in standard locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath(getDefaultConfigDir())
	}

	// Read environment variables
	v.SetEnvPrefix("LLMD_EDGE")
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found, use defaults
	}

	var config Config
	if err := v.UnmarshalExact(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Auto-detect platform if set to "auto"
	if config.Edge.Platform == "auto" || config.Edge.Platform == "" {
		config.Edge.Platform = detectPlatform()
	}

	// Expand paths in model configurations
	if err := expandModelPaths(&config); err != nil {
		return nil, fmt.Errorf("failed to expand model paths: %w", err)
	}

	// Expand environment variables in headers
	expandTokenEnvVars(&config)

	// Apply platform-specific overrides
	if err := applyPlatformOverrides(&config); err != nil {
		return nil, fmt.Errorf("failed to apply platform overrides: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// expandModelPaths expands ~ and environment variables in model paths
func expandModelPaths(config *Config) error {
	for i := range config.Edge.Models.Local {
		if config.Edge.Models.Local[i].Path != "" {
			expandedPath, err := expandPath(config.Edge.Models.Local[i].Path)
			if err != nil {
				return fmt.Errorf("failed to expand path for model %s: %w", config.Edge.Models.Local[i].Name, err)
			}
			config.Edge.Models.Local[i].Path = expandedPath
		}
	}
	return nil
}

// expandPath expands ~ to home directory and cleans the path
func expandPath(path string) (string, error) {
	// Handle empty path
	if path == "" {
		return path, nil
	}

	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		
		if len(path) == 1 {
			path = home
		} else if path[1] == '/' || path[1] == filepath.Separator {
			path = filepath.Join(home, path[2:])
		} else {
			path = filepath.Join(home, path[1:])
		}
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	// Clean the path (removes redundant separators, resolves . and ..)
	path = filepath.Clean(path)

	return path, nil
}

// expandHeaderEnvVars expands environment variables in header values and auth token
func expandTokenEnvVars(config *Config) {
	// Expand environment variables in auth_token
	if config.Edge.Models.Remote.AuthToken != "" {
		config.Edge.Models.Remote.AuthToken = os.ExpandEnv(config.Edge.Models.Remote.AuthToken)
	}
	
	// Expand environment variables in headers
	for key, value := range config.Edge.Models.Remote.Headers {
		// Expand environment variables in the format ${VAR} or $VAR
		config.Edge.Models.Remote.Headers[key] = os.ExpandEnv(value)
	}
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("edge.platform", "auto")
	v.SetDefault("edge.device.type", "auto")
	v.SetDefault("edge.device.memory", "auto")
	v.SetDefault("edge.device.storage", "auto")
	v.SetDefault("edge.routing.policy", "hybrid")
	v.SetDefault("edge.routing.fallback", "remote")
	v.SetDefault("edge.models.remote.timeout", 60)
}

// detectPlatform detects the current platform
func detectPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	case "linux":
		// Could be Android or Linux desktop
		if isAndroid() {
			return "android"
		}
		return "linux"
	case "ios":
		return "ios"
	default:
		return runtime.GOOS
	}
}

// isAndroid checks if running on Android
func isAndroid() bool {
	// Simple heuristic: check for Android-specific paths
	_, err := os.Stat("/system/build.prop")
	return err == nil
}

// getDefaultConfigDir returns the default configuration directory
func getDefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "llm-d")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "llm-d")
		}
		return filepath.Join(home, "AppData", "Roaming", "llm-d")
	default:
		return filepath.Join(home, ".llm-d")
	}
}

// applyPlatformOverrides applies platform-specific configuration overrides
func applyPlatformOverrides(config *Config) error {
	platform := config.Edge.Platform
	overrides, ok := config.Edge.PlatformOverrides[platform]
	if !ok {
		return nil
	}

	// This is a simplified version - in production, you'd use a more sophisticated merge
	// For now, we'll just note that overrides exist
	_ = overrides

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate routing policy
	validPolicies := map[RoutingPolicy]bool{
		PolicyLocalFirst:       true,
		PolicyRemoteFirst:      true,
		PolicyHybrid:           true,
		PolicyCostOptimized:    true,
		PolicyLatencyOptimized: true,
		PolicyMobileOptimized:  true,
	}

	if !validPolicies[config.Edge.Routing.Policy] {
		return fmt.Errorf("invalid routing policy: %s", config.Edge.Routing.Policy)
	}

	// Validate fallback
	validFallbacks := map[string]bool{
		"remote": true,
		"local":  true,
		"fail":   true,
	}

	if !validFallbacks[config.Edge.Routing.Fallback] {
		return fmt.Errorf("invalid fallback: %s", config.Edge.Routing.Fallback)
	}

	// Validate that at least one model is configured
	if len(config.Edge.Models.Local) == 0 && config.Edge.Models.Remote.ClusterURL == "" {
		return fmt.Errorf("no models configured (need at least local or remote)")
	}

	return nil
}

// GetModelStorageDir returns the directory for storing models
func GetModelStorageDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./models"
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "llm-d", "models")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "llm-d", "models")
		}
		return filepath.Join(home, "AppData", "Roaming", "llm-d", "models")
	default:
		return filepath.Join(home, ".llm-d", "models")
	}
}

// GetCacheDir returns the directory for caching
func GetCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./cache"
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Caches", "llm-d")
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			return filepath.Join(localAppData, "llm-d", "cache")
		}
		return filepath.Join(home, "AppData", "Local", "llm-d", "cache")
	default:
		return filepath.Join(home, ".cache", "llm-d")
	}
}

// Made with Bob

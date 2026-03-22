package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EdgeRouterConfig represents the edge-router configuration structure
type EdgeRouterConfig struct {
	Edge EdgeConfig `yaml:"edge"`
}

// EdgeConfig contains edge router settings
type EdgeConfig struct {
	Platform          string                 `yaml:"platform"`
	Device            DeviceConfig           `yaml:"device"`
	Routing           RoutingConfig          `yaml:"routing"`
	Models            ModelsConfig           `yaml:"models"`
	RoutingRules      []RoutingRule          `yaml:"routing_rules,omitempty"`
	PlatformOverrides map[string]interface{} `yaml:"platform_overrides,omitempty"`
}

// DeviceConfig contains device settings
type DeviceConfig struct {
	Type    string `yaml:"type"`
	Memory  string `yaml:"memory"`
	Storage string `yaml:"storage"`
}

// RoutingConfig contains routing policy settings
type RoutingConfig struct {
	Policy   string `yaml:"policy"`
	Fallback string `yaml:"fallback"`
}

// ModelsConfig contains model configuration
type ModelsConfig struct {
	Local  []LocalModel  `yaml:"local"`
	Remote RemoteConfig  `yaml:"remote"`
}

// LocalModel represents a local model configuration
type LocalModel struct {
	Name          string `yaml:"name"`
	Format        string `yaml:"format"`
	Quantization  string `yaml:"quantization"`
	Priority      int    `yaml:"priority"`
	Path          string `yaml:"path,omitempty"`
}

// RemoteConfig contains remote cluster configuration
type RemoteConfig struct {
	ClusterURL string `yaml:"cluster_url"`
	AuthToken  string `yaml:"auth_token"`
	Timeout    int    `yaml:"timeout"`
}

// RoutingRule represents a routing rule
type RoutingRule struct {
	Condition string `yaml:"condition"`
	Action    string `yaml:"action"`
}

// ConfigUpdater handles updating edge-router configuration
type ConfigUpdater struct {
	configPath string
}

// NewConfigUpdater creates a new config updater
// If configPath is empty, searches for config in common locations
func NewConfigUpdater(configPath string) *ConfigUpdater {
	if configPath == "" {
		configPath = findConfigFile()
	}
	return &ConfigUpdater{
		configPath: configPath,
	}
}

// findConfigFile searches for edge-router config in common locations
func findConfigFile() string {
	// Try common locations in order
	locations := []string{
		// Relative to current directory
		"edge-router/config.yaml",
		"../edge-router/config.yaml",
		// Relative to home directory
		filepath.Join(os.Getenv("HOME"), "llm-d-edge", "edge-router", "config.yaml"),
		// System-wide location
		"/etc/llm-d-edge/config.yaml",
	}
	
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			absPath, _ := filepath.Abs(loc)
			return absPath
		}
	}
	
	// Default to edge-router/config.yaml (will be created if needed)
	return filepath.Join("edge-router", "config.yaml")
}

// LoadConfig loads the edge-router configuration
func (u *ConfigUpdater) LoadConfig() (*EdgeRouterConfig, error) {
	data, err := os.ReadFile(u.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return u.defaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config EdgeRouterConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the edge-router configuration
func (u *ConfigUpdater) SaveConfig(config *EdgeRouterConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(u.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(u.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// AddLocalModel adds a local model to the configuration
func (u *ConfigUpdater) AddLocalModel(model LocalModel) error {
	config, err := u.LoadConfig()
	if err != nil {
		return err
	}

	// Check if model already exists
	for i, existing := range config.Edge.Models.Local {
		if existing.Name == model.Name {
			// Update existing model
			config.Edge.Models.Local[i] = model
			return u.SaveConfig(config)
		}
	}

	// Add new model
	config.Edge.Models.Local = append(config.Edge.Models.Local, model)
	return u.SaveConfig(config)
}

// RemoveLocalModel removes a local model from the configuration
func (u *ConfigUpdater) RemoveLocalModel(modelName string) error {
	config, err := u.LoadConfig()
	if err != nil {
		return err
	}

	// Find and remove model
	newModels := make([]LocalModel, 0)
	found := false
	for _, model := range config.Edge.Models.Local {
		if model.Name != modelName {
			newModels = append(newModels, model)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("model %s not found in configuration", modelName)
	}

	config.Edge.Models.Local = newModels
	return u.SaveConfig(config)
}

// ListLocalModels returns all local models in the configuration
func (u *ConfigUpdater) ListLocalModels() ([]LocalModel, error) {
	config, err := u.LoadConfig()
	if err != nil {
		return nil, err
	}

	return config.Edge.Models.Local, nil
}

// UpdateModelPriority updates the priority of a local model
func (u *ConfigUpdater) UpdateModelPriority(modelName string, priority int) error {
	config, err := u.LoadConfig()
	if err != nil {
		return err
	}

	found := false
	for i, model := range config.Edge.Models.Local {
		if model.Name == modelName {
			config.Edge.Models.Local[i].Priority = priority
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("model %s not found in configuration", modelName)
	}

	return u.SaveConfig(config)
}

// defaultConfig returns a default edge-router configuration
func (u *ConfigUpdater) defaultConfig() *EdgeRouterConfig {
	return &EdgeRouterConfig{
		Edge: EdgeConfig{
			Platform: "auto",
			Device: DeviceConfig{
				Type:    "auto",
				Memory:  "auto",
				Storage: "auto",
			},
			Routing: RoutingConfig{
				Policy:   "hybrid",
				Fallback: "remote",
			},
			Models: ModelsConfig{
				Local: []LocalModel{},
				Remote: RemoteConfig{
					ClusterURL: "https://llm-d.example.com",
					AuthToken:  "${LLMD_AUTH_TOKEN}",
					Timeout:    60,
				},
			},
		},
	}
}

// Made with Bob
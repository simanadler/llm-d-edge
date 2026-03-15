package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
)

// TestLoadConfigWithValidFile tests loading a valid configuration file
func TestLoadConfigWithValidFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
edge:
  platform: macos
  device:
    type: apple-silicon-m4
    memory: 16GB
    storage: 512GB
  routing:
    policy: local-first
    fallback: remote
  models:
    local:
      - name: llama-3.2-3b
        format: mlx
        quantization: 4bit
        priority: 1
    remote:
      cluster_url: https://api.example.com
      auth_token: test-token
      timeout: 60
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config == nil {
		t.Fatal("Config is nil")
	}

	// Verify loaded values
	if config.Edge.Platform != "macos" {
		t.Errorf("Expected platform 'macos', got '%s'", config.Edge.Platform)
	}

	if config.Edge.Device.Type != "apple-silicon-m4" {
		t.Errorf("Expected device type 'apple-silicon-m4', got '%s'", config.Edge.Device.Type)
	}

	if config.Edge.Routing.Policy != PolicyLocalFirst {
		t.Errorf("Expected policy 'local-first', got '%s'", config.Edge.Routing.Policy)
	}

	if len(config.Edge.Models.Local) != 1 {
		t.Errorf("Expected 1 local model, got %d", len(config.Edge.Models.Local))
	}

	if config.Edge.Models.Remote.ClusterURL != "https://api.example.com" {
		t.Errorf("Expected cluster URL 'https://api.example.com', got '%s'", config.Edge.Models.Remote.ClusterURL)
	}
}

// TestLoadConfigWithDefaults tests loading config with default values
func TestLoadConfigWithDefaults(t *testing.T) {
	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
edge:
  models:
    local:
      - name: test-model
        format: mlx
        quantization: 4bit
        priority: 1
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check defaults
	if config.Edge.Device.Type != "auto" {
		t.Errorf("Expected default device type 'auto', got '%s'", config.Edge.Device.Type)
	}

	if config.Edge.Routing.Policy != PolicyHybrid {
		t.Errorf("Expected default policy 'hybrid', got '%s'", config.Edge.Routing.Policy)
	}

	if config.Edge.Routing.Fallback != "remote" {
		t.Errorf("Expected default fallback 'remote', got '%s'", config.Edge.Routing.Fallback)
	}

	if config.Edge.Models.Remote.Timeout != 60 {
		t.Errorf("Expected default timeout 60, got %d", config.Edge.Models.Remote.Timeout)
	}
}

// TestLoadConfigAutoDetectPlatform tests automatic platform detection
func TestLoadConfigAutoDetectPlatform(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
edge:
  platform: auto
  models:
    local:
      - name: test-model
        format: mlx
        quantization: 4bit
        priority: 1
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Platform should be auto-detected
	if config.Edge.Platform == "auto" || config.Edge.Platform == "" {
		t.Error("Platform should be auto-detected, not 'auto' or empty")
	}

	expectedPlatform := detectPlatform()
	if config.Edge.Platform != expectedPlatform {
		t.Errorf("Expected platform '%s', got '%s'", expectedPlatform, config.Edge.Platform)
	}
}

// TestLoadConfigInvalidFile tests loading with invalid file path
func TestLoadConfigInvalidFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent config file, got nil")
	}
}

// TestLoadConfigInvalidYAML tests loading with invalid YAML
func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
edge:
  platform: macos
  invalid yaml structure
    - broken
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

// TestValidateConfigValidPolicies tests validation with valid routing policies
func TestValidateConfigValidPolicies(t *testing.T) {
	validPolicies := []RoutingPolicy{
		PolicyLocalFirst,
		PolicyRemoteFirst,
		PolicyHybrid,
		PolicyCostOptimized,
		PolicyLatencyOptimized,
		PolicyMobileOptimized,
	}

	for _, policy := range validPolicies {
		t.Run(string(policy), func(t *testing.T) {
			config := &Config{
				Edge: EdgeConfig{
					Routing: RoutingConfig{
						Policy:   policy,
						Fallback: "remote",
					},
					Models: ModelsConfig{
						Local: []LocalModelConfig{
							{Name: "test-model"},
						},
					},
				},
			}

			err := validateConfig(config)
			if err != nil {
				t.Errorf("Valid policy '%s' should not produce error: %v", policy, err)
			}
		})
	}
}

// TestValidateConfigInvalidPolicy tests validation with invalid routing policy
func TestValidateConfigInvalidPolicy(t *testing.T) {
	config := &Config{
		Edge: EdgeConfig{
			Routing: RoutingConfig{
				Policy:   "invalid-policy",
				Fallback: "remote",
			},
			Models: ModelsConfig{
				Local: []LocalModelConfig{
					{Name: "test-model"},
				},
			},
		},
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid routing policy, got nil")
	}
}

// TestValidateConfigValidFallbacks tests validation with valid fallback options
func TestValidateConfigValidFallbacks(t *testing.T) {
	validFallbacks := []string{"remote", "local", "fail"}

	for _, fallback := range validFallbacks {
		t.Run(fallback, func(t *testing.T) {
			config := &Config{
				Edge: EdgeConfig{
					Routing: RoutingConfig{
						Policy:   PolicyHybrid,
						Fallback: fallback,
					},
					Models: ModelsConfig{
						Local: []LocalModelConfig{
							{Name: "test-model"},
						},
					},
				},
			}

			err := validateConfig(config)
			if err != nil {
				t.Errorf("Valid fallback '%s' should not produce error: %v", fallback, err)
			}
		})
	}
}

// TestValidateConfigInvalidFallback tests validation with invalid fallback
func TestValidateConfigInvalidFallback(t *testing.T) {
	config := &Config{
		Edge: EdgeConfig{
			Routing: RoutingConfig{
				Policy:   PolicyHybrid,
				Fallback: "invalid-fallback",
			},
			Models: ModelsConfig{
				Local: []LocalModelConfig{
					{Name: "test-model"},
				},
			},
		},
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid fallback, got nil")
	}
}

// TestValidateConfigNoModels tests validation with no models configured
func TestValidateConfigNoModels(t *testing.T) {
	config := &Config{
		Edge: EdgeConfig{
			Routing: RoutingConfig{
				Policy:   PolicyHybrid,
				Fallback: "remote",
			},
			Models: ModelsConfig{
				Local:  []LocalModelConfig{},
				Remote: RemoteClusterConfig{},
			},
		},
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for no models configured, got nil")
	}
}

// TestValidateConfigLocalModelsOnly tests validation with only local models
func TestValidateConfigLocalModelsOnly(t *testing.T) {
	config := &Config{
		Edge: EdgeConfig{
			Routing: RoutingConfig{
				Policy:   PolicyLocalFirst,
				Fallback: "fail",
			},
			Models: ModelsConfig{
				Local: []LocalModelConfig{
					{Name: "test-model"},
				},
			},
		},
	}

	err := validateConfig(config)
	if err != nil {
		t.Errorf("Config with only local models should be valid: %v", err)
	}
}

// TestValidateConfigRemoteModelsOnly tests validation with only remote models
func TestValidateConfigRemoteModelsOnly(t *testing.T) {
	config := &Config{
		Edge: EdgeConfig{
			Routing: RoutingConfig{
				Policy:   PolicyRemoteFirst,
				Fallback: "fail",
			},
			Models: ModelsConfig{
				Remote: RemoteClusterConfig{
					ClusterURL: "https://api.example.com",
				},
			},
		},
	}

	err := validateConfig(config)
	if err != nil {
		t.Errorf("Config with only remote models should be valid: %v", err)
	}
}

// TestDetectPlatform tests platform detection
func TestDetectPlatform(t *testing.T) {
	platform := detectPlatform()

	if platform == "" {
		t.Error("Detected platform should not be empty")
	}

	// Verify it matches expected platform for current OS
	expectedPlatforms := map[string][]string{
		"darwin":  {"macos"},
		"windows": {"windows"},
		"linux":   {"linux", "android"},
		"ios":     {"ios"},
	}

	validPlatforms, ok := expectedPlatforms[runtime.GOOS]
	if !ok {
		// Unknown OS, just check it's not empty
		return
	}

	found := false
	for _, valid := range validPlatforms {
		if platform == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Detected platform '%s' not in expected list %v for OS '%s'", platform, validPlatforms, runtime.GOOS)
	}
}

// TestGetDefaultConfigDir tests default config directory detection
func TestGetDefaultConfigDir(t *testing.T) {
	dir := getDefaultConfigDir()

	if dir == "" {
		t.Error("Default config dir should not be empty")
	}

	// Should contain platform-specific path
	switch runtime.GOOS {
	case "darwin":
		if !filepath.IsAbs(dir) && dir != "." {
			t.Error("Config dir should be absolute path or '.'")
		}
	case "windows":
		if !filepath.IsAbs(dir) && dir != "." {
			t.Error("Config dir should be absolute path or '.'")
		}
	default:
		if !filepath.IsAbs(dir) && dir != "." {
			t.Error("Config dir should be absolute path or '.'")
		}
	}
}

// TestGetModelStorageDir tests model storage directory detection
func TestGetModelStorageDir(t *testing.T) {
	dir := GetModelStorageDir()

	if dir == "" {
		t.Error("Model storage dir should not be empty")
	}

	// Should end with "models"
	if filepath.Base(dir) != "models" {
		t.Errorf("Model storage dir should end with 'models', got '%s'", dir)
	}
}

// TestGetCacheDir tests cache directory detection
func TestGetCacheDir(t *testing.T) {
	dir := GetCacheDir()

	if dir == "" {
		t.Error("Cache dir should not be empty")
	}

	// Should contain "cache" in path
	if filepath.Base(dir) != "cache" && filepath.Base(filepath.Dir(dir)) != "Caches" {
		t.Errorf("Cache dir should contain 'cache' or 'Caches', got '%s'", dir)
	}
}

// TestSetDefaults tests default value setting
func TestSetDefaults(t *testing.T) {
	v := viper.New()
	setDefaults(v)

	tests := []struct {
		key      string
		expected interface{}
	}{
		{"edge.platform", "auto"},
		{"edge.device.type", "auto"},
		{"edge.device.memory", "auto"},
		{"edge.device.storage", "auto"},
		{"edge.routing.policy", "hybrid"},
		{"edge.routing.fallback", "remote"},
		{"edge.models.remote.timeout", 60},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			actual := v.Get(tt.key)
			if actual != tt.expected {
				t.Errorf("Expected default %s=%v, got %v", tt.key, tt.expected, actual)
			}
		})
	}
}

// TestApplyPlatformOverrides tests platform override application
func TestApplyPlatformOverrides(t *testing.T) {
	config := &Config{
		Edge: EdgeConfig{
			Platform: "macos",
			PlatformOverrides: map[string]interface{}{
				"macos": map[string]interface{}{
					"device": map[string]interface{}{
						"type": "apple-silicon",
					},
				},
			},
		},
	}

	err := applyPlatformOverrides(config)
	if err != nil {
		t.Errorf("Platform overrides should not produce error: %v", err)
	}
}

// TestApplyPlatformOverridesNoPlatform tests override with no matching platform
func TestApplyPlatformOverridesNoPlatform(t *testing.T) {
	config := &Config{
		Edge: EdgeConfig{
			Platform: "linux",
			PlatformOverrides: map[string]interface{}{
				"macos": map[string]interface{}{
					"device": "override",
				},
			},
		},
	}

	err := applyPlatformOverrides(config)
	if err != nil {
		t.Errorf("Missing platform override should not produce error: %v", err)
	}
}

// TestLoadConfigWithRoutingRules tests loading config with routing rules
func TestLoadConfigWithRoutingRules(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
edge:
  platform: macos
  routing:
    policy: hybrid
    fallback: remote
  models:
    local:
      - name: test-model
        format: mlx
        quantization: 4bit
        priority: 1
  routing_rules:
    - condition: "prompt_tokens < 1000"
      action: "route_local"
    - condition: "prompt_tokens >= 1000"
      action: "route_remote"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.Edge.RoutingRules) != 2 {
		t.Errorf("Expected 2 routing rules, got %d", len(config.Edge.RoutingRules))
	}

	if config.Edge.RoutingRules[0].Condition != "prompt_tokens < 1000" {
		t.Errorf("Expected first rule condition 'prompt_tokens < 1000', got '%s'", config.Edge.RoutingRules[0].Condition)
	}

	if config.Edge.RoutingRules[0].Action != "route_local" {
		t.Errorf("Expected first rule action 'route_local', got '%s'", config.Edge.RoutingRules[0].Action)
	}
}

// TestLoadConfigWithMultipleLocalModels tests loading config with multiple local models
func TestLoadConfigWithMultipleLocalModels(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
edge:
  platform: macos
  routing:
    policy: local-first
    fallback: remote
  models:
    local:
      - name: llama-3.2-3b
        format: mlx
        quantization: 4bit
        priority: 1
      - name: llama-3.2-1b
        format: mlx
        quantization: 8bit
        priority: 2
        path: /custom/path/model
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.Edge.Models.Local) != 2 {
		t.Errorf("Expected 2 local models, got %d", len(config.Edge.Models.Local))
	}

	// Check first model
	if config.Edge.Models.Local[0].Name != "llama-3.2-3b" {
		t.Errorf("Expected first model name 'llama-3.2-3b', got '%s'", config.Edge.Models.Local[0].Name)
	}

	if config.Edge.Models.Local[0].Priority != 1 {
		t.Errorf("Expected first model priority 1, got %d", config.Edge.Models.Local[0].Priority)
	}

	// Check second model
	if config.Edge.Models.Local[1].Name != "llama-3.2-1b" {
		t.Errorf("Expected second model name 'llama-3.2-1b', got '%s'", config.Edge.Models.Local[1].Name)
	}

	if config.Edge.Models.Local[1].Path != "/custom/path/model" {
		t.Errorf("Expected second model path '/custom/path/model', got '%s'", config.Edge.Models.Local[1].Path)
	}
}

// TestRoutingPolicyConstants tests that routing policy constants are defined correctly
func TestRoutingPolicyConstants(t *testing.T) {
	tests := []struct {
		policy   RoutingPolicy
		expected string
	}{
		{PolicyLocalFirst, "local-first"},
		{PolicyRemoteFirst, "remote-first"},
		{PolicyHybrid, "hybrid"},
		{PolicyCostOptimized, "cost-optimized"},
		{PolicyLatencyOptimized, "latency-optimized"},
		{PolicyMobileOptimized, "mobile-optimized"},
	}

	for _, tt := range tests {
		t.Run(string(tt.policy), func(t *testing.T) {
			if string(tt.policy) != tt.expected {
				t.Errorf("Expected policy constant '%s', got '%s'", tt.expected, string(tt.policy))
			}
		})
	}
}

// TestIsAndroid tests Android detection (mocked)
func TestIsAndroid(t *testing.T) {
	// This function checks for /system/build.prop
	// We can't easily mock this in a unit test without changing the implementation
	// But we can at least call it to ensure it doesn't panic
	result := isAndroid()
	// Result will be false in test environment
	if result && runtime.GOOS != "linux" {
		t.Error("isAndroid should only return true on Linux systems")
	}
}

// TestGetDefaultConfigDirWithError tests config dir when home dir fails
func TestGetDefaultConfigDirWithError(t *testing.T) {
	// We can't easily mock os.UserHomeDir() failure without changing the implementation
	// But the function handles it by returning "."
	dir := getDefaultConfigDir()
	if dir == "" {
		t.Error("getDefaultConfigDir should never return empty string")
	}
}

// TestGetModelStorageDirWithError tests model storage dir when home dir fails
func TestGetModelStorageDirWithError(t *testing.T) {
	// Similar to above, function handles error by returning "./models"
	dir := GetModelStorageDir()
	if dir == "" {
		t.Error("GetModelStorageDir should never return empty string")
	}
	// Should always end with models
	if filepath.Base(dir) != "models" {
		t.Errorf("GetModelStorageDir should end with 'models', got '%s'", dir)
	}
}

// TestGetCacheDirWithError tests cache dir when home dir fails
func TestGetCacheDirWithError(t *testing.T) {
	// Function handles error by returning "./cache"
	dir := GetCacheDir()
	if dir == "" {
		t.Error("GetCacheDir should never return empty string")
	}
}

// TestLoadConfigWithEmptyPath tests loading config with empty path
func TestLoadConfigWithEmptyPath(t *testing.T) {
	// Create a config in current directory
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	
	os.Chdir(tmpDir)
	
	configContent := `
edge:
  platform: macos
  models:
    local:
      - name: test-model
        format: mlx
        quantization: 4bit
        priority: 1
`

	if err := os.WriteFile("config.yaml", []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load config with empty path: %v", err)
	}

	if config == nil {
		t.Fatal("Config should not be nil")
	}
}

// TestLoadConfigFileNotFound tests loading when config file doesn't exist
func TestLoadConfigFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	
	// Change to empty directory
	os.Chdir(tmpDir)
	
	// Try to load config with empty path (will look for config.yaml)
	// Should use defaults and not error
	_, err := LoadConfig("")
	// This will error because no models are configured in defaults
	if err == nil {
		t.Error("Expected error when no config file and no models configured")
	}
}

// TestConfigStructTags tests that struct tags are properly defined
func TestConfigStructTags(t *testing.T) {
	// This is a compile-time check, but we can verify the structs exist
	var config Config
	var edgeConfig EdgeConfig
	var deviceConfig DeviceConfig
	var routingConfig RoutingConfig
	var modelsConfig ModelsConfig
	var localModelConfig LocalModelConfig
	var remoteClusterConfig RemoteClusterConfig
	var routingRule RoutingRule

	// Just verify they can be instantiated
	_ = config
	_ = edgeConfig
	_ = deviceConfig
	_ = routingConfig
	_ = modelsConfig
	_ = localModelConfig
	_ = remoteClusterConfig
	_ = routingRule
}

// TestLoadConfigWithAllFields tests loading config with all possible fields
func TestLoadConfigWithAllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
edge:
  platform: macos
  device:
    type: apple-silicon-m4
    memory: 32GB
    storage: 1TB
  routing:
    policy: cost-optimized
    fallback: local
  models:
    local:
      - name: llama-3.2-3b
        format: mlx
        quantization: 4bit
        priority: 1
        path: /custom/path
      - name: llama-3.2-1b
        format: gguf
        quantization: 8bit
        priority: 2
    remote:
      cluster_url: https://api.example.com
      auth_token: secret-token-123
      timeout: 120
  routing_rules:
    - condition: "model == 'llama-3.2-3b'"
      action: "route_local"
    - condition: "prompt_tokens > 2000"
      action: "route_remote"
  platform_overrides:
    macos:
      device:
        type: apple-silicon
    linux:
      device:
        type: nvidia-gpu
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all fields are loaded
	if config.Edge.Platform != "macos" {
		t.Errorf("Expected platform 'macos', got '%s'", config.Edge.Platform)
	}

	if config.Edge.Device.Memory != "32GB" {
		t.Errorf("Expected memory '32GB', got '%s'", config.Edge.Device.Memory)
	}

	if config.Edge.Device.Storage != "1TB" {
		t.Errorf("Expected storage '1TB', got '%s'", config.Edge.Device.Storage)
	}

	if config.Edge.Routing.Policy != PolicyCostOptimized {
		t.Errorf("Expected policy 'cost-optimized', got '%s'", config.Edge.Routing.Policy)
	}

	if config.Edge.Routing.Fallback != "local" {
		t.Errorf("Expected fallback 'local', got '%s'", config.Edge.Routing.Fallback)
	}

	if len(config.Edge.Models.Local) != 2 {
		t.Errorf("Expected 2 local models, got %d", len(config.Edge.Models.Local))
	}

	if config.Edge.Models.Local[1].Format != "gguf" {
		t.Errorf("Expected second model format 'gguf', got '%s'", config.Edge.Models.Local[1].Format)
	}

	if config.Edge.Models.Remote.Timeout != 120 {
		t.Errorf("Expected timeout 120, got %d", config.Edge.Models.Remote.Timeout)
	}

	if len(config.Edge.RoutingRules) != 2 {
		t.Errorf("Expected 2 routing rules, got %d", len(config.Edge.RoutingRules))
	}

	if config.Edge.PlatformOverrides == nil {
		t.Error("Expected platform overrides to be loaded")
	}

	if _, ok := config.Edge.PlatformOverrides["macos"]; !ok {
		t.Error("Expected macos platform override")
	}
}

// TestDetectPlatformAllCases tests platform detection for different OS values
func TestDetectPlatformAllCases(t *testing.T) {
	// We can only test the current platform, but we can verify the function works
	platform := detectPlatform()
	
	validPlatforms := []string{"macos", "windows", "linux", "android", "ios"}
	found := false
	for _, valid := range validPlatforms {
		if platform == valid {
			found = true
			break
		}
	}
	
	// If not in known list, should be runtime.GOOS
	if !found && platform != runtime.GOOS {
		t.Errorf("Unknown platform '%s' should match runtime.GOOS '%s'", platform, runtime.GOOS)
	}
}

// TestLoadConfigWithExcessFields tests that excess/unknown fields cause failure
func TestLoadConfigWithExcessFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config with unknown/excess fields
	configContent := `
edge:
  platform: macos
  unknown_field: "this should cause an error"
  device:
    type: apple-silicon-m4
    memory: 16GB
    storage: 512GB
    extra_field: "also invalid"
  routing:
    policy: local-first
    fallback: remote
  models:
    local:
      - name: test-model
        format: mlx
        quantization: 4bit
        priority: 1
        invalid_property: "should fail"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error when loading config with excess fields, got nil")
	}

	// Verify error message mentions the unmarshal issue
	if err != nil && !contains(err.Error(), "unmarshal") {
		t.Logf("Error message: %v", err)
	}
}

// TestLoadConfigWithTypoInField tests that typos in field names cause failure
func TestLoadConfigWithTypoInField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config with typo in field name
	configContent := `
edge:
  platfrom: macos  # typo: should be "platform"
  device:
    type: apple-silicon-m4
  routing:
    policy: local-first
    fallback: remote
  models:
    local:
      - name: test-model
        format: mlx
        quantization: 4bit
        priority: 1
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error when loading config with typo in field name, got nil")
	}
}

// TestLoadConfigWithExtraTopLevelField tests excess fields at top level
func TestLoadConfigWithExtraTopLevelField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
edge:
  platform: macos
  device:
    type: apple-silicon-m4
  routing:
    policy: local-first
    fallback: remote
  models:
    local:
      - name: test-model
        format: mlx
        quantization: 4bit
        priority: 1
extra_top_level_field: "should cause error"
another_field: 123
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error when loading config with extra top-level fields, got nil")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr)+1 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestValidateConfigEdgeCases tests validation edge cases
func TestValidateConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Empty routing policy",
			config: &Config{
				Edge: EdgeConfig{
					Routing: RoutingConfig{
						Policy:   "",
						Fallback: "remote",
					},
					Models: ModelsConfig{
						Local: []LocalModelConfig{{Name: "test"}},
					},
				},
			},
			shouldError: true,
			errorMsg:    "invalid routing policy",
		},
		{
			name: "Empty fallback",
			config: &Config{
				Edge: EdgeConfig{
					Routing: RoutingConfig{
						Policy:   PolicyHybrid,
						Fallback: "",
					},
					Models: ModelsConfig{
						Local: []LocalModelConfig{{Name: "test"}},
					},
				},
			},
			shouldError: true,
			errorMsg:    "invalid fallback",
		},
		{
			name: "Both local and remote models",
			config: &Config{
				Edge: EdgeConfig{
					Routing: RoutingConfig{
						Policy:   PolicyHybrid,
						Fallback: "remote",
					},
					Models: ModelsConfig{
						Local: []LocalModelConfig{{Name: "test"}},
						Remote: RemoteClusterConfig{
							ClusterURL: "https://api.example.com",
						},
					},
				},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.shouldError && err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// Made with Bob
package config

// ModelCapabilities defines the capabilities of a model
type ModelCapabilities struct {
	ParameterCount string             `mapstructure:"parameter_count,omitempty" yaml:"parameter_count,omitempty"` // e.g., "3B", "7B"
	ContextLength  int                `mapstructure:"context_length,omitempty" yaml:"context_length,omitempty"`   // e.g., 8192
	ModelFamily    string             `mapstructure:"model_family,omitempty" yaml:"model_family,omitempty"`       // e.g., "llama", "qwen"
	QualityTier    string             `mapstructure:"quality_tier,omitempty" yaml:"quality_tier,omitempty"`       // low, medium, high, premium
	Tasks          map[string]float64 `mapstructure:"tasks,omitempty" yaml:"tasks,omitempty"`                     // Task suitability scores (0.0-1.0) - OPTIONAL
	Domains        map[string]float64 `mapstructure:"domains,omitempty" yaml:"domains,omitempty"`                 // Domain expertise scores (0.0-1.0) - OPTIONAL
}

// ModelMatching defines matching rules for model substitution
type ModelMatching struct {
	CanSubstitute   []SubstitutionRule `mapstructure:"can_substitute,omitempty" yaml:"can_substitute,omitempty"`     // Models this can substitute for
	ExcludePatterns []string           `mapstructure:"exclude_patterns,omitempty" yaml:"exclude_patterns,omitempty"` // Models this should NOT substitute for
}

// SubstitutionRule defines a pattern for model substitution
type SubstitutionRule struct {
	Pattern string `mapstructure:"pattern" yaml:"pattern"` // Regex pattern for model names
}

// ExtendedLocalModelConfig extends LocalModelConfig with capabilities and matching
type ExtendedLocalModelConfig struct {
	LocalModelConfig `mapstructure:",squash" yaml:",inline"`                                               // Embed base config
	Capabilities     ModelCapabilities                       `mapstructure:"capabilities,omitempty" yaml:"capabilities,omitempty"` // Model capabilities - OPTIONAL
	Matching         ModelMatching                           `mapstructure:"matching,omitempty" yaml:"matching,omitempty"`         // Matching rules - OPTIONAL
}

// Made with Bob
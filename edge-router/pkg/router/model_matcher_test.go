package router

import (
	"testing"

	"github.com/llm-d-incubation/llm-d-edge/pkg/config"
	"go.uber.org/zap"
)

func TestModelMatcher_FindCandidates_ExactMatch(t *testing.T) {
	models := []config.ExtendedLocalModelConfig{
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "meta-llama/Llama-3.2-3B",
				Priority: 1,
			},
		},
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "Qwen/Qwen3-0.6B",
				Priority: 2,
			},
		},
	}

	matcher := NewModelMatcher(models, zap.NewNop())
	candidates := matcher.FindCandidates("meta-llama/Llama-3.2-3B")

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	if candidates[0].MatchType != MatchTypeExact {
		t.Errorf("expected exact match, got %s", candidates[0].MatchType)
	}

	if candidates[0].MatchScore != 1.0 {
		t.Errorf("expected match score 1.0, got %f", candidates[0].MatchScore)
	}

	if candidates[0].Model.Name != "meta-llama/Llama-3.2-3B" {
		t.Errorf("expected model name 'meta-llama/Llama-3.2-3B', got %s", candidates[0].Model.Name)
	}
}

func TestModelMatcher_FindCandidates_SubstitutionMatch(t *testing.T) {
	models := []config.ExtendedLocalModelConfig{
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "meta-llama/Llama-3.2-3B",
				Priority: 1,
			},
			Matching: config.ModelMatching{
				CanSubstitute: []config.SubstitutionRule{
					{Pattern: "gpt-3.5*"},
					{Pattern: "llama*3*"},
				},
			},
		},
	}

	matcher := NewModelMatcher(models, zap.NewNop())
	candidates := matcher.FindCandidates("gpt-3.5-turbo")

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	if candidates[0].MatchType != MatchTypeSubstitution {
		t.Errorf("expected substitution match, got %s", candidates[0].MatchType)
	}

	if candidates[0].MatchScore != 0.8 {
		t.Errorf("expected match score 0.8, got %f", candidates[0].MatchScore)
	}

	if candidates[0].Model.Name != "meta-llama/Llama-3.2-3B" {
		t.Errorf("expected model name 'meta-llama/Llama-3.2-3B', got %s", candidates[0].Model.Name)
	}
}

func TestModelMatcher_FindCandidates_ExcludePattern(t *testing.T) {
	models := []config.ExtendedLocalModelConfig{
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "Qwen/Qwen3-0.6B",
				Priority: 1,
			},
			Matching: config.ModelMatching{
				CanSubstitute: []config.SubstitutionRule{
					{Pattern: "gpt*"}, // Matches all GPT models
				},
				ExcludePatterns: []string{
					"gpt-4*", // But exclude GPT-4
				},
			},
		},
	}

	matcher := NewModelMatcher(models, zap.NewNop())

	// Should match gpt-3.5-turbo via substitution
	candidates := matcher.FindCandidates("gpt-3.5-turbo")
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate for gpt-3.5-turbo, got %d", len(candidates))
	}
	if candidates[0].MatchType != MatchTypeSubstitution {
		t.Errorf("expected substitution match for gpt-3.5-turbo, got %s", candidates[0].MatchType)
	}

	// Should NOT match gpt-4 via substitution (excluded), but will fallback
	candidates = matcher.FindCandidates("gpt-4")
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate for gpt-4 (fallback), got %d", len(candidates))
	}
	// Should be fallback match, not substitution
	if candidates[0].MatchType != MatchTypeFallback {
		t.Errorf("expected fallback match for gpt-4 (excluded from substitution), got %s", candidates[0].MatchType)
	}
	if candidates[0].MatchScore != 0.3 {
		t.Errorf("expected match score 0.3 for fallback, got %f", candidates[0].MatchScore)
	}
}

func TestModelMatcher_FindCandidates_FamilyMatch(t *testing.T) {
	models := []config.ExtendedLocalModelConfig{
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "meta-llama/Llama-3.2-3B",
				Priority: 1,
			},
			Capabilities: config.ModelCapabilities{
				ModelFamily: "llama",
			},
		},
	}

	matcher := NewModelMatcher(models, zap.NewNop())
	candidates := matcher.FindCandidates("llama-3.2-7b")

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	if candidates[0].MatchType != MatchTypeFamily {
		t.Errorf("expected family match, got %s", candidates[0].MatchType)
	}

	if candidates[0].MatchScore != 0.7 {
		t.Errorf("expected match score 0.7, got %f", candidates[0].MatchScore)
	}
}

func TestModelMatcher_FindCandidates_FallbackMatch(t *testing.T) {
	models := []config.ExtendedLocalModelConfig{
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "meta-llama/Llama-3.2-3B",
				Priority: 1,
			},
		},
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "Qwen/Qwen3-0.6B",
				Priority: 2,
			},
		},
	}

	matcher := NewModelMatcher(models, zap.NewNop())
	candidates := matcher.FindCandidates("some-unknown-model")

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates (fallback), got %d", len(candidates))
	}

	for _, candidate := range candidates {
		if candidate.MatchType != MatchTypeFallback {
			t.Errorf("expected fallback match, got %s", candidate.MatchType)
		}
		if candidate.MatchScore != 0.3 {
			t.Errorf("expected match score 0.3, got %f", candidate.MatchScore)
		}
	}
}

func TestModelMatcher_FindCandidates_MultipleMatches_Ranking(t *testing.T) {
	models := []config.ExtendedLocalModelConfig{
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "Qwen/Qwen3-0.6B",
				Priority: 2, // Lower priority
			},
			Matching: config.ModelMatching{
				CanSubstitute: []config.SubstitutionRule{
					{Pattern: "gpt-3.5*"},
				},
			},
		},
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "meta-llama/Llama-3.2-3B",
				Priority: 1, // Higher priority
			},
			Matching: config.ModelMatching{
				CanSubstitute: []config.SubstitutionRule{
					{Pattern: "gpt-3.5*"},
				},
			},
		},
	}

	matcher := NewModelMatcher(models, zap.NewNop())
	candidates := matcher.FindCandidates("gpt-3.5-turbo")

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	// First candidate should be Llama (higher priority = lower number)
	if candidates[0].Model.Name != "meta-llama/Llama-3.2-3B" {
		t.Errorf("expected first candidate to be Llama-3.2-3B, got %s", candidates[0].Model.Name)
	}

	// Second candidate should be Qwen
	if candidates[1].Model.Name != "Qwen/Qwen3-0.6B" {
		t.Errorf("expected second candidate to be Qwen3-0.6B, got %s", candidates[1].Model.Name)
	}
}

func TestModelMatcher_FindCandidates_DifferentScores_Ranking(t *testing.T) {
	models := []config.ExtendedLocalModelConfig{
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "meta-llama/Llama-3.2-3B",
				Priority: 1,
			},
			Capabilities: config.ModelCapabilities{
				ModelFamily: "llama",
			},
		},
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "Qwen/Qwen3-0.6B",
				Priority: 1,
			},
			Matching: config.ModelMatching{
				CanSubstitute: []config.SubstitutionRule{
					{Pattern: "llama*"},
				},
			},
		},
	}

	matcher := NewModelMatcher(models, zap.NewNop())
	candidates := matcher.FindCandidates("llama-3.2-7b")

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	// First should be substitution match (0.8) over family match (0.7)
	if candidates[0].Model.Name != "Qwen/Qwen3-0.6B" {
		t.Errorf("expected first candidate to be Qwen (substitution 0.8), got %s", candidates[0].Model.Name)
	}
	if candidates[0].MatchScore != 0.8 {
		t.Errorf("expected match score 0.8, got %f", candidates[0].MatchScore)
	}

	// Second should be family match (0.7)
	if candidates[1].Model.Name != "meta-llama/Llama-3.2-3B" {
		t.Errorf("expected second candidate to be Llama (family 0.7), got %s", candidates[1].Model.Name)
	}
	if candidates[1].MatchScore != 0.7 {
		t.Errorf("expected match score 0.7, got %f", candidates[1].MatchScore)
	}
}

func TestModelMatcher_matchesPattern(t *testing.T) {
	matcher := NewModelMatcher([]config.ExtendedLocalModelConfig{}, zap.NewNop())

	tests := []struct {
		name     string
		model    string
		pattern  string
		expected bool
	}{
		{"exact match", "gpt-3.5-turbo", "gpt-3.5-turbo", true},
		{"wildcard prefix", "gpt-3.5-turbo", "gpt-3.5*", true},
		{"wildcard suffix", "gpt-3.5-turbo", "*turbo", true},
		{"wildcard middle", "gpt-3.5-turbo", "gpt*turbo", true},
		{"wildcard both", "gpt-3.5-turbo", "*3.5*", true},
		{"no match", "gpt-4", "gpt-3.5*", false},
		{"case sensitive", "GPT-3.5-turbo", "gpt-3.5*", false},
		{"multiple wildcards", "meta-llama/Llama-3.2-3B", "*llama*3*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.matchesPattern(tt.model, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, expected %v",
					tt.model, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestModelMatcher_extractModelFamily(t *testing.T) {
	matcher := NewModelMatcher([]config.ExtendedLocalModelConfig{}, zap.NewNop())

	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"llama with path", "meta-llama/Llama-3.2-3B", "llama"},
		{"llama simple", "llama-3.2-7b", "llama"},
		{"qwen with path", "Qwen/Qwen3-0.6B", "qwen"},
		{"qwen simple", "qwen2-7b", "qwen"},
		{"gpt", "gpt-3.5-turbo", "gpt"},
		{"claude", "claude-3-opus", "claude"},
		{"mistral", "mistralai/Mistral-7B", "mistral"},
		{"phi", "microsoft/phi-2", "phi"},
		{"gemma", "google/gemma-7b", "gemma"},
		{"unknown", "some-unknown-model", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.extractModelFamily(tt.model)
			if result != tt.expected {
				t.Errorf("extractModelFamily(%q) = %q, expected %q",
					tt.model, result, tt.expected)
			}
		})
	}
}

func TestModelMatcher_isExcluded(t *testing.T) {
	matcher := NewModelMatcher([]config.ExtendedLocalModelConfig{}, zap.NewNop())

	excludePatterns := []string{
		"gpt-4*",
		"claude-3-opus*",
		"*13b*",
	}

	tests := []struct {
		name     string
		model    string
		expected bool
	}{
		{"excluded gpt-4", "gpt-4", true},
		{"excluded gpt-4-turbo", "gpt-4-turbo", true},
		{"not excluded gpt-3.5", "gpt-3.5-turbo", false},
		{"excluded claude-3-opus", "claude-3-opus", true},
		{"not excluded claude-3-sonnet", "claude-3-sonnet", false},
		{"excluded 13b", "llama-3.2-13b", true},
		{"not excluded 7b", "llama-3.2-7b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.isExcluded(tt.model, excludePatterns)
			if result != tt.expected {
				t.Errorf("isExcluded(%q) = %v, expected %v",
					tt.model, result, tt.expected)
			}
		})
	}
}

func TestModelMatcher_containsModel(t *testing.T) {
	matcher := NewModelMatcher([]config.ExtendedLocalModelConfig{}, zap.NewNop())

	candidates := []ModelCandidate{
		{
			Model: config.ExtendedLocalModelConfig{
				LocalModelConfig: config.LocalModelConfig{
					Name: "model-a",
				},
			},
		},
		{
			Model: config.ExtendedLocalModelConfig{
				LocalModelConfig: config.LocalModelConfig{
					Name: "model-b",
				},
			},
		},
	}

	if !matcher.containsModel(candidates, "model-a") {
		t.Error("expected to find model-a")
	}

	if !matcher.containsModel(candidates, "model-b") {
		t.Error("expected to find model-b")
	}

	if matcher.containsModel(candidates, "model-c") {
		t.Error("expected not to find model-c")
	}
}

func TestModelMatcher_NoModels(t *testing.T) {
	matcher := NewModelMatcher([]config.ExtendedLocalModelConfig{}, zap.NewNop())
	candidates := matcher.FindCandidates("any-model")

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates when no models configured, got %d", len(candidates))
	}
}

func TestModelMatcher_ExactMatchTakesPrecedence(t *testing.T) {
	models := []config.ExtendedLocalModelConfig{
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "gpt-3.5-turbo",
				Priority: 1,
			},
		},
		{
			LocalModelConfig: config.LocalModelConfig{
				Name:     "meta-llama/Llama-3.2-3B",
				Priority: 1,
			},
			Matching: config.ModelMatching{
				CanSubstitute: []config.SubstitutionRule{
					{Pattern: "gpt-3.5*"},
				},
			},
		},
	}

	matcher := NewModelMatcher(models, zap.NewNop())
	candidates := matcher.FindCandidates("gpt-3.5-turbo")

	// Should return only exact match, not substitution
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate (exact match only), got %d", len(candidates))
	}

	if candidates[0].Model.Name != "gpt-3.5-turbo" {
		t.Errorf("expected exact match 'gpt-3.5-turbo', got %s", candidates[0].Model.Name)
	}

	if candidates[0].MatchType != MatchTypeExact {
		t.Errorf("expected exact match type, got %s", candidates[0].MatchType)
	}
}

// Made with Bob
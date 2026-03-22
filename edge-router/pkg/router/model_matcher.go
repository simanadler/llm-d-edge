package router

import (
	"regexp"
	"sort"
	"strings"

	"github.com/llm-d-incubation/llm-d-edge/pkg/config"
	"go.uber.org/zap"
)

// ModelMatcher handles matching requested models to available local models
type ModelMatcher struct {
	localModels []config.ExtendedLocalModelConfig
	logger      *zap.Logger
}

// NewModelMatcher creates a new model matcher
func NewModelMatcher(localModels []config.ExtendedLocalModelConfig, logger *zap.Logger) *ModelMatcher {
	return &ModelMatcher{
		localModels: localModels,
		logger:      logger,
	}
}

// MatchType represents the type of match found
type MatchType string

const (
	MatchTypeExact        MatchType = "exact"        // Exact model name match
	MatchTypeSubstitution MatchType = "substitution" // Substitution rule match
	MatchTypeFallback     MatchType = "fallback"     // Generic fallback
)

// ModelCandidate represents a candidate model with match information
type ModelCandidate struct {
	Model      config.ExtendedLocalModelConfig
	MatchType  MatchType
	MatchScore float64 // 0.0-1.0: confidence in this match
}

// FindCandidates returns ranked list of local models that could handle the request
func (m *ModelMatcher) FindCandidates(requestedModel string) []ModelCandidate {
	candidates := []ModelCandidate{}

	m.logger.Debug("finding candidates for requested model",
		zap.String("requested_model", requestedModel),
		zap.Int("available_models", len(m.localModels)))

	// 1. Exact match (highest priority)
	for _, model := range m.localModels {
		if model.Name == requestedModel {
			m.logger.Info("found exact match",
				zap.String("requested_model", requestedModel),
				zap.String("matched_model", model.Name))

			candidates = append(candidates, ModelCandidate{
				Model:      model,
				MatchType:  MatchTypeExact,
				MatchScore: 1.0,
			})
			return candidates // Return immediately for exact match
		}
	}

	// 2. Substitution match (check matching rules)
	for _, model := range m.localModels {
		for _, rule := range model.Matching.CanSubstitute {
			if m.matchesPattern(requestedModel, rule.Pattern) {
				// Check if excluded
				if m.isExcluded(requestedModel, model.Matching.ExcludePatterns) {
					m.logger.Debug("model excluded by exclude pattern",
						zap.String("requested_model", requestedModel),
						zap.String("candidate_model", model.Name))
					continue
				}

				m.logger.Info("found substitution match",
					zap.String("requested_model", requestedModel),
					zap.String("matched_model", model.Name),
					zap.String("pattern", rule.Pattern))

				candidates = append(candidates, ModelCandidate{
					Model:      model,
					MatchType:  MatchTypeSubstitution,
					MatchScore: 0.8, // High confidence for explicit substitution rules
				})
			}
		}
	}

	// 3. Fallback: any available model (lowest priority)
	if len(candidates) == 0 {
		m.logger.Warn("no specific matches found, using fallback",
			zap.String("requested_model", requestedModel))

		for _, model := range m.localModels {
			candidates = append(candidates, ModelCandidate{
				Model:      model,
				MatchType:  MatchTypeFallback,
				MatchScore: 0.3,
			})
		}
	}

	// Sort by match score (descending) and priority (ascending)
	sort.Slice(candidates, func(i, j int) bool {
		// Primary sort: match score (higher is better)
		if candidates[i].MatchScore != candidates[j].MatchScore {
			return candidates[i].MatchScore > candidates[j].MatchScore
		}
		// Secondary sort: priority (lower number is higher priority)
		return candidates[i].Model.Priority < candidates[j].Model.Priority
	})

	m.logger.Info("found candidates",
		zap.String("requested_model", requestedModel),
		zap.Int("candidate_count", len(candidates)))

	return candidates
}

// matchesPattern checks if a model name matches a pattern (supports wildcards)
func (m *ModelMatcher) matchesPattern(modelName, pattern string) bool {
	// Convert wildcard pattern to regex
	// Replace * with .* and escape other regex special characters
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, ".*")
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, modelName)
	if err != nil {
		m.logger.Warn("invalid pattern",
			zap.String("pattern", pattern),
			zap.Error(err))
		return false
	}

	return matched
}

// isExcluded checks if a model name matches any exclude pattern
func (m *ModelMatcher) isExcluded(modelName string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if m.matchesPattern(modelName, pattern) {
			return true
		}
	}
	return false
}

// containsModel checks if candidates already contains a model with the given name
func (m *ModelMatcher) containsModel(candidates []ModelCandidate, modelName string) bool {
	for _, candidate := range candidates {
		if candidate.Model.Name == modelName {
			return true
		}
	}
	return false
}

// Made with Bob
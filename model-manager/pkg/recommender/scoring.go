package recommender

import (
	"fmt"
	"math"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// ScoringEngine calculates scores for model recommendations
type ScoringEngine struct {
	weights types.ScoringWeights
}

// NewScoringEngine creates a new scoring engine with default weights
func NewScoringEngine() *ScoringEngine {
	return &ScoringEngine{
		weights: types.DefaultScoringWeights(),
	}
}

// NewScoringEngineWithWeights creates a scoring engine with custom weights
func NewScoringEngineWithWeights(weights types.ScoringWeights) *ScoringEngine {
	return &ScoringEngine{
		weights: weights,
	}
}

// ScoreModel calculates a composite score for a model
func (e *ScoringEngine) ScoreModel(
	model types.ModelMetadata,
	compat types.ModelCompatibility,
	device types.DeviceProfile,
	needs types.UserNeeds,
) (float64, types.ScoreBreakdown) {
	breakdown := types.ScoreBreakdown{
		DeviceFit:     e.scoreDeviceFit(compat, device, needs),
		TaskAlignment: e.scoreTaskAlignment(model, needs),
		Quality:       e.scoreQuality(model, needs),
		Efficiency:    e.scoreEfficiency(compat, model, needs),
		Accessibility: e.scoreAccessibility(model),
	}

	// Calculate weighted composite score
	score := breakdown.DeviceFit*e.weights.DeviceFit +
		breakdown.TaskAlignment*e.weights.TaskAlignment +
		breakdown.Quality*e.weights.Quality +
		breakdown.Efficiency*e.weights.Efficiency +
		breakdown.Accessibility*e.weights.Accessibility

	return score, breakdown
}

// scoreDeviceFit scores how well the model fits the device
func (e *ScoringEngine) scoreDeviceFit(compat types.ModelCompatibility, device types.DeviceProfile, needs types.UserNeeds) float64 {
	if !compat.Compatible {
		return 0.0
	}

	// Base score from compatibility confidence
	score := compat.Confidence

	// Adjust based on memory fit
	memoryRatio := compat.EstimatedMemoryGB / device.Memory.TotalGB
	if memoryRatio > 0.8 {
		// Using >80% of memory, penalize
		score *= 0.7
	} else if memoryRatio > 0.6 {
		// Using >60% of memory, slight penalty
		score *= 0.9
	}

	// Adjust for performance based on latency tolerance
	// High latency tolerance (e.g., batch jobs) = don't penalize slow models
	// Low latency tolerance (e.g., interactive) = favor fast models
	if needs.Declared.LatencyToleranceMS > 0 {
		if needs.Declared.LatencyToleranceMS >= 5000 {
			// High tolerance (5+ seconds) - batch jobs, don't care about speed
			// No performance adjustment
		} else if needs.Declared.LatencyToleranceMS >= 2000 {
			// Medium tolerance (2-5 seconds) - slight preference for speed
			if compat.EstimatedTokensPerSec > 50 {
				score *= 1.05
			} else if compat.EstimatedTokensPerSec < 15 {
				score *= 0.95
			}
		} else {
			// Low tolerance (<2 seconds) - strong preference for speed
			if compat.EstimatedTokensPerSec > 30 {
				score *= 1.1
			} else if compat.EstimatedTokensPerSec < 10 {
				score *= 0.8
			}
		}
	} else {
		// Default behavior (medium tolerance)
		if compat.EstimatedTokensPerSec > 30 {
			score *= 1.1
		} else if compat.EstimatedTokensPerSec < 10 {
			score *= 0.8
		}
	}

	return math.Min(score, 1.0)
}

// scoreTaskAlignment scores how well the model aligns with user tasks
func (e *ScoringEngine) scoreTaskAlignment(model types.ModelMetadata, needs types.UserNeeds) float64 {
	if len(needs.Combined.Tasks) == 0 {
		// No task preferences, use neutral score
		return 0.7
	}

	var totalScore float64
	var totalWeight float64

	for task, importance := range needs.Combined.Tasks {
		if capability, ok := model.Tasks[task]; ok {
			totalScore += capability * importance
			totalWeight += importance
		}
	}

	if totalWeight == 0 {
		return 0.5 // No matching tasks
	}

	return totalScore / totalWeight
}

// scoreQuality scores the model's quality tier and parameter count
func (e *ScoringEngine) scoreQuality(model types.ModelMetadata, needs types.UserNeeds) float64 {
	// Base quality from tier
	qualityScores := map[string]float64{
		"base":    0.5,
		"instruct": 0.7,
		"chat":    0.8,
		"premium": 1.0,
	}

	tierScore, ok := qualityScores[model.QualityTier]
	if !ok {
		tierScore = 0.6 // Default for unknown tiers
	}

	// Parameter count quality multiplier
	// Larger models generally produce better quality
	paramScore := e.scoreParameterQuality(model.ParameterCount)
	
	// Combine tier and parameter scores (70% tier, 30% params)
	baseScore := tierScore*0.7 + paramScore*0.3

	// Adjust based on user quality preference
	prefScores := map[string]float64{
		"low":     0.5,
		"medium":  0.7,
		"high":    0.9,
		"premium": 1.0,
	}

	if prefScore, ok := prefScores[needs.Declared.QualityPreference]; ok {
		// Calculate quality alignment: how well does the model match the preference?
		// High preference + high quality model = boost
		// High preference + low quality model = penalize
		// Low preference + any quality = less penalty for lower quality
		
		qualityDiff := baseScore - prefScore
		
		if needs.Declared.QualityPreference == "high" || needs.Declared.QualityPreference == "premium" {
			// User wants high quality - reward high-quality models, penalize low-quality
			if qualityDiff >= 0 {
				// Model meets or exceeds preference - boost it
				score := baseScore * (1.0 + qualityDiff*0.4)
				return math.Min(score, 1.0)
			} else {
				// Model is below preference - penalize it significantly
				score := baseScore * (1.0 + qualityDiff*0.6)
				return math.Max(score, 0.3)
			}
		} else if needs.Declared.QualityPreference == "low" {
			// User wants low quality - don't penalize lower-quality models
			// Actually boost smaller/lower-quality models slightly
			if baseScore <= 0.7 {
				// Lower quality models get a boost
				score := baseScore * 1.1
				return math.Min(score, 1.0)
			}
			// Higher quality models don't get penalized, just no boost
			return baseScore
		}
		// Medium preference - use base score with minimal adjustment
		return baseScore
	}

	return baseScore
}

// scoreParameterQuality scores based on parameter count
// Larger models generally produce higher quality outputs
func (e *ScoringEngine) scoreParameterQuality(paramCount string) float64 {
	paramScores := map[string]float64{
		"0.5B": 0.50,
		"1B":   0.55,
		"1.7B": 0.60,
		"3B":   0.70,
		"7B":   0.80,
		"8B":   0.82,
		"13B":  0.88,
		"14B":  0.89,
		"30B":  0.94,
		"32B":  0.95,
		"70B":  0.98,
		"72B":  0.99,
	}
	
	if score, ok := paramScores[paramCount]; ok {
		return score
	}
	
	return 0.65 // Default for unknown sizes
}

// scoreEfficiency scores the model's efficiency (size vs capability)
func (e *ScoringEngine) scoreEfficiency(compat types.ModelCompatibility, model types.ModelMetadata, needs types.UserNeeds) float64 {
	// Efficiency can mean different things based on use case:
	// - For latency-sensitive: tokens/sec per GB (speed efficiency)
	// - For batch jobs: quality per GB (resource efficiency)
	
	if model.DownloadSizeGB == 0 {
		return 0.5
	}

	// Determine if user cares about speed
	highLatencyTolerance := needs.Declared.LatencyToleranceMS >= 5000
	
	if highLatencyTolerance {
		// For batch jobs: efficiency = quality per GB of storage
		// Favor larger, higher-quality models that use resources well
		paramQuality := e.scoreParameterQuality(model.ParameterCount)
		efficiency := paramQuality / model.DownloadSizeGB
		
		// Normalize to 0-1 range (assuming 0.05 quality/GB is excellent)
		score := math.Min(efficiency/0.05, 1.0)
		return score
	} else {
		// For interactive use: efficiency = tokens/sec per GB (speed efficiency)
		// Favor smaller, faster models
		efficiency := float64(compat.EstimatedTokensPerSec) / model.DownloadSizeGB
		
		// Normalize to 0-1 range (assuming 10 tokens/sec/GB is excellent)
		score := math.Min(efficiency/10.0, 1.0)
		return score
	}
}

// scoreAccessibility scores how easy it is to access/install the model
func (e *ScoringEngine) scoreAccessibility(model types.ModelMetadata) float64 {
	score := 1.0

	// Penalize very large downloads
	if model.DownloadSizeGB > 20 {
		score *= 0.6
	} else if model.DownloadSizeGB > 10 {
		score *= 0.8
	}

	// Boost for permissive licenses
	permissiveLicenses := []string{"apache-2.0", "mit", "bsd"}
	for _, license := range permissiveLicenses {
		if model.License == license {
			score *= 1.1
			break
		}
	}

	return math.Min(score, 1.0)
}

// RankModels ranks a list of models and generates recommendations
func (e *ScoringEngine) RankModels(
	compatibilities []types.ModelCompatibility,
	device types.DeviceProfile,
	needs types.UserNeeds,
) []types.ModelRecommendation {
	recommendations := make([]types.ModelRecommendation, 0, len(compatibilities))

	for _, compat := range compatibilities {
		if !compat.Compatible {
			continue // Skip incompatible models
		}

		score, breakdown := e.ScoreModel(compat.Model, compat, device, needs)

		rec := types.ModelRecommendation{
			Model:          compat.Model,
			Score:          score,
			ScoreBreakdown: breakdown,
			Explanation:    e.generateExplanation(compat, breakdown),
			Setup:          e.generateSetupInfo(compat),
		}

		recommendations = append(recommendations, rec)
	}

	// Sort by score (descending)
	for i := 0; i < len(recommendations); i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Score > recommendations[i].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	// Assign ranks
	for i := range recommendations {
		recommendations[i].Rank = i + 1
	}

	return recommendations
}

// generateExplanation creates a human-readable explanation
func (e *ScoringEngine) generateExplanation(
	compat types.ModelCompatibility,
	breakdown types.ScoreBreakdown,
) types.Explanation {
	strengths := []string{}
	tradeoffs := []string{}

	// Analyze strengths
	if breakdown.DeviceFit > 0.8 {
		strengths = append(strengths, "Excellent fit for your device hardware")
	}
	if breakdown.TaskAlignment > 0.8 {
		strengths = append(strengths, "Well-suited for your typical tasks")
	}
	if breakdown.Efficiency > 0.8 {
		strengths = append(strengths, "Highly efficient (good performance for size)")
	}
	if compat.EstimatedTokensPerSec > 30 {
		strengths = append(strengths, fmt.Sprintf("Fast inference (~%d tokens/sec)", compat.EstimatedTokensPerSec))
	}

	// Analyze tradeoffs
	if breakdown.Quality < 0.7 {
		tradeoffs = append(tradeoffs, "Lower quality tier - may produce less accurate results")
	}
	if breakdown.DeviceFit < 0.7 {
		tradeoffs = append(tradeoffs, "May strain device resources")
	}
	if compat.EstimatedMemoryGB > 8 {
		tradeoffs = append(tradeoffs, fmt.Sprintf("Requires %.1f GB of memory", compat.EstimatedMemoryGB))
	}

	summary := fmt.Sprintf("Score: %.2f - ", breakdown.DeviceFit+breakdown.TaskAlignment+breakdown.Quality+breakdown.Efficiency+breakdown.Accessibility)
	if len(strengths) > 0 {
		summary += strengths[0]
	} else {
		summary += "Balanced option for your device"
	}

	return types.Explanation{
		Summary:      summary,
		Strengths:    strengths,
		Tradeoffs:    tradeoffs,
		Alternatives: []string{}, // Would be populated with similar models
	}
}

// generateSetupInfo creates installation instructions
func (e *ScoringEngine) generateSetupInfo(compat types.ModelCompatibility) types.SetupInfo {
	estimatedMinutes := int(compat.Model.DownloadSizeGB * 2) // Rough estimate: 2 min per GB
	
	return types.SetupInfo{
		InstallCommand: fmt.Sprintf("model-manager install %s --format %s --quantization %s",
			compat.Model.Name,
			compat.RecommendedFormat,
			compat.RecommendedQuantization),
		EstimatedTime: fmt.Sprintf("%d minutes", estimatedMinutes),
		Requirements: []string{
			fmt.Sprintf("%.1f GB free storage", compat.Model.DownloadSizeGB),
			fmt.Sprintf("%.1f GB available memory", compat.EstimatedMemoryGB),
		},
		PostInstall: []string{
			"Model will be available for inference immediately",
			"Configure edge-router to use this model",
		},
	}
}

// Made with Bob

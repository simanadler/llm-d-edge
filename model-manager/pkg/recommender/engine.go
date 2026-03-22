package recommender

import (
	"context"
	"fmt"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/platform"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// Engine is the main recommendation engine
type Engine struct {
	profiler      platform.DeviceProfiler
	matcher       *ModelMatcher
	scoringEngine *ScoringEngine
}

// NewEngine creates a new recommendation engine
func NewEngine(platformName string) (*Engine, error) {
	profiler, err := platform.NewDeviceProfiler(platformName)
	if err != nil {
		return nil, fmt.Errorf("failed to create device profiler: %w", err)
	}

	return &Engine{
		profiler:      profiler,
		matcher:       NewModelMatcher(),
		scoringEngine: NewScoringEngine(),
	}, nil
}

// NewEngineWithCustomWeights creates an engine with custom scoring weights
func NewEngineWithCustomWeights(platformName string, weights types.ScoringWeights) (*Engine, error) {
	profiler, err := platform.NewDeviceProfiler(platformName)
	if err != nil {
		return nil, fmt.Errorf("failed to create device profiler: %w", err)
	}

	return &Engine{
		profiler:      profiler,
		matcher:       NewModelMatcher(),
		scoringEngine: NewScoringEngineWithWeights(weights),
	}, nil
}

// GenerateRecommendations generates model recommendations for the device
func (e *Engine) GenerateRecommendations(ctx context.Context, needs types.UserNeeds) ([]types.ModelRecommendation, error) {
	// Step 1: Profile the device
	deviceProfile, err := e.profiler.Profile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to profile device: %w", err)
	}

	// Step 2: Adapt scoring weights based on device capabilities
	adaptiveWeights := types.AdaptiveScoringWeights(deviceProfile.Memory.TotalGB)
	e.scoringEngine = NewScoringEngineWithWeights(adaptiveWeights)

	// Step 3: Find compatible models
	compatibilities, err := e.matcher.FindCandidates(ctx, *deviceProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to find model candidates: %w", err)
	}

	if len(compatibilities) == 0 {
		return nil, fmt.Errorf("no compatible models found for this device")
	}

	// Step 4: Score and rank models
	recommendations := e.scoringEngine.RankModels(compatibilities, *deviceProfile, needs)

	return recommendations, nil
}

// ProfileDevice profiles the current device without generating recommendations
func (e *Engine) ProfileDevice(ctx context.Context) (*types.DeviceProfile, error) {
	return e.profiler.Profile(ctx)
}

// CheckModelCompatibility checks if a specific model is compatible
func (e *Engine) CheckModelCompatibility(ctx context.Context, model types.ModelMetadata) (*types.ModelCompatibility, error) {
	deviceProfile, err := e.profiler.Profile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to profile device: %w", err)
	}

	return e.matcher.CheckCompatibility(ctx, model, *deviceProfile)
}

// Made with Bob

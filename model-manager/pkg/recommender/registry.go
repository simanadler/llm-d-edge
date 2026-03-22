package recommender

import (
	"context"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// ModelRegistry is an interface for discovering models from various sources
type ModelRegistry interface {
	// GetModels returns available models from this registry
	GetModels(ctx context.Context) ([]types.ModelMetadata, error)
	
	// SearchModels searches for models matching a query
	SearchModels(ctx context.Context, query string, limit int) ([]types.ModelMetadata, error)
	
	// GetModel retrieves metadata for a specific model
	GetModel(ctx context.Context, modelID string) (*types.ModelMetadata, error)
	
	// Name returns the registry name
	Name() string
}

// CompositeRegistry combines multiple model registries
type CompositeRegistry struct {
	registries []ModelRegistry
}

// NewCompositeRegistry creates a registry that queries multiple sources
func NewCompositeRegistry(registries ...ModelRegistry) *CompositeRegistry {
	return &CompositeRegistry{
		registries: registries,
	}
}

// GetModels returns models from all registries
func (r *CompositeRegistry) GetModels(ctx context.Context) ([]types.ModelMetadata, error) {
	allModels := make([]types.ModelMetadata, 0)
	seen := make(map[string]bool)
	
	for _, registry := range r.registries {
		models, err := registry.GetModels(ctx)
		if err != nil {
			// Log error but continue with other registries
			continue
		}
		
		// Deduplicate by HuggingFaceRepo or Name
		for _, model := range models {
			key := model.HuggingFaceRepo
			if key == "" {
				key = model.Name
			}
			
			if !seen[key] {
				allModels = append(allModels, model)
				seen[key] = true
			}
		}
	}
	
	return allModels, nil
}

// SearchModels searches across all registries
func (r *CompositeRegistry) SearchModels(ctx context.Context, query string, limit int) ([]types.ModelMetadata, error) {
	allModels := make([]types.ModelMetadata, 0)
	seen := make(map[string]bool)
	
	for _, registry := range r.registries {
		models, err := registry.SearchModels(ctx, query, limit)
		if err != nil {
			continue
		}
		
		for _, model := range models {
			key := model.HuggingFaceRepo
			if key == "" {
				key = model.Name
			}
			
			if !seen[key] {
				allModels = append(allModels, model)
				seen[key] = true
				
				if len(allModels) >= limit {
					return allModels, nil
				}
			}
		}
	}
	
	return allModels, nil
}

// GetModel tries to find a model in any registry
func (r *CompositeRegistry) GetModel(ctx context.Context, modelID string) (*types.ModelMetadata, error) {
	for _, registry := range r.registries {
		model, err := registry.GetModel(ctx, modelID)
		if err == nil && model != nil {
			return model, nil
		}
	}
	
	return nil, nil
}

// Name returns the composite registry name
func (r *CompositeRegistry) Name() string {
	return "composite"
}

// AddRegistry adds a new registry to the composite
func (r *CompositeRegistry) AddRegistry(registry ModelRegistry) {
	r.registries = append(r.registries, registry)
}

// Made with Bob

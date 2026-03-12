package stub

import (
	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"go.uber.org/zap"
)

func init() {
	// Register stub engine factory for simulating local inference
	// Register under multiple names for compatibility
	engine.RegisterEngine("stub", func() engine.InferenceEngine {
		logger := zap.NewNop()
		return NewStubEngine(logger)
	})
	engine.RegisterEngine("local-stub", func() engine.InferenceEngine {
		logger := zap.NewNop()
		return NewStubEngine(logger)
	})
	engine.RegisterEngine("test", func() engine.InferenceEngine {
		logger := zap.NewNop()
		return NewStubEngine(logger)
	})
}

// Made with Bob

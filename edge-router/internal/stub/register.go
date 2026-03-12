package stub

import (
	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"go.uber.org/zap"
)

func init() {
	// Register local-stub engine factory for simulating local inference
	engine.RegisterEngine("local-stub", func() engine.InferenceEngine {
		// Create a no-op logger for the local-stub engine
		logger := zap.NewNop()
		return NewStubEngine(logger)
	})
}

// Made with Bob

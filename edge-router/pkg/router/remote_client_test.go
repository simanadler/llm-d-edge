package router

import (
	"testing"

	"github.com/llm-d-incubation/llm-d-edge/pkg/config"
	"go.uber.org/zap"
)

func TestGetModelNameWithoutProvider(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Qwen model with provider",
			input:    "Qwen/Qwen2.5-72B-Instruct",
			expected: "qwen2-5-72b-instruct",
		},
		{
			name:     "Model without provider",
			input:    "Qwen2.5-72B-Instruct",
			expected: "qwen2-5-72b-instruct",
		},
		{
			name:     "Meta Llama model",
			input:    "meta-llama/Llama-3.1-8B-Instruct",
			expected: "llama-3-1-8b-instruct",
		},
		{
			name:     "Simple model name",
			input:    "gpt-4",
			expected: "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModelNameWithoutProvider(tt.input)
			if result != tt.expected {
				t.Errorf("getModelNameWithoutProvider(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildClusterURL(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name       string
		clusterURL string
		model      string
		expected   string
	}{
		{
			name:       "RITS URL with Qwen model",
			clusterURL: "https://inference-3scale-apicast-production.apps.rits.fmaas.res.ibm.com",
			model:      "Qwen/Qwen2.5-72B-Instruct",
			expected:   "https://inference-3scale-apicast-production.apps.rits.fmaas.res.ibm.com/qwen2-5-72b-instruct",
		},
		{
			name:       "RITS URL with trailing slash",
			clusterURL: "https://inference-3scale-apicast-production.apps.rits.fmaas.res.ibm.com/",
			model:      "Qwen/Qwen2.5-72B-Instruct",
			expected:   "https://inference-3scale-apicast-production.apps.rits.fmaas.res.ibm.com/qwen2-5-72b-instruct",
		},
		{
			name:       "RITS URL with model already in path",
			clusterURL: "https://inference-3scale-apicast-production.apps.rits.fmaas.res.ibm.com/qwen2-5-72b-instruct",
			model:      "Qwen/Qwen2.5-72B-Instruct",
			expected:   "https://inference-3scale-apicast-production.apps.rits.fmaas.res.ibm.com/qwen2-5-72b-instruct",
		},
		{
			name:       "Non-RITS URL should not be modified",
			clusterURL: "https://api.openai.com",
			model:      "Qwen/Qwen2.5-72B-Instruct",
			expected:   "https://api.openai.com",
		},
		{
			name:       "RITS URL with different model",
			clusterURL: "https://inference-3scale-apicast-production.apps.rits.fmaas.res.ibm.com",
			model:      "meta-llama/Llama-3.1-8B-Instruct",
			expected:   "https://inference-3scale-apicast-production.apps.rits.fmaas.res.ibm.com/llama-3-1-8b-instruct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &RemoteClient{
				config: config.RemoteClusterConfig{
					ClusterURL: tt.clusterURL,
				},
				logger: logger,
			}

			result := rc.buildClusterURL(tt.model)
			if result != tt.expected {
				t.Errorf("buildClusterURL(%q, %q) = %q, want %q", tt.clusterURL, tt.model, result, tt.expected)
			}
		})
	}
}

// Made with Bob

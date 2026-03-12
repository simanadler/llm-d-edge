package router

import (
	"sync"
	"time"
)

// Metrics tracks router performance metrics
type Metrics struct {
	mu sync.RWMutex

	// Routing decisions
	totalRoutingDecisions int64
	routingDecisionTimeMs []int64

	// Inference metrics
	localInferences       int64
	remoteInferences      int64
	localSuccesses        int64
	remoteSuccesses       int64
	localFailures         int64
	remoteFailures        int64
	localLatencyMs        []int64
	remoteLatencyMs       []int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		routingDecisionTimeMs: make([]int64, 0, 1000),
		localLatencyMs:        make([]int64, 0, 1000),
		remoteLatencyMs:       make([]int64, 0, 1000),
	}
}

// RecordRoutingDecision records a routing decision
func (m *Metrics) RecordRoutingDecision(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRoutingDecisions++
	m.routingDecisionTimeMs = append(m.routingDecisionTimeMs, duration.Milliseconds())

	// Keep only last 1000 samples
	if len(m.routingDecisionTimeMs) > 1000 {
		m.routingDecisionTimeMs = m.routingDecisionTimeMs[1:]
	}
}

// RecordInference records an inference result
func (m *Metrics) RecordInference(target RouteTarget, latencyMs int64, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if target == TargetLocal {
		m.localInferences++
		if success {
			m.localSuccesses++
		} else {
			m.localFailures++
		}
		m.localLatencyMs = append(m.localLatencyMs, latencyMs)
		if len(m.localLatencyMs) > 1000 {
			m.localLatencyMs = m.localLatencyMs[1:]
		}
	} else {
		m.remoteInferences++
		if success {
			m.remoteSuccesses++
		} else {
			m.remoteFailures++
		}
		m.remoteLatencyMs = append(m.remoteLatencyMs, latencyMs)
		if len(m.remoteLatencyMs) > 1000 {
			m.remoteLatencyMs = m.remoteLatencyMs[1:]
		}
	}
}

// GetStats returns current statistics
func (m *Metrics) GetStats() MetricsStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MetricsStats{
		TotalRoutingDecisions:    m.totalRoutingDecisions,
		AvgRoutingDecisionTimeMs: m.calculateAverage(m.routingDecisionTimeMs),
		LocalInferences:          m.localInferences,
		RemoteInferences:         m.remoteInferences,
		LocalSuccessRate:         m.calculateSuccessRate(m.localSuccesses, m.localInferences),
		RemoteSuccessRate:        m.calculateSuccessRate(m.remoteSuccesses, m.remoteInferences),
		AvgLocalLatencyMs:        m.calculateAverage(m.localLatencyMs),
		AvgRemoteLatencyMs:       m.calculateAverage(m.remoteLatencyMs),
		P50LocalLatencyMs:        m.calculatePercentile(m.localLatencyMs, 0.5),
		P90LocalLatencyMs:        m.calculatePercentile(m.localLatencyMs, 0.9),
		P99LocalLatencyMs:        m.calculatePercentile(m.localLatencyMs, 0.99),
		P50RemoteLatencyMs:       m.calculatePercentile(m.remoteLatencyMs, 0.5),
		P90RemoteLatencyMs:       m.calculatePercentile(m.remoteLatencyMs, 0.9),
		P99RemoteLatencyMs:       m.calculatePercentile(m.remoteLatencyMs, 0.99),
	}
}

// MetricsStats contains aggregated metrics
type MetricsStats struct {
	TotalRoutingDecisions    int64
	AvgRoutingDecisionTimeMs float64
	LocalInferences          int64
	RemoteInferences         int64
	LocalSuccessRate         float64
	RemoteSuccessRate        float64
	AvgLocalLatencyMs        float64
	AvgRemoteLatencyMs       float64
	P50LocalLatencyMs        int64
	P90LocalLatencyMs        int64
	P99LocalLatencyMs        int64
	P50RemoteLatencyMs       int64
	P90RemoteLatencyMs       int64
	P99RemoteLatencyMs       int64
}

func (m *Metrics) calculateAverage(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := int64(0)
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func (m *Metrics) calculateSuccessRate(successes, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(successes) / float64(total) * 100
}

func (m *Metrics) calculatePercentile(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}

	// Simple percentile calculation (not sorting for performance)
	// In production, use a proper percentile algorithm
	sorted := make([]int64, len(values))
	copy(sorted, values)

	// Bubble sort (simple but inefficient - use quicksort in production)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}

// Made with Bob

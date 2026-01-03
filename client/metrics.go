package client

import (
	"sync"
	"time"
)

// Metrics 指标收集器
type Metrics struct {
	mu                   sync.RWMutex
	totalRequests        int64
	successfulRequests   int64
	failedRequests       int64
	totalRequestDuration time.Duration
	reconnectCount       int64
	circuitBreakerState  CircuitBreakerState
	lastRequestTimestamp time.Time
}

// NewMetrics 创建新的指标收集器
func NewMetrics() *Metrics {
	return &Metrics{
		lastRequestTimestamp: time.Now(),
	}
}

// RecordRequest 记录请求指标
func (m *Metrics) RecordRequest(success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	if success {
		m.successfulRequests++
	} else {
		m.failedRequests++
	}
	m.totalRequestDuration += duration
	m.lastRequestTimestamp = time.Now()
}

// RecordReconnect 记录重连指标
func (m *Metrics) RecordReconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconnectCount++
}

// UpdateCircuitBreakerState 更新熔断器状态指标
func (m *Metrics) UpdateCircuitBreakerState(state CircuitBreakerState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.circuitBreakerState = state
}

// GetMetrics 获取当前指标快照
func (m *Metrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgDuration time.Duration
	if m.totalRequests > 0 {
		avgDuration = time.Duration(int64(m.totalRequestDuration) / m.totalRequests)
	}

	return map[string]interface{}{
		"total_requests":              m.totalRequests,
		"successful_requests":         m.successfulRequests,
		"failed_requests":             m.failedRequests,
		"success_rate":                float64(m.successfulRequests) / float64(m.totalRequests) * 100,
		"average_request_duration_ms": avgDuration.Milliseconds(),
		"reconnect_count":             m.reconnectCount,
		"circuit_breaker_state":       m.circuitBreakerState.String(),
		"last_request_timestamp":      m.lastRequestTimestamp.Format(time.RFC3339),
	}
}
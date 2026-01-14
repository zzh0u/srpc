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



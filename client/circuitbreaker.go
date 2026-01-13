package client

import (
	"sync"
	"time"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	CBStateClosed   CircuitBreakerState = iota // 关闭：允许请求
	CBStateOpen                                // 开启：拒绝请求
	CBStateHalfOpen                            // 半开：尝试部分请求
)

// String 方法用于CircuitBreakerState
func (s CircuitBreakerState) String() string {
	switch s {
	case CBStateClosed:
		return "CLOSED"
	case CBStateOpen:
		return "OPEN"
	case CBStateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	state             CircuitBreakerState
	failureCount      int
	successCount      int
	lastStateChange   time.Time
	openDuration      time.Duration
	failureThreshold  int
	successThreshold  int
	halfOpenMaxCalls  int
	halfOpenCallCount int
	mu                sync.RWMutex
}

// NewCircuitBreaker 创建新的熔断器
func NewCircuitBreaker(failureThreshold, successThreshold int, openDuration time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CBStateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		openDuration:     openDuration,
		halfOpenMaxCalls: 3, // 半开状态下允许的最大请求数
	}
}

// AllowRequest 检查是否允许请求
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CBStateClosed:
		return true
	case CBStateOpen:
		// 检查是否应该切换到半开状态
		if time.Since(cb.lastStateChange) >= cb.openDuration {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = CBStateHalfOpen
			cb.halfOpenCallCount = 0
			cb.lastStateChange = time.Now()
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case CBStateHalfOpen:
		if cb.halfOpenCallCount < cb.halfOpenMaxCalls {
			return true
		}
		return false
	default:
		return false
	}
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CBStateClosed:
		cb.successCount++
		cb.failureCount = 0
	case CBStateHalfOpen:
		cb.halfOpenCallCount++
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = CBStateClosed
			cb.successCount = 0
			cb.failureCount = 0
			cb.lastStateChange = time.Now()
		}
	default:
		// TODO
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CBStateClosed:
		cb.failureCount++
		cb.successCount = 0
		if cb.failureCount >= cb.failureThreshold {
			cb.state = CBStateOpen
			cb.lastStateChange = time.Now()
		}
	case CBStateHalfOpen:
		cb.halfOpenCallCount++
		cb.failureCount++
		cb.successCount = 0
		cb.state = CBStateOpen
		cb.lastStateChange = time.Now()
	default:
		// TODO
	}
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

package client

import "time"

// Semaphore 信号量，用于资源管理
type Semaphore struct {
	sem chan struct{}
}

// NewSemaphore 创建新的信号量
func NewSemaphore(maxConcurrent int) *Semaphore {
	return &Semaphore{
		sem: make(chan struct{}, maxConcurrent),
	}
}

// Acquire 获取信号量
func (s *Semaphore) Acquire() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release 释放信号量
func (s *Semaphore) Release() {
	<-s.sem
}

// TryAcquire 尝试获取信号量，带超时
func (s *Semaphore) TryAcquire(timeout time.Duration) bool {
	select {
	case s.sem <- struct{}{}:
		return true
	case <-time.After(timeout):
		return false
	}
}
package client

import (
	"time"
)

// executeWithRetry 执行带重试的操作
func (c *GRPCClient) executeWithRetry(operation func() error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// 检查是否正在关闭
		if c.IsShutting() {
			c.slogger.Info("客户端正在关闭，取消重试")
			return
		}

		// 如果不是第一次尝试，等待重试延迟
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second // 指数退避: 1,4,9秒...
			if backoff > 10*time.Second {
				backoff = 10 * time.Second // 最大10秒
			}
			c.slogger.Info("重试等待", map[string]interface{}{"attempt": attempt, "backoff": backoff})
			time.Sleep(backoff)
		}

		// 执行操作
		err := operation()
		if err == nil {
			return // 成功
		}

		lastErr = err

		// 检查是否是致命错误（无需重试）
		if isFatalError(err) {
			c.slogger.Error("遇到致命错误，停止重试", map[string]interface{}{"error": err})
			break
		}

		// 如果是最后一次尝试，退出循环
		if attempt == c.config.MaxRetries {
			c.slogger.Error("达到最大重试次数，最终失败", map[string]interface{}{"max_retries": c.config.MaxRetries, "error": err})
			break
		}

		c.slogger.Warn("请求失败，准备重试", map[string]interface{}{"current_attempt": attempt + 1, "total_attempts": c.config.MaxRetries + 1, "error": err})
	}

	if lastErr != nil {
		c.slogger.Error("所有重试尝试均失败", map[string]interface{}{"error": lastErr})
	}
}

// isFatalError 检查是否为致命错误（无需重试）
func isFatalError(err error) bool {
	// TODO: 可以根据具体的gRPC错误码来判断
	// 例如：无效参数、权限拒绝等错误无需重试
	return false
}

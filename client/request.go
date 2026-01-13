package client

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/grpc/metadata"

	pb "srpc/proto"
)

// calculateJitteredInterval 计算带抖动的间隔时间
func (c *GRPCClient) calculateJitteredInterval() time.Duration {
	if c.config.JitterPercent <= 0 {
		return c.config.RequestInterval
	}

	// 计算抖动的范围
	jitterRange := float64(c.config.JitterPercent) / 100.0 * float64(c.config.RequestInterval)

	// 生成随机抖动值（-jitterRange/2 到 +jitterRange/2）
	rand.Seed(time.Now().UnixNano())
	jitter := rand.Float64()*jitterRange - jitterRange/2

	// 计算最终间隔
	interval := float64(c.config.RequestInterval) + jitter

	// 确保间隔不小于 1 毫秒
	if interval < float64(time.Millisecond) {
		return time.Millisecond
	}

	return time.Duration(interval)
}

// mainLoop 主循环
func (c *GRPCClient) mainLoop() {
	defer c.wg.Done()

	for {
		// 计算带抖动的等待时间
		waitInterval := c.calculateJitteredInterval()

		select {
		case <-c.ctx.Done():
			c.slogger.Info("主循环收到关闭信号，正在退出")
			return
		case <-time.After(waitInterval):
			c.makeRequest()
		}
	}
}

// makeRequest 发起 gRPC 请求
func (c *GRPCClient) makeRequest() {
	c.mu.RLock()
	isShutting := c.isShutting
	state := c.connectionState
	c.mu.RUnlock()

	if isShutting {
		return
	}

	// 检查熔断器
	if !c.circuitBreaker.AllowRequest() {
		cbState := c.circuitBreaker.GetState()
		c.slogger.Info("熔断器状态，跳过本次请求", map[string]interface{}{"circuit_breaker_state": cbState})
		return
	}

	// 检查连接状态
	switch state {
	case StateDisconnected:
		c.slogger.Info("连接已断开，跳过本次请求")
		return
	case StateConnecting:
		c.slogger.Info("正在连接中，跳过本次请求")
		return
	case StateDegraded:
		c.slogger.Info("连接降级，尝试恢复")
		c.reconnect()
		return
	case StateConnected:
		// 连接正常，执行请求
		c.executeSayHello()
	default:
		c.slogger.Warn("未知连接状态", map[string]interface{}{"state": state})
	}
}

// executeSayHello 执行 SayHello RPC 调用
func (c *GRPCClient) executeSayHello() {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	// 生成请求 ID（如果启用）
	var requestID string
	if c.config.GenerateRequestID && c.idGenerator != nil {
		requestID = c.idGenerator.Generate()
		// 将请求 ID 添加到 context metadata 中，以便服务端追踪
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	}

	// 创建请求
	req := &pb.HelloRequest{
		Name: fmt.Sprintf("Client-%d", time.Now().Unix()),
	}

	// 执行带重试的请求
	c.executeWithRetry(func() error {
		start := time.Now()
		resp, err := c.greeter.SayHello(ctx, req)
		elapsed := time.Since(start)

		// 构建日志字段
		logFields := map[string]interface{}{
			"duration":  elapsed.String(),
			"operation": "SayHello",
		}
		if requestID != "" {
			logFields["request_id"] = requestID
		}

		if err != nil {
			logFields["error"] = err.Error()
			c.slogger.Error("SayHello请求失败", logFields)
			// 记录熔断器失败
			c.circuitBreaker.RecordFailure()
			// 记录指标
			c.metrics.RecordRequest(false, elapsed)
			return err
		}

		logFields["response"] = resp.GetMessage()
		c.slogger.Info("SayHello请求成功", logFields)
		// 记录熔断器成功
		c.circuitBreaker.RecordSuccess()
		// 记录指标
		c.metrics.RecordRequest(true, elapsed)
		return nil
	})
}

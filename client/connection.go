package client

import (
	"context"
	"fmt"
	pb "srpc/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ConnectionState 连接状态
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota // 断开连接
	StateConnecting                          // 连接中
	StateConnected                           // 已连接
	StateDegraded                            // 降级（部分功能不可用）
)

// connect 建立 gRPC 连接
func (c *GRPCClient) connect() error {
	c.mu.Lock()
	c.connectionState = StateConnecting
	c.mu.Unlock()

	// 构建连接选项
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// 如果启用压缩，添加压缩选项
	if c.config.EnableCompression && c.config.CompressionType != "" {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(c.config.CompressionType)))
	}

	c.slogger.Info("正在连接到 gRPC 服务器", map[string]interface{}{
		"server_addr":      c.config.ServerAddr,
		"compression":      c.config.EnableCompression,
		"compression_type": c.config.CompressionType,
	})

	conn, err := grpc.NewClient(c.config.ServerAddr, opts...)
	if err != nil {
		c.mu.Lock()
		c.connectionState = StateDisconnected
		c.lastError = err
		c.mu.Unlock()
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.greeter = pb.NewGreeterClient(conn)
	c.connectionState = StateConnected
	c.lastError = nil
	c.reconnectCount++
	c.mu.Unlock()

	c.slogger.Info("成功连接到 gRPC 服务器", map[string]interface{}{
		"server_addr":     c.config.ServerAddr,
		"reconnect_count": c.reconnectCount,
	})
	return nil
}

// startHealthChecker 启动健康检查
func (c *GRPCClient) startHealthChecker() {
	c.wg.Add(1)
	go c.healthCheckLoop()
}

// healthCheckLoop 健康检查循环
func (c *GRPCClient) healthCheckLoop() {
	defer c.wg.Done()

	healthTicker := time.NewTicker(c.config.KeepAliveInterval)
	defer healthTicker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			c.slogger.Info("健康检查收到关闭信号，正在退出")
			return
		case <-healthTicker.C:
			c.checkConnectionHealth()
		}
	}
}

// checkConnectionHealth 检查连接健康状态
func (c *GRPCClient) checkConnectionHealth() {
	c.mu.RLock()
	state := c.connectionState
	conn := c.conn
	c.mu.RUnlock()

	// 如果正在关闭，跳过健康检查
	if c.IsShutting() {
		return
	}

	// 检查连接状态
	switch state {
	case StateDisconnected:
		c.slogger.Info("连接已断开，尝试重新连接")
		c.reconnect()
	case StateConnected:
		// 执行健康检查请求
		if conn != nil {
			ctx, cancel := context.WithTimeout(c.ctx, 3*time.Second)
			defer cancel()

			// 发送简单的 SayHello 请求作为健康检查
			req := &pb.HelloRequest{Name: "health-check"}
			_, err := c.greeter.SayHello(ctx, req)
			if err != nil {
				c.slogger.Error("健康检查失败，连接可能已断开", map[string]interface{}{"error": err})
				c.mu.Lock()
				c.connectionState = StateDisconnected
				c.lastError = err
				c.mu.Unlock()
				c.reconnect()
			} else {
				c.slogger.Info("健康检查通过")
			}
		}
	case StateConnecting:
		// 正在连接中，等待完成
		c.slogger.Info("连接中，跳过健康检查")
	case StateDegraded:
		// 降级状态，尝试恢复
		c.slogger.Info("连接降级，尝试恢复")
		c.reconnect()
	}
}

// reconnect 尝试重新连接
func (c *GRPCClient) reconnect() {
	// 检查是否正在关闭
	if c.IsShutting() {
		return
	}

	c.mu.Lock()
	oldConn := c.conn
	c.connectionState = StateConnecting
	c.mu.Unlock()

	// 关闭旧连接
	if oldConn != nil {
		oldConn.Close()
	}

	// 尝试重新连接
	var retryCount int
	maxReconnectRetries := 5

	for retryCount < maxReconnectRetries {
		if c.IsShutting() {
			return
		}

		c.slogger.Info("重新连接尝试", map[string]interface{}{"current_attempt": retryCount + 1, "max_attempts": maxReconnectRetries})

		err := c.connect()
		if err == nil {
			c.slogger.Info("重新连接成功")
			c.metrics.RecordReconnect()
			return
		}

		c.slogger.Error("重新连接失败", map[string]interface{}{"error": err})

		// 指数退避等待
		backoff := time.Duration(retryCount*retryCount+1) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}

		c.slogger.Info("等待后重试", map[string]interface{}{"backoff": backoff})
		time.Sleep(backoff)
		retryCount++
	}

	// 重连失败
	c.mu.Lock()
	c.connectionState = StateDisconnected
	c.lastError = fmt.Errorf("重连失败，已尝试 %d 次", maxReconnectRetries)
	c.mu.Unlock()

	c.slogger.Error("重连失败，已达到最大重试次数", map[string]interface{}{"max_retries": maxReconnectRetries})
}

// getConnectionState 获取连接状态
// func (c *GRPCClient) getConnectionState() ConnectionState {
// 	c.mu.RLock()
// 	defer c.mu.RUnlock()
// 	return c.connectionState
// }

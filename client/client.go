package client

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	_ "srpc/pkg/compress" // 确保压缩器被注册
	"srpc/pkg/log"
	"srpc/pkg/tools"
	_ "srpc/pkg/tools"
	pb "srpc/proto"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

// Config 客户端配置
type Config struct {
	ServerAddr            string        // gRPC 服务器地址
	KeepAliveInterval     time.Duration // 连接保活间隔
	RequestInterval       time.Duration // 请求间隔时间
	MaxRetries            int           // 最大重试次数
	JitterPercent         int           // 随机抖动百分比（0-100）
	MaxConcurrentRequests int           // 最大并发请求数
	EnableCompression     bool          // 是否启用压缩
	CompressionType       string        // 压缩类型：snappy（目前只支持 snappy）
	GenerateRequestID     bool          // 是否为每个请求生成唯一 ID
}

// GRPCClient gRPC 客户端
type GRPCClient struct {
	config          Config
	conn            *grpc.ClientConn
	greeter         pb.GreeterClient
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	stopChan        chan struct{}
	mu              sync.RWMutex
	isShutting      bool
	connectionState ConnectionState   // 连接状态
	lastError       error             // 最后错误
	reconnectCount  int               // 重连次数
	circuitBreaker  *CircuitBreaker   // 熔断器
	slogger         *log.Slogger      // 日志记录器
	metrics         *Metrics          // 指标收集器
	semaphore       *Semaphore        // 信号量，用于并发控制
	idGenerator     tools.IDGenerator // ID 生成器（如果启用）
}

// NewGRPCClient 创建新的 gRPC 客户端
func NewGRPCClient(config Config) (*GRPCClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 设置默认最大并发请求数
	maxConcurrent := config.MaxConcurrentRequests
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // 默认值
	}

	// 设置压缩类型默认值
	compressionType := config.CompressionType
	if config.EnableCompression && compressionType == "" {
		compressionType = "snappy" // 默认使用 snappy 压缩
	}

	// 初始化 ID 生成器（如果启用）
	var idGenerator tools.IDGenerator
	if config.GenerateRequestID {
		idGenerator = tools.GetDefaultIDGenerator()
	}

	client := &GRPCClient{
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		stopChan:        make(chan struct{}),
		connectionState: StateDisconnected,
		reconnectCount:  0,
		circuitBreaker:  NewCircuitBreaker(5, 3, 30*time.Second), // 5次失败触发，3次成功恢复，开启30秒
		slogger:         log.NewLogger(),
		metrics:         NewMetrics(),
		semaphore:       NewSemaphore(maxConcurrent),
		idGenerator:     idGenerator,
	}

	// 更新配置中的压缩类型（如果启用了压缩但类型为空）
	if client.config.EnableCompression && client.config.CompressionType == "" {
		client.config.CompressionType = "snappy"
	}

	// 建立 gRPC 连接
	if err := client.connect(); err != nil {
		// 释放 context 持有的资源，避免资源泄露
		cancel()
		return nil, fmt.Errorf("连接gRPC服务器失败: %v", err)
	}

	// 启动健康检查
	client.startHealthChecker()

	return client, nil
}

// Run 启动客户端主循环
func (c *GRPCClient) Run() error {
	c.slogger.Info("启动 gRPC 客户端", map[string]interface{}{
		"server_addr": c.config.ServerAddr,
	})

	// 启动信号处理
	c.setupSignalHandler()

	// 启动主工作 goroutine
	c.wg.Add(1)
	go c.mainLoop()

	// 等待终止
	c.wg.Wait()

	// 清理资源
	return c.cleanup()
}

// setupSignalHandler 设置信号处理器
func (c *GRPCClient) setupSignalHandler() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		c.slogger.Info("收到信号，开始关闭", map[string]interface{}{"signal": sig})
		c.Shutdown()
	}()
}

// Shutdown 关闭客户端
func (c *GRPCClient) Shutdown() {
	c.mu.Lock()
	if c.isShutting {
		c.mu.Unlock()
		return
	}
	c.isShutting = true
	c.mu.Unlock()

	c.slogger.Info("开始关闭")

	// 发送停止信号
	c.cancel()

	// 等待主循环退出
	close(c.stopChan)

	// 等待所有 goroutine 完成
	c.wg.Wait()
}

// cleanup 清理资源
func (c *GRPCClient) cleanup() error {
	c.slogger.Info("清理资源")

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("关闭gRPC连接失败: %v", err)
		}
		c.slogger.Info("gRPC 连接已关闭")
	}

	c.slogger.Info("客户端已完全关闭")
	return nil
}

// GetConfig 获取配置
func (c *GRPCClient) GetConfig() Config {
	return c.config
}

// IsShutting 检查是否正在关闭
func (c *GRPCClient) IsShutting() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isShutting
}

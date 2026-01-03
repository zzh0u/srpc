package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"srpc/client"
	_ "srpc/pkg/compress" // 确保压缩器被注册
)

func main() {
	slog.Info("启动gRPC客户端")

	// 读取配置
	config := loadConfig()

	// 创建客户端
	grpcClient, err := client.NewGRPCClient(config)
	if err != nil {
		slog.Error("创建gRPC客户端失败", "error", err)
		os.Exit(1)
	}

	// 运行客户端
	if err := grpcClient.Run(); err != nil {
		slog.Error("客户端运行失败", "error", err)
		os.Exit(1)
	}

	slog.Info("客户端已正常退出")
}

// loadConfig 从环境变量加载配置
func loadConfig() client.Config {
	// 获取服务器地址，默认为localhost:50051
	serverAddr := getEnv("GRPC_SERVER_ADDR", "localhost:50051")

	// 获取请求间隔，默认为30秒
	requestIntervalSec := getEnvAsInt("REQUEST_INTERVAL_SEC", 30)
	requestInterval := time.Duration(requestIntervalSec) * time.Second

	// 获取最大重试次数，默认为3
	maxRetries := getEnvAsInt("MAX_RETRIES", 3)

	// 获取连接保活间隔，默认为 20 秒
	keepAliveSec := getEnvAsInt("KEEP_ALIVE_SEC", 20)
	keepAliveInterval := time.Duration(keepAliveSec) * time.Second

	// 获取抖动百分比，默认为10%（0-100）
	jitterPercent := getEnvAsInt("JITTER_PERCENT", 10)
	// 限制在 0-100 范围内
	if jitterPercent < 0 {
		jitterPercent = 0
	} else if jitterPercent > 100 {
		jitterPercent = 100
	}

	// 获取最大并发请求数，默认为 5
	maxConcurrentRequests := getEnvAsInt("MAX_CONCURRENT_REQUESTS", 5)
	if maxConcurrentRequests < 1 {
		maxConcurrentRequests = 1
	}

	// 获取是否启用压缩，默认为 false
	enableCompression := getEnvAsBool("ENABLE_COMPRESSION", false)

	// 获取压缩类型，默认为 snappy
	compressionType := getEnv("COMPRESSION_TYPE", "snappy")

	// 获取是否生成请求ID，默认为 true
	generateRequestID := getEnvAsBool("GENERATE_REQUEST_ID", true)

	return client.Config{
		ServerAddr:            serverAddr,
		RequestInterval:       requestInterval,
		MaxRetries:            maxRetries,
		KeepAliveInterval:     keepAliveInterval,
		JitterPercent:         jitterPercent,
		MaxConcurrentRequests: maxConcurrentRequests,
		EnableCompression:     enableCompression,
		CompressionType:       compressionType,
		GenerateRequestID:     generateRequestID,
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取整数环境变量，如果不存在则返回默认值
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		slog.Warn("环境变量不是有效的整数，使用默认值", "key", key, "value", value, "default", defaultValue)
	}
	return defaultValue
}

// getEnvAsBool 获取布尔值环境变量，如果不存在则返回默认值
// 支持的值：true, false, 1, 0, yes, no
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch value {
		case "true", "1", "yes", "YES", "Yes":
			return true
		case "false", "0", "no", "NO", "No":
			return false
		default:
			slog.Warn("环境变量不是有效的布尔值，使用默认值", "key", key, "value", value, "default", defaultValue)
		}
	}
	return defaultValue
}

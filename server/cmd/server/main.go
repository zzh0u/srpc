package main

import (
	"log"
	_ "srpc/pkg/compress" // 确保压缩器被注册
	"srpc/server"
)

func main() {
	log.Println("启动gRPC服务端...")
	if err := server.RunServer(); err != nil {
		log.Fatalf("服务器运行失败: %v", err)
	}
}

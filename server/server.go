package server

import (
	"context"
	"fmt"
	"io"
	srpclog "srpc/pkg/log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "srpc/proto"
	_ "srpc/pkg/compress" // 确保压缩器被注册
)

var logger = srpclog.NewLogger()

// server结构体实现GreeterServer接口
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello实现普通RPC
func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	// 从metadata中获取请求ID
	var requestID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ids := md.Get("x-request-id"); len(ids) > 0 {
			requestID = ids[0]
		}
	}

	if requestID != "" {
		logger.Info(fmt.Sprintf("收到SayHello请求 [ID: %s]: %v", requestID, req.GetName()))
	} else {
		logger.Info(fmt.Sprintf("收到SayHello请求: %v", req.GetName()))
	}

	return &pb.HelloReply{
		Message: fmt.Sprintf("Hello %s!", req.GetName()),
	}, nil
}

// GetStream 实现服务端流模式
func (s *server) GetStream(req *pb.StreamReqData, stream pb.Greeter_GetStreamServer) error {
	logger.Info(fmt.Sprintf("收到GetStream请求: %v", req.GetData()))

	// 发送5条流式响应
	for i := 1; i <= 5; i++ {
		response := &pb.StreamResData{
			Data: fmt.Sprintf("服务端流数据 %d: %s", i, req.GetData()),
		}
		if err := stream.Send(response); err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("发送流数据: %v", response.GetData()))
		time.Sleep(500 * time.Millisecond) // 模拟处理延迟
	}

	return nil
}

// PutStream 实现客户端流模式
func (s *server) PutStream(stream pb.Greeter_PutStreamServer) error {
	logger.Info("开始接收客户端流数据")

	var messageCount int32 = 0
	var lastMessage string

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// 客户端流结束
			logger.Info(fmt.Sprintf("客户端流结束，共接收 %d 条消息", messageCount))
			return stream.SendAndClose(&pb.StreamResData{
				Data: fmt.Sprintf("成功接收 %d 条消息，最后一条: %s", messageCount, lastMessage),
			})
		}
		if err != nil {
			return err
		}

		messageCount++
		lastMessage = req.GetData()
		logger.Info(fmt.Sprintf("接收客户端流数据 %d: %v", messageCount, lastMessage))
	}
}

// AllStream实现双向流模式
func (s *server) AllStream(stream pb.Greeter_AllStreamServer) error {
	logger.Info("开始双向流通信")

	// 启动goroutine接收客户端消息
	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				logger.Info("客户端流结束")
				return
			}
			if err != nil {
				logger.Error(fmt.Sprintf("接收客户端消息错误: %v", err))
				return
			}
			logger.Info(fmt.Sprintf("接收客户端消息: %v", req.GetData()))

			// 立即回应
			response := &pb.StreamResData{
				Data: fmt.Sprintf("回应: %s", req.GetData()),
			}
			if err := stream.Send(response); err != nil {
				logger.Error(fmt.Sprintf("发送回应错误: %v", err))
				return
			}
		}
	}()

	// 主goroutine发送一些初始消息
	for i := 1; i <= 3; i++ {
		response := &pb.StreamResData{
			Data: fmt.Sprintf("服务端初始消息 %d", i),
		}
		if err := stream.Send(response); err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("发送服务端初始消息: %v", response.GetData()))
		time.Sleep(1 * time.Second)
	}

	// 等待流结束
	<-stream.Context().Done()
	return nil
}

// RunServer 启动 gRPC 服务器
func RunServer() error {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return fmt.Errorf("监听失败: %v", err)
	}

	// 创建gRPC服务器
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})

	logger.Info("gRPC服务器启动，监听端口: 50051")

	// 优雅关闭处理
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stopChan
		logger.Info("收到关闭信号，开始优雅关闭...")
		s.GracefulStop()
		logger.Info("gRPC服务器已关闭")
	}()

	// 启动服务器
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("服务器启动失败: %v", err)
	}

	return nil
}

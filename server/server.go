package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	_ "srpc/pkg/compress" // 确保压缩器被注册
	srpclog "srpc/pkg/log"
	pb "srpc/proto"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var slogger = srpclog.NewLogger()

// server 结构体实现 GreeterServer 接口
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello 实现普通RPC
func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	// 从 metadata 中获取请求 ID
	var requestID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ids := md.Get("x-request-id"); len(ids) > 0 {
			requestID = ids[0]
		}
	}

	if requestID != "" {
		slogger.Info(fmt.Sprintf("收到 SayHello 请求 [ID: %s]: %v", requestID, req.GetName()))
	} else {
		slogger.Info(fmt.Sprintf("收到 SayHello 请求: %v", req.GetName()))
	}

	return &pb.HelloReply{
		Message: fmt.Sprintf("Hello %s!", req.GetName()),
	}, nil
}

// GetStream 实现服务端流模式
func (s *server) GetStream(req *pb.StreamReqData, stream pb.Greeter_GetStreamServer) error {
	slogger.Info(fmt.Sprintf("收到 GetStream 请求: %v", req.GetData()))

	// 发送 5 条流式响应
	for i := 1; i <= 5; i++ {
		response := &pb.StreamResData{
			Data: fmt.Sprintf("服务端流数据 %d: %s", i, req.GetData()),
		}
		if err := stream.Send(response); err != nil {
			return err
		}
		slogger.Info(fmt.Sprintf("发送流数据: %v", response.GetData()))
		time.Sleep(500 * time.Millisecond) // 模拟处理延迟
	}

	return nil
}

// PutStream 实现客户端流模式
func (s *server) PutStream(stream pb.Greeter_PutStreamServer) error {
	slogger.Info("开始接收客户端流数据")

	var messageCount int32 = 0
	var lastMessage string

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// 客户端流结束
			slogger.Info(fmt.Sprintf("客户端流结束，共接收 %d 条消息", messageCount))
			return stream.SendAndClose(&pb.StreamResData{
				Data: fmt.Sprintf("成功接收 %d 条消息，最后一条: %s", messageCount, lastMessage),
			})
		}
		if err != nil {
			return err
		}

		messageCount++
		lastMessage = req.GetData()
		slogger.Info(fmt.Sprintf("接收客户端流数据 %d: %v", messageCount, lastMessage))
	}
}

// AllStream 实现双向流模式
func (s *server) AllStream(stream pb.Greeter_AllStreamServer) error {
	slogger.Info("开始双向流通信")

	// 启动goroutine接收客户端消息
	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				slogger.Info("客户端流结束")
				return
			}
			if err != nil {
				slogger.Error(fmt.Sprintf("接收客户端消息错误: %v", err))
				return
			}
			slogger.Info(fmt.Sprintf("接收客户端消息: %v", req.GetData()))

			// 立即回应
			response := &pb.StreamResData{
				Data: fmt.Sprintf("回应: %s", req.GetData()),
			}
			if err := stream.Send(response); err != nil {
				slogger.Error(fmt.Sprintf("发送回应错误: %v", err))
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
		slogger.Info(fmt.Sprintf("发送服务端初始消息: %v", response.GetData()))
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

	// 创建 gRPC 服务器
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})

	slogger.Info("gRPC 服务器启动，监听端口: 50051")

	// 关闭处理
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stopChan
		slogger.Info("收到关闭信号，开始关闭...")
		s.GracefulStop()
		slogger.Info("gRPC 服务器已关闭")
	}()

	// 启动服务器
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("服务器启动失败: %v", err)
	}

	return nil
}

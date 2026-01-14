# 待实现的功能

1. 拦截器实现认证/日志，错误处理，deadline/timeout 机制
2. 负载均衡：多服务端实例场景下的负载均衡策略 (google.golang.org/grpc/balancer/roundrobin)
3. 如何集成服务发现，多次均衡，监控（prometheus）
4. 高级资源管理：请求队列、连接池控制
5. 事件驱动触发：支持 Kafka/RabbitMQ/Webhook 等外部事件源触发请求

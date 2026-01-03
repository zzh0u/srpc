# client/Dockerfile
# 多阶段构建：第一阶段 - 构建阶段
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

# 复制整个项目
COPY . .

WORKDIR /app/client

# 运行go mod download
RUN go mod download

# 构建静态链接的可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o client ./cmd/client

# 第二阶段 - 运行阶段
FROM alpine:3.21

# 安装CA证书（用于TLS连接）
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从构建阶段复制可执行文件
COPY --from=builder /app/client .

# 创建非root用户运行应用
RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser && \
    chown -R appuser:appuser /root/client
USER appuser

# 健康检查 - 检查进程是否运行
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD ps aux | grep -q "[c]lient" || exit 1

# 运行客户端
CMD ["./client"]
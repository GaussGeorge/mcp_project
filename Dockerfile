# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 预下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 🔥【AMD64 专用优化】
# 显式指定 GOARCH=amd64，适配 Windows/Linux 服务器环境
# CGO_ENABLED=0 确保静态链接，不依赖系统库
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/backend ./cmd/backend/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/client ./cmd/client3/mainv2.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/gateway ./cmd/server/main.go

# Final Stage
FROM alpine:latest

WORKDIR /app

# 复制二进制文件
COPY --from=builder /bin/gateway /app/gateway
COPY --from=builder /bin/backend /app/backend
COPY --from=builder /bin/client /app/client

# 赋予执行权限 (Windows Git 有时会丢失权限位，这一步很关键)
RUN chmod +x /app/gateway /app/backend /app/client

# 暴露端口
EXPOSE 8080

# 默认入口
CMD ["/app/gateway"]
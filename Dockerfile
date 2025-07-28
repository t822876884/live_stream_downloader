# 使用官方Go镜像作为构建环境
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o live_stream_downloader ./cmd/server

# 使用轻量级的alpine镜像作为运行环境
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 设置时区为亚洲/上海
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建非root用户
RUN adduser -D -h /app appuser

# 创建数据目录并设置权限
RUN mkdir -p /app/data && \
    chown -R appuser:appuser /app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /app/live_stream_downloader .

# 复制web目录
COPY --from=builder /app/web ./web

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 设置数据目录为卷，方便持久化和外部访问
VOLUME ["/app/data"]

# Dockerfile中添加
ENV SERVER_ADDR=:8080
ENV DATA_DIR=/app/data

# 修改CMD为
CMD ["./live_stream_downloader"]
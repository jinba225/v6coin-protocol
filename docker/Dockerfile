# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git cmake make openssl-dev build-essential

# 复制依赖文件
COPY code/go/go.mod code/go/go.sum* ./code/go/

# 下载 Go 依赖
WORKDIR /app/code/go
RUN go mod download

# 复制 Go 源代码
COPY code/go ./go

# 构建 Go 节点
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o v6coin-node ./cmd/node

# 构建阶段（C代码）
FROM alpine:latest AS c-builder

WORKDIR /app

RUN apk add --no-cache cmake make openssl-dev gcc musl-dev

COPY code/c ./c

WORKDIR /app/c
RUN cmake -B build -DCMAKE_BUILD_TYPE=Release
RUN cmake --build build --config Release

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 从构建阶段复制二进制文件
COPY --from=builder /app/code/go/v6coin-node /usr/local/bin/v6coin-node

# 创建数据目录
RUN mkdir -p /data /config

# 暴露端口
EXPOSE 38901 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD v6coin-node status || exit 1

# 设置默认命令
CMD ["v6coin-node", "start", "--datadir=/data", "--config=/config/config.toml"]

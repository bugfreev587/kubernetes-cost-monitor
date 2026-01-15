# -----------------------
# Stage 1: Build
# -----------------------
FROM golang:1.25-alpine AS builder

# 安装 git（用于 go mod）和必要依赖
RUN apk add --no-cache git bash ca-certificates

# 设置工作目录
WORKDIR /app

# 复制 go.mod / go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建可执行文件
RUN go build -o /bin/api-server ./cmd/server

# -----------------------
# Stage 2: Production image
# -----------------------
FROM alpine:3.18

# 添加证书
RUN apk add --no-cache ca-certificates bash

# 设置工作目录
WORKDIR /app

# 复制二进制文件
COPY --from=builder /bin/api-server /bin/api-server

# 复制配置目录（可选，如果你打算在容器内使用 dev.yaml）
COPY conf ./conf

# 设置默认环境变量（可覆盖）
# Note: The server uses API_SERVER_CONF_FILE or ENVIRONMENT to determine config file
# Default paths: /app/conf/api-server-dev.yaml or /app/conf/api-server-prod.yaml

# 暴露端口
EXPOSE 8080

# 默认启动命令
CMD ["/bin/api-server"]
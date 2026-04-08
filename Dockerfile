# /root/workspace/reservation_sys_go/Dockerfile

# ==================== 构建阶段 ====================
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum，利用缓存加速构建
COPY go.mod go.sum ./

# 配置 Go 代理（加速模块下载）
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
RUN go mod download

# 复制源代码
COPY . .

# 构建三个服务的二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /v1 ./cmd/api/v1
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /v2 ./cmd/api/v2
# RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /v3 ./cmd/api/v3

# ==================== 运行阶段 ====================
FROM alpine:3.19

WORKDIR /app

# 安装 ca-certificates（用于 HTTPS 请求）
RUN apk add --no-cache ca-certificates tzdata

# 从构建阶段复制二进制文件
COPY --from=builder /v1 /app/v1
COPY --from=builder /v2 /app/v2
# COPY --from=builder /v3 /app/v3

# 复制菜单同步工具
COPY --from=builder /sync_menu /app/tools/sync_menu

# 复制配置文件和静态资源
COPY configs/ /app/configs/
COPY internal/reservation/frontend/ /app/internal/reservation/frontend/

# 设置时区
ENV TZ=Asia/Shanghai

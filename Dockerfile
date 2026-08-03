# 预约场地审核系统 — 单体架构 Docker 镜像
# 构建: docker build -t reservation-server:latest .
# 运行: docker run -v ./configs:/app/configs:ro reservation-server:latest

FROM golang:1.24-alpine AS builder

WORKDIR /build

# 依赖下载（利用 Docker 缓存）
COPY go.mod go.sum ./
RUN go mod download

# 构建二进制
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server

# 运行阶段
FROM alpine:3.21

RUN apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY --from=builder /build/server .

EXPOSE 8080

CMD ["./server"]

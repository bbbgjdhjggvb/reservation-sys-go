# ==================== 构建阶段 ====================
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gateway ./service/gateway/cmd

# ==================== 运行阶段 ====================
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /gateway /app/gateway
COPY service/gateway/configs/ /app/service/gateway/configs/

ENV TZ=Asia/Shanghai

CMD ["/app/gateway"]

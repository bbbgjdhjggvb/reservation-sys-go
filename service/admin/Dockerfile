# ==================== 构建阶段 ====================
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /admin ./service/admin/cmd

# ==================== 运行阶段 ====================
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /admin /app/admin
COPY service/admin/configs/ /app/service/admin/configs/
COPY service/admin/frontend/ /app/service/admin/frontend/

ENV TZ=Asia/Shanghai

CMD ["/app/admin"]

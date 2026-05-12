#!/bin/bash
# 从 proto 文件生成 Go gRPC 代码
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="$PATH:$(go env GOPATH)/bin"

echo "Generating Go code from proto files..."

# notification proto → gateway service
protoc --go_out="$PROJECT_ROOT/service/gateway/api/gen" --go_opt=paths=source_relative \
  --go-grpc_out="$PROJECT_ROOT/service/gateway/api/gen" --go-grpc_opt=paths=source_relative \
  -I "$PROJECT_ROOT/service/gateway/api/proto" \
  "$PROJECT_ROOT/service/gateway/api/proto/notification/notification.proto"

# 同时生成到 admin service（admin 需要引用 notification 客户端）
mkdir -p "$PROJECT_ROOT/service/admin/api/gen/notification"
cp "$PROJECT_ROOT/service/gateway/api/gen/notification/"*.pb.go \
   "$PROJECT_ROOT/service/admin/api/gen/notification/"

# account proto → gateway service（账号验证服务，供 admin 调用）
protoc --go_out="$PROJECT_ROOT/service/gateway/api/gen" --go_opt=paths=source_relative \
  --go-grpc_out="$PROJECT_ROOT/service/gateway/api/gen" --go-grpc_opt=paths=source_relative \
  -I "$PROJECT_ROOT/service/gateway/api/proto" \
  "$PROJECT_ROOT/service/gateway/api/proto/account/account.proto"

# 同时生成到 admin service（admin 需要引用 account 客户端）
mkdir -p "$PROJECT_ROOT/service/admin/api/gen/account"
cp "$PROJECT_ROOT/service/gateway/api/gen/account/"*.pb.go \
   "$PROJECT_ROOT/service/admin/api/gen/account/"

echo "Done. Generated files in service/*/api/gen/"

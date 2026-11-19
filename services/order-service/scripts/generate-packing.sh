#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
root=../../packages/proto
protoc -I "$root" --go_out=internal/gen --go_opt=paths=source_relative \
 --go_opt=Mpacking/v1/packing.proto=github.com/omamx/order-service/internal/gen/packing/v1 \
 --go-grpc_out=internal/gen --go-grpc_opt=paths=source_relative \
 --go-grpc_opt=Mpacking/v1/packing.proto=github.com/omamx/order-service/internal/gen/packing/v1 \
 "$root/packing/v1/packing.proto"

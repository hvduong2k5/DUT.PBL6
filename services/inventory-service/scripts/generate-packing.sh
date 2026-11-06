#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
root=../../packages/proto
protoc -I "$root" --go_out=pkg/proto --go_opt=paths=source_relative \
 --go_opt=Mpacking/v1/packing.proto=dut-pbl6/inventory-service/pkg/proto/packing/v1 \
 --go-grpc_out=pkg/proto --go-grpc_opt=paths=source_relative \
 --go-grpc_opt=Mpacking/v1/packing.proto=dut-pbl6/inventory-service/pkg/proto/packing/v1 \
 "$root/packing/v1/packing.proto"

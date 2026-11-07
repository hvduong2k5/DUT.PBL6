#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p internal/gen
opts=''
for file in common/v1/types common/v1/errors common/v1/workflow fulfillment/v1/fulfillment order/v1/order; do
 dir=$(dirname "$file")
 opts="$opts --go_opt=M$file.proto=github.com/omamx/fulfillment-service/internal/gen/$dir --go-grpc_opt=M$file.proto=github.com/omamx/fulfillment-service/internal/gen/$dir"
done
protoc -I contracts --go_out=internal/gen --go_opt=paths=source_relative \
 --go-grpc_out=internal/gen --go-grpc_opt=paths=source_relative \
 $opts --go_opt=Mpacking.proto=github.com/omamx/fulfillment-service/internal/gen/packing \
 --go-grpc_opt=Mpacking.proto=github.com/omamx/fulfillment-service/internal/gen/packing \
 contracts/common/v1/types.proto contracts/common/v1/errors.proto contracts/common/v1/workflow.proto \
 contracts/fulfillment/v1/fulfillment.proto contracts/order/v1/order.proto contracts/packing.proto
mkdir -p internal/gen/packing
mv internal/gen/packing.pb.go internal/gen/packing_grpc.pb.go internal/gen/packing/

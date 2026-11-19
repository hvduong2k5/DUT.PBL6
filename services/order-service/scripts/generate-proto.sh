#!/bin/sh
set -eu
sh "$(dirname "$0")/generate-packing.sh"
cd "$(dirname "$0")/.."
proto_root="../../packages/proto"
mkdir -p internal/gen
protoc -I "$proto_root" \
 --go_out=internal/gen --go_opt=paths=source_relative \
 --go_opt=Mcommon/v1/types.proto=github.com/omamx/order-service/internal/gen/common/v1 \
 --go_opt=Mcommon/v1/errors.proto=github.com/omamx/order-service/internal/gen/common/v1 \
 --go_opt=Mcommon/v1/workflow.proto=github.com/omamx/order-service/internal/gen/common/v1 \
 --go_opt=Minventory/v1/inventory.proto=github.com/omamx/order-service/internal/gen/inventory/v1 \
 --go_opt=Morder/v1/order.proto=github.com/omamx/order-service/internal/gen/order/v1 \
 --go_opt=Mshipping/v2/shipping.proto=github.com/omamx/order-service/internal/gen/shipping/v2 \
 --go_opt=Mfulfillment/v1/fulfillment.proto=github.com/omamx/order-service/internal/gen/fulfillment/v1 \
 --go_opt=Mpromotion/v1/promotion.proto=github.com/omamx/order-service/internal/gen/promotion/v1 \
 --go-grpc_out=internal/gen --go-grpc_opt=paths=source_relative \
 --go-grpc_opt=Mcommon/v1/types.proto=github.com/omamx/order-service/internal/gen/common/v1 \
 --go-grpc_opt=Mcommon/v1/errors.proto=github.com/omamx/order-service/internal/gen/common/v1 \
 --go-grpc_opt=Mcommon/v1/workflow.proto=github.com/omamx/order-service/internal/gen/common/v1 \
 --go-grpc_opt=Minventory/v1/inventory.proto=github.com/omamx/order-service/internal/gen/inventory/v1 \
 --go-grpc_opt=Morder/v1/order.proto=github.com/omamx/order-service/internal/gen/order/v1 \
 --go-grpc_opt=Mshipping/v2/shipping.proto=github.com/omamx/order-service/internal/gen/shipping/v2 \
 --go-grpc_opt=Mfulfillment/v1/fulfillment.proto=github.com/omamx/order-service/internal/gen/fulfillment/v1 \
 --go-grpc_opt=Mpromotion/v1/promotion.proto=github.com/omamx/order-service/internal/gen/promotion/v1 \
 "$proto_root/common/v1/types.proto" "$proto_root/common/v1/errors.proto" "$proto_root/common/v1/workflow.proto" "$proto_root/order/v1/order.proto" "$proto_root/inventory/v1/inventory.proto" "$proto_root/shipping/v2/shipping.proto" "$proto_root/fulfillment/v1/fulfillment.proto" "$proto_root/promotion/v1/promotion.proto"

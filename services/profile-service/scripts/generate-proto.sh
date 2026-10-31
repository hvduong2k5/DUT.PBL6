#!/bin/sh
set -eu
# Run from services/profile-service; protoc and pinned Go plugins must be installed.
proto_root="../../packages/proto"
mkdir -p internal/gen
protoc -I "$proto_root" \
  --go_out=internal/gen --go_opt=paths=source_relative \
  --go_opt=Mcommon/v1/types.proto=github.com/omamx/profile-service/internal/gen/common/v1 \
  --go_opt=Mcommon/v1/errors.proto=github.com/omamx/profile-service/internal/gen/common/v1 \
  --go_opt=Mprofile/v1/profile.proto=github.com/omamx/profile-service/internal/gen/profile/v1 \
  --go-grpc_out=internal/gen --go-grpc_opt=paths=source_relative \
  --go-grpc_opt=Mprofile/v1/profile.proto=github.com/omamx/profile-service/internal/gen/profile/v1 \
  "$proto_root/common/v1/types.proto" "$proto_root/common/v1/errors.proto" "$proto_root/profile/v1/profile.proto"

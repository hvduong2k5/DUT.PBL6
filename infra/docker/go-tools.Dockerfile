FROM golang:1.22
RUN apt-get update -qq && apt-get install -y -qq protobuf-compiler && rm -rf /var/lib/apt/lists/*
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
# Bind-mounted checkouts may have a host-only worktree pointer or different UID.
# Evidence manifests record the source SHA separately from disposable binaries.
ENV GOFLAGS=-buildvcs=false
ARG SERVICE_PATH=services/order-service
WORKDIR /workspace/${SERVICE_PATH}

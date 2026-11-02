package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryUnaryServerInterceptor_NormalExecution(t *testing.T) {
	interceptor := recoveryUnaryServerInterceptor()
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/inventory.v1.InventoryService/GetStockLevel"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "success-resp", nil
	}

	resp, err := interceptor(ctx, "req", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "success-resp", resp)
}

func TestRecoveryUnaryServerInterceptor_CatchesPanic_ReturnsInternalError(t *testing.T) {
	interceptor := recoveryUnaryServerInterceptor()
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/inventory.v1.InventoryService/ReserveStock"}

	handler := func(ctx context.Context, req any) (any, error) {
		// Simulate unexpected runtime panic (e.g., nil pointer dereference)
		panic("nil pointer dereference simulation")
	}

	// Must NOT crash the process
	assert.NotPanics(t, func() {
		resp, err := interceptor(ctx, "req", info, handler)
		assert.Nil(t, resp)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Equal(t, "internal server error", st.Message())
	})
}

func TestRecoveryStreamServerInterceptor_NormalExecution(t *testing.T) {
	interceptor := recoveryStreamServerInterceptor()
	info := &grpc.StreamServerInfo{FullMethod: "/inventory.v1.InventoryService/StreamStock"}

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	err := interceptor(nil, nil, info, handler)
	require.NoError(t, err)
}

func TestRecoveryStreamServerInterceptor_CatchesPanic_ReturnsInternalError(t *testing.T) {
	interceptor := recoveryStreamServerInterceptor()
	info := &grpc.StreamServerInfo{FullMethod: "/inventory.v1.InventoryService/StreamStock"}

	handler := func(srv any, ss grpc.ServerStream) error {
		panic("stream unexpected panic")
	}

	assert.NotPanics(t, func() {
		err := interceptor(nil, nil, info, handler)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Equal(t, "internal server error", st.Message())
	})
}

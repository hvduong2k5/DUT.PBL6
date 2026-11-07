package grpc

import (
	"context"
	api "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

func TestReadOnlyCatalogCredential(t *testing.T) {
	s := &TerminalServer{Token: strings.Repeat("o", 32), CatalogToken: strings.Repeat("c", 32)}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+s.CatalogToken))
	require.NoError(t, s.authorizeRead(ctx))
	_, err := s.GetStockLevel(ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = s.GetStockLevel(context.Background(), nil)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = s.ReserveStock(ctx, &api.ReserveStockRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = s.ReleaseReservation(ctx, nil)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = s.FinalizeReservation(ctx, nil)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = s.ReleaseReservationByOrder(ctx, nil)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	duplicate := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+s.Token, "authorization", "Bearer "+s.CatalogToken))
	require.Error(t, s.authorizeRead(duplicate))
}

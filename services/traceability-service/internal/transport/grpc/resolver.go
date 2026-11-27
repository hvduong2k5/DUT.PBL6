package grpctransport

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const MaxRequestBytes = 256 * 1024
const MaxResponseBytes = 512 * 1024

type Resolver struct {
	traceabilityv1.UnimplementedBatchLinkResolverServer
	Service *application.Resolver
	Token   string
}

func NewServer(service *application.Resolver, token string, options ...grpc.ServerOption) (*grpc.Server, error) {
	if service == nil || len(token) < 32 {
		return nil, errors.New("resolver and dedicated service credential required")
	}
	options = append(options, grpc.MaxRecvMsgSize(MaxRequestBytes), grpc.MaxSendMsgSize(MaxResponseBytes), grpc.MaxHeaderListSize(32*1024), grpc.ChainUnaryInterceptor(requestContext))
	server := grpc.NewServer(options...)
	traceabilityv1.RegisterBatchLinkResolverServer(server, &Resolver{Service: service, Token: token})
	return server, nil
}

func requestContext(ctx context.Context, request any, _ *grpc.UnaryServerInfo, next grpc.UnaryHandler) (response any, err error) {
	defer func() {
		if recover() != nil {
			response = nil
			err = status.Error(codes.Internal, "Unable to process request")
		}
	}()
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("x-request-id")
	id := ""
	if len(values) == 1 && validRequestID(values[0]) {
		id = values[0]
	}
	if id == "" {
		id, err = domain.NewUUID()
		if err != nil {
			return nil, status.Error(codes.Internal, "Unable to process request")
		}
	}
	_ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", id))
	return next(ctx, request)
}
func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

func (s *Resolver) ResolveBatchLinks(ctx context.Context, request *traceabilityv1.ResolveBatchLinksRequest) (*traceabilityv1.ResolveBatchLinksResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	auth := md.Get("authorization")
	if len(auth) != 1 {
		return nil, status.Error(codes.Unauthenticated, "Service identity required")
	}
	parts := strings.Fields(auth[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, status.Error(codes.Unauthenticated, "Service identity required")
	}
	supplied, expected := sha256.Sum256([]byte(parts[1])), sha256.Sum256([]byte(s.Token))
	if len(s.Token) < 32 || subtle.ConstantTimeCompare(supplied[:], expected[:]) != 1 {
		return nil, status.Error(codes.Unauthenticated, "Service identity required")
	}
	items := make([]domain.LinkBinding, 0, len(request.GetItems()))
	for _, b := range request.GetItems() {
		items = append(items, domain.LinkBinding{SourceNamespace: b.GetSourceNamespace(), BatchID: b.GetBatchId(), SKUCode: b.GetSkuCode()})
	}
	result, err := s.Service.Resolve(ctx, items)
	if errors.Is(err, domain.ErrInvalidCommand) {
		return nil, status.Error(codes.InvalidArgument, "Invalid batch bindings")
	}
	if err != nil {
		return nil, status.Error(codes.Unavailable, "Required storage unavailable")
	}
	response := &traceabilityv1.ResolveBatchLinksResponse{Items: make([]*traceabilityv1.ResolvedBatchLink, 0, len(result.Items)), Total: int32(result.Total)}
	for _, entry := range result.Items {
		state := traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_UNAVAILABLE
		switch entry.Status {
		case "AVAILABLE":
			state = traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_AVAILABLE
		case "NOT_AVAILABLE":
			state = traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_NOT_AVAILABLE
		case "NO_LONGER_PUBLIC":
			state = traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_NO_LONGER_PUBLIC
		}
		response.Items = append(response.Items, &traceabilityv1.ResolvedBatchLink{Binding: &traceabilityv1.TraceabilityBatchBinding{SourceNamespace: entry.SourceNamespace, BatchId: entry.BatchID, SkuCode: entry.SKUCode}, Status: state, QrUrl: entry.QRURL})
	}
	return response, nil
}

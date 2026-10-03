package grpc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/domain/entity"
	commonv1 "dut-pbl6/inventory-service/pkg/proto/common/v1"
	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InventoryGRPCHandler implements inventoryv1.InventoryServiceServer.
type InventoryGRPCHandler struct {
	inventoryv1.UnimplementedInventoryServiceServer
	reserveStockUC        *usecase.ReserveStockUseCase
	releaseReservationUC  *usecase.ReleaseReservationUseCase
	getStockLevelUC       *usecase.GetStockLevelUseCase
	getBatchFEFODetailsUC *usecase.GetBatchFEFODetailsUseCase
}

// NewInventoryGRPCHandler constructs an InventoryGRPCHandler.
func NewInventoryGRPCHandler(
	reserveStockUC *usecase.ReserveStockUseCase,
	releaseReservationUC *usecase.ReleaseReservationUseCase,
	getStockLevelUC *usecase.GetStockLevelUseCase,
	getBatchFEFODetailsUC *usecase.GetBatchFEFODetailsUseCase,
) *InventoryGRPCHandler {
	return &InventoryGRPCHandler{
		reserveStockUC:        reserveStockUC,
		releaseReservationUC:  releaseReservationUC,
		getStockLevelUC:       getStockLevelUC,
		getBatchFEFODetailsUC: getBatchFEFODetailsUC,
	}
}

// Helper to convert internal domain errors and codes into gRPC Status with attached ErrorDetail.
func toGRPCError(code codes.Code, errCode commonv1.ErrorCode, userMsg, devMsg string, metadata map[string]string) error {
	st := status.New(code, devMsg)
	errDetail := &commonv1.ErrorDetail{
		ErrorCode:   errCode,
		UserMessage: userMsg,
		DevMessage:  devMsg,
		Domain:      "inventory",
		Metadata:    metadata,
	}
	stWithDetails, err := st.WithDetails(errDetail)
	if err == nil {
		return stWithDetails.Err()
	}
	return st.Err()
}

// ReserveStock handles stock reservation requests.
func (h *InventoryGRPCHandler) ReserveStock(ctx context.Context, req *inventoryv1.ReserveStockRequest) (*inventoryv1.ReserveStockResponse, error) {
	if req == nil {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Yêu cầu không hợp lệ", "request payload cannot be nil", nil)
	}
	orderID := strings.TrimSpace(req.OrderId)
	if orderID == "" {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Mã đơn hàng không được để trống", "order_id is required", nil)
	}
	if len(req.Items) == 0 {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Danh sách sản phẩm giữ chỗ không được để trống", "items list cannot be empty", nil)
	}

	items := make([]usecase.ReserveItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		sku := strings.TrimSpace(item.SkuCode)
		if sku == "" || item.Quantity <= 0 {
			return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
				"Thông tin mặt hàng không hợp lệ", fmt.Sprintf("invalid sku_code or quantity for item: %s", item.SkuCode),
				map[string]string{"sku_code": item.SkuCode})
		}
		items = append(items, usecase.ReserveItemInput{
			SKU: sku,
			Qty: int(item.Quantity),
		})
	}

	reservation, err := h.reserveStockUC.Execute(ctx, orderID, items)
	if err != nil {
		if errors.Is(err, entity.ErrInsufficientStock) {
			return nil, toGRPCError(codes.ResourceExhausted, commonv1.ErrorCode_ERR_INVENTORY_INSUFFICIENT_STOCK,
				"Số lượng tồn kho không đủ để giữ chỗ", err.Error(), nil)
		}
		if errors.Is(err, entity.ErrItemNotFound) {
			return nil, toGRPCError(codes.NotFound, commonv1.ErrorCode_ERR_INVENTORY_SKU_NOT_FOUND,
				"Không tìm thấy sản phẩm trong danh mục kho", err.Error(), nil)
		}
		if errors.Is(err, entity.ErrConcurrentUpdate) {
			return nil, toGRPCError(codes.Aborted, commonv1.ErrorCode_ERR_INTERNAL_SERVER_ERROR,
				"Hệ thống đang bận xử lý tồn kho, vui lòng thử lại", err.Error(), nil)
		}
		if errors.Is(err, entity.ErrInvalidBatchData) || errors.Is(err, entity.ErrInvalidQuantity) {
			return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
				"Dữ liệu giữ chỗ không hợp lệ", err.Error(), nil)
		}
		return nil, toGRPCError(codes.Internal, commonv1.ErrorCode_ERR_INTERNAL_SERVER_ERROR,
			"Lỗi máy chủ nội bộ khi giữ chỗ tồn kho", err.Error(), nil)
	}

	resID := reservation.ID.String()
	return &inventoryv1.ReserveStockResponse{
		Status:        inventoryv1.ReservationStatus_RESERVATION_STATUS_SUCCESS,
		ReservationId: &resID,
		ExpiresAt:     timestamppb.New(reservation.ExpiresAt),
	}, nil
}

// ReleaseReservation handles stock release requests.
func (h *InventoryGRPCHandler) ReleaseReservation(ctx context.Context, req *inventoryv1.ReleaseReservationRequest) (*inventoryv1.ReleaseReservationResponse, error) {
	if req == nil {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Yêu cầu không hợp lệ", "request payload cannot be nil", nil)
	}
	orderID := strings.TrimSpace(req.OrderId)
	resID := strings.TrimSpace(req.ReservationId)
	if orderID == "" && resID == "" {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Cần cung cấp order_id hoặc reservation_id", "order_id or reservation_id required", nil)
	}

	totalRestored, err := h.releaseReservationUC.ExecuteWithResult(ctx, orderID, resID)
	if err != nil {
		if errors.Is(err, entity.ErrReservationNotFound) || errors.Is(err, entity.ErrItemNotFound) {
			return nil, toGRPCError(codes.NotFound, commonv1.ErrorCode_ERR_INVENTORY_RESERVATION_NOT_FOUND,
				"Không tìm thấy phiếu giữ chỗ tồn kho", err.Error(), nil)
		}
		if errors.Is(err, entity.ErrInvalidBatchData) {
			return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
				"Dữ liệu yêu cầu giải phóng không hợp lệ", err.Error(), nil)
		}
		return nil, toGRPCError(codes.Internal, commonv1.ErrorCode_ERR_INTERNAL_SERVER_ERROR,
			"Lỗi máy chủ nội bộ khi giải phóng tồn kho", err.Error(), nil)
	}

	return &inventoryv1.ReleaseReservationResponse{
		IsReleased:         true,
		TotalItemsRestored: int32(totalRestored),
		Message:            "Giải phóng tồn kho thành công",
	}, nil
}

// GetStockLevel handles inventory stock level queries for SKUs.
func (h *InventoryGRPCHandler) GetStockLevel(ctx context.Context, req *inventoryv1.GetStockLevelRequest) (*inventoryv1.GetStockLevelResponse, error) {
	if req == nil {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Yêu cầu không hợp lệ", "request payload cannot be nil", nil)
	}
	hasValidSKU := false
	for _, s := range req.SkuCodes {
		if strings.TrimSpace(s) != "" {
			hasValidSKU = true
			break
		}
	}
	if !hasValidSKU {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Danh sách mã SKU không được để trống", "sku_codes list cannot be empty or contain only blank values", nil)
	}

	results, err := h.getStockLevelUC.Execute(ctx, req.SkuCodes)
	if err != nil {
		if errors.Is(err, entity.ErrItemNotFound) {
			return nil, toGRPCError(codes.NotFound, commonv1.ErrorCode_ERR_INVENTORY_SKU_NOT_FOUND,
				"Không tìm thấy thông tin sản phẩm trong kho", err.Error(), nil)
		}
		return nil, toGRPCError(codes.Internal, commonv1.ErrorCode_ERR_INTERNAL_SERVER_ERROR,
			"Lỗi nội bộ khi truy vấn mức tồn kho", err.Error(), nil)
	}

	protoItems := make([]*inventoryv1.StockLevelItem, 0, len(results))
	for _, r := range results {
		protoItems = append(protoItems, &inventoryv1.StockLevelItem{
			SkuCode:           r.SKU,
			PhysicalQuantity:  int32(r.PhysicalQty),
			ReservedQuantity:  int32(r.ReservedQty),
			AvailableQuantity: int32(r.AvailableQty),
			StockStatus:       r.StockStatus,
		})
	}

	return &inventoryv1.GetStockLevelResponse{
		Items: protoItems,
	}, nil
}

// GetBatchFEFODetails handles queries for batches in FEFO order.
func (h *InventoryGRPCHandler) GetBatchFEFODetails(ctx context.Context, req *inventoryv1.GetBatchFEFODetailsRequest) (*inventoryv1.GetBatchFEFODetailsResponse, error) {
	if req == nil {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Yêu cầu không hợp lệ", "request payload cannot be nil", nil)
	}
	sku := strings.TrimSpace(req.SkuCode)
	if sku == "" {
		return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
			"Mã SKU không được để trống", "sku_code is required", nil)
	}

	batches, err := h.getBatchFEFODetailsUC.Execute(ctx, sku)
	if err != nil {
		if errors.Is(err, entity.ErrItemNotFound) {
			return nil, toGRPCError(codes.NotFound, commonv1.ErrorCode_ERR_INVENTORY_SKU_NOT_FOUND,
				"Không tìm thấy sản phẩm trong danh mục kho", err.Error(), nil)
		}
		if errors.Is(err, entity.ErrInvalidBatchData) {
			return nil, toGRPCError(codes.InvalidArgument, commonv1.ErrorCode_ERR_INVALID_ARGUMENT,
				"Mã SKU không hợp lệ", err.Error(), nil)
		}
		return nil, toGRPCError(codes.Internal, commonv1.ErrorCode_ERR_INTERNAL_SERVER_ERROR,
			"Lỗi nội bộ khi truy vấn chi tiết lô hàng FEFO", err.Error(), nil)
	}

	now := time.Now().UTC()
	protoBatches := make([]*inventoryv1.BatchItemDetail, 0, len(batches))
	for _, b := range batches {
		daysUntil := int32(math.Ceil(b.ExpDate.Sub(now).Hours() / 24))
		if daysUntil < 0 {
			daysUntil = 0
		}

		qualityStatus := inventoryv1.BatchQualityStatus_BATCH_QUALITY_STATUS_ACTIVE_FEFO
		if b.PhysicalQty <= 0 {
			qualityStatus = inventoryv1.BatchQualityStatus_BATCH_QUALITY_STATUS_DEPLETED
		} else if b.Status == entity.BatchStatusQuarantine {
			qualityStatus = inventoryv1.BatchQualityStatus_BATCH_QUALITY_STATUS_QUARANTINED
		} else if b.Status == entity.BatchStatusExpired || !now.Before(b.ExpDate) {
			qualityStatus = inventoryv1.BatchQualityStatus_BATCH_QUALITY_STATUS_QUARANTINED
		} else if b.Status == entity.BatchStatusNearExpiry || (daysUntil > 0 && daysUntil <= 45) {
			qualityStatus = inventoryv1.BatchQualityStatus_BATCH_QUALITY_STATUS_EXPIRING_SOON
		}

		protoBatches = append(protoBatches, &inventoryv1.BatchItemDetail{
			BatchCode:         b.BatchCode,
			ManufacturedAt:    timestamppb.New(b.MfgDate),
			BestBefore:        timestamppb.New(b.ExpDate),
			DaysUntilExpiry:   daysUntil,
			RemainingQuantity: int32(b.AvailableQty()),
			QualityStatus:     qualityStatus,
			OcopTraceQrCode:   fmt.Sprintf("https://trace.omama.vn/batch/%s", b.BatchCode),
		})
	}

	return &inventoryv1.GetBatchFEFODetailsResponse{
		SkuCode: sku,
		Batches: protoBatches,
	}, nil
}

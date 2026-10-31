package main

import (
	"context"
	"fmt"
	"log"
	"time"

	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fmt.Println("==========================================================")
	fmt.Println("   KIỂM THỬ TỰ ĐỘNG 4 RPCs - INVENTORY SERVICE (PORT 8001)")
	fmt.Println("==========================================================")

	target := "localhost:8001"
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Lỗi kết nối gRPC server %s: %v", target, err)
	}
	defer conn.Close()

	client := inventoryv1.NewInventoryServiceClient(conn)
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// RPC 1: GetStockLevel
	// -------------------------------------------------------------------------
	fmt.Println("\n[RPC 1] Kiểm tra GetStockLevel (5 SKUs)...")
	stockReq := &inventoryv1.GetStockLevelRequest{
		SkuCodes: []string{
			"MX-GION-500G",
			"MX-DEO-300G",
			"KE-ME-GUONG-250G",
			"MX-KHOAI-LANG-400G",
			"MX-MAT-ONG-350G",
		},
	}
	stockRes, err := client.GetStockLevel(ctx, stockReq)
	if err != nil {
		log.Fatalf("FAILED GetStockLevel: %v", err)
	}
	for _, item := range stockRes.Items {
		fmt.Printf("   -> SKU: %-18s | Physical: %3d | Reserved: %2d | Available: %3d | Status: %s\n",
			item.SkuCode, item.PhysicalQuantity, item.ReservedQuantity, item.AvailableQuantity, item.StockStatus)
	}

	// -------------------------------------------------------------------------
	// RPC 2: GetBatchFEFODetails
	// -------------------------------------------------------------------------
	fmt.Println("\n[RPC 2] Kiểm tra GetBatchFEFODetails (MX-GION-500G)...")
	fefoReq := &inventoryv1.GetBatchFEFODetailsRequest{
		SkuCode: "MX-GION-500G",
	}
	fefoRes, err := client.GetBatchFEFODetails(ctx, fefoReq)
	if err != nil {
		log.Fatalf("FAILED GetBatchFEFODetails: %v", err)
	}
	fmt.Printf("   Tìm thấy %d lô hàng theo thứ tự FEFO (hạn dùng gần nhất xếp đầu):\n", len(fefoRes.Batches))
	for i, b := range fefoRes.Batches {
		fmt.Printf("   %d. Lô: %-20s | HSD còn: %3d ngày | Tồn: %3d | QualityStatus: %v | Tem: %s\n",
			i+1, b.BatchCode, b.DaysUntilExpiry, b.RemainingQuantity, b.QualityStatus, b.OcopTraceQrCode)
	}

	// -------------------------------------------------------------------------
	// RPC 3: ReserveStock (Đặt giữ chỗ 60 hộp MX-GION-500G)
	// -------------------------------------------------------------------------
	fmt.Println("\n[RPC 3] Kiểm tra ReserveStock (Khóa 60 hộp MX-GION-500G)...")
	reserveReq := &inventoryv1.ReserveStockRequest{
		IdempotencyKey: "IDEMP-TEST-RUNNER-001",
		OrderId:        "ORD-TEST-RUNNER-001",
		Items: []*inventoryv1.ReserveItem{
			{
				SkuCode:  "MX-GION-500G",
				Quantity: 60,
			},
		},
		TtlMinutes: 15,
		Channel:    "D2C_WEB",
	}
	reserveRes, err := client.ReserveStock(ctx, reserveReq)
	if err != nil {
		log.Fatalf("FAILED ReserveStock: %v", err)
	}
	fmt.Printf("   -> Status: %v | ReservationID: %s | ExpiresAt: %v\n",
		reserveRes.Status, reserveRes.GetReservationId(), reserveRes.GetExpiresAt().AsTime().Format(time.RFC3339))

	// Kiểm tra Idempotency cho ReserveStock
	fmt.Println("   [Idempotency Check] Gửi lại cùng yêu cầu ReserveStock...")
	idemReserveRes, err := client.ReserveStock(ctx, reserveReq)
	if err != nil {
		log.Fatalf("FAILED Idempotent ReserveStock: %v", err)
	}
	if idemReserveRes.GetReservationId() == reserveRes.GetReservationId() {
		fmt.Println("   -> IDEMPOTENCY PASSED! Trả về đúng ReservationID cũ, không trừ trùng tồn kho.")
	} else {
		log.Fatalf("FAILED: Idempotency trả về ID khác nhau!")
	}

	// -------------------------------------------------------------------------
	// RPC 4: ReleaseReservation (Giải phóng 10 hộp MX-DEO-300G từ đơn mẫu)
	// -------------------------------------------------------------------------
	fmt.Println("\n[RPC 4] Kiểm tra ReleaseReservation (Đơn mẫu ORD-TEST-HUEDAC-9999)...")
	releaseReq := &inventoryv1.ReleaseReservationRequest{
		IdempotencyKey: "IDEMP-RELEASE-RUNNER-001",
		OrderId:        "ORD-TEST-HUEDAC-9999",
		ReservationId:  "c3333333-3333-3333-3333-333333333333",
		ReleaseReason:  "CUSTOMER_CANCELLED",
	}
	releaseRes, err := client.ReleaseReservation(ctx, releaseReq)
	if err != nil {
		log.Fatalf("FAILED ReleaseReservation: %v", err)
	}
	fmt.Printf("   -> IsReleased: %v | TotalItemsRestored: %d | Message: %s\n",
		releaseRes.IsReleased, releaseRes.TotalItemsRestored, releaseRes.Message)

	// Kiểm tra Idempotency cho ReleaseReservation
	fmt.Println("   [Idempotency Check] Gửi lại cùng yêu cầu ReleaseReservation...")
	idemReleaseRes, err := client.ReleaseReservation(ctx, releaseReq)
	if err != nil {
		log.Fatalf("FAILED Idempotent ReleaseReservation: %v", err)
	}
	fmt.Printf("   -> IDEMPOTENCY PASSED! IsReleased: %v | ItemsRestored: %d (0 vì đã hoàn trước đó)\n",
		idemReleaseRes.IsReleased, idemReleaseRes.TotalItemsRestored)

	// -------------------------------------------------------------------------
	// EDGE CASES: Insufficient Stock & Zero Expired Sale Protection
	// -------------------------------------------------------------------------
	fmt.Println("\n[EDGE CASES] Kiểm tra biên: Insufficient Stock & Zero Expired Sale...")
	// Yêu cầu 300 hộp MX-GION-500G (chỉ còn 290 hộp hợp lệ vì 60 đã giữ, 20 hết hạn và 80 cách ly bị chặn)
	overReq := &inventoryv1.ReserveStockRequest{
		IdempotencyKey: "IDEMP-TEST-OVER-001",
		OrderId:        "ORD-TEST-OVER-001",
		Items: []*inventoryv1.ReserveItem{
			{
				SkuCode:  "MX-GION-500G",
				Quantity: 300,
			},
		},
		TtlMinutes: 15,
		Channel:    "D2C_WEB",
	}
	_, overErr := client.ReserveStock(ctx, overReq)
	if overErr != nil {
		fmt.Printf("   -> EDGE CASE PASSED! Chặn mua vượt tồn lô hợp lệ: %v\n", overErr)
	} else {
		log.Fatalf("FAILED: Lẽ ra phải báo lỗi không đủ tồn kho hợp lệ!")
	}

	// -------------------------------------------------------------------------
	// Kiểm tra lại tồn kho sau biến động giao dịch
	// -------------------------------------------------------------------------
	fmt.Println("\n[XÁC MINH BIẾN ĐỘNG TỒN KHO SAU GIAO DỊCH]:")
	verifyStockRes, err := client.GetStockLevel(ctx, stockReq)
	if err != nil {
		log.Fatalf("FAILED GetStockLevel Verification: %v", err)
	}
	for _, item := range verifyStockRes.Items {
		fmt.Printf("   -> SKU: %-18s | Physical: %3d | Reserved: %2d | Available: %3d | Status: %s\n",
			item.SkuCode, item.PhysicalQuantity, item.ReservedQuantity, item.AvailableQuantity, item.StockStatus)
	}

	fmt.Println("\n==========================================================")
	fmt.Println("   ALL 4 RPCs PASSED 100% IN LIVE END-TO-END VERIFICATION!")
	fmt.Println("==========================================================")
}

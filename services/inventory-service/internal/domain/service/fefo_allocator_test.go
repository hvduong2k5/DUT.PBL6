package service_test

import (
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"dut-pbl6/inventory-service/internal/domain/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createBatchHelper(t *testing.T, code string, expDays int, qty int, status entity.BatchStatus) *entity.Batch {
	t.Helper()
	now := time.Now()
	mfgDate := now.Add(-30 * 24 * time.Hour)
	expDate := now.Add(time.Duration(expDays) * 24 * time.Hour)

	b, err := entity.NewBatch(
		code,
		"SKU-OC-MEXUNG-001",
		uuid.New(),
		mfgDate,
		expDate,
		qty,
	)
	require.NoError(t, err)
	b.Status = status
	return b
}

// UT-INV-FEFO-01: Case đủ hàng từ 1 lô duy nhất
func TestUT_INV_FEFO_01_SingleBatchSufficient(t *testing.T) {
	// Given: 2 lô hàng (Lô A: 100 chiếc, hết hạn T+30; Lô B: 100 chiếc, hết hạn T+60). Yêu cầu = 50
	batchA := createBatchHelper(t, "BATCH-A", 30, 100, entity.BatchStatusActive)
	batchB := createBatchHelper(t, "BATCH-B", 60, 100, entity.BatchStatusActive)
	batches := []*entity.Batch{batchB, batchA} // Pass in unordered

	// When: Chạy hàm AllocateByFEFO(batches, 50)
	results, err := service.AllocateByFEFO(batches, 50)

	// Then: Trả về kết quả phân bổ lấy 50 chiếc từ Lô A. Lô B không bị trừ.
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, batchA.ID, results[0].BatchID)
	assert.Equal(t, 50, results[0].AllocatedQty)
	assert.Equal(t, 50, batchA.ReservedQty)
	assert.Equal(t, 50, batchA.AvailableQty())
	assert.Equal(t, 0, batchB.ReservedQty)
	assert.Equal(t, 100, batchB.AvailableQty())
}

// UT-INV-FEFO-02: Case cần gom từ nhiều lô theo thứ tự FEFO (exp_date ASC)
func TestUT_INV_FEFO_02_MultiBatchAllocation(t *testing.T) {
	// Given: Lô A (50 chiếc, hết hạn sớm), Lô B (100 chiếc, hết hạn muộn hơn). Yêu cầu = 120
	batchA := createBatchHelper(t, "BATCH-A", 15, 50, entity.BatchStatusActive)
	batchB := createBatchHelper(t, "BATCH-B", 45, 100, entity.BatchStatusActive)
	batches := []*entity.Batch{batchB, batchA}

	// When: Chạy hàm AllocateByFEFO(batches, 120)
	results, err := service.AllocateByFEFO(batches, 120)

	// Then: Phân bổ thành công: Lấy 50 chiếc từ Lô A, 70 chiếc từ Lô B
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, batchA.ID, results[0].BatchID)
	assert.Equal(t, 50, results[0].AllocatedQty)
	assert.Equal(t, 50, batchA.ReservedQty)
	assert.Equal(t, 0, batchA.AvailableQty())

	assert.Equal(t, batchB.ID, results[1].BatchID)
	assert.Equal(t, 70, results[1].AllocatedQty)
	assert.Equal(t, 70, batchB.ReservedQty)
	assert.Equal(t, 30, batchB.AvailableQty())
}

// UT-INV-FEFO-03: Case tổng tồn khả dụng không đủ (ErrInsufficientStock)
func TestUT_INV_FEFO_03_InsufficientStock(t *testing.T) {
	// Given: Lô A (50), Lô B (50). Yêu cầu = 150
	batchA := createBatchHelper(t, "BATCH-A", 20, 50, entity.BatchStatusActive)
	batchB := createBatchHelper(t, "BATCH-B", 40, 50, entity.BatchStatusActive)
	batches := []*entity.Batch{batchA, batchB}

	// When: Chạy hàm AllocateByFEFO(batches, 150)
	results, err := service.AllocateByFEFO(batches, 150)

	// Then: Trả về lỗi ErrInsufficientStock và không có danh sách phân bổ nào được tạo
	assert.ErrorIs(t, err, entity.ErrInsufficientStock)
	assert.Nil(t, results)
	assert.Equal(t, 0, batchA.ReservedQty, "Batch A should remain untouched")
	assert.Equal(t, 0, batchB.ReservedQty, "Batch B should remain untouched")
}

// UT-INV-FEFO-04: Case có lô xen kẽ bị quá hạn hoặc trạng thái không phải ACTIVE
func TestUT_INV_FEFO_04_SkipInactiveAndExpiredBatches(t *testing.T) {
	// Given: Lô A (50, EXPIRED), Lô B (50, QUARANTINE), Lô C (100, ACTIVE, hết hạn xa nhất). Yêu cầu = 50
	batchA := createBatchHelper(t, "BATCH-A-EXP", -5, 50, entity.BatchStatusExpired)
	batchB := createBatchHelper(t, "BATCH-B-QUAR", 30, 50, entity.BatchStatusQuarantine)
	batchC := createBatchHelper(t, "BATCH-C-ACT", 90, 100, entity.BatchStatusActive)
	batches := []*entity.Batch{batchA, batchB, batchC}

	// When: Chạy hàm AllocateByFEFO(batches, 50)
	results, err := service.AllocateByFEFO(batches, 50)

	// Then: Bỏ qua Lô A và Lô B. Phân bổ lấy toàn bộ 50 chiếc từ Lô C
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, batchC.ID, results[0].BatchID)
	assert.Equal(t, 50, results[0].AllocatedQty)
	assert.Equal(t, 0, batchA.ReservedQty)
	assert.Equal(t, 0, batchB.ReservedQty)
	assert.Equal(t, 50, batchC.ReservedQty)
	assert.Equal(t, 50, batchC.AvailableQty())
}

// UT-INV-FEFO-05: Case biên (Edge Cases) - Lấy số lượng <= 0
func TestUT_INV_FEFO_05_InvalidAllocationQuantity(t *testing.T) {
	batch := createBatchHelper(t, "BATCH-A", 30, 100, entity.BatchStatusActive)
	batches := []*entity.Batch{batch}

	// When & Then: Yêu cầu = 0 -> ErrInvalidAllocationRequest
	results0, err0 := service.AllocateByFEFO(batches, 0)
	assert.ErrorIs(t, err0, entity.ErrInvalidAllocationRequest)
	assert.Nil(t, results0)

	// When & Then: Yêu cầu = -10 -> ErrInvalidAllocationRequest
	resultsNeg, errNeg := service.AllocateByFEFO(batches, -10)
	assert.ErrorIs(t, errNeg, entity.ErrInvalidAllocationRequest)
	assert.Nil(t, resultsNeg)
}

// UT-INV-FEFO-06: Case biên (Edge Cases) - Lấy chính xác 1 đơn vị khả dụng cuối cùng của kho hàng
func TestUT_INV_FEFO_06_AllocateLastSingleUnit(t *testing.T) {
	// Given: Duy nhất 1 lô còn đúng 1 đơn vị khả dụng (Q_avail = 1). Yêu cầu mua đúng 1 đơn vị (req_qty = 1)
	batch := createBatchHelper(t, "BATCH-LAST-1", 10, 1, entity.BatchStatusActive)
	batches := []*entity.Batch{batch}

	assert.Equal(t, 1, batch.AvailableQty())

	// When: Chạy hàm AllocateByFEFO(batches, 1)
	results, err := service.AllocateByFEFO(batches, 1)

	// Then: Phân bổ thành công đúng 1 đơn vị từ lô này, số dư khả dụng còn lại của lô đạt đúng bằng 0
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, batch.ID, results[0].BatchID)
	assert.Equal(t, 1, results[0].AllocatedQty)
	assert.Equal(t, 1, batch.ReservedQty)
	assert.Equal(t, 0, batch.AvailableQty(), "Available quantity must reach exactly 0")
}

func TestAllocateByFEFO_ExpiredActiveBatchRejected(t *testing.T) {
	// A batch with Status = ACTIVE but ExpDate in the past must not be allocated (INV-BI-02)
	now := time.Now()
	expiredBatch := createBatchHelper(t, "BATCH-EXPIRED-ACTIVE", -2, 50, entity.BatchStatusActive)
	validBatch := createBatchHelper(t, "BATCH-VALID", 30, 50, entity.BatchStatusActive)

	batches := []*entity.Batch{expiredBatch, validBatch}

	results, err := service.AllocateByFEFOAt(batches, 50, now)
	require.NoError(t, err)
	require.Len(t, results, 1)
	// Must allocate from the valid batch, completely ignoring the expired one
	assert.Equal(t, validBatch.ID, results[0].BatchID)
	assert.Equal(t, 0, expiredBatch.ReservedQty)
	assert.Equal(t, entity.BatchStatusExpired, expiredBatch.Status)
}

func TestAllocateByFEFO_MixedSKUsRejected(t *testing.T) {
	batchA := createBatchHelper(t, "BATCH-A", 30, 50, entity.BatchStatusActive)
	batchB := createBatchHelper(t, "BATCH-B", 30, 50, entity.BatchStatusActive)
	batchB.SKU = "SKU-OTHER-002" // Different SKU

	batches := []*entity.Batch{batchA, batchB}

	results, err := service.AllocateByFEFO(batches, 60)
	assert.ErrorIs(t, err, entity.ErrInvalidBatchData)
	assert.Nil(t, results)
}

func TestAllocateByFEFO_NearExpiryPrioritized(t *testing.T) {
	// Near expiry batch (exp in 15 days, status NEAR_EXPIRY) must be allocated before normal batch (exp in 60 days)
	batchNear := createBatchHelper(t, "BATCH-NEAR", 15, 30, entity.BatchStatusNearExpiry)
	batchNormal := createBatchHelper(t, "BATCH-NORM", 60, 50, entity.BatchStatusActive)

	batches := []*entity.Batch{batchNormal, batchNear}

	allocator := service.NewFEFOAllocator()
	results, err := allocator.Allocate(batches, 40)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, batchNear.ID, results[0].BatchID, "Near expiry batch must be allocated first")
	assert.Equal(t, 30, results[0].AllocatedQty)
	assert.Equal(t, batchNormal.ID, results[1].BatchID)
	assert.Equal(t, 10, results[1].AllocatedQty)
}

func TestAllocationResult_ToReservationAllocation(t *testing.T) {
	resID := uuid.New()
	batchID := uuid.New()
	ar := service.AllocationResult{
		BatchID:      batchID,
		BatchCode:    "LOT-1",
		SKU:          "SKU-001",
		AllocatedQty: 15,
	}
	resAlloc := ar.ToReservationAllocation(resID)
	assert.Equal(t, resID, resAlloc.ReservationID)
	assert.Equal(t, batchID, resAlloc.BatchID)
	assert.Equal(t, "SKU-001", resAlloc.SKU)
	assert.Equal(t, 15, resAlloc.AllocatedQty)
	assert.NotEqual(t, uuid.Nil, resAlloc.ID)
}

func TestAllocateByFEFOAt_DeterministicTimestamp(t *testing.T) {
	now := time.Now()
	// Batch expiring in 10 days
	batch := createBatchHelper(t, "BATCH-DET", 10, 50, entity.BatchStatusActive)
	batches := []*entity.Batch{batch}

	// 1. Simulated now is at Day 5: batch is still valid, allocation succeeds and reserves
	simulatedNow := now.Add(5 * 24 * time.Hour)
	results, err := service.AllocateByFEFOAt(batches, 20, simulatedNow)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 20, results[0].AllocatedQty)
	assert.Equal(t, 20, batch.ReservedQty)

	// 2. Simulated now is at Day 12: batch is expired relative to futureNow, allocation fails
	futureNow := now.Add(12 * 24 * time.Hour)
	batch2 := createBatchHelper(t, "BATCH-DET-2", 10, 50, entity.BatchStatusActive)
	results2, err2 := service.AllocateByFEFOAt([]*entity.Batch{batch2}, 10, futureNow)
	assert.ErrorIs(t, err2, entity.ErrInsufficientStock)
	assert.Nil(t, results2)
	assert.Equal(t, entity.BatchStatusExpired, batch2.Status)
}

package entity_test

import (
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UT-INV-DOMAIN-01: Kiểm tra INV-BI-01: Cấm tuyệt đối tồn kho âm (Q_avail = Q_phy - Q_res >= 0)
func TestUT_INV_DOMAIN_01_NonNegativeAvailableStock(t *testing.T) {
	// Given: Một sản phẩm có Physical Quantity = 100, Reserved Quantity = 90
	item, err := entity.NewInventoryItem("SKU-OC-MEXUNG-001", entity.DefaultWarehouseID, 100)
	require.NoError(t, err)
	item.ReservedQty = 90

	assert.Equal(t, 10, item.AvailableQty())

	// When: Thực hiện yêu cầu giữ thêm (reserve) số lượng 20 đơn vị
	err = item.Reserve(20)

	// Then: Hàm trả về lỗi ErrInsufficientStock, các thuộc tính số lượng không bị thay đổi
	assert.ErrorIs(t, err, entity.ErrInsufficientStock)
	assert.Equal(t, 100, item.PhysicalQty, "Physical quantity must remain unchanged")
	assert.Equal(t, 90, item.ReservedQty, "Reserved quantity must remain unchanged")
	assert.Equal(t, 10, item.AvailableQty(), "Available quantity must remain 10")
}

// UT-INV-DOMAIN-02: Kiểm tra INV-BI-02: Lô có hạn sử dụng quá khứ tự động chuyển EXPIRED
func TestUT_INV_DOMAIN_02_ExpiredBatchHandling(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	mfgDate := yesterday.Add(-30 * 24 * time.Hour)

	// Given: Một lô hàng đang ở trạng thái ACTIVE, ngày hết hạn (exp_date) là ngày hôm qua
	batch, err := entity.NewBatch(
		"LOT-2026-001",
		"SKU-OC-MEXUNG-001",
		uuid.New(),
		mfgDate,
		yesterday,
		100,
	)
	require.NoError(t, err)
	assert.Equal(t, entity.BatchStatusActive, batch.Status)

	// When: Gọi phương thức CheckExpiry(current_date)
	isExpired := batch.CheckExpiry(now)

	// Then: Trạng thái lô hàng đổi thành EXPIRED và CanAllocate() trả về false
	assert.True(t, isExpired)
	assert.Equal(t, entity.BatchStatusExpired, batch.Status)
	assert.False(t, batch.CanAllocate(), "Expired batch must not be allocatable")
}

// UT-INV-DOMAIN-03: Kiểm tra INV-BI-04: Tính hợp lệ của dữ liệu lô khi tạo mới
func TestUT_INV_DOMAIN_03_BatchValidationInvariants(t *testing.T) {
	now := time.Now()
	mfgDate := now
	expDateInvalid := now.Add(-10 * 24 * time.Hour) // mfg > exp
	expDateValid := now.Add(90 * 24 * time.Hour)
	supplierID := uuid.New()

	// Given 1: Thông tin khởi tạo với mfg_date sau exp_date
	// When & Then: Trả về lỗi ErrInvalidBatchData và đối tượng trả về là nil
	batchInvalidDate, err1 := entity.NewBatch(
		"LOT-2026-ERR1",
		"SKU-OC-MEXUNG-001",
		supplierID,
		mfgDate,
		expDateInvalid,
		100,
	)
	assert.ErrorIs(t, err1, entity.ErrInvalidBatchData)
	assert.Nil(t, batchInvalidDate)

	// Given 2: Thông tin khởi tạo với physical_qty = 0
	// When & Then: Trả về lỗi ErrInvalidBatchData và đối tượng trả về là nil
	batchZeroQty, err2 := entity.NewBatch(
		"LOT-2026-ERR2",
		"SKU-OC-MEXUNG-001",
		supplierID,
		mfgDate,
		expDateValid,
		0,
	)
	assert.ErrorIs(t, err2, entity.ErrInvalidBatchData)
	assert.Nil(t, batchZeroQty)

	// Given 3: physical_qty < 0
	batchNegativeQty, err3 := entity.NewBatch(
		"LOT-2026-ERR3",
		"SKU-OC-MEXUNG-001",
		supplierID,
		mfgDate,
		expDateValid,
		-5,
	)
	assert.ErrorIs(t, err3, entity.ErrInvalidBatchData)
	assert.Nil(t, batchNegativeQty)
}

func TestBatch_NearExpiryTransition(t *testing.T) {
	now := time.Now()
	// Batch expiring in 30 days (< 45 days)
	expDate := now.Add(30 * 24 * time.Hour)
	mfgDate := now.Add(-30 * 24 * time.Hour)

	batch, err := entity.NewBatch("LOT-NEAR", "SKU-001", uuid.New(), mfgDate, expDate, 100)
	require.NoError(t, err)
	assert.Equal(t, entity.BatchStatusActive, batch.Status)

	// When: CheckExpiry called
	isExpired := batch.CheckExpiry(now)

	// Then: Status becomes NEAR_EXPIRY, not expired, and CanAllocate returns true
	assert.False(t, isExpired)
	assert.Equal(t, entity.BatchStatusNearExpiry, batch.Status)
	assert.True(t, batch.CanAllocate())
}

func TestBatch_CanAllocateAutoExpire(t *testing.T) {
	now := time.Now()
	// Batch expired 1 hour ago
	expDate := now.Add(-1 * time.Hour)
	mfgDate := now.Add(-60 * 24 * time.Hour)

	batch, err := entity.NewBatch("LOT-AUTOEXP", "SKU-001", uuid.New(), mfgDate, expDate, 100)
	require.NoError(t, err)

	// Without calling CheckExpiry explicitly, CanAllocate must detect past expiry date and return false
	assert.False(t, batch.CanAllocate())
	assert.Equal(t, entity.BatchStatusExpired, batch.Status)
}

func TestBatch_ReserveReleaseCommit(t *testing.T) {
	now := time.Now()
	batch, err := entity.NewBatch("LOT-RRC", "SKU-001", uuid.New(), now.Add(-10*24*time.Hour), now.Add(60*24*time.Hour), 100)
	require.NoError(t, err)

	// Reserve invalid
	assert.ErrorIs(t, batch.Reserve(0), entity.ErrInvalidQuantity)
	assert.ErrorIs(t, batch.Reserve(-5), entity.ErrInvalidQuantity)
	assert.ErrorIs(t, batch.Reserve(120), entity.ErrInsufficientStock)

	// Reserve valid
	require.NoError(t, batch.Reserve(40))
	assert.Equal(t, 40, batch.ReservedQty)
	assert.Equal(t, 60, batch.AvailableQty())

	// Release invalid
	assert.ErrorIs(t, batch.Release(0), entity.ErrInvalidQuantity)
	assert.ErrorIs(t, batch.Release(50), entity.ErrInvalidQuantity)

	// Release valid
	require.NoError(t, batch.Release(10))
	assert.Equal(t, 30, batch.ReservedQty)
	assert.Equal(t, 70, batch.AvailableQty())

	// Commit invalid
	assert.ErrorIs(t, batch.Commit(0), entity.ErrInvalidQuantity)
	assert.ErrorIs(t, batch.Commit(40), entity.ErrInsufficientStock)

	// Commit valid
	require.NoError(t, batch.Commit(30))
	assert.Equal(t, 70, batch.PhysicalQty)
	assert.Equal(t, 0, batch.ReservedQty)
	assert.Equal(t, 70, batch.AvailableQty())
}

func TestBatch_ReserveAt(t *testing.T) {
	now := time.Now()
	expDate := now.Add(30 * 24 * time.Hour)
	batch, err := entity.NewBatch("LOT-RAT", "SKU-001", uuid.New(), now.Add(-10*24*time.Hour), expDate, 100)
	require.NoError(t, err)

	simulatedNow := now.Add(10 * 24 * time.Hour)

	// Invalid quantities
	assert.ErrorIs(t, batch.ReserveAt(0, simulatedNow), entity.ErrInvalidQuantity)
	assert.ErrorIs(t, batch.ReserveAt(-10, simulatedNow), entity.ErrInvalidQuantity)

	// Valid reservation before expiry
	require.NoError(t, batch.ReserveAt(25, simulatedNow))
	assert.Equal(t, 25, batch.ReservedQty)
	assert.Equal(t, 75, batch.AvailableQty())

	// Quantity exceeds available
	assert.ErrorIs(t, batch.ReserveAt(80, simulatedNow), entity.ErrInsufficientStock)

	// Reservation after expiry fails and auto-expires batch
	futureExpired := expDate.Add(1 * time.Hour)
	assert.ErrorIs(t, batch.ReserveAt(10, futureExpired), entity.ErrBatchCannotAllocate)
	assert.Equal(t, entity.BatchStatusExpired, batch.Status)

	// Non-allocatable status (Quarantine)
	batchQuarantine, err := entity.NewBatch("LOT-QUAR", "SKU-001", uuid.New(), now.Add(-10*24*time.Hour), expDate, 100)
	require.NoError(t, err)
	batchQuarantine.Status = entity.BatchStatusQuarantine
	assert.ErrorIs(t, batchQuarantine.ReserveAt(10, simulatedNow), entity.ErrBatchCannotAllocate)
}


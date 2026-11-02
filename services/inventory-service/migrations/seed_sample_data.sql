-- =============================================================================
-- SEED DATA: DỮ LIỆU MẪU KHO HÀNG & LÔ HÀNG FEFO (O MẠ HUẾ - MÈ XỬNG OCOP)
-- Database: om_inventory_db
-- Dịch vụ: MS-01 Inventory Service (gRPC Port 8001)
-- =============================================================================

-- 1. DỌN DẸP DỮ LIỆU CŨ ĐỂ ĐẢM BẢO TÍNH BẤT BIẾN (IDEMPOTENT RE-RUN)
TRUNCATE TABLE stock_reservation_allocations, stock_reservations, stock_adjustments, batches, inventory_items, outbox_events, idempotency_keys RESTART IDENTITY CASCADE;

-- 2. ĐỊNH NGHĨA DANH MỤC TỒN KHO THEO SKU (INVENTORY_ITEMS)
-- Kho Tổng O Mạ Huế (Warehouse ID: 'a1111111-1111-1111-1111-111111111111')
-- Lưu ý: available_qty là cột GENERATED ALWAYS AS (physical_qty - reserved_qty) STORED

INSERT INTO inventory_items (sku, warehouse_id, physical_qty, reserved_qty, status, updated_at)
VALUES
    -- SKU 1: Mè xửng giòn Huế Thượng Hạng (Hộp 500g) - SKU chủ lực OCOP 4 sao
    ('MX-GION-500G', 'a1111111-1111-1111-1111-111111111111', 450, 0, 'ACTIVE', CURRENT_TIMESTAMP),

    -- SKU 2: Mè xửng dẻo Cung Đình truyền thống (Gói 300g) - Có sẵn 10 gói đang bị tạm khóa giữ chỗ
    ('MX-DEO-300G', 'a1111111-1111-1111-1111-111111111111', 230, 10, 'ACTIVE', CURRENT_TIMESTAMP),

    -- SKU 3: Kẹo mè gương đậu phộng Huế O Mạ (Hộp 250g)
    ('KE-ME-GUONG-250G', 'a1111111-1111-1111-1111-111111111111', 85, 0, 'ACTIVE', CURRENT_TIMESTAMP),

    -- SKU 4: Mè xửng khoai lang tím Huế (Hộp 400g) - Tồn ít (LOW_STOCK <= 10) để test cảnh báo
    ('MX-KHOAI-LANG-400G', 'a1111111-1111-1111-1111-111111111111', 8, 0, 'ACTIVE', CURRENT_TIMESTAMP),

    -- SKU 5: Mè xửng mật ong A Lưới (Hộp 350g) - Hết hàng / Tạm ngưng (SUSPENDED / OUT_OF_STOCK)
    ('MX-MAT-ONG-350G', 'a1111111-1111-1111-1111-111111111111', 0, 0, 'SUSPENDED', CURRENT_TIMESTAMP);

-- 3. DANH SÁCH LÔ HÀNG SẢN XUẤT (BATCHES) VỚI CÁC TRẠNG THÁI FEFO ĐA DẠNG
-- Nhà xưởng Mè Xửng O Mạ Kim Long: 'b2222222-2222-2222-2222-222222222222'
-- Nhà xưởng Kẹo Mè Gương Vỹ Dạ:     'b3333333-3333-3333-3333-333333333333'

-- =============================================================================
-- [SKU 1: MX-GION-500G] - Bộ dữ liệu toàn diện kiểm thử thuật toán FEFO & Zero Expired Sale
-- =============================================================================
INSERT INTO batches (id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at)
VALUES
    -- Lô 1: ĐÃ HẾT HẠN (EXPIRED) - Hết hạn cách đây 10 ngày.
    -- Kịch bản: Thuật toán FEFO PHẢI BỎ QUA lô này, tuyệt đối không xuất bán (Zero Expired Sale).
    (
        'd1111111-0001-0000-0000-000000000001',
        'LOT-OMA-MXG-EXP-01',
        'MX-GION-500G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '120 days')::date,
        (CURRENT_DATE - INTERVAL '10 days')::date,
        20, 0, 'EXPIRED',
        CURRENT_TIMESTAMP - INTERVAL '120 days'
    ),

    -- Lô 2: CẬN DATE (NEAR_EXPIRY) - Còn 25 ngày nữa hết hạn (ngưỡng cận date OCOP là <= 45 ngày).
    -- Kịch bản: Ưu tiên xuất kho ĐẦU TIÊN theo FEFO (First Expired, First Out).
    (
        'd1111111-0001-0000-0000-000000000002',
        'LOT-OMA-MXG-NEAR-02',
        'MX-GION-500G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '65 days')::date,
        (CURRENT_DATE + INTERVAL '25 days')::date,
        50, 0, 'NEAR_EXPIRY',
        CURRENT_TIMESTAMP - INTERVAL '65 days'
    ),

    -- Lô 3: ĐANG LƯU HÀNH BÌNH THƯỜNG (ACTIVE) - Còn 80 ngày nữa hết hạn.
    -- Kịch bản: Ưu tiên xuất kho THỨ HAI sau khi lô NEAR-02 được cấp phát hết.
    (
        'd1111111-0001-0000-0000-000000000003',
        'LOT-OMA-MXG-ACT-03',
        'MX-GION-500G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '20 days')::date,
        (CURRENT_DATE + INTERVAL '80 days')::date,
        100, 0, 'ACTIVE',
        CURRENT_TIMESTAMP - INTERVAL '20 days'
    ),

    -- Lô 4: LÔ MỚI XUẤT XƯỞNG (ACTIVE) - Còn 150 ngày nữa hết hạn.
    -- Kịch bản: Ưu tiên xuất kho CUỐI CÙNG trong nhóm hợp lệ.
    (
        'd1111111-0001-0000-0000-000000000004',
        'LOT-OMA-MXG-ACT-04',
        'MX-GION-500G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '3 days')::date,
        (CURRENT_DATE + INTERVAL '150 days')::date,
        200, 0, 'ACTIVE',
        CURRENT_TIMESTAMP - INTERVAL '3 days'
    ),

    -- Lô 5: ĐANG CÁCH LY / KIỂM ĐỊNH (QUARANTINE) - Đang chờ kết quả mẫu vi sinh OCOP Huế.
    -- Kịch bản: Dù còn hạn 180 ngày, FEFO và ReserveStock TUYỆT ĐỐI KHÔNG ĐƯỢC PHÉP khóa/xuất lô này.
    (
        'd1111111-0001-0000-0000-000000000005',
        'LOT-OMA-MXG-QRN-05',
        'MX-GION-500G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '1 days')::date,
        (CURRENT_DATE + INTERVAL '180 days')::date,
        80, 0, 'QUARANTINE',
        CURRENT_TIMESTAMP - INTERVAL '1 days'
    );

-- =============================================================================
-- [SKU 2: MX-DEO-300G] - Dữ liệu phục vụ kiểm thử ReleaseReservation & Allocation
-- =============================================================================
INSERT INTO batches (id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at)
VALUES
    -- Lô cận date có 10 sản phẩm đang bị tạm khóa giữ chỗ bởi đơn hàng mẫu ORD-TEST-HUEDAC-9999
    (
        'd2222222-0002-0000-0000-000000000001',
        'LOT-OMA-MXD-NEAR-01',
        'MX-DEO-300G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '70 days')::date,
        (CURRENT_DATE + INTERVAL '18 days')::date,
        30, 10, 'NEAR_EXPIRY',
        CURRENT_TIMESTAMP - INTERVAL '70 days'
    ),
    -- Lô tiêu chuẩn đang lưu thông bình thường
    (
        'd2222222-0002-0000-0000-000000000002',
        'LOT-OMA-MXD-ACT-02',
        'MX-DEO-300G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '15 days')::date,
        (CURRENT_DATE + INTERVAL '90 days')::date,
        150, 0, 'ACTIVE',
        CURRENT_TIMESTAMP - INTERVAL '15 days'
    ),
    -- Lô cách ly kiểm định
    (
        'd2222222-0002-0000-0000-000000000003',
        'LOT-OMA-MXD-QRN-03',
        'MX-DEO-300G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '2 days')::date,
        (CURRENT_DATE + INTERVAL '120 days')::date,
        50, 0, 'QUARANTINE',
        CURRENT_TIMESTAMP - INTERVAL '2 days'
    );

-- =============================================================================
-- [SKU 3: KE-ME-GUONG-250G] & [SKU 4: MX-KHOAI-LANG-400G]
-- =============================================================================
INSERT INTO batches (id, batch_code, sku, supplier_id, mfg_date, exp_date, physical_qty, reserved_qty, status, created_at)
VALUES
    (
        'd3333333-0003-0000-0000-000000000001',
        'LOT-OMA-KMG-NEAR-01',
        'KE-ME-GUONG-250G',
        'b3333333-3333-3333-3333-333333333333',
        (CURRENT_DATE - INTERVAL '60 days')::date,
        (CURRENT_DATE + INTERVAL '30 days')::date,
        25, 0, 'NEAR_EXPIRY',
        CURRENT_TIMESTAMP - INTERVAL '60 days'
    ),
    (
        'd3333333-0003-0000-0000-000000000002',
        'LOT-OMA-KMG-ACT-02',
        'KE-ME-GUONG-250G',
        'b3333333-3333-3333-3333-333333333333',
        (CURRENT_DATE - INTERVAL '10 days')::date,
        (CURRENT_DATE + INTERVAL '100 days')::date,
        60, 0, 'ACTIVE',
        CURRENT_TIMESTAMP - INTERVAL '10 days'
    ),
    (
        'd4444444-0004-0000-0000-000000000001',
        'LOT-OMA-MXKL-ACT-01',
        'MX-KHOAI-LANG-400G',
        'b2222222-2222-2222-2222-222222222222',
        (CURRENT_DATE - INTERVAL '5 days')::date,
        (CURRENT_DATE + INTERVAL '110 days')::date,
        8, 0, 'ACTIVE',
        CURRENT_TIMESTAMP - INTERVAL '5 days'
    );

-- =============================================================================
-- 4. PHIẾU GIỮ CHỖ MẪU (STOCK RESERVATIONS) ĐỂ TEST RPC ReleaseReservation
-- =============================================================================
-- Phiếu giữ chỗ này tương ứng với 10 hộp MX-DEO-300G đang bị khóa ở lô LOT-OMA-MXD-NEAR-01
INSERT INTO stock_reservations (id, order_id, status, expires_at, created_at, updated_at)
VALUES
    (
        'c3333333-3333-3333-3333-333333333333',
        'ORD-TEST-HUEDAC-9999',
        'PENDING',
        CURRENT_TIMESTAMP + INTERVAL '2 hours',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    );

INSERT INTO stock_reservation_allocations (id, reservation_id, sku, batch_id, allocated_qty)
VALUES
    (
        'e1111111-0001-0000-0000-000000000001',
        'c3333333-3333-3333-3333-333333333333',
        'MX-DEO-300G',
        'd2222222-0002-0000-0000-000000000001',
        10
    );

-- =============================================================================
-- XÁC NHẬN TỔNG KHO SAU KHI SEED (VERIFICATION QUERY)
-- =============================================================================
-- SELECT sku, physical_qty, reserved_qty, available_qty, status FROM inventory_items ORDER BY sku;
-- SELECT batch_code, sku, mfg_date, exp_date, physical_qty, reserved_qty, status FROM batches ORDER BY sku, exp_date ASC;

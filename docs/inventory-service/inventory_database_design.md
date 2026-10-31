# Thiết kế Cơ sở Dữ liệu Dịch vụ Kho hàng (Inventory Database Design)

## 1. Schema PostgreSQL (inventory_db)
Được thiết kế chuẩn 3NF, tối ưu cho xử lý Transaction.

### 1.1. DDL Script
```sql
CREATE TABLE inventory_items (
    sku VARCHAR(50) PRIMARY KEY,
    warehouse_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001', -- Kho trung tâm xưởng O Mạ (Huế) trong giai đoạn MVP
    physical_qty INT NOT NULL DEFAULT 0 CHECK (physical_qty >= 0),
    reserved_qty INT NOT NULL DEFAULT 0 CHECK (reserved_qty >= 0),
    available_qty INT GENERATED ALWAYS AS (physical_qty - reserved_qty) STORED,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_avail_qty CHECK (physical_qty >= reserved_qty)
);

CREATE TABLE batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_code VARCHAR(100) NOT NULL,
    sku VARCHAR(50) NOT NULL REFERENCES inventory_items(sku),
    supplier_id UUID NOT NULL,
    mfg_date DATE NOT NULL,
    exp_date DATE NOT NULL,
    physical_qty INT NOT NULL DEFAULT 0 CHECK (physical_qty >= 0),
    reserved_qty INT NOT NULL DEFAULT 0 CHECK (reserved_qty >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'QUARANTINE', -- QUARANTINE, ACTIVE, NEAR_EXPIRY, EXPIRED
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_dates CHECK (mfg_date < exp_date),
    CONSTRAINT check_batch_avail_qty CHECK (physical_qty >= reserved_qty)
);

CREATE TABLE stock_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id VARCHAR(50) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- PENDING, COMMITTED, RELEASED, EXPIRED
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE stock_reservation_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id UUID NOT NULL REFERENCES stock_reservations(id),
    sku VARCHAR(50) NOT NULL,
    batch_id UUID NOT NULL REFERENCES batches(id),
    allocated_qty INT NOT NULL CHECK (allocated_qty > 0)
);

CREATE TABLE stock_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adjustment_code VARCHAR(50) NOT NULL UNIQUE,
    sku VARCHAR(50) NOT NULL REFERENCES inventory_items(sku),
    batch_id UUID REFERENCES batches(id),
    adjustment_type VARCHAR(30) NOT NULL, -- DISCREPANCY, DAMAGED, LOSS, RECLASSIFICATION
    quantity_delta INT NOT NULL,
    reason TEXT,
    adjusted_by UUID NOT NULL,
    approved_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE idempotency_keys (
    key_name VARCHAR(100) PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT
);
```

### 1.2. Indexes (Chỉ mục) tối ưu hóa truy vấn
- **Chỉ mục tối ưu thuật toán FEFO:**
  Cần tìm nhanh các batch còn hạn, trạng thái ACTIVE hoặc NEAR_EXPIRY, sắp xếp theo ngày hết hạn (First-Expired, First-Out).
  ```sql
  CREATE INDEX idx_batches_fefo ON batches(sku, exp_date ASC) WHERE status IN ('ACTIVE', 'NEAR_EXPIRY');
  ```
- **Chỉ mục Worker dọn dẹp Reservation:**
  Giúp Cron Worker quét nhanh các Reservation đã hết hạn.
  ```sql
  CREATE INDEX idx_reservations_pending_expire ON stock_reservations(expires_at) WHERE status = 'PENDING';
  ```
- **Chỉ mục Worker kiểm tra hạn sử dụng:**
  ```sql
  CREATE INDEX idx_batches_expiry ON batches(exp_date) WHERE status IN ('ACTIVE', 'NEAR_EXPIRY');
  ```
- **Chỉ mục Transactional Outbox Poller:**
  Giúp Outbox Relay quét nhanh các sự kiện chờ phát lên Kafka theo thứ tự thời gian.
  ```sql
  CREATE INDEX idx_outbox_pending ON outbox_events(created_at ASC) WHERE status = 'PENDING';
  ```

## 2. Thiết kế Caching và Distributed Lock (Redis Key Schema)

### 2.1. Distributed Lock (Redlock)
Mục đích: Khóa dòng thực thi logic với cùng 1 SKU tại 1 thời điểm trên nhiều pod/server (Concurrency Control).
- **Key Pattern:** `lock:inventory:sku:{sku}`
- **Type:** String
- **TTL:** 3000ms (3 giây). Ngắn để tránh kẹt lock, nhưng đủ dài cho 1 db transaction.
- **Value:** UUID ngẫu nhiên của worker giữ lock.

### 2.2. Stock Level Cache
Mục đích: Trả về số lượng khả dụng `available_qty` ngay lập tức cho API `GetStockLevel` mà không cần hit DB.
- **Key Pattern:** `cache:inventory:stock:{sku}`
- **Type:** String (Integer string)
- **TTL:** 10 phút.
- **Value:** Số lượng khả dụng hiện tại. (Được Invalidate hoặc Update đè mỗi khi có thao tác ReserveStock, ReleaseStock, ReceiveGoods thành công).

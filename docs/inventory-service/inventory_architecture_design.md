# Thiết kế Kiến trúc Dịch vụ Kho hàng (Inventory Service)

## 1. Mục tiêu và Ràng buộc hệ thống
- **Vai trò:** Dịch vụ cốt lõi (Core Domain), nằm trên Critical Path của luồng Thanh toán (Checkout).
- **Yêu cầu phi chức năng:** 
  - SLA: 99.99%.
  - Độ trễ (Latency): < 50ms cho các API gRPC nội bộ.
  - Đồng thời (Concurrency): Chịu tải cao, chống hiện tượng đua (Race Condition) và bán lố (Overselling) trong các dịp Flash Sale.
  - Tính nhất quán dữ liệu (Consistency): Đảm bảo giao dịch ACID tuyệt đối trong CSDL.

## 2. Quyết định Công nghệ & Ngôn ngữ (Tech Stack)

### 2.1. Ngôn ngữ lập trình: Go (Golang)
- **Lý do chọn Go:**
  - **Hiệu suất và Độ trễ cực thấp:** Go là ngôn ngữ biên dịch trực tiếp ra mã máy, không có máy ảo cồng kềnh như Java, thời gian khởi động nhanh và độ trễ dọn rác (GC pause) cực nhỏ, dễ dàng đáp ứng SLA < 50ms.
  - **Xử lý đồng thời (Concurrency) xuất sắc:** Với `goroutines` nhẹ và `channels`, Go xử lý hàng chục nghìn connection và luồng logic đồng thời rất hiệu quả.
  - **Tương thích hoàn hảo với gRPC:** Go được Google hỗ trợ sâu sắc về gRPC/Protobuf, code sinh ra cực kỳ nhẹ và tối ưu.

### 2.2. Cơ sở dữ liệu: PostgreSQL
- **Lý do chọn PostgreSQL:**
  - Hỗ trợ ACID Transactions nghiêm ngặt.
  - Hỗ trợ cơ chế khóa bản ghi `SELECT ... FOR UPDATE` mạnh mẽ để chống Over-selling ở tầng Database.
  - Hiệu suất truy vấn phức tạp (như FEFO) rất tốt, Indexing thông minh.

### 2.3. Caching & Distributed Lock: Redis
- Cung cấp cơ chế **Redlock** (thông qua thư viện như `redsync`) để tạo Distributed Lock chặn Race Condition trước khi chạm vào CSDL.
- Phục vụ Cache thông tin tồn kho `Q_avail` với tốc độ truy xuất < 1ms.

### 2.4. Message Broker: Apache Kafka
- Giải quyết bài toán Saga Pattern và Event-Driven Communication, đảm bảo các logic Asynchronous (trừ tiền, giải phóng tồn kho) đáng tin cậy.

## 3. Kiến trúc nội bộ: Clean Architecture / Hexagonal Architecture
Service được thiết kế theo tư tưởng **Clean Architecture** kết hợp **Domain-Driven Design (DDD)**. Nguyên tắc Inversion of Control (IoC) và Dependency Injection (DI) được áp dụng triệt để để tách biệt logic nghiệp vụ khỏi công nghệ cơ sở hạ tầng.

### 3.1. Cấu trúc thư mục (Directory Structure)
```text
inventory-service/
├── cmd/
│   └── server/             # Điểm vào (Entrypoint), khởi tạo gRPC server, DI container.
├── internal/
│   ├── domain/             # Tầng Core (Entities, Value Objects, Domain Services).
│   │   ├── entity/         # Các struct Batch, InventoryItem, StockReservation.
│   │   └── service/        # Các thuật toán nghiệp vụ (ví dụ: feco_allocator.go).
│   ├── application/        # Tầng Use-Cases (Logic điều phối).
│   │   ├── command/        # CQRS: Các logic thay đổi trạng thái (Reserve, Release).
│   │   ├── query/          # CQRS: Các logic lấy dữ liệu đọc (GetStock).
│   │   └── port/           # Interfaces (Repository, Broker, Cache, Lock).
│   ├── infrastructure/     # Tầng Hạ tầng (Implementations của Port).
│   │   ├── postgres/       # SQL queries, Transaction manager.
│   │   ├── redis/          # Distributed lock, Cache.
│   │   └── kafka/          # Message producers/consumers.
│   └── presentation/       # Tầng Giao tiếp.
│       ├── grpc/           # Các gRPC Handlers, Protobuf mapping.
│       └── event_handler/  # Nhận sự kiện từ Kafka.
├── pkg/                    # Các package dùng chung (Logger, Errors, Utils).
├── proto/                  # Định nghĩa Protobuf (.proto).
└── docs/                   # Tài liệu.
```

### 3.2. Sơ đồ Luồng Nghiệp vụ (Mermaid Diagrams)

#### Sơ đồ Tuần tự: Luồng gRPC ReserveStock (Giữ chỗ Tồn kho)
```mermaid
sequenceDiagram
    autonumber
    participant Order_Service
    participant Inventory_gRPC
    participant Redis_Redlock
    participant Postgres_DB
    
    Order_Service->>Inventory_gRPC: ReserveStock(sku, qty, order_id)
    Inventory_gRPC->>Postgres_DB: Check idempotency_key (order_id)
    Postgres_DB-->>Inventory_gRPC: Not processed
    
    Inventory_gRPC->>Redis_Redlock: AcquireLock("lock:inventory:sku:{sku}", TTL=3s)
    Redis_Redlock-->>Inventory_gRPC: Lock acquired
    
    Inventory_gRPC->>Postgres_DB: BEGIN TRANSACTION (Read Committed)
    Inventory_gRPC->>Postgres_DB: SELECT * FROM inventory_items WHERE sku FOR UPDATE
    Postgres_DB-->>Inventory_gRPC: Current stock status
    
    alt available_qty < qty
        Inventory_gRPC->>Postgres_DB: ROLLBACK
        Inventory_gRPC->>Redis_Redlock: ReleaseLock()
        Inventory_gRPC-->>Order_Service: Error (INSUFFICIENT_STOCK)
    else
        Inventory_gRPC->>Postgres_DB: SELECT * FROM batches WHERE sku AND status='ACTIVE' ORDER BY exp_date ASC FOR UPDATE
        Postgres_DB-->>Inventory_gRPC: List of active batches
        
        Inventory_gRPC->>Inventory_gRPC: Execute FEFO Allocation Algorithm
        Inventory_gRPC->>Postgres_DB: UPDATE inventory_items (reserved_qty += qty)
        Inventory_gRPC->>Postgres_DB: UPDATE batches (reserved_qty += alloc_qty)
        Inventory_gRPC->>Postgres_DB: INSERT INTO stock_reservations
        Inventory_gRPC->>Postgres_DB: INSERT INTO stock_reservation_allocations
        Inventory_gRPC->>Postgres_DB: COMMIT TRANSACTION
        
        Inventory_gRPC->>Redis_Redlock: ReleaseLock()
        Inventory_gRPC-->>Order_Service: Success (reservation_id)
    end
```

#### Sơ đồ Tuần tự: Luồng Saga Release / Cron Worker Hết hạn (Đã Tối ưu & Chuẩn hóa)
```mermaid
sequenceDiagram
    autonumber
    participant Order_Saga (Kafka)
    participant Worker (Cron/Ticker)
    participant App_UseCase
    participant Redis_Redlock
    participant Postgres_DB
    
    alt Saga Compensation
        Order_Saga->>App_UseCase: ReleaseStockReservationCommand (order_id)
    else TTL 15 mins Expired
        Worker->>App_UseCase: Scan PENDING reservations where expires_at < NOW()
    end

    App_UseCase->>Postgres_DB: SELECT * FROM stock_reservations WHERE order_id = {order_id}
    Postgres_DB-->>App_UseCase: Reservation data
    
    alt status != 'PENDING'
        App_UseCase-->>Order_Saga: Return (Idempotent: Already processed/cancelled)
    else status == 'PENDING'
        App_UseCase->>Redis_Redlock: AcquireLock("lock:inventory:sku:{sku}") (Sorted SKUs)
        Redis_Redlock-->>App_UseCase: Locks acquired
        
        App_UseCase->>Postgres_DB: BEGIN TRANSACTION (Read Committed)
        App_UseCase->>Postgres_DB: SELECT * FROM stock_reservation_allocations WHERE reservation_id = {id}
        Postgres_DB-->>App_UseCase: List of allocations (batch_id, alloc_qty)
        
        Note over App_UseCase,Postgres_DB: Lock Hierarchy một chiều: Items (SKU ASC) -> Batches (UUID ASC)
        App_UseCase->>Postgres_DB: UPDATE inventory_items SET reserved_qty = reserved_qty - {total_qty} WHERE sku = {sku}
        loop For each allocated batch (Canonical UUID ASC)
            App_UseCase->>Postgres_DB: UPDATE batches SET reserved_qty = reserved_qty - {alloc_qty} WHERE id = {batch_id}
        end
        
        App_UseCase->>Postgres_DB: UPDATE stock_reservations SET status = 'RELEASED' / 'EXPIRED', updated_at = NOW() WHERE id = {id}
        App_UseCase->>Postgres_DB: COMMIT TRANSACTION
        
        App_UseCase->>Redis_Redlock: ReleaseLock() (defer guaranteed)
        App_UseCase-->>Order_Saga: Success
    end
```


#### Sơ đồ Trạng thái: Vòng đời của Lô hàng (Batch Lifecycle)
```mermaid
stateDiagram-v2
    [*] --> QUARANTINE : Received (Pending QC)
    QUARANTINE --> ACTIVE : Approved (Quality OK)
    QUARANTINE --> [*] : Rejected (Failed QC)
    ACTIVE --> NEAR_EXPIRY : < 45 days to expiry
    ACTIVE --> EXPIRED : Passed expiry date
    NEAR_EXPIRY --> EXPIRED : Passed expiry date
    EXPIRED --> [*] : Disposed
```

#### Sơ đồ Trạng thái: Vòng đời của Phiếu Giữ chỗ (Reservation Lifecycle)
```mermaid
stateDiagram-v2
    [*] --> PENDING : ReserveStock (Created)
    PENDING --> COMMITTED : CommitStockDeduction (Order Paid)
    PENDING --> RELEASED : ReleaseStock (Order Canceled / Payment Failed)
    PENDING --> EXPIRED : Worker (15 mins timeout reached)
    COMMITTED --> [*]
    RELEASED --> [*]
    EXPIRED --> [*]
```

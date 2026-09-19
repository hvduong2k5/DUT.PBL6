# ĐẶC TẢ NGHIỆP VỤ & KỸ THUẬT: INVENTORY-SERVICE (MS-01)

> **Dự án:** Hệ sinh thái thương mại điện tử đa kênh & Hỗ trợ quyết định chiến lược cho Nông đặc sản OCOP Huế (Mè xửng O Mạ) — DUT.PBL6  
> **Tài liệu nguồn:** Tổng hợp từ `docs/01_requirements/epics/EPIC_09_Inventory.md`, `docs/02_architecture/bounded_context.md`, `docs/02_architecture/service_boundary.md`, `docs/02_architecture/high_level_design.md` và `docs/02_architecture/domain_design.md`.

---

## 1. THÔNG TIN ĐỊNH DANH & PHÂN LOẠI DỊCH VỤ

| Thuộc tính | Giá trị |
| :--- | :--- |
| **Service Identifier** | `MS-01: inventory-service` |
| **Bounded Context** | `BC-01: Inventory & Batch Management Context` |
| **Phân loại Miền (Domain Type)** | 🔴 **Core Domain** (Lõi cạnh tranh, SLA 99.9%, Nằm trên Critical Path đặt hàng) |
| **Giai đoạn (Phase)** | MVP (Bắt buộc bàn giao ở chu kỳ 1) |
| **Đội ngũ phụ trách (Team Owner)** | Warehouse & Supply Chain Engineering Team |
| **Cổng mạng nội bộ (Internal Port)** | `8001` (gRPC Sync & REST HTTP Management) |
| **Hệ quản trị CSDL chính (Primary DB)** | PostgreSQL 16 (`inventory_db`, Schema độc quyền) |
| **Bộ nhớ đệm & Khóa phân tán (Cache & Lock)**| Redis 7.2 (`inventory:*`, Sử dụng thuật toán Redis Redlock) |
| **Trục truyền thông sự kiện (Message Bus)** | Apache Kafka (`inventory.events.v1` & topics chuyên biệt) |

---

## 2. TRÁCH NHIỆM NGHIỆP VỤ (BUSINESS RESPONSIBILITY)

`inventory-service` là **Nguồn chân lý duy nhất (Single Source of Truth - SSOT)** về số lượng vật lý, số lượng đang tạm giữ, số lượng cách ly/hỏng và số lượng khả dụng của mọi mặt hàng nông sản OCOP.

### 2.1. Nhiệm vụ Cốt lõi
1. **Quản lý tồn kho đa chiều theo SKU và Batch/Lot:**
   - Theo dõi từng lô nhập hàng gắn liền với Mã lô (`batch_code`), Ngày sản xuất (NSX), Hạn sử dụng (HSD), và Nhà cung cấp (`supplier_id`).
   - Đảm bảo tính minh bạch để phục vụ truy xuất nguồn gốc OCOP (liên kết với `traceability-service`).
2. **Khóa chống bán vượt mức (Anti-Overselling):**
   - Đảm bảo tính nguyên tử (Atomicity) khi kiểm tra và tạm giữ tồn kho cho đơn hàng mới từ Web, Mobile hay Sàn thương mại điện tử (Marketplace Shopee/TikTok Shop).
3. **Phân bổ xuất kho theo nguyên tắc FEFO (First-Expired, First-Out):**
   - Ưu tiên xuất các lô hàng cận hạn sử dụng nhất nhưng vẫn nằm trong khung chất lượng an toàn, giảm thiểu hao hụt thực phẩm.
4. **Quản trị rủi ro cận hạn (Near-Expiry Risk Management):**
   - Tự động phát hiện các lô hàng có hạn sử dụng dưới ngưỡng an toàn (**< 45 ngày**), kích hoạt cảnh báo đến bộ phận Khuyến mãi/Bán hàng để đẩy nhanh tốc độ tiêu thụ.
   - Cách ly tuyệt đối các lô hàng đã quá hạn ($ExpiryDate \le CurrentDate$), không cho phép đưa vào tồn khả dụng.
5. **Kiểm kê & Điều chỉnh tồn kho có kiểm soát:**
   - Ghi nhận chênh lệch kiểm kê thực tế, khai báo thất thoát, hao hụt hoặc hàng hỏng.
   - Bắt buộc ghi nhận loại nghiệp vụ, lý do, người thao tác và phát sinh Audit Trail (liên kết `audit-service`).

### 2.2. Ranh giới Phủ định (Những việc DỊCH VỤ KHÔNG ĐƯỢC PHÉP LÀM)
- **Không quản lý thông tin sản phẩm:** Tên sản phẩm, hình ảnh, mô tả, giá niêm yết thuộc quyền sở hữu của `MS-05: catalog-service`.
- **Không quản lý vòng đời đơn hàng:** Trạng thái đơn (Created, Paid, Cancelled) thuộc sở hữu của `MS-04: order-service`.
- **Không lưu trữ thông tin nhà cung cấp:** Hồ sơ, hợp đồng NCC thuộc về `MS-08: procurement-service`.
- **Không ghi hình đóng gói:** Bằng chứng video đóng kiện thuộc về `MS-02: fulfillment-service`.
- **Không trực tiếp gửi email/SMS/FCM cho khách:** Gửi thông báo thuộc về `MS-17: notification-service`.

---

## 3. THUẬT NGỮ CHUYÊN NGÀNH MIỀN (UBIQUITOUS LANGUAGE)

| Thuật ngữ | Định nghĩa kỹ thuật trong Ngữ cảnh BC-01 |
| :--- | :--- |
| **SKU (Stock Keeping Unit)** | Đơn vị định danh hàng hóa lưu kho thấp nhất, gắn với khối lượng và quy cách đóng gói cụ thể. |
| **Batch / Lot (Lô hàng)** | Lô sản xuất cụ thể của một SKU, chứa thông tin NSX, HSD, NCC; là đơn vị để truy vết chất lượng. |
| **Physical Quantity ($Q_{phy}$)** | Tổng số lượng vật lý thực tế đang nằm trên kệ/kho (Tổng nhập - Tổng xuất thực tế). |
| **Reserved Quantity ($Q_{res}$)** | Số lượng đang được hệ thống tạm giữ cho các đơn hàng đang trong quá trình thanh toán/chờ đóng gói. |
| **Available Quantity ($Q_{avail}$)** | Số lượng khả dụng có thể bán tiếp: $Q_{avail} = Q_{phy} - Q_{res} \ge 0$. |
| **FEFO Allocation** | Cơ chế thuật toán phân bổ xuất kho ưu tiên lô có hạn sử dụng sớm nhất (`min(expiry_date)`). |
| **Stock Reservation** | Phiếu tạm giữ tồn kho gắn với một đơn hàng cụ thể, có thời gian sống (TTL = 15 phút). |
| **Near-Expiry Threshold** | Ngưỡng thời gian cảnh báo cận hạn cho thực phẩm mè xửng (Mặc định: **45 ngày** trước khi hết hạn). |
| **Quarantine Stock** | Lô hàng bị khóa cách ly do hỏng hóc, lỗi kiểm định hoặc hết hạn; bị loại bỏ 100% khỏi $Q_{avail}$. |
| **Stock Adjustment** | Giao dịch điều chỉnh số dư kho có lưu vết phục vụ đối soát tài chính và kiểm toán. |

---

## 4. CÁC QUY TẮC BẤT BIẾN NGHIỆP VỤ (BUSINESS INVARIANTS)

1. **`INV-BI-01`: Không tồn tại Tồn kho Khả dụng Âm (Non-Negative Available Stock)**  
   $$Q_{avail} = Q_{phy} - Q_{res} \ge 0$$  
   Trong mọi giao dịch tranh chấp đồng thời (concurrency), nếu $Q_{requested} > Q_{avail}$, giao dịch tạm giữ bắt buộc phải bị từ chối ngay lập tức.
2. **`INV-BI-02`: Khóa Hàng Quá Hạn (Zero Expired Sale)**  
   Mọi Batch có $ExpiryDate \le CurrentDate$ lập tức bị chuyển trạng thái thành `EXPIRED`/`QUARANTINE`. Tuyệt đối CẤM phân bổ các lô này vào tồn khả dụng để bán hoặc gán cho đơn hàng.
3. **`INV-BI-03`: Bắt buộc Phân bổ theo Thứ tự Hạn sử dụng (FEFO Enforcement)**  
   Khi tạo phiếu xuất kho hoặc phân bổ lô cho đơn hàng, thuật toán phải duyệt danh sách lô còn hạn theo thứ tự $ExpiryDate$ tăng dần (`ORDER BY expiry_date ASC, created_at ASC`).
4. **`INV-BI-04`: Tính Toàn vẹn Dữ liệu Lô (Batch Completeness)**  
   Một Batch chỉ được kích hoạt đủ điều kiện bán khi có đầy đủ 5 trường: `batch_code`, `manufacturing_date`, `expiry_date`, `physical_quantity > 0`, và `supplier_id`. Trong đó bắt buộc $manufacturing\_date < expiry\_date$.
5. **`INV-BI-05`: Giới hạn Thời gian Sống Tạm giữ (Reservation TTL = 15 Minutes)**  
   Một Reservation chỉ tồn tại tối đa 15 phút kể từ thời điểm tạo ($expires\_at = created\_at + 15\text{ phút}$). Nếu quá thời hạn mà không nhận được lệnh thanh toán/xác nhận (`CommitStockDeductionCommand`), hệ thống phải tự động hoàn trả số lượng đã giữ về lại $Q_{avail}$.

---

## 5. MÔ HÌNH MIỀN DDD & CẤU TRÚC AGGREGATE ROOTS

### 5.1. Aggregate Root: `InventoryItem`
Quản lý tổng tồn kho và danh sách các lô hàng của một SKU tại một kho hàng:

```text
InventoryItem (Aggregate Root)
├── id: UUID
├── sku: String (Unique Identifier per Warehouse)
├── warehouse_id: UUID
├── total_physical_qty: Integer (>= 0)
├── total_reserved_qty: Integer (>= 0)
├── status: ItemStatus (ACTIVE | SUSPENDED)
├── version: Long (Optimistic Locking)
├── created_at / updated_at: Timestamp
└── batches: List<Batch> (Entities)
    ├── id: UUID
    ├── batch_code: String (Mã lô từ nhà máy/NCC)
    ├── manufacturing_date: Date
    ├── expiry_date: Date
    ├── physical_qty: Integer
    ├── reserved_qty: Integer
    ├── status: BatchStatus (ACTIVE | NEAR_EXPIRY | EXPIRED | QUARANTINE)
    └── supplier_id: UUID
```

### 5.2. Aggregate Root: `StockReservation`
Quản lý vòng đời tạm giữ hàng phục vụ chu trình Checkout/Saga:

```text
StockReservation (Aggregate Root)
├── reservation_id: UUID
├── order_id: UUID (Reference to MS-04)
├── channel: OrderChannel (D2C_WEB | MOBILE | SHOPEE | TIKTOK | POS)
├── status: ReservationStatus (PENDING | COMMITTED | RELEASED | EXPIRED)
├── expires_at: Timestamp (created_at + 15 mins)
├── created_at / updated_at: Timestamp
└── allocations: List<ReservationItemAllocation> (Value Objects)
    ├── sku: String
    ├── batch_id: UUID
    └── allocated_qty: Integer
```

### 5.3. Entity: `StockAdjustment`
Lưu vết các nghiệp vụ can thiệp thủ công từ nhân viên kho:

```text
StockAdjustment (Audit Entity)
├── id: UUID
├── adjustment_code: String (ADJ-YYYYMMDD-XXXX)
├── sku: String
├── batch_id: UUID (Optional)
├── adjustment_type: AdjustmentType (DISCREPANCY | DAMAGED | LOSS | RECLASSIFICATION)
├── quantity_delta: Integer (+/- Qty)
├── reason: String
├── adjusted_by: UUID (User ID của ACT-06/ACT-15)
├── approved_by: UUID (User ID của Manager nếu vượt hạn mức)
└── created_at: Timestamp
```

---

## 6. GIAO DIỆN & GIAO THỨC TRUYỀN THÔNG (INTERFACES & PROTOCOLS)

### 6.1. Đồng bộ (Synchronous Internal Path — gRPC Protobuf)
Cổng giao tiếp dành cho các dịch vụ nằm trên Critical Path:

```protobuf
syntax = "proto3";
package inventory.v1;

service InventoryService {
  // 1. Tạm giữ tồn kho phục vụ Checkout (Strict timeout: 2000ms, Circuit Breaker)
  rpc ReserveStock(ReserveStockRequest) returns (ReserveStockResponse);

  // 2. Lấy thông tin số dư tồn kho tức thời (Dành cho Catalog, Order, Channel)
  rpc GetStockLevel(GetStockLevelRequest) returns (GetStockLevelResponse);

  // 3. Giải phóng tạm giữ thủ công hoặc cưỡng chế (Dành cho Saga Compensation)
  rpc ReleaseReservation(ReleaseReservationRequest) returns (ReleaseReservationResponse);

  // 4. Nhập kho lô hàng mới (Từ Procurement hoặc Thủ kho)
  rpc ReceiveGoods(ReceiveGoodsRequest) returns (ReceiveGoodsResponse);

  // 5. Điều chỉnh kiểm kê kho (Dành cho Warehouse Staff)
  rpc AdjustStock(AdjustStockRequest) returns (AdjustStockResponse);
}
```

### 6.2. Bất đồng bộ (Asynchronous Event-Driven — Kafka Events & Commands)

#### A. Commands tiếp nhận (Consume)
- `order.stock.commit` (`CommitStockDeductionCommand`):  
  Nhận từ `MS-04: order-service` khi thanh toán hoàn tất. Trừ $Q_{phy}$ và $Q_{res}$ chính thức theo các lô đã được gán trước đó, giải phóng `StockReservation` sang trạng thái `COMMITTED`.
- `order.stock.release` (`ReleaseStockReservationCommand`):  
  Nhận từ `MS-04: order-service` khi khách chủ động hủy đơn hoặc thanh toán thất bại. Trả $Q_{res}$ về $Q_{avail}$.
- `procurement.goods.received` (`GoodsReceivedEvent`):  
  Nhận từ `MS-08: procurement-service`. Tự động tạo bản ghi Batch mới và tăng $Q_{phy}$.
- `care.return.approved` (`ReturnApprovedEvent`):  
  Nhận từ `MS-06: care-service`. Tạo phiếu nhập kho cách ly (Quarantine) chờ kiểm định.

#### B. Events phát hành (Publish) — Topic `inventory.events.v1`
Mọi payload sự kiện đều **TUYỆT ĐỐI KHÔNG CHỨA PII** (Không có tên, địa chỉ, SĐT người mua):
- `StockReservedEvent`: Phát ra khi tạm giữ thành công -> Báo cho Saga Orchestrator của Order.
- `StockDeductedEvent`: Phát ra khi trừ kho thực tế hoàn tất -> Gửi cho `MS-09: finance-service` ghi nhận giá vốn (COGS) và `MS-11: analytics-service`.
- `StockReservationExpiredEvent`: Phát ra khi worker phát hiện reservation quá 15 phút chưa commit -> Báo cho `MS-04` để hủy đơn hàng quá hạn.
- `StockLevelChangedEvent`: Phát ra mỗi khi $Q_{avail}$ thay đổi -> Báo cho `MS-13: channel-service` đồng bộ tồn sang Shopee/TikTok Shop và `MS-05: catalog-service` cập nhật trạng thái còn/hết hàng.
- `ExpiryWarningEvent`: Phát ra từ Cron Job hàng ngày lúc 01:00 AM khi phát hiện các Batch có $ExpiryDate - CurrentDate < 45\text{ ngày}$ -> Gửi cho `MS-17: notification-service` và `MS-07: promotion-service` để xả hàng cận date.
- `StockAdjustedEvent`: Phát ra khi có thao tác điều chỉnh thủ công -> Gửi sang `MS-18: audit-service` (Kèm Hash Chain).

---

## 7. CƠ CHẾ KỸ THUẬT & KIẾN TRÚC ĐẶC THÙ

### 7.1. Chống Đua Tồn Kho (Concurrency & Anti-Overselling Pattern)
- **Tầng 1 (Distributed Locking):** Sử dụng **Redis Redlock** với khóa `lock:inventory:sku:{sku_code}` (TTL 3.000ms). Mọi yêu cầu tạm giữ đồng thời cùng 1 SKU phải xếp hàng lấy khóa phân tán, ngăn ngừa race condition tại mức vi giây.
- **Tầng 2 (Database Transaction):** Sử dụng `SELECT ... FOR UPDATE` trên bảng `inventory_items` trong PostgreSQL với Transaction Isolation Level `READ COMMITTED` (hoặc `REPEATABLE READ`).
- **Cơ chế Idempotency:** Mọi gRPC request và Kafka command đều mang theo `idempotency_key` (hoặc `order_id`). Hệ thống kiểm tra bảng `processed_idempotency_keys` trước khi thực hiện để đảm bảo không bị trừ kho/giữ kho 2 lần nếu mạng retry.

### 7.2. Thuật toán Xuất kho FEFO (First-Expired, First-Out)
Khi khách mua số lượng $N$ của một SKU:
1. Lấy toàn bộ các Batch còn hiệu lực ($ExpiryDate > CurrentDate$ và `status = 'ACTIVE'`) sắp xếp theo $ExpiryDate$ tăng dần.
2. Duyệt qua từng Batch:
   - Tính lượng có thể giữ của Batch: $BatchAvail = PhysicalQty - ReservedQty$.
   - Nếu $BatchAvail \le 0$, bỏ qua.
   - Nếu $BatchAvail \ge Remainder$, phân bổ $Remainder$ vào Batch này và kết thúc.
   - Nếu $BatchAvail < Remainder$, lấy hết $BatchAvail$ của Batch này, giảm $Remainder$, tiếp tục duyệt Batch kế tiếp.
3. Nếu duyệt hết toàn bộ các Batch mà tổng phân bổ $< N$ -> Hủy toàn bộ giao dịch, báo lỗi `INSUFFICIENT_STOCK`.

### 7.3. Scheduled Worker & Tự động Giải phóng Tồn kho
- **Worker 1 (Mỗi 60 giây):** Quét bảng `stock_reservations` lấy các bản ghi có `status = 'PENDING'` và `expires_at < NOW()`. Thực hiện hoàn trả $Q_{res}$, đổi trạng thái sang `EXPIRED`, và phát sinh sự kiện `StockReservationExpiredEvent`.
- **Worker 2 (Hàng ngày lúc 01:00 AM):**
  - Quét các Batch có $ExpiryDate \le NOW()$: Đổi trạng thái sang `EXPIRED`, trừ ra khỏi khả dụng, phát cảnh báo.
  - Quét các Batch có $NOW() < ExpiryDate \le NOW() + 45\text{ ngày}$: Gắn nhãn `NEAR_EXPIRY` và phát sự kiện `ExpiryWarningEvent`.

# TÀI LIỆU THIẾT KẾ CHI TIẾT (LLD): MS-04 ORDER SERVICE
## PHÂN HỆ 4: CA SỬ DỤNG & ĐIỀU PHỐI SAGA (USE CASES & SAGA ORCHESTRATION)

---

> **ĐIỀU HƯỚNG TÀI LIỆU:**
> - 📍 **Vị trí:** Phân hệ 4 / 6 của bộ thiết kế LLD `MS-04 order-service`.
> - ⬅️ [03_database_and_persistence.md — Mô hình dữ liệu & Tầng lưu trữ](03_database_and_persistence.md)
> - 🔼 [README.md — Bản đồ điều hướng & Kiến trúc tổng thể](README.md)
> - ➡️ [05_api_contracts_and_transports.md — Giao thức mạng & Đặc tả hợp đồng](05_api_contracts_and_transports.md)

---

## 6. BƯỚC 6: TẦNG ỨNG DỤNG & ĐỘNG CƠ ĐIỀU PHỐI SAGA (APPLICATION USE CASES & SAGA ENGINE)

Sau khi tầng lưu trữ đã được trừu tượng hóa qua các interface ở Phân hệ 3, Bước 6 thiết kế chi tiết luồng điều phối nghiệp vụ, phân ranh giới Order Domain vs Saga Orchestrator, kiểm soát độ trễ (Latency Budget), và quản trị các ca sử dụng then chốt.

---

### 6.1. Kiến Trúc Phân Lớp Bên Trong: Order Domain vs Saga Orchestrator

- **Order Domain Component:** Phụ trách tính toán số học, kiểm tra điều kiện chuyển trạng thái FSM nội bộ, chụp snapshot địa chỉ và tạo bản ghi lưu trữ.
- **Saga Orchestrator Component:** Quản lý vòng đời phân tán, phát sinh Command sang Inventory, lưu trạng thái bước vào `saga_instances` và chịu trách nhiệm 100% kích hoạt lệnh đền bù (Centralized Compensation Model A).

---

### 6.2. Dự Toán Độ Trễ Thực Thi (Checkout Latency Budget cho P95 < 200ms)

Để bảo đảm cam kết **NFR-01 (Độ trễ P95 < 200ms)**, chuỗi thực thi của `CheckoutD2CUseCase` được thiết kế song song hóa và phân bổ ngân sách thời gian (Latency Budget) nghiêm ngặt:

| Công Đoạn Xử Lý | Cơ Chế Kỹ Thuật | Ngân Sách Dự Toán (Budget) | Ghi Chú Tối Ưu Hóa |
| :--- | :--- | :---: | :--- |
| **API Gateway Transit** | Kong JWT Local Validation & TLS Term | **10 ms** | Xác thực qua Cached JWKS, không gọi mạng nội bộ. |
| **Idempotency Check** | Redis `GET idempotency:checkout:{key}` | **5 ms** | In-memory lookup. |
| **Parallel Verification** | `errgroup` chạy song song 3 gRPC: | **35 ms** | Chạy đồng thời 3 luồng gRPC: |
| ├── *Catalog Service* | gRPC `ValidatePriceAndSKU` | *(30 ms)* | Đọc từ Redis Cache của Catalog. |
| ├── *Promotion Service* | gRPC `ValidateVoucher` | *(25 ms)* | Kiểm tra ngân sách mã giảm giá. |
| └── *Shipping Service* | gRPC `CalculateShippingFee` | *(20 ms)* | Tra cứu ma trận cước địa lý. |
| **Stock Reservation** | gRPC `ReserveStock` sang MS-01 | **45 ms** | Kho chạy `SELECT FOR UPDATE` trên index SKU. |
| **Database Transaction** | PostgreSQL Local ACID Commit | **30 ms** | Ghi 1 lệnh gom: Order, Items, Address, Saga, Outbox. |
| **Network & Serialization** | Protobuf / JSON Marshalling | **15 ms** | Zero-copy buffer serialization. |
| **TỔNG THỜI GIAN P95 DỰ TOÁN** | **Toàn trình từ Client $\rightarrow$ Response** | **$\mathbf{\sim 140\ ms}$** | **Thỏa mãn vượt trội NFR-01 (< 200ms)**. |

> [!NOTE]
> Timeout cấu hình cho gRPC là **2.0 giây** — Đây là ngưỡng chịu lỗi cực hạn (Circuit Breaker Deadline) để cô lập sự cố khi service đối tác sập hoàn toàn, **không phải thời gian chạy bình thường (P95 thông thường luôn $\le 45$ms)**.

---

### 6.3. Use Case 1: `CheckoutD2CUseCase` (Critical Path Synchronous)

```text
[BƯỚC 1]: Kiểm tra Idempotency Key trên Redis bằng lệnh SET key "PROCESSING" EX 86400 NX.
          └──> Nếu đã tồn tại -> Trả về ngay kết quả phản hồi đã lưu từ trước.
[BƯỚC 2A]: Chuẩn bị dữ liệu Địa chỉ Giao nhận (Address Resolution):
          ├──> Nếu Client truyền `shipping_address_id` (Khách hàng đăng nhập chọn sổ địa chỉ):
          │    └──> Gọi gRPC: MS-15: ProfileService.GetDeliveryAddress(shipping_address_id, customer_id)
          │         (Độ trễ P99 <= 5ms từ Dual-Layer Cache của MS-15).
          └──> Nếu Client là Guest: Lấy trực tiếp thông tin địa chỉ từ request payload.
[BƯỚC 2B]: Kích hoạt golang.org/x/sync/errgroup phát 3 cuộc gọi gRPC đồng thời:
          ├──> MS-05: CatalogService.ValidatePriceAndSKU(items)
          ├──> MS-07: PromotionService.ValidateVoucher(voucher_code, customer_id)
          └──> MS-12: ShippingService.CalculateShippingFee(resolved_address.ward_code, total_weight)
[BƯỚC 3]: Đánh giá kết quả xác thực:
          └──> Nếu có bất kỳ lỗi nào (Giá sai, Voucher hết hạn, Địa chỉ không tồn tại) -> Hủy luồng, trả về HTTP 400.
[BƯỚC 4]: Gọi gRPC Synchronous: MS-01: InventoryService.ReserveStock(order_id, items, ttl=15m).
          └──> Nếu trả về INSUFFICIENT_STOCK -> Dừng luồng, trả về danh sách SKU thiếu hàng.
[BƯỚC 5]: Application sinh UUID v7 cho order_id, outbox_id, saga_id.
[BƯỚC 6]: Mở Local Database Transaction (PostgreSQL ACID):
          BEGIN;
            INSERT INTO orders (id, order_code, ..., status='PENDING_PAYMENT', expires_at=NOW()+15m);
            INSERT INTO order_line_items (...);
            INSERT INTO order_address_snapshots (...); -- Lưu Snapshot địa chỉ 2 cấp từ MS-15/Guest
            INSERT INTO order_voucher_snapshots (...);
            INSERT INTO saga_instances (id, order_id, saga_type='CHECKOUT_D2C_SAGA', status='IN_PROGRESS', ...);
            INSERT INTO outbox_events (id, aggregate_id, event_type='vn.omama.order.placed.v1', topic='order.events.v1', ...);
          COMMIT;
[BƯỚC 7]: Tạo link VietQR động: https://img.vietqr.io/image/970422-0905123456-compact2.png?amount=...&addInfo=OMAMA%20ORD...
[BƯỚC 8]: Lưu kết quả CheckoutResponse vào Redis Idempotency Key -> Trả về HTTP 201 cho Client.
```

---

### 6.4. Use Case 2: `VietQRWebhookCallbackUseCase` (Chốt Luật Khớp Số Tiền Tuyệt Đối)

> [!IMPORTANT]
> **Quy Chuẩn Đối Soát Số Tiền Thanh Toán D2C VietQR:**
> Tuyệt đối cấm quy tắc lỏng lẻo `amount_paid >= final_amount`. Với giao dịch chuyển khoản VietQR tự động, **BẮT BUỘC KHỚP TUYỆT ĐỐI (`amount_paid == final_amount`)**.

#### Bảng Xử Lý Toàn Diện 5 Kịch Bản Thanh Toán:
| Kịch Bản Thanh Toán | Điều Kiện So Khớp | Trạng Thái Đơn Hàng | Hành Động Hệ Thống & Kế Toán |
| :--- | :--- | :--- | :--- |
| **1. Khớp Chuẩn (Exact Match)** | `amount == final_amount` | Chuyển $\rightarrow$ `PAID` | Ghi Outbox `OrderPaidEvent`, kích hoạt đóng gói và trừ tồn kho committed. |
| **2. Chuyển Thiếu (Underpaid)** | `amount < final_amount` | Giữ nguyên `PENDING_PAYMENT` | Ghi log thanh toán một phần, gửi SMS/ZNS: *"Quý khách chuyển thiếu X đồng, vui lòng chuyển nốt!"*. |
| **3. Chuyển Thừa (Overpaid)** | `amount > final_amount` | Chuyển $\rightarrow$ `PAID` | Đơn hàng vẫn được đóng gói giao kẹo; hệ thống tự động tạo Ticket kế toán tại MS-09 để hoàn lại số tiền thừa cho khách. |
| **4. Giao Dịch Trùng (Duplicate Webhook)** | Trùng `(payment_provider, provider_tx_id)` | Không đổi | Bị chặn bởi Unique Constraint CSDL; trả về ngay HTTP 200 OK, không xử lý lại. |
| **5. Chuyển Tiền Sau Khi Hủy (Paid After Timeout)**| Đơn đã `CANCELLED_TIMEOUT` | Giữ nguyên `CANCELLED_TIMEOUT` | **CẤM chuyển PAID**. Ghi nhận `RECONCILIATION_REQUIRED`, kích hoạt hoàn tiền 100% cho khách vì tồn kho có thể đã bị giải phóng. |

---

### 6.5. Use Case 3: `OrderTimeoutCancelCompensationUseCase` (Đền Bù Tập Trung Model A)

```text
                               Saga Orchestrator (Order Service)
                                              │
                      ┌───────────────────────┴───────────────────────┐
                      │ gRPC ReleaseReservation                       │ gRPC ReleaseVoucher
                      ▼                                               ▼
             MS-01: INVENTORY SERVICE                        MS-07: PROMOTION SERVICE
         (Hoàn trả tồn kho khả dụng)                     (Mở khóa voucher cho khách)
```

```text
[BƯỚC 1]: Cron Job Sweeper chạy mỗi 15 giây, thực hiện truy vấn quét:
          SELECT id, order_code, version FROM orders 
          WHERE status = 'PENDING_PAYMENT' AND expires_at < CURRENT_TIMESTAMP LIMIT 50;
[BƯỚC 2]: Với mỗi đơn hàng quá hạn, thực thi Atomic CAS Transition trên PostgreSQL:
          UPDATE orders 
          SET status = 'CANCELLED_TIMEOUT', version = version + 1, updated_at = NOW()
          WHERE id = $1 AND status = 'PENDING_PAYMENT' AND version = $2;
[BƯỚC 3]: Kiểm tra kết quả RowsAffected:
          ├──> Nếu RowsAffected == 0: Luồng khác đã thanh toán hoặc cập nhật trước -> Bỏ qua.
          └──> Nếu RowsAffected == 1: Lệnh hủy đơn thành công. Bắt đầu đền bù tập trung (Model A):
               ├──> Cập nhật saga_instances status = 'COMPENSATING'.
               ├──> Ghi Outbox: OrderCancelledEvent (Mục đích thông báo thuần túy, KHÔNG PHẢI LỆNH ĐỀN BÙ).
               ├──> Trực tiếp gọi gRPC: MS-01: InventoryService.ReleaseReservation(order_id).
               ├──> Trực tiếp gọi gRPC: MS-07: PromotionService.ReleaseVoucher(voucher_code, customer_id).
               └──> Nếu gRPC thành công:
                    └──> Cập nhật saga_instances status = 'COMPENSATED'.
               └──> Nếu gRPC thất bại (lỗi mạng):
                    └──> Cập nhật saga_instances status = 'RETRYING', retry_count = retry_count + 1.
                    └──> RetryWorker sẽ tự động thử lại sau (Exponential Backoff: 1s, 2s, 4s, 8s, 16s).
                    └──> Nếu quá 5 lần vẫn lỗi -> Chuyển status = 'DEAD_LETTER', kích hoạt cảnh báo Slack/PagerDuty cho đội Ops.
```

---

### 6.6. Use Case 4: `MarketplaceInboundSagaUseCase` (Tiếp Nhận Đơn Sàn & Khóa Tồn Tập Trung)
1. Consumer trong `order-service` tiêu thụ sự kiện `MarketplaceOrderImportedEvent` từ topic `channel.events.v1`.
2. Khởi tạo `SagaInstance` với `saga_type = 'MARKETPLACE_INBOUND_SAGA'`.
3. Gọi gRPC Synchronous `ReserveStock` sang `inventory-service`:
   - **Thành công:** Tạo đơn hàng nội bộ với `channel = MARKETPLACE_SHOPEE`, trạng thái `PAID`. Ghi Outbox `OrderPaidEvent` để xưởng đóng kẹo. Cập nhật Saga $\rightarrow$ `COMPLETED`.
   - **Thất bại (Hết hàng tại xưởng Huế):** Cập nhật Saga $\rightarrow$ `FAILED`. Ghi Outbox `MarketplaceOrderStockFailedEvent`. `channel-service` tiêu thụ sự kiện này để tự động gửi API báo hủy đơn lên sàn Shopee/TikTok.

---

### 6.7. Use Case 5: `CreatePOSOrderUseCase` (Bán Trực Tiếp Tại Quầy Xưởng Hương Thủy)
- Phục vụ khách du lịch mua kẹo mè xửng trực tiếp tại xưởng hoặc showroom Huế qua thiết bị POS offline.
- Thu ngân bấm thanh toán $\rightarrow$ MS-13 gọi gRPC `CreatePOSOrder` sang MS-04.
- Đơn hàng được tạo thẳng ở trạng thái `PAID`, ghi nhận thanh toán tiền mặt/quẹt thẻ, xuất bản `OrderPaidEvent` để trừ tồn kho vật lý và in hóa đơn VAT ngay tại quầy trong vòng **dưới 50 mili-giây**.

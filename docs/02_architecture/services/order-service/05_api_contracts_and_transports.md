# TÀI LIỆU THIẾT KẾ CHI TIẾT (LLD): MS-04 ORDER SERVICE
## PHÂN HỆ 5: GIAO THỨC MẠNG & ĐẶC TẢ HỢP ĐỒNG (DELIVERY & CONTRACTS)

---

> **ĐIỀU HƯỚNG TÀI LIỆU:**
> - 📍 **Vị trí:** Phân hệ 5 / 6 của bộ thiết kế LLD `MS-04 order-service`.
> - ⬅️ [04_usecases_and_saga_orchestration.md — Ca sử dụng & Điều phối Saga](04_usecases_and_saga_orchestration.md)
> - 🔼 [README.md — Bản đồ điều hướng & Kiến trúc tổng thể](README.md)
> - ➡️ [06_testing_and_failure_recovery.md — Kiểm thử, Khôi phục sự cố & Truy vết](06_testing_and_failure_recovery.md)

---

## 7. BƯỚC 7: TẦNG VẬN CHUYỂN & ĐẶC TẢ HỢP ĐỒNG GIAO TIẾP (DELIVERY LAYER & CONTRACTS)

Bước 7 xác định các cổng giao tiếp mạng (North-South qua API Gateway và East-West giữa các microservices nội bộ), cùng đặc tả các hợp đồng giao tiếp (REST OpenAPI, gRPC Protobuf, Kafka CloudEvents).

---

### 7.1. Cổng Biên HTTP RESTful APIs (North - South via Kong Gateway)

Toàn bộ các REST API đều tuân thủ chuẩn OpenAPI 3.0 tại [`docs/03_api_specs/openapi_d2c.yaml`](../../docs/03_api_specs/openapi_d2c.yaml):

#### 1. Khởi Tạo Phiên Đặt Hàng (`POST /api/v1/checkout`)
- **Headers yêu cầu:** `Authorization: Bearer <jwt>`, `Idempotency-Key: <uuid>`, `Content-Type: application/json`.
- **Payload Request:**
```json
{
  "customer_id": "018f3a5b-9c12-7def-a890-123456789abc",
  "channel": "D2C_WEB",
  "shipping_address": {
    "recipient_name": "Nguyễn Hoàng Nam",
    "phone_number": "0905123456",
    "street_address": "15 Lê Lợi",
    "ward_code": "VN-HUE-PHUHOI",
    "ward_name": "Phường Phú Hội",
    "province_code": "VN-HUE",
    "province_name": "Thành phố Huế",
    "latitude": 16.4673,
    "longitude": 107.5905
  },
  "items": [
    { "sku_code": "MX-GION-500G", "quantity": 2 },
    { "sku_code": "MX-DEO-HOMEMADE-300G", "quantity": 1 }
  ],
  "voucher_code": "OMAMA_TET2026",
  "payment_method": "VIETQR",
  "customer_note": "Gói kỹ chống vỡ giúp em, mang vào Sài Gòn làm quà!"
}
```

- **Payload Response (HTTP 201 Created):**
```json
{
  "order_id": "018f3a5b-9c12-7def-a890-123456789abc",
  "order_code": "ORD-20261015-0042",
  "status": "PENDING_PAYMENT",
  "pricing": {
    "subtotal_amount": 350000,
    "discount_amount": 30000,
    "shipping_fee": 25000,
    "final_amount": 345000,
    "currency": "VND"
  },
  "payment": {
    "method": "VIETQR",
    "status": "PENDING",
    "vietqr_url": "https://img.vietqr.io/image/970422-0905123456-compact2.png?amount=345000&addInfo=OMAMA%20ORD202610150042",
    "expires_at": "2026-10-15T08:45:00.000Z",
    "countdown_seconds": 900
  },
  "created_at": "2026-10-15T08:30:00.000Z"
}
```

---

### 7.2. Cổng Nội Bộ gRPC Services (East - West Server)

Khớp 100% tệp Protobuf [`packages/proto/order/v1/order.proto`](../../packages/proto/order/v1/order.proto):

```protobuf
syntax = "proto3";
package omamx.order.v1;

service OrderService {
  // Tạo đơn hàng trực tiếp tại quầy POS offline (MS-13 gọi)
  rpc CreatePOSOrder(CreatePOSOrderRequest) returns (CreatePOSOrderResponse);

  // Truy vấn chi tiết đơn hàng kèm snapshot giao vận
  rpc GetOrderDetail(GetOrderDetailRequest) returns (GetOrderDetailResponse);

  // Hủy đơn hàng và kích hoạt đền bù tập trung
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
}
```

---

### 7.3. Hợp Đồng Gọi gRPC Ngoại Vi (East - West Client on Critical Path)

1. **Khóa tồn kho:** Gọi `MS-01: InventoryService.ReserveStock` (Port 8001). Strict Timeout: `2000ms`.
2. **Giải phóng tồn kho (Đền bù):** Gọi `MS-01: InventoryService.ReleaseReservation` (Port 8001).
3. **Thẩm định giá:** Gọi `MS-05: CatalogService.ValidatePriceAndSKU` (Port 8005).
4. **Thẩm định voucher:** Gọi `MS-07: PromotionService.ValidateVoucher` (Port 8007).
5. **Giải phóng voucher (Đền bù):** Gọi `MS-07: PromotionService.ReleaseVoucher` (Port 8007).
6. **Tính phí ship:** Gọi `MS-12: ShippingService.CalculateShippingFee` (Port 8012).
7. **Lấy địa chỉ giao hàng (Saved Address Hydration):** Gọi `MS-15: ProfileService.GetDeliveryAddress` (Port 50051 / 8015). Strict Timeout: `500ms`, SLA $P99 \le 5\text{ms}$.
   - Contract Protobuf:
     ```protobuf
     message GetDeliveryAddressRequest {
       string address_id = 1;
       string customer_id = 2;
     }
     message DeliveryAddressResponse {
       string id = 1;
       string customer_id = 2;
       string recipient_name = 3;
       string phone_number = 4;
       string street_address = 5;
       string ward_code = 6;
       string ward_name = 7;
       string province_code = 8;
       string province_name = 9;
       double latitude = 10;
       double longitude = 11;
       bool is_default = 12;
     }
     ```

---

### 7.4. Danh Mục Sự Kiện Kafka Xuất Bản (Transactional Outbox Producer)

Tuân thủ chuẩn **CNCF CloudEvents 1.0 JSON Schema** tại [`packages/events/schemas/order/v1/`](../../packages/events/schemas/order/v1/):

| Tên Sự Kiện | Event Type | Kafka Topic | Partition Key | Ý Nghĩa Nghiệp Vụ (Semantic) |
| :--- | :--- | :--- | :---: | :--- |
| `OrderPlacedEvent` | `vn.omama.order.placed.v1` | `order.events.v1` | `order_id` | Khách tạo đơn thành công, đang chờ quét mã VietQR trong 15 phút. |
| `OrderPaidEvent` | `vn.omama.order.paid.v1` | `order.events.v1` | `order_id` | Tiền đã về tài khoản, kích hoạt xưởng đóng kẹo, trừ kho committed và xuất VAT. |
| `OrderCancelledEvent` | `vn.omama.order.cancelled.v1` | `order.events.v1` | `order_id` | **Thông báo đơn đã bị hủy (Business Fact)**. Không chứa command đền bù kho. |
| `OrderCompletedEvent` | `vn.omama.order.completed.v1`| `order.events.v1` | `order_id` | Đơn hoàn tất sau 7 ngày, kích hoạt tích điểm 1% loyalty cho khách. |
| `MarketplaceOrderStockFailedEvent` | `vn.omama.order.marketplace.failed.v1` | `order.events.v1` | `order_id` | Báo cho MS-13 xử lý hủy đơn sàn khi kho xưởng Huế hết hàng. |

- **Payload Chuẩn Của `OrderPaidEvent` (Bảo Toàn Quyền Riêng Tư PII):**
```json
{
  "specversion": "1.0",
  "id": "018f3a5b-9c12-7def-a890-123456789abc",
  "source": "https://omama.vn/services/order-service",
  "type": "vn.omama.order.paid.v1",
  "subject": "order_id:018f3a5b-9c12-7def-a890-123456789abc",
  "time": "2026-10-15T08:32:11.000Z",
  "datacontenttype": "application/json",
  "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
  "data": {
    "order_id": "018f3a5b-9c12-7def-a890-123456789abc",
    "order_code": "ORD-20261015-0042",
    "customer_id": "018f3a5b-9c12-7def-a890-999999999abc",
    "shipping_address_id": "addr-snap-018f3a5b",
    "channel": "D2C_WEB",
    "currency": "VND",
    "subtotal_amount": 350000,
    "discount_amount": 30000,
    "shipping_fee": 25000,
    "final_amount": 345000,
    "payment": {
      "provider": "VIETQR_NAPAS",
      "provider_transaction_id": "FT2628891048201",
      "paid_at": "2026-10-15T08:32:10.000Z"
    },
    "items": [
      { "sku_code": "MX-GION-500G", "quantity": 2, "unit_price": 110000 },
      { "sku_code": "MX-DEO-HOMEMADE-300G", "quantity": 1, "unit_price": 130000 }
    ]
  }
}
```

---

### 7.5. Danh Mục Sự Kiện Kafka Tiêu Thụ (Inbound Consumer)

| Nguồn Phát | Kafka Topic | Event Type | Xử Lý Phía `order-service` |
| :--- | :--- | :--- | :--- |
| `channel-service` (MS-13) | `channel.events.v1` | `vn.omama.channel.marketplace.order.imported.v1` | Khởi chạy `MarketplaceInboundSagaUseCase`, gọi `ReserveStock` kho. |
| `fulfillment-service` (MS-02) | `fulfillment.events.v1` | `vn.omama.fulfillment.package.sealed.v1` | Cập nhật đơn $\rightarrow$ `PACKED`, lưu `seal_code` và link bằng chứng S3. |
| `shipping-service` (MS-12) | `shipping.events.v1` | `vn.omama.shipping.shipment.delivered.v1` | Cập nhật đơn $\rightarrow$ `DELIVERED`, bắt đầu đếm 7 ngày khiếu nại. |

---

## 8. BƯỚC 8: CƠ CHẾ KỸ THUẬT XUYÊN SUỐT & PHÒNG VỆ (CROSS-CUTTING CONCERNS)

---

### 8.1. Đảm Bảo At-least-once Delivery & Idempotent Consumer (Effectively-once Business Outcome)

> [!IMPORTANT]
> **Định Nghĩa Chuẩn Về Tính Nhất Quán Trong Hệ Phân Tán:**
> Trong kiến trúc hướng sự kiện (Event-Driven Architecture) với Transactional Outbox:
> $$\mathbf{Transactional\ Outbox\ =\ At\text{-}least\text{-}once\ Delivery}$$
> Do máy chủ Outbox Publisher có thể gặp sự cố mạng hoặc crash ngay sau khi bắn thành công lên Kafka nhưng chưa kịp cập nhật trạng thái `PUBLISHED` trong CSDL, thông điệp có thể bị bắn lại nhiều lần khi tiến trình khởi động lại.
> 
> Vì vậy, đẳng thức toàn vẹn nghiệp vụ của hệ thống Mè Xửng O Mạ được xác lập:
> $$\mathbf{At\text{-}least\text{-}once\ Delivery\ +\ Idempotent\ Consumer\ =\ Effectively\text{-}once\ Business\ Outcome}$$

Mọi Consumer trong toàn bộ 18 Microservices (bao gồm cả các consumer bên trong `order-service`) đều bắt buộc phải triển khai cơ chế kiểm tra trùng lặp (Idempotent Message Handling):
- Sử dụng bảng kiểm tra sự kiện đã xử lý `processed_events (event_id UUID PRIMARY KEY, processed_at TIMESTAMPTZ)` ngay bên trong transaction của consumer.
- Nếu `event_id` đã tồn tại $\rightarrow$ Bỏ qua việc thực thi nghiệp vụ, xác nhận Commit Offset Kafka ngay lập tức.

---

### 8.2. Hệ Thống Idempotency Đa Tầng (Multi-Tier Idempotency Shield)

1. **Tầng 1 (Memory / Redis Filter):** Sử dụng `idempotency:checkout:{key}` với TTL 24h.
2. **Tầng 2 (PostgreSQL Unique Constraint):** Bảng `payments` áp dụng ràng buộc duy nhất `UNIQUE (payment_provider, provider_transaction_id)`. Bất kỳ nỗ lực ghi đúp nào cũng bị cơ sở dữ liệu chặn đứng với mã lỗi vi phạm khóa duy nhất (`23505 UniqueViolation`).

---

### 8.3. Chiến Lược Cầu Dao Ngắt Mạch & Quá Giờ Nghiêm Ngặt (Circuit Breaker & Strict Timeout)

- **Strict Timeout (2000ms):** Bọc toàn bộ các cuộc gọi gRPC ngoại vi trong Context có deadline 2 giây.
- **Circuit Breaker:** Ngưỡng mở cầu dao: $\ge 50\%$ request lỗi trong 10 giây. Khi cầu dao OPEN: Trả về ngay mã lỗi nhanh `ERR_CIRCUIT_BREAKER_OPEN` (HTTP 503) trong vòng 1ms, không làm nghẽn thread pool. Thời gian ngủ thăm dò (Sleep window): 5 giây.

---

### 8.4. Bảo Mật Zero-Trust & Xác Thực Webhook HMAC-SHA256

- **Local JWT Validation:** API Gateway tiêm các Header định danh sạch: `X-User-Id`, `X-User-Roles`. `order-service` kiểm tra quyền sở hữu đơn hàng trực tiếp từ Header.
- **Xác Thực Chữ Ký Webhook Ngân Hàng:**
  - Ngân hàng gửi kèm Header `X-VietQR-Signature`.
  - `order-service` tính toán HMAC-SHA256 từ binary payload gốc và Secret Key, đối chiếu bằng thuật toán an toàn thời gian cố định `crypto/subtle.ConstantTimeCompare` để loại trừ tấn công vét cạn theo thời gian (Timing Attack).

---

### 8.5. Tích Hợp Mặt Phẳng Kiểm Toán & Giám Sát Viễn Trắc (Audit & Observability)

- **Mặt Phẳng Kiểm Toán (Audit Plane):** Mọi thao tác hủy đơn, duyệt hoàn tiền, điều chỉnh giá đều xuất bản sự kiện sang `audit.events.v1` kèm chuỗi băm Tamper-Evident Hash Chain SHA-256.
- **Mặt Phẳng Viễn Trắc (Observability Plane):** Tự động lan truyền W3C `traceparent` qua gRPC Metadata và Kafka Headers; đẩy tín hiệu OTLP sang OpenTelemetry Collector.

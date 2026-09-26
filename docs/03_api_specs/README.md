# CẨM NANG ĐẶC TẢ GIAO DIỆN & HỢP ĐỒNG HỆ THỐNG (API & CONTRACTS SPECIFICATIONS)
## HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ & CHUỖI CUNG ỨNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

Thư mục này là **Single Source of Truth** về mặt tài liệu giao tiếp kỹ thuật của toàn bộ 18 Microservices, phục vụ trực tiếp cho quá trình lập trình song song, kiểm thử tự động (CI/CD Contract Test) và tài liệu bảo vệ đồ án.

---

## 1. Bản Đồ Hợp Đồng Đa Giao Thức (Multi-Protocol Contracts Map)

Hệ thống phân định ranh giới giao tiếp rõ rệt theo 3 chiều:

```text
                                 CLIENTS & ĐỐI TÁC NGOÀI
           (Web Clients, Mobile Apps, Quầy POS, VietQR, Sàn Shopee/TikTok)
                                            │
                                            ▼  [North - South Edge]
                                     API GATEWAY (KONG)
                    ┌───────────────────────┼───────────────────────┐
                    │ REST (OpenAPI 3.0)    │ GraphQL (/graphql)    │ Webhook HTTPS Sàn TMĐT
                    ▼                       ▼                       ▼
          MS-04: ORDER SERVICE       MOBILE BFF (Node.js)    MS-13: CHANNEL SERVICE
            ▲       │                  (Gom query 4G)               │               │
            │       │                  │       │                    │ gRPC Sync     │ Kafka Async
            │       │                  │ gRPC  │ gRPC               │ CreatePOS     │ MarketplaceOrder
            │       │  [Critical Path] ├──► MS-05: catalog          │ Order         │ Imported
            │       │  (HTTP/2 Proto)  ├──► MS-07: promotion        ▼ (Quầy POS)    │
            │       ├──► MS-01: inv    ├──► MS-15: profile   [MS-04 Order]          ▼
            │       ├──► MS-05: cat    └──► MS-04: order             APACHE KAFKA BROKER CLUSTER
            │       ├──► MS-07: promotion-service    [MS-04 Order]          ▼
            │       ├──► MS-12: shipping-service                 APACHE KAFKA BROKER CLUSTER
            │       ├──► MS-15: profile-service (Address Snapshot)          │
            │       └──► MS-16: identity-service (ABAC Dynamic)  [Workflows & Chuỗi Cung Ứng]
            │                                                    ├──► MS-01: inventory-service
            └────────────────────────────────────────────────────┼──► MS-02: fulfillment-service
                        (Saga Inbound Consumer)                  ├──► MS-06: care-service
                                                                 ├──► MS-07: promotion-service
                                                                 ├──► MS-08: procurement-service
                                                                 ├──► MS-09: finance-service
                                                                 ├──► MS-11: analytics-service
                                                                 ├──► MS-17: notification-service
                                                                 └──► MS-18: audit-service
```

> [!IMPORTANT]
> **Làm Rõ Tuyệt Đối Về Vai Trò Của `CreatePOSOrder` vs Luồng Đơn Sàn:**
> - **Đơn Sàn Shopee / TikTok Shop:** Tuyệt đối KHÔNG gọi gRPC sang `order-service`. MS-13 chỉ đóng vai trò ACL tiếp nhận Webhook và phát sự kiện `MarketplaceOrderImportedEvent` lên topic `channel.events.v1`. Saga Orchestrator trong MS-04 tiêu thụ sự kiện này từ Kafka để tự động điều phối khóa tồn kho, bảo đảm 100% nguyên tắc **Centralized Compensation** (Mục 3.4).
> - **Quầy Bán Lẻ Trực Tiếp Offline (POS Xưởng Hương Thủy & Cửa hàng Huế):** Khi khách mua kẹo trực tiếp tại quầy, thu ngân bấm thanh toán trên thiết bị POS offline. Vì khách đang đứng trực tiếp chờ lấy hàng mang về ngay, MS-13 gọi gRPC Sync `CreatePOSOrder` sang MS-04 để in hóa đơn và hoàn tất đơn tức thời.
> - **Đơn Sỉ Doanh Nghiệp B2B & Quản Trị Viên:** Tạo qua giao diện Admin Web CMS, đi qua REST API Gateway `/api/v1/admin/orders`.

---

## 2. Bảng Ánh Xạ Mô Hình Lỗi Thống Nhất (Unified Error Model Mapping Table)

Để Frontend/Mobile chỉ cần quản lý 1 bộ mã lỗi duy nhất, API Gateway tự động dịch mã lỗi từ gRPC của các microservices nội bộ sang HTTP Status Code tương ứng:

| Mã Lỗi Nghiệp Vụ (`ErrorCode`) | gRPC Status Code | HTTP Status Code | Thông Điệp Người Dùng (Tiếng Việt) | Hướng Xử Lý Phía Client (UX Action) |
| :--- | :--- | :---: | :--- | :--- |
| `ERR_INVENTORY_INSUFFICIENT_STOCK` | `FAILED_PRECONDITION` (9) | `400 Bad Request` | *"Sản phẩm đã hết hàng hoặc số lượng khả dụng không đủ."* | Hiển thị badge "Hết hàng", cập nhật lại giỏ hàng về số lượng tối đa. |
| `ERR_INVENTORY_RESERVATION_EXPIRED` | `DEADLINE_EXCEEDED` (4) | `410 Gone` | *"Phiếu giữ kẹo đã hết hạn sau 15 phút. Vui lòng đặt lại."* | Chuyển hướng về giỏ hàng, thông báo giữ lại thông tin đơn. |
| `ERR_CATALOG_PRICE_TAMPERED` | `INVALID_ARGUMENT` (3) | `400 Bad Request` | *"Giá sản phẩm đã thay đổi. Hệ thống đã cập nhật mức giá mới."* | Refresh lại giá niêm yết mới nhất từ Catalog Service. |
| `ERR_PROMOTION_VOUCHER_EXPIRED` | `FAILED_PRECONDITION` (9) | `400 Bad Request` | *"Mã giảm giá đã hết hạn sử dụng."* | Gỡ voucher khỏi đơn hàng, cho phép khách nhập mã khác. |
| `ERR_PROMOTION_MIN_ORDER_AMOUNT_NOT_MET` | `FAILED_PRECONDITION` (9) | `400 Bad Request` | *"Giá trị đơn kẹo chưa đạt mức tối thiểu để áp dụng mã này."* | Gợi ý khách mua thêm kẹo mè xửng giòn/dẻo để đủ điều kiện. |
| `ERR_PROMOTION_VOUCHER_ALREADY_LOCKED` | `ABORTED` (10) | `409 Conflict` | *"Mã giảm giá đang được sử dụng ở một phiên thanh toán khác."* | Chờ 15 phút hoặc chọn mã khuyến mãi khác. |
| `ERR_PAYMENT_HMAC_VERIFICATION_FAILED` | `UNAUTHENTICATED` (16) | `401 Unauthorized` | *"Xác thực chữ ký số Webhook ngân hàng thất bại."* | Từ chối cập nhật trạng thái đơn, ghi log cảnh báo an ninh. |
| `ERR_IDEMPOTENCY_CONFLICT` | `ALREADY_EXISTS` (6) | `409 Conflict` | *"Yêu cầu đang được xử lý hoặc đã hoàn tất trước đó."* | Giữ nguyên trạng thái hiển thị, không gửi lại lệnh duplicate. |
| `ERR_CIRCUIT_BREAKER_OPEN` | `UNAVAILABLE` (14) | `503 Service Unavailable`| *"Hệ thống kho đang quá tải, quý khách vui lòng thử lại sau giây lát!"* | Hiển thị popup thử lại kèm đếm ngược 5 giây (Retry with backoff). |

---

## 3. Hướng Dẫn Vận Hành Mock Server Cho Thành Viên Nhóm (Parallel Development)

Để 18 thành viên có thể code song song độc lập mà không cần chờ đợi service đối tác hoàn thành, sử dụng Mock Server sinh trực tiếp từ các file Contract đã chốt:

### 3.1. Chạy Mock Server gRPC Cục Bộ (Ví dụ: Mock `inventory-service` cho bạn làm `order-service`)
Sử dụng công cụ `grpc-mock` hoặc `BloomRPC / Postman gRPC`:
```bash
# Cài đặt công cụ giả lập gRPC từ file .proto
npx @improbable-eng/grpc-web-fake-server --proto packages/proto/inventory/v1/inventory.proto
```
- Mock server sẽ tự động lắng nghe trên port `8001` và trả về payload mẫu hợp lệ theo đúng message `ReserveStockResponse`.
- Bạn làm Order Service có thể tự tin test luồng Happy Path và luồng Out-of-stock mà không cần bạn làm Kho phải dựng database PostgreSQL xong trước.

### 3.2. Chạy Mock Server REST Cho Frontend & Các Bên Tích Hợp
Sử dụng công cụ **Prism** đọc file [`openapi_d2c.yaml`](./openapi_d2c.yaml):
```bash
# Chạy Prism mock server trên port 4010
npx @stoplight/prism-cli mock docs/03_api_specs/openapi_d2c.yaml -p 4010
```
- Mọi endpoint trong 9 nhóm API (Auth, Catalog, Giỏ hàng, Checkout, Profile, Voucher, Review, OCOP QR, Webhooks) sẽ phản hồi tức thì với dữ liệu mẫu chuẩn Schema và mô phỏng chính xác mã lỗi HTTP 400, 401, 403, 404, 409, 503.

### 3.3. Xem & Trải Nghiệm Trực Quan Trên Swagger UI
Bạn có thể mở giao diện tương tác Swagger UI cho [`openapi_d2c.yaml`](./openapi_d2c.yaml) bằng một trong hai cách:
```bash
# Cách 1: Sử dụng npx swagger-ui-watcher (tự động mở trình duyệt và tự reload khi sửa file)
npx swagger-ui-watcher docs/03_api_specs/openapi_d2c.yaml -p 8080

# Cách 2: Sử dụng Docker container chính thức
docker run -p 8080:8080 -e SWAGGER_JSON=/spec/openapi_d2c.yaml -v ${PWD}/docs/03_api_specs:/spec swaggerapi/swagger-ui
```
Truy cập `http://localhost:8080` để xem đầy đủ 32 endpoint được phân nhóm theo 9 Swagger Tag trực quan.

### 3.4. Chạy Mock Server & GraphQL Playground Cho Mobile BFF
Ứng dụng di động sử dụng tầng đệm **Mobile BFF** ([`schema.graphql`](./schema.graphql)) theo chuẩn **NFR-02 (Mobile First)** để gom toàn bộ dữ liệu màn hình (Home, Chi tiết sản phẩm, Giỏ hàng, Vận đơn) trong 1 Single Round-trip mạng 4G:
```bash
# Chạy Apollo Server / GraphQL Yoga Mock Server từ file schema
npx @graphql-tools/mock-cli docs/03_api_specs/schema.graphql --port 4000
```
- Mở `http://localhost:4000` (hoặc Apollo Studio Sandbox) để gửi các query gom màn hình: `homeScreenData`, `productDetailScreen`, `cartScreenData`, `orderTracking`.

---

## 4. Danh Sách Tệp Đặc Tả Chi Tiết Trong Thư Mục
1. [01_grpc_contracts.md](./01_grpc_contracts.md): Đặc tả chi tiết 8 gRPC Services, SLA timeout 2.0s, và cơ chế Retry Idempotent.
2. [02_kafka_event_catalog.md](./02_kafka_event_catalog.md): Danh bạ Kafka Topics phủ 100% 18 microservices, Partition Keys, định dạng CloudEvents 1.0 và cơ chế Dead-Letter-Queue.
3. [openapi_d2c.yaml](./openapi_d2c.yaml): Đặc tả chuẩn OpenAPI 3.0 cho Tầng Biên REST API & Webhooks (32 endpoints, 9 nhóm thẻ nghiệp vụ, 40 schemas).
4. [schema.graphql](./schema.graphql): Đặc tả GraphQL Schema cho Mobile BFF (Backend-For-Frontend) tối ưu mạng 4G cho Native Mobile App (NFR-02).


# ĐẶC TẢ CHI TIẾT CÁC HỢP ĐỒNG ĐỒNG BỘ gRPC (EAST - WEST SYNC CONTRACTS)
## HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ & CHUỖI CUNG ỨNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

Các hợp đồng này được định nghĩa bằng **Protocol Buffers v3** đặt tại thư mục [`packages/proto/`](../../packages/proto/), sử dụng giao thức **gRPC trên nền tảng HTTP/2** cho các tương tác nội bộ nằm trên đường găng nghiệp vụ (**Critical Path**).

---

## 1. Nguyên Tắc Vận Hành gRPC Toàn Hệ Thống

1. **Quy Tắc Quá Giờ Bắt Buộc (Strict Timeout 2.0s):**
   - Mọi lời gọi gRPC giữa các microservices đều phải cấu hình `deadline: 2000ms`. Nếu quá 2 giây service đối tác không phản hồi, client lập tức hủy kết nối và ném lỗi `DEADLINE_EXCEEDED`.
2. **Cầu Dao Ngắt Mạch (Circuit Breaker Policy):**
   - Nếu trong cửa sổ trượt 10 giây có trên 50% số request gRPC sang một service bị thất bại (hoặc timeout), Circuit Breaker sẽ tự động **BẬT MỞ (OPEN)** trong 5 giây, trả về ngay lỗi `ERR_CIRCUIT_BREAKER_OPEN` thay vì làm nghẽn luồng gọi.
3. **Chống Lặp Idempotent Bắt Buộc (Mandatory Idempotency Key):**
   - 100% các request command ghi dữ liệu (`ReserveStock`, `ReleaseReservation`, `ValidateAndLockVoucher`, `CreatePOSOrder`...) đều chứa `string idempotency_key = 1`. Service nhận lưu khóa này vào Redis với TTL 24h để đảm bảo khi mạng giật retry thì chỉ thực thi đúng 1 lần.
4. **An Toàn Kiểu Số Học Tiền Tệ (BigInt Money Safety):**
   - Mọi số tiền đều đóng gói trong message `omamx.common.v1.Money { string currency_code; int64 units; int32 nanos; }`.
   - Phía **Node.js** (`order-service`, `api-gateway`) bắt buộc cấu hình `forceLong=bigint` (đã khai báo trong `buf.gen.yaml`) để đọc trường `units` thành kiểu dữ liệu `BigInt`, ngăn ngừa hoàn toàn nguy cơ làm tròn sai số khi doanh thu vượt $2^{53}-1$ (9.007.199.254.740.991 VND).

---

## 2. Danh Mục Chi Tiết 8 Dịch Vụ gRPC Nội Bộ

### 2.1. MS-01: `InventoryService` (Port nội bộ: 8001)
- **Tệp định nghĩa:** [`packages/proto/inventory/v1/inventory.proto`](../../packages/proto/inventory/v1/inventory.proto)
- **Bên Cung Cấp (Provider):** `inventory-service` (Go)
- **Bên Tiêu Thụ Duy Nhất (Consumer):** `order-service` (Saga Orchestrator)

#### Chi tiết các RPC Methods:
1. `ReserveStock(ReserveStockRequest) returns (ReserveStockResponse)`:
   - **Mục đích:** Tạm khóa tồn kho khả dụng cho đơn hàng trong 15 phút.
   - **Xử lý nội bộ:** Chạy câu lệnh SQL `SELECT physical_quantity, reserved_quantity FROM inventory_items WHERE sku_code = $1 FOR UPDATE;` trong một Database Transaction. Nếu `available = physical - reserved >= requested`, cập nhật `reserved = reserved + requested` và tạo bản ghi trong `stock_reservations`.
   - **Phản hồi:** Trả về `reservation_id` và thời điểm hết hạn `expires_at` (`google.protobuf.Timestamp`). Nếu không đủ hàng, trả về danh sách `unavailable_items` chi tiết số lượng còn thiếu.
2. `ReleaseReservation(ReleaseReservationRequest) returns (ReleaseReservationResponse)`:
   - **Mục đích:** Nhả tồn kho đã khóa khi đơn hàng bị hủy hoặc timeout 15m.
   - **Xử lý nội bộ:** Trừ `reserved_quantity`, đánh dấu phiếu giữ chỗ là `RELEASED`.
3. `GetStockLevel(GetStockLevelRequest) returns (GetStockLevelResponse)`:
   - **Mục đích:** Lấy số lượng tồn kho vật lý và khả dụng của danh sách SKU để hiển thị lên Website.
4. `GetBatchFEFODetails(GetBatchFEFODetailsRequest) returns (GetBatchFEFODetailsResponse)`:
   - **Mục đích:** Tra cứu thông tin các lô kẹo mè xửng đang lưu kho sắp xếp theo tiêu chí Hạn dùng gần nhất xuất trước (FEFO - First Expired, First Out).

---

### 2.2. MS-04: `OrderService` (Port nội bộ: 8004)
- **Tệp định nghĩa:** [`packages/proto/order/v1/order.proto`](../../packages/proto/order/v1/order.proto)
- **Bên Cung Cấp (Provider):** `order-service` (Node.js)
- **Bên Tiêu Thụ (Consumer):**
  - `channel-service` (MS-13 - Phân hệ POS Quầy Bán Lẻ Trực Tiếp Offline tại xưởng Hương Thủy và cửa hàng Huế).
  - *Lưu ý sống còn:* **TUYỆT ĐỐI CẤM** dùng RPC này cho đơn sàn Shopee/TikTok (đơn sàn đi qua Kafka `MarketplaceOrderImportedEvent` vào Saga Orchestrator). Đơn sỉ B2B từ Quản trị viên đi qua REST API Gateway `/api/v1/admin/orders`.

#### Chi tiết các RPC Methods:
1. `CreatePOSOrder(CreatePOSOrderRequest) returns (CreatePOSOrderResponse)`:
   - **Mục đích:** Tạo đơn hàng tức thời cho khách mua kẹo trực tiếp tại quầy thu ngân offline của xưởng/cửa hàng Huế.
   - **Xử lý nội bộ:** Khởi tạo đơn hàng `PAID` ngay lập tức (nếu thanh toán tiền mặt/thẻ tại quầy) hoặc cấp mã VietQR động trên màn hình phụ của quầy POS.
2. `GetOrderDetail(GetOrderDetailRequest) returns (GetOrderDetailResponse)`:
   - **Mục đích:** Truy vấn chi tiết đơn hàng, snapshot địa chỉ, mã tem Seal O Mạ và mã vận đơn.
3. `CancelOrder(CancelOrderRequest) returns (CancelOrderResponse)`:
   - **Mục đích:** Kích hoạt hủy đơn và điều phối đền bù Saga.

---

### 2.3. MS-05: `CatalogService` (Port nội bộ: 8005)
- **Tệp định nghĩa:** [`packages/proto/catalog/v1/catalog.proto`](../../packages/proto/catalog/v1/catalog.proto)
- **Bên Cung Cấp (Provider):** `catalog-service` (Node.js/Go)
- **Bên Tiêu Thụ (Consumer):** `order-service`, `api-gateway`, `channel-service`

#### Chi tiết các RPC Methods:
1. `ValidatePriceAndSKU(ValidatePriceAndSKURequest) returns (ValidatePriceAndSKUResponse)`:
   - **Mục đích:** So sánh giá do Client gửi lên với giá niêm yết chính thống trong CSDL Catalog để chặn đứng hành vi can thiệp F12/Inspect sửa giá hộp mè xửng từ 110.000đ xuống 1.000đ.
   - **Phản hồi:** Trả về `is_valid: true/false`, kèm theo `canonical_subtotal` (tổng tiền chuẩn tính lại từ server) và danh sách `discrepancies` nếu phát hiện sai lệch.
2. `GetProductVariant(GetProductVariantRequest) returns (GetProductVariantResponse)`:
   - **Mục đích:** Lấy thông tin đầy đủ về quy cách đóng gói, khối lượng tịnh (`Weight`), xếp hạng sao OCOP, thời hạn sử dụng.
3. `ListProductVariants(ListProductVariantsRequest) returns (ListProductVariantsResponse)`:
   - **Mục đích:** Phục vụ các màn hình tìm kiếm danh mục có phân trang.

---

### 2.4. MS-07: `PromotionService` (Port nội bộ: 8007)
- **Tệp định nghĩa:** [`packages/proto/promotion/v1/promotion.proto`](../../packages/proto/promotion/v1/promotion.proto)
- **Bên Cung Cấp (Provider):** `promotion-service`
- **Bên Tiêu Thụ (Consumer):** `order-service`

#### Chi tiết các RPC Methods:
1. `ValidateAndLockVoucher(ValidateAndLockVoucherRequest) returns (ValidateAndLockVoucherResponse)`:
   - **Mục đích:** Kiểm tra điều kiện áp mã (Hạn dùng, số lượt còn lại, giá trị đơn tối thiểu) và **tạm khóa 1 lượt dùng trong 15 phút** để tránh tình trạng 1 mã voucher giới hạn 100 lượt bị 200 người dùng đồng thời.
   - **Phản hồi:** Trả về `voucher_lock_id`, số tiền giảm giá `discount_amount` (`Money`), và `expires_at`.
2. `ReleaseVoucher(ReleaseVoucherRequest) returns (ReleaseVoucherResponse)`:
   - **Mục đích:** Mở khóa voucher hoàn lại lượt dùng khi khách hủy đơn hoặc không thanh toán.

---

### 2.5. MS-12: `ShippingService` (Port nội bộ: 8012)
- **Tệp định nghĩa:** [`packages/proto/shipping/v1/shipping.proto`](../../packages/proto/shipping/v1/shipping.proto)
- **Bên Cung Cấp (Provider):** `shipping-service`
- **Bên Tiêu Thụ (Consumer):** `order-service`, `fulfillment-service`

#### Chi tiết các RPC Methods:
1. `CalculateShippingFee(CalculateShippingFeeRequest) returns (CalculateShippingFeeResponse)`:
   - **Mục đích:** Tính toán cước phí vận chuyển chính xác từ Kho xưởng Hương Thủy (Huế) đến địa chỉ người nhận, tự động tính thêm phụ phí bọc xốp bóng khí bảo vệ kẹo giòn dễ vỡ nếu `is_fragile = true`.
2. `CreateShipment(CreateShipmentRequest) returns (CreateShipmentResponse)`:
   - **Mục đích:** Bắn lệnh tạo vận đơn sang hãng giao vận 3PL (GHN/ViettelPost), lấy mã vận đơn `tracking_code` và đường link nhãn in nhiệt PDF A6.

---

### 2.6. MS-02: `FulfillmentService` (Port nội bộ: 8002)
- **Tệp định nghĩa:** [`packages/proto/fulfillment/v1/fulfillment.proto`](../../packages/proto/fulfillment/v1/fulfillment.proto)
- **Bên Cung Cấp (Provider):** `fulfillment-service`
- **Bên Tiêu Thụ (Consumer):** `order-service`, `care-service` (CSKH khiếu nại)

#### Chi tiết các RPC Methods:
1. `GetPackingVideoUrl(GetPackingVideoUrlRequest) returns (GetPackingVideoUrlResponse)`:
   - **Mục đích:** Sinh đường link Pre-signed S3 URL an toàn (TTL 15 phút - NFR-09) của video đóng gói kẹo có dán tem Seal O Mạ, kèm mã băm SHA-256 để CSKH đối soát khi khách khiếu nại kẹo vỡ.
2. `SubmitPackingEvidence(SubmitPackingEvidenceRequest) returns (SubmitPackingEvidenceResponse)`:
   - **Mục đích:** Thợ xưởng Huế nạp mã tem Seal O Mạ, mã lô FEFO và Object Key của video hoàn tất đóng gói.

---

### 2.7. MS-15: `ProfileService` (Port nội bộ: 8015)
- **Tệp định nghĩa:** [`packages/proto/profile/v1/profile.proto`](../../packages/proto/profile/v1/profile.proto)
- **Bên Cung Cấp (Provider):** `profile-service`
- **Bên Tiêu Thụ (Consumer):** `order-service` (MS-04), `care-service` (MS-06), `fulfillment-service` (MS-02)

#### Chi tiết các RPC Methods:
1. `GetCustomerProfile(GetCustomerProfileRequest) returns (GetCustomerProfileResponse)`:
   - **Mục đích:** Lấy thông tin họ tên, số điện thoại, hạng thành viên loyalty.
2. `GetDeliveryAddress(GetDeliveryAddressRequest) returns (GetDeliveryAddressResponse)`:
   - **Mục đích:** Phục vụ tính năng **Address Snapshot** (database_design.md mục 4.1). Khi khách đặt hàng, Order Service gọi method này để lấy địa chỉ chuẩn xác và chụp bản sao bất biến vào bảng `orders`, ngăn chặn việc khách đổi địa chỉ trong sổ địa chỉ làm sai lệch đơn hàng đang giao.

---

### 2.8. MS-16: `IdentityService` (Port nội bộ: 8016)
- **Tệp định nghĩa:** [`packages/proto/identity/v1/identity.proto`](../../packages/proto/identity/v1/identity.proto)
- **Bên Cung Cấp (Provider):** `identity-service`
- **Bên Tiêu Thụ (Consumer):** `finance-service` (MS-09), `order-service` (MS-04), `api-gateway`

#### Chi tiết các RPC Methods:
1. `CheckSpecializedPermission(SpecializedPermissionRequest) returns (SpecializedPermissionResponse)`:
   - **Mục đích:** Thẩm duyệt phân quyền động (ABAC/PBAC) cho các hành động có rủi ro tài chính cao (HLD Mục 3.5 & 4.7):
     - *"Kế toán duyệt Refund > 10 triệu VND"* (Cần Giám đốc phê duyệt).
     - *"Giám đốc duyệt chiết khấu B2B > 25%"*.
2. `GetJwksPublicKey(GetJwksPublicKeyRequest) returns (GetJwksPublicKeyResponse)`:
   - **Mục đích:** Cung cấp khóa công khai JWKS để API Gateway nạp cache cục bộ định kỳ, xác thực JWT ngay tại biên mà không gọi blocking vào Identity Service.

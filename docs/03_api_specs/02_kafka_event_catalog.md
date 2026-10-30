# DANH MỤC SỰ KIỆN PHÂN TÁN KAFKA (EAST - WEST ASYNC EVENT CATALOG)
## HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ & CHUỖI CUNG ỨNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

Tất cả các sự kiện bất đồng bộ được quản lý trong thư mục [`packages/events/`](../../packages/events/), tuân thủ định dạng **CNCF CloudEvents 1.0 JSON Schema**.

---

## 1. Bảng Ma Trận Topics & Đăng Ký Consumers Toàn Hệ Thống (Phủ 100% 18 Microservices)

> [!IMPORTANT]
> **Khớp 100% Ma Trận HLD Mục 3.6:** Bảng dưới đây định nghĩa đầy đủ danh sách Producer (Bên phát sinh sự kiện) và 100% Consumer (Bên đăng ký tiêu thụ) trên toàn bộ 18 dịch vụ, giải quyết triệt để tình trạng thiếu hụt consumer dẫn tới vỡ hợp đồng khi tích hợp:

| Topic Tên | Partition Key | Publisher (Bên Bắn) | Danh Sách Consumers Cam Kết (Khớp 100% HLD 3.6) | Mục Đích Nghiệp Vụ |
| :--- | :---: | :--- | :--- | :--- |
| **`order.events.v1`** | `order_id` | `order-service` (MS-04) | • `fulfillment-service` (MS-02 - tạo Picking Task)<br/>• `inventory-service` (MS-01 - trừ kho vật lý FEFO)<br/>• `promotion-service` (MS-07 - ghi nhận voucher đã dùng, tích điểm 1% loyalty)<br/>• `finance-service` (MS-09 - ghi nhận doanh thu, xuất hóa đơn VAT điện tử)<br/>• `care-service` (MS-06 - mở quyền viết review sau khi OrderCompleted)<br/>• `notification-service` (MS-17 - bắn Zalo ZNS / Push App)<br/>• `analytics-service` (MS-11 - cập nhật GMV real-time) | Vòng đời đơn hàng D2C & POS (`OrderPlaced`, `OrderPaid`, `OrderCancelled`, `OrderCompleted`, `MarketplaceOrderStockFailed`). |
| **`inventory.events.v1`** | `sku_code` | `inventory-service` (MS-01) | • `catalog-service` (MS-05 - làm mới cache tồn Redis)<br/>• `channel-service` (MS-13 - đồng bộ tồn khả dụng lên Shopee/TikTok)<br/>• `procurement-service` (MS-08 - tự động dự thảo PO khi nguyên liệu dưới ngưỡng)<br/>• `marketing-service` (MS-14 - phát động chiến dịch xả hàng cận date)<br/>• `promotion-service` (MS-07 - tạo Flash Sale xả hàng cận date)<br/>• `notification-service` (MS-17 - cảnh báo thủ kho)<br/>• `analytics-service` (MS-11) | Biến động số lượng tồn kho (`StockLevelChanged`, `StockReserved`, `StockReleased`), cảnh báo kẹo cận date 45 ngày (`ExpiryWarningEvent`). |
| **`fulfillment.events.v1`** | `order_id` | `fulfillment-service` (MS-02) | • `shipping-service` (MS-12 - tạo vận đơn 3PL lấy hàng)<br/>• `order-service` (MS-04 - chuyển trạng thái PACKED)<br/>• `traceability-service` (MS-03 - kích hoạt mã tem QR OCOP)<br/>• `notification-service` (MS-17) | Thợ xưởng Huế hoàn tất đóng gói, dán tem Seal O Mạ và nạp video kiểm định lên S3 (`PackingCompleted`). |
| **`shipping.events.v1`** | `tracking_code` | `shipping-service` (MS-12) | • `order-service` (MS-04 - chuyển trạng thái SHIPPED/DELIVERED)<br/>• `finance-service` (MS-09 - đối soát công nợ COD và cước 3PL)<br/>• `notification-service` (MS-17) | Cập nhật tiến độ giao vận từ bưu tá 3PL (`ShipmentCreated`, `ShipmentDelivered`, `ShipmentFailed`). |
| **`channel.events.v1`** | `marketplace_order_id` | `channel-service` (MS-13) | • `order-service` (MS-04 - Saga Orchestrator độc quyền tiêu thụ để khóa tồn và tạo đơn nội bộ)<br/>• `analytics-service` (MS-11) | Tiếp nhận đơn sàn Shopee/TikTok Shop từ Webhook đối tác, bảo đảm nguyên tắc Centralized Compensation. |
| **`finance.events.v1`** | `order_id` | `finance-service` (MS-09) | • `care-service` (MS-06 - xác nhận tiền hoàn về thành công cho khách)<br/>• `notification-service` (MS-17 - gửi hóa đơn VAT PDF) | Thông báo tài chính (`InvoiceIssued`, `RefundProcessed`). |
| **`promotion.events.v1`** | `customer_id` | `promotion-service` (MS-07) | • `analytics-service` (MS-11 - phân tích hiệu quả ROI voucher)<br/>• `notification-service` (MS-17 - thông báo cộng điểm thưởng) | Ghi nhận voucher đã dùng (`VoucherUsed`) và tích điểm loyalty (`LoyaltyPointsEarned`). |
| **`care.events.v1`** | `order_id` | `care-service` (MS-06) | • `finance-service` (MS-09 - kích hoạt hoàn tiền sau đối soát)<br/>• `inventory-service` (MS-01 - nhập lại kho hàng đổi trả còn nguyên vẹn)<br/>• `notification-service` (MS-17) | CSKH xử lý khiếu nại kẹo vỡ/đổi trả (`ReturnTicketApproved`, `ReturnInspected`, `ReviewSubmitted`). |
| **`procurement.events.v1`** | `sku_code` | `procurement-service` (MS-08) | • `inventory-service` (MS-01 - tạo Lô hàng FEFO mới)<br/>• `finance-service` (MS-09 - ghi nhận công nợ AP với HTX mè Huế)<br/>• `traceability-service` (MS-03 - ghi nhận vùng nguyên liệu mè cát Quảng Điền) | Quản lý thu mua nguyên liệu mè/đậu Huế (`GoodsReceived`, `PurchaseOrderApproved`). |
| **`profile.events.v1`** | `customer_id` / `employee_id` | `profile-service` (MS-15) | • `analytics-service` (MS-11 - đồng bộ nhân khẩu học)<br/>• `marketing-service` (MS-14 - cập nhật sở thích kẹo)<br/>• `notification-service` (MS-17 - cảnh báo chứng chỉ VSATTP cho quản đốc)<br/>• `fulfillment-service` (MS-02 - phân quyền nhân sự đóng gói mới) | Vòng đời hồ sơ khách hàng & nhân sự (`ProfileUpdated`, `DefaultAddressSwitched`, `EmployeeOnboarded`, `StaffComplianceWarningEvent`). |
| **`identity.events.v1`** | `user_id` | `identity-service` (MS-16) | • `profile-service` (MS-15 - tự động tạo hồ sơ rỗng ban đầu)<br/>• `marketing-service` (MS-14 - kích hoạt chiến dịch chào mừng khách mới) | Quản lý tài khoản định danh (`UserRegistered`, `UserDeactivated`, `PasswordChanged`). |
| **`audit.events.v1`** | `entity_id` | Tất cả 18 Services | • `audit-service` (MS-18 - Audit Plane) | Ghi nhận chứng cứ kiểm toán pháp lý không thể chối bỏ bằng Hash Chain SHA-256 (`AuditRecordEvent`). |

---

## 2. Chi Tiết Các Sự Kiện Lõi Của Chuỗi Cung Ứng & Phân Vùng

### 2.1. Sự Kiện `OrderPaidEvent` (`vn.omama.order.paid.v1`)
- **Tệp Schema:** [`packages/events/schemas/order/v1/order_paid.event.json`](../../packages/events/schemas/order/v1/order_paid.event.json)
- **Topic:** `order.events.v1` | **Partition Key:** `order_id`
- **Thời điểm bắn:** Ngay khi Webhook ngân hàng VietQR báo tiền về khớp số tiền và đúng chữ ký HMAC.
- **Tác vụ của các Consumer:**
  - `fulfillment-service`: Nhận đơn, tạo phiếu bốc hàng (Picking Task) cho thợ xưởng Hương Thủy.
  - `inventory-service`: Trừ kho vật lý vĩnh viễn theo nguyên tắc FEFO (lấy kẹo trong lô có hạn dùng gần nhất).
  - `promotion-service`: Đánh dấu voucher đã sử dụng chính thức.
  - `finance-service`: Ghi nhận doanh thu và kích hoạt quy trình xuất hóa đơn VAT điện tử.
  - `notification-service`: Bắn thông báo Zalo ZNS / SMS / Push App: *"O Mạ đã nhận được thanh toán, xưởng đang chuẩn bị kẹo cho bạn!"*.
  - `analytics-service`: Cập nhật chỉ số doanh thu thời gian thực (Real-time GMV).

### 2.2. Sự Kiện `MarketplaceOrderImportedEvent` (`vn.omama.channel.marketplace.order.imported.v1`)
- **Tệp Schema:** [`packages/events/schemas/channel/v1/marketplace_order_imported.event.json`](../../packages/events/schemas/channel/v1/marketplace_order_imported.event.json)
- **Topic:** `channel.events.v1` | **Partition Key:** `marketplace_order_id`
- **Thời điểm bắn:** Khi Channel Service nhận Webhook đơn hàng mới từ sàn Shopee hoặc TikTok Shop.
- **Tác vụ của các Consumer:**
  - `order-service` (Saga Orchestrator): Tiêu thụ sự kiện, tạo đơn hàng nội bộ và độc quyền gọi gRPC `ReserveStock` sang Inventory. Nếu kho hết hàng, bắn sự kiện `MarketplaceOrderStockFailedEvent` để Channel Service báo hủy đơn lên sàn, đảm bảo **100% Centralized Compensation**.

### 2.3. Sự Kiện `InvoiceIssuedEvent` & `RefundProcessedEvent`
- **Tệp Schema:** [`packages/events/schemas/finance/v1/invoice_issued.event.json`](../../packages/events/schemas/finance/v1/invoice_issued.event.json) & [`refund_processed.event.json`](../../packages/events/schemas/finance/v1/refund_processed.event.json)
- **Topic:** `finance.events.v1` | **Partition Key:** `order_id`
- **Tác vụ của các Consumer:**
  - `care-service`: Đóng ticket khiếu nại sau khi tiền hoàn đã về tài khoản khách hàng.
  - `notification-service`: Gửi đường link file PDF hóa đơn VAT điện tử qua Email cho khách hàng doanh nghiệp B2B.

### 2.4. Sự Kiện `ExpiryWarningEvent` (`vn.omama.inventory.batch.expiry.warning.v1`)
- **Tệp Schema:** [`packages/events/schemas/inventory/v1/expiry_warning.event.json`](../../packages/events/schemas/inventory/v1/expiry_warning.event.json)
- **Topic:** `inventory.events.v1` | **Partition Key:** `sku_code`
- **Thời điểm bắn:** Định kỳ 01:00 AM hằng ngày, Scheduled Job quét thấy lô kẹo mè xửng còn dưới 45 ngày hạn sử dụng (FEFO Rule).
- **Tác vụ của các Consumer:**
  - `promotion-service`: Tự động kích hoạt mã giảm giá Flash Sale 20% - 30% xả hàng.
  - `marketing-service`: Đẩy banner khuyến mãi "Tri ân mè xửng Huế chuẩn vị - Ưu đãi có hạn" lên trang chủ.

### 2.5. Sự Kiện `GoodsReceivedEvent` (`vn.omama.procurement.goods.received.v1`)
- **Tệp Schema:** [`packages/events/schemas/procurement/v1/goods_received.event.json`](../../packages/events/schemas/procurement/v1/goods_received.event.json)
- **Topic:** `procurement.events.v1` | **Partition Key:** `sku_code`
- **Tác vụ của các Consumer:**
  - `inventory-service`: Nhập kho và tạo Lot sản xuất mới.
  - `finance-service`: Ghi nhận khoản phải trả người bán (Accounts Payable - AP) cho Hợp tác xã mè/đậu Huế.
  - `traceability-service`: Ghi nhận tọa độ và thông tin nông hộ trồng mè sạch phục vụ tem QR OCOP.

### 2.6. Sự Kiện Kiểm Toán `AuditRecordEvent` (`vn.omama.audit.record.v1`)
- **Tệp Schema:** [`packages/events/schemas/audit/v1/audit_record.event.json`](../../packages/events/schemas/audit/v1/audit_record.event.json)
- **Topic:** `audit.events.v1` | **Partition Key:** `entity_id`
- **Cơ chế:** Hash Chain SHA-256:
  $$\text{current\_entity\_hash} = \text{SHA-256}(\text{prev\_entity\_hash} + \text{actor\_id} + \text{action} + \text{after\_state} + \text{timestamp})$$

### 2.7. Sự Kiện Cảnh Báo Tuân Thủ VSATTP Nhân Sự `StaffComplianceWarningEvent` (`vn.omama.profile.staff.compliance.warning.v1`)
- **Topic:** `profile.events.v1` | **Partition Key:** `employee_id`
- **Thời điểm bắn:** Định kỳ 02:00 AM hằng ngày, Scheduled Job trong Profile Service quét thấy thợ nấu kẹo/nhân viên đóng gói có chứng chỉ VSATTP còn $\le 30$ ngày hoặc $\le 7$ ngày.
- **Tác vụ của các Consumer:**
  - `notification-service`: Gửi email và thông báo Zalo ZNS cảnh báo đỏ cho Quản đốc xưởng Hương Thủy và Trưởng phòng HR lên danh sách gia hạn tập huấn kịp thời, đảm bảo 100% điều kiện pháp lý của chuẩn OCOP 4 sao.
  - `fulfillment-service`: Cảnh báo điều phối không xếp ca phụ trách dán tem kiểm định cho nhân sự có chứng chỉ hết hạn.


# DANH MỤC SỰ KIỆN PHÂN TÁN & QUY CHUẨN KAFKA CLOUDEVENTS (EVENT CATALOG)
## HỆ SINH THÁI THƯƠNG MẠI ĐIỆN TỬ & CHUỖI CUNG ỨNG ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

Thư mục này chứa toàn bộ các bản hợp đồng sự kiện bất đồng bộ (Asynchronous Event Schemas) của hệ thống, được chuẩn hóa theo định dạng **CNCF CloudEvents 1.0** và lưu trữ dưới dạng **JSON Schema (Draft-07)**.

---

## 1. Bảng Ma Trận Phân Vùng & Đăng Ký Consumers Toàn Hệ Thống (Đầy Đủ 18 Microservices)

> [!CRITICAL]
> **Quy Tắc Sống Còn Của Kafka: Thứ Tự Xử Lý Chỉ Được Bảo Đảm Trong Cùng Một Partition!**
> Việc lựa chọn Partition Key quyết định thứ tự xử lý của chuỗi sự kiện. Bảng dưới đây ánh xạ 100% các Topic, Partition Key và danh sách Consumer cam kết theo đúng HLD Mục 3.6:

| Topic Kafka | Partition Key | Publisher (Bên Bắn) | Danh Sách Consumers Cam Kết (Đầy Đủ 18 Services) | Rationale (Lý Do Kỹ Thuật) |
| :--- | :---: | :--- | :--- | :--- |
| **`order.events.v1`** | `order_id` | `order-service` (MS-04) | • `fulfillment-service` (MS-02 - tạo Picking Task)<br/>• `inventory-service` (MS-01 - trừ kho vật lý FEFO)<br/>• `promotion-service` (MS-07 - ghi nhận dùng voucher, tích điểm 1% loyalty)<br/>• `finance-service` (MS-09 - ghi nhận doanh thu, xuất hóa đơn điện tử VAT)<br/>• `care-service` (MS-06 - mở quyền viết review sau OrderCompleted)<br/>• `notification-service` (MS-17 - bắn Zalo ZNS/Push App)<br/>• `analytics-service` (MS-11 - cập nhật GMV real-time) | Đảm bảo toàn bộ chuỗi sự kiện của 1 đơn hàng (`Placed` ➔ `Paid` ➔ `Cancelled` ➔ `Completed`) luôn vào cùng 1 partition theo tuần tự thời gian. |
| **`inventory.events.v1`** | `sku_code` | `inventory-service` (MS-01) | • `catalog-service` (MS-05 - làm mới cache tồn Redis)<br/>• `channel-service` (MS-13 - đồng bộ tồn khả dụng lên Shopee/TikTok)<br/>• `procurement-service` (MS-08 - tự động dự thảo PO khi nguyên liệu dưới ngưỡng)<br/>• `marketing-service` (MS-14 - phát động chiến dịch xả hàng cận date)<br/>• `promotion-service` (MS-07 - tạo Flash Sale xả hàng cận date)<br/>• `notification-service` (MS-17 - cảnh báo thủ kho)<br/>• `analytics-service` (MS-11) | Đảm bảo các cập nhật biến động tồn kho và cảnh báo hạn dùng kẹo mè xửng luôn tuần tự, không bị race-condition. |
| **`fulfillment.events.v1`** | `order_id` | `fulfillment-service` (MS-02) | • `shipping-service` (MS-12 - gọi API 3PL tạo vận đơn)<br/>• `order-service` (MS-04 - chuyển trạng thái PACKED)<br/>• `traceability-service` (MS-03 - kích hoạt mã tem QR OCOP)<br/>• `notification-service` (MS-17) | Đảm bảo các mốc đóng gói xưởng và dán tem Seal O Mạ được ráp nối chuẩn xác theo đơn hàng. |
| **`shipping.events.v1`** | `tracking_code` | `shipping-service` (MS-12) | • `order-service` (MS-04 - chuyển trạng thái SHIPPED/DELIVERED)<br/>• `finance-service` (MS-09 - đối soát phí ship 3PL)<br/>• `notification-service` (MS-17) | Đảm bảo các checkpoint hành trình từ 3PL (GHN/ViettelPost) được cập nhật theo đúng diễn biến thời gian. |
| **`channel.events.v1`** | `marketplace_order_id` | `channel-service` (MS-13) | • `order-service` (MS-04 - Inbound Saga Orchestrator độc quyền khóa tồn)<br/>• `analytics-service` (MS-11) | Tiếp nhận đơn sàn Shopee/TikTok, bảo đảm 100% nguyên tắc Centralized Compensation (Mục 3.4). |
| **`finance.events.v1`** | `order_id` | `finance-service` (MS-09) | • `care-service` (MS-06 - xác nhận tiền hoàn về cho khách)<br/>• `notification-service` (MS-17 - gửi hóa đơn VAT PDF) | Thông báo hóa đơn điện tử (`InvoiceIssued`) hoặc hoàn tiền (`RefundProcessed`). |
| **`promotion.events.v1`** | `customer_id` | `promotion-service` (MS-07) | • `analytics-service` (MS-11 - tính toán ROI chiến dịch)<br/>• `notification-service` (MS-17) | Ghi nhận voucher đã dùng (`VoucherUsed`) và cộng điểm thành viên (`LoyaltyPointsEarned`). |
| **`care.events.v1`** | `order_id` | `care-service` (MS-06) | • `finance-service` (MS-09 - hoàn tiền khiếu nại)<br/>• `inventory-service` (MS-01 - nhập lại kho hàng đổi trả)<br/>• `notification-service` (MS-17) | Chấp thuận đổi trả sau đối soát video tem Seal (`ReturnTicketApproved`). |
| **`procurement.events.v1`** | `sku_code` | `procurement-service` (MS-08) | • `inventory-service` (MS-01 - tạo Lô hàng FEFO mới)<br/>• `finance-service` (MS-09 - ghi nhận công nợ AP với HTX mè Huế)<br/>• `traceability-service` (MS-03 - ghi nhận vùng nguyên liệu) | Nhập nguyên liệu mè/đậu Huế hoặc kẹo thành phẩm (`GoodsReceived`). |
| **`audit.events.v1`** | `entity_id` | Tất cả 18 Services | • `audit-service` (MS-18 - Audit Plane) | Đảm bảo chuỗi băm Hash Chain (`prev_entity_hash` ➔ `current_entity_hash`) liên tục, chống sửa xóa dữ liệu. |

---

## 2. Cấu Trúc Khung Tiêu Chuẩn CloudEvents 1.0

Mọi thông điệp bắn lên Kafka đều phải bọc trong Envelope chuẩn:

```json
{
  "specversion": "1.0",
  "id": "evt-<uuid-v4>",
  "source": "https://omama.vn/services/<service-name>",
  "type": "vn.omama.<domain>.<event-name>.<version>",
  "subject": "<entity-type>:<entity-id>",
  "time": "2026-10-15T08:30:15.120Z",
  "datacontenttype": "application/json",
  "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
  "data": {
    "... payload nghiệp vụ đặc thù ..."
  }
}
```

- **Tối thiểu hóa PII (Privacy by Design):** Không truyền trực tiếp Số CMND/CCCD, Số điện thoại đầy đủ, Địa chỉ chi tiết trong các Event phát tán rộng. Chỉ dùng các khóa định danh liên kết (`customer_id`, `shipping_address_id`).

---

## 3. Chính Sách Quản Trị Schema Registry & Dead-Letter-Queue (DLQ)

1. **Chế Độ Tương Thích:** Bắt buộc áp dụng **`BACKWARD COMPATIBILITY`**.
   - Nhà sản xuất (Producer) có thể bổ sung trường mới nhưng trường đó phải là optional (hoặc có giá trị mặc định).
   - Tuyệt đối cấm xóa trường hoặc đổi kiểu dữ liệu của trường đang có trong phiên bản `v1`.
2. **Xử Lý Lỗi Consumer (DLQ Policy):**
   - Nếu Consumer xử lý sự kiện thất bại: Thử lại 3 lần với thuật toán **Exponential Backoff** (1s, 2s, 4s).
   - Nếu sau 3 lần vẫn lỗi: Đẩy sự kiện sang Topic Dead-Letter-Queue tương ứng (ví dụ: `order.events.v1.dlq`) kèm Header lý do lỗi (`x-dead-letter-reason`) để đội Vận hành kiểm tra thủ công, không làm tắc nghẽn Consumer Group.

# UI handoff — quote, lịch sử đơn và tracking

Cập nhật 31/10/2026. Đây là thay đổi backend/contract để Web và Mobile triển khai;
chưa xác nhận app đã bật remote hoặc mọi provider đã nối thật.

## 1. MVP mua toàn bộ giỏ

`POST /api/v1/checkout/quote` dùng `cart_revision`, `payment_method` và **đúng một**
trong `shipping_address_id` hoặc `address`. Không nhận `selected_item_ids`/`items`.
Backend báo giá toàn bộ giỏ ở revision đó, chưa giữ hàng trong kho. Thay số lượng,
địa chỉ hoặc phương thức thanh toán thì yêu cầu quote mới và khách xác nhận lại.

UI hiển thị items/tổng tiền/expiry từ quote, không tính phí hoặc tạo order ID riêng.
Khi operation thành công, đọc lại giỏ server. Cleanup phía Order chỉ xóa đúng cart
ID/revision đã mua; giỏ thay đổi hoặc bị thay thế được giữ lại. Cleanup chạy bất đồng
bộ nên giỏ có thể còn xuất hiện tạm thời. Không clear giỏ trên client.

Schema đã loại trường hợp gửi thiếu hoặc đồng thời hai cách truyền địa chỉ.
Runtime hiện trả `ADDRESS_REQUIRED` (422) khi thiếu, `AMBIGUOUS_ADDRESS` (400) khi
cả hai được cung cấp. Voucher/Guest/COD vẫn theo gate cũ, không mở bằng PR này.

## 2. Lịch sử đơn

```http
GET /api/v1/orders?limit=20
GET /api/v1/orders?limit=20&status=PAID
GET /api/v1/orders?limit=20&status=PAID&cursor=<URL-encoded next_cursor>
```

- `limit`: 1–100, mặc định 20; giá trị sai trả 400 `INVALID_LIMIT`.
- `status`: trạng thái **đơn**, không phải trạng thái tiền. So khớp chính xác; bỏ
  hoặc rỗng nghĩa là không lọc. Giá trị chưa biết trả danh sách rỗng.
- `cursor`: chuỗi opaque do server ký; gửi lại nguyên trạng, không tự giải mã.
  Ràng buộc đúng principal và bộ lọc status. Đổi status phải bỏ cursor.
- Thứ tự `created_at DESC, id DESC`, gồm tie-break khi nhiều đơn cùng thời điểm.
  Không phải ảnh chụp cố định qua nhiều lần đọc; đơn mới thấy khi refresh từ đầu.
- Response giữ `{items, next_cursor}`. `next_cursor: ""` là hết trang, kể cả trang
  cuối vẫn có items. Không có `total`, `page` hoặc `pageSize`.
- `q`, `page`, `pageSize`, query khác hoặc query trùng tên trả 400; không âm thầm
  bỏ qua tìm kiếm để UI hiểu nhầm kết quả. Chưa có tìm kiếm server theo mã đơn.
- Refresh thay danh sách từ đầu. "Tải thêm" append và có thể dedup theo Order UUID.
  Hiển thị `order_code` cho khách, dùng UUID cho route; không chuyển mã thành UUID.

Test PostgreSQL/Redis thật kiểm trang đầu/cuối/trang vừa đúng limit, thời gian bằng
nhau, không trùng/mất đơn, cursor đổi bộ lọc/khách, query không hỗ trợ và dữ liệu rỗng.
Hành vi queue/admin cũ không được thay đổi trong lát cắt này.

## 3. Tracking tối thiểu

`GET /api/v1/orders/{id}/tracking` hiện trả DTO được khai báo rõ:

```json
{"shipment_id": null, "order_status": "PAID"}
```

```json
{"shipment_id": "11111111-1111-4111-8111-111111111111", "order_status": "SHIPPED"}
```

`shipment_id=null` nghĩa là Order chưa ghi tham chiếu shipment, không chứng minh đã
hủy/đã giao. Endpoint kiểm ownership, khách khác nhận 404. Không có carrier, mã vận
đơn, checkpoints, seal/video hoặc thời điểm từng mốc. UI hiển thị trạng thái thật
và "Chưa có dữ liệu hành trình"; không tự dựng timeline. Delivered khác Completed.
Tracking nhiều kiện và carrier timeline cần provider projection/contract riêng sau.

## 4. Địa giới: còn chặn picker thêm/sửa địa chỉ

Profile có bảng `administrative_units` nhưng migration 000002 đang seed mã mẫu như
`WARD-HN-001`, không phải bằng chứng danh mục hành chính chính thức đầy đủ. Không
công bố các mã này là nguồn canonical production hoặc tự sinh mã từ tên.

Trong lúc chờ, UI có thể làm luồng địa chỉ đã lưu hợp lệ với backend môi trường
đang dùng; không coi mã fixture là hợp lệ với Shipping thật. Việc thêm/sửa địa chỉ
production cần nguồn dữ liệu được chủ sở hữu xác nhận, version/cập nhật và API lookup
tỉnh/phường theo cùng dataset mà Profile/Shipping xác thực. Không mở picker giả để
đánh dấu G2 hoàn thành. PR này không thêm endpoint location dựa trên seed mẫu.

## 5. Cách cập nhật UI

1. Pin public OpenAPI mới trong PR này; giữ snapshot lịch sử nguyên vẹn.
2. Web/Mobile đổi list request/response theo mục 2 và DTO tracking theo mục 3.
3. Mobile nối các RestMvp repositories vào remote cohort có Identity thật trước khi
   nghiệm thu; local account/session không phải tài khoản Keycloak.
4. Kiểm với gateway/BFF thật: PublicError, header Idempotency-Key và identity mapping.
   Test service trực tiếp không tự nghiệm thu adapter gateway hoặc toàn bộ checkout.

Các quyết định trên cụ thể hóa hướng full-cart đã đề xuất trong TEAM-DECISIONS;
tracking chỉ đặc tả dữ liệu backend đang có. G2, search, rich tracking và provider
checkout acceptance vẫn mở. Registry toàn cục tiếp tục là integration-target.

## Kiểm chứng ngày 31/10/2026

- Linux Go 1.22: test app/HTTP và `go vet ./...` PASS.
- `TestCustomerHistoryCursorAndMinimalTracking` chạy với `-race -count=1` trên
  PostgreSQL/Redis thật PASS. Catalog/Profile/Inventory trong harness là simulator,
  không dùng kết quả này làm bằng chứng provider checkout thật đã nghiệm thu.
- MVP validator PASS: 28 OpenAPI, 48 event schemas, 14 protobuf, bốn wire baseline,
  20 negative guards; Order design verifier PASS qua descriptor đã compile.
- CI trên PR phải kiểm lại toàn bộ regression/race và runtime contract trước merge.

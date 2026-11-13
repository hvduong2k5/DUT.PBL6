# Kết nối Kong và trang tra cứu QR

**Ngày lập:** 13/11/2026 (UTC+7) · **Mốc:** `dev@239c8db` · **Trạng thái:** Chốt thiết kế ngày 13/11/2026

Web đã có trang `/trace/{qrCode}` và BFF customer-core. Kong chưa route đến Traceability. OpenAPI customer hiện yêu cầu một số trường chứng nhận luôn có giá trị, trong khi DTO Traceability có trường nullable và metadata xác minh.

| Lớp | Thiết kế |
| --- | --- |
| Web | Giữ URL `/trace/{qrCode}` để mở từ QR. BFF dùng `TRACEABILITY_UPSTREAM_URL` trỏ tới prefix `/api/v2/trace` qua Kong. Reviews, vouchers và loyalty tiếp tục dùng customer-core. |
| Kong public | Chuyển `GET /api/v2/trace/{qr_code}` đến path `/api/v1/trace/{qr_code}` của Traceability. Cho phép truy cập ẩn danh và kiểm thử mã hợp lệ qua BFF. |
| Kong admin | Chuyển `/api/v1/admin/trace/records` và các thao tác con theo OpenAPI Traceability. Giữ ranh giới xác thực với luồng public. Health và metrics phục vụ mạng nội bộ. |
| Contract | Dùng DTO Traceability cho API mới `GET /api/v2/trace/{qr_code}` và cấu hình Kong rewrite tới `/api/v1/trace/{qr_code}` nội bộ. BFF mới dùng prefix `/api/v2/trace`. Giữ contract v1 cho client hiện có. Thêm OpenAPI v2 và cập nhật state matrix cùng thay đổi code. |

## Trạng thái hiển thị

| Phản hồi | Web hiển thị |
| --- | --- |
| `200` có dữ liệu | Hiển thị bản công bố, thời điểm và phương thức xác minh. Với `MANUAL`, ghi rõ xác minh thủ công. |
| `200` có trường nullable | Ẩn trường hoặc ghi chưa có dữ liệu cho nghệ nhân, chứng nhận an toàn thực phẩm hoặc video. Không điền dữ liệu mẫu vào hồ sơ thật. |
| `404` | Không tìm thấy mã. Cho phép nhập lại. |
| `410` | Hồ sơ đã ngừng công bố. Không hiển thị như mã chưa từng tồn tại. |
| `503` | Dịch vụ tạm thời gián đoạn. Cho phép thử lại. |

## Nghiệm thu

- Dùng trình duyệt mở QR của hồ sơ vừa publish qua Web → BFF → Kong → Traceability. API trả đúng revision và không lộ bằng chứng nội bộ.
- Sửa nháp mới không đổi bản công bố đang hiển thị. Sau khi đình chỉ, cùng URL hiển thị trạng thái ngừng công bố và BFF giữ HTTP `410`.
- Test mã lạ, dữ liệu nullable, trạng thái xác minh thủ công và upstream gián đoạn. Kiểm tra truy cập ẩn danh public và truy cập admin khi thiếu token.
- Ghi cấu hình route, SHA Web/Gateway, request ID và kết quả browser test trong biên bản nghiệm thu.

Nguồn đối chiếu [customer OpenAPI](../../../03_api_specs/customer-stable.openapi.yaml), [Traceability OpenAPI](../../../03_api_specs/traceability-service.openapi.yaml) và [cấu hình Kong](../../../../services/api-gateway/config/kong.yml).

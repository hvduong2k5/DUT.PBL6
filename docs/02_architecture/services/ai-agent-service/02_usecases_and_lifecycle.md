# 02 — Use case và vòng đời

## Use case đợt 1

| Use case | Kích hoạt | Kết quả |
| --- | --- | --- |
| Tạo phiên chat | `POST /api/v1/ai/shopping/conversations` (có hoặc không có Bearer JWT) | Phiên thuộc loại `guest` hoặc `customer`, kèm `expires_at` theo retention |
| Chat tư vấn | Gửi tin nhắn vào phiên | Luồng SSE: token → product_cards/citations → done (chi tiết [chương 09](09_shopping_workflow_design.md)) |
| Hỏi FAQ | Câu hỏi khớp nhóm chủ đề FAQ (hạn dùng, bảo quản, làng nghề, địa chỉ…) | Trả lời từ kho tri thức kèm trích dẫn tên tài liệu |
| Tra trạng thái đơn | Câu hỏi có mã đơn dạng `XX` + số (ví dụ `DH1001`) hoặc từ khóa đơn hàng | Chỉ khách đăng nhập, chỉ đơn của chính khách; đơn của người khác không phân biệt được với đơn không tồn tại |
| Hỏi phí ship | Câu hỏi phí vận chuyển + tên tỉnh/thành | Chỉ báo cần địa chỉ/kiện hàng đầy đủ và chuyển sang checkout; không bịa quote từ dữ liệu thiếu |
| Gợi ý mua kèm trong chat | Câu hỏi “mua kèm” + SKU trong tin nhắn | Thẻ sản phẩm từ placement `cart` của Analytics |
| Xem gợi ý ngoài chat | `GET /api/v1/ai/recommendations?placement=home\|product_detail\|cart` | Danh sách sản phẩm đã resolve về Catalog (chi tiết [chương 11](11_recommendation_consumption.md)) |
| Feedback | `POST /api/v1/ai/shopping/feedback` với signal `helpful / not_helpful / clicked / dismissed` | Ghi vào `recommendation_feedback` (hoặc bộ nhớ ở chế độ fixture) |
| Đồng ý lưu lịch sử chat | `GET/PUT /api/v1/ai/consent` (chỉ khách đăng nhập) | Xem consent lưu lịch sử chat của MS-19 — khác consent cá nhân hóa của Analytics |

## Vòng đời phiên và retention

- Phiên được cấp `expires_at` ngay lúc tạo: **khách đăng nhập 30 ngày, khách vãng lai 15 ngày** (quyết định 09/11/2026; cấu hình `AI_RETENTION_CUSTOMER_DAYS` / `AI_RETENTION_GUEST_DAYS`).
- Runtime production tự chạy cleanup theo `AI_CLEANUP_INTERVAL_SECONDS` (mặc định mỗi giờ); lệnh `python -m ai_agent_service cleanup` vẫn dùng được cho thao tác vận hành thủ công.
- Quyền sở hữu: phiên `customer` chỉ truy cập được bởi đúng `customer_id`; phiên `guest` cần đúng token riêng của phiên. Sai chủ trả **404**, thiếu/sai guest token trả **401** mà không cấp dữ liệu.

## Consent lưu lịch sử chat — cơ chế thực thi

Consent mặc định là **chưa đồng ý**. Với khách đăng nhập chưa có consent:

- Phiên vẫn hoạt động đầy đủ (slot làm việc vẫn lưu trong `conversation_contexts` để workflow nhớ ngữ cảnh trong phiên).
- Tin nhắn **không** được ghi xuống store: workflow chạy qua một store proxy chặn riêng thao tác ghi/đọc tin nhắn (`_EphemeralMessages`), nên lịch sử không tồn tại để rò rỉ hay dùng lại — đây là chặn ở tầng lưu trữ, không phải ẩn ở tầng hiển thị.

Khi khách bật consent (`PUT /api/v1/ai/consent`), các lượt từ thời điểm đó mới bắt đầu được lưu. Bản ghi consent nằm ở `consent_records` với số phiên bản tăng mỗi lần đổi.

## Slot làm việc

Mỗi phiên giữ một bộ slot nhu cầu (`budget_vnd`, `recipient`, `occasion`, `tastes`, bộ đếm `_clarifications`) trong `conversation_contexts.slots` (JSONB). Slot là dữ liệu làm việc của workflow, tách khỏi lịch sử tin nhắn, nên vẫn tồn tại khi khách chưa consent lưu lịch sử. Chi tiết trích xuất và kiểm hợp lệ ở [chương 09](09_shopping_workflow_design.md).

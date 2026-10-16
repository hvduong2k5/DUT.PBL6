# ADR-004 — Duyệt hủy tại Order và sự kiện hoàn tiền thủ công

Ngày: 15/10/2026. Quyết định: accepted cho đợt giao đầu.

## Quyết định

Manager duyệt/từ chối một cancellation request qua API có quyền và audit. Quyết định đầu tiên bất biến; retry cùng nội dung đọc lại kết quả, đổi quyết định trả conflict. Duyệt chỉ tạo intent/hold, không xác nhận việc đóng gói đã dừng. Worker vẫn phải thắng CancelTask và nhận kết quả đảo kho trước khi hủy terminal. Inventory simulator truy vấn approval từ Order qua API nội bộ có xác thực; không tin chuỗi approval ID tùy ý. Capability này chưa thay thế proof protocol của Inventory production.

Tiền được hoàn thủ công phát `vn.omama.order.manual_refund_verified.v1` trên `order.manual_refunds.v1`. Payload chỉ có định danh, số tiền/VND, cách thực hiện và hash bằng chứng; không mang tài khoản ngân hàng hay PII. Chỉ phát sau người thứ hai đối chiếu giao dịch ngân hàng. Sự kiện và trạng thái SUCCEEDED lưu cùng transaction, retry không phát lại một effect mới.

## Lý do

Baseline v3 refund_succeeded đòi provider proof mà manual transfer không có. Không bịa proof hoặc nới lỏng schema cũ. Một fact riêng cho biết rõ mức bằng chứng là xác nhận vận hành; Finance/Notification phải chủ động đăng ký consumer và xử lý idempotent trước khi cần phản ứng theo fact này.

## Trade-off và rollout

Thêm topic/schema và consumer rollout. Runtime Order hiện vẫn phát sự kiện v2 cho các luồng cũ; không tự tuyên bố đã nâng toàn bộ runtime lên 3.2/v3. Không có consumer không làm mất nghĩa vụ hoàn tiền vì ledger/audit vẫn ở PostgreSQL, nhưng thông báo/kế toán tự động chưa được nghiệm thu.

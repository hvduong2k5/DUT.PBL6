# MS-14 · Lập lịch và thực thi thủ công

Mỗi kênh của revision APPROVED có tối đa một execution. Lệnh schedule dùng lịch của Channel Plan đã duyệt. Campaign revision tiếp tục biểu thị quyết định duyệt. Execution có state và version riêng để các kênh tiến triển độc lập.

| Lệnh | Chuyển trạng thái | Điều kiện |
| --- | --- | --- |
| Schedule | mới → SCHEDULED | Revision hiện hành, đã duyệt, kênh thuộc snapshot, kỳ kênh chưa kết thúc |
| Start | SCHEDULED hoặc PAUSED → ACTIVE | Trong kỳ kênh, operator xác nhận readiness bằng evidenceRef và actor có quyền thực thi |
| Pause | ACTIVE → PAUSED | Lý do bắt buộc |
| Complete | ACTIVE → COMPLETED | Tham chiếu bằng chứng bắt buộc |
| Cancel | SCHEDULED, ACTIVE hoặc PAUSED → CANCELLED | Lý do bắt buộc |

Operator thực hiện kiểm tra thủ công trước Start, bao gồm Product, Article, Promotion, Media và consent khi kênh cần. evidenceRef là mã hồ sơ nội bộ, chỉ nhận ký tự ASCII chữ/số và . _ : / -. Lệnh chỉ ghi nhận hành động thủ công. Worker, gọi provider và tự động chuyển trạng thái theo đồng hồ thuộc phần tiếp theo.

Bảng campaign_executions lưu revision, approval request, kênh, cửa sổ thời gian, trạng thái và version. Bảng campaign_execution_history lưu actor, request ID, chuyển trạng thái, lý do và evidence. Mỗi command lấy lock Campaign trước, ghi execution, lịch sử và idempotency trong cùng transaction. Schedule và action dùng expectedVersion; schedule kiểm version của revision, action kiểm version của execution. Khi Campaign còn execution chưa kết thúc, tạo revision mới trả 409. Quyền đọc dùng MARKETING_CAMPAIGN_VIEW. Lệnh dùng MARKETING_EXECUTION_MANAGE theo DEPARTMENT hoặc GLOBAL.

Admin API draft nằm dưới revision, gồm GET/POST executions và POST executions/{executionId}/actions. Identity BFF chuyển tiếp cùng suffix. Nghiệm thu tập trung vào schedule lặp, thao tác ngoài kỳ, thay đổi đồng thời, trạng thái theo từng kênh và audit.

### Đọc lịch sử execution

GET executions/{executionId}/history dùng cùng scope với danh sách execution. API trả actor, request ID, trạng thái trước/sau, thời điểm, lý do và mã bằng chứng. UI dùng afterVersion để tải trang tiếp theo theo thứ tự ghi. PostgreSQL đọc execution và lịch sử trong cùng snapshot để total và cursor nhất quán. Scope ngoài phòng ban và execution không thuộc revision đều trả 404. Mã bằng chứng là tham chiếu nội bộ, UI hiển thị dạng văn bản.

### Kiểm chứng ở lớp HTTP

Test tích hợp chạy Spring MVC và PostgreSQL 16, với Identity fixture cấp grant theo phòng ban. Test gọi các route execution từ schedule đến complete, kiểm replay cùng key, trùng lịch bằng key khác, thiếu evidence, version cũ và phân trang history. Đọc history từ phòng ban khác trả 404. Nghiệm thu qua Kong và Identity BFF với phiên thật là bước kiểm riêng.

### Thu hồi phê duyệt

Executive có MARKETING_PLAN_APPROVE theo scope có thể thu hồi approval của revision hiện hành với expectedVersion và lý do. Quyết định APPROVED lịch sử được giữ, còn bản ghi thu hồi riêng là nguồn xác định approval đã mất hiệu lực. GET revision trả bản ghi này trong approvalRequest.revocation. Thu hồi tăng version revision và chặn mọi schedule mới.

Trong cùng transaction, SCHEDULED chuyển CANCELLED vì chưa bắt đầu. ACTIVE và PAUSED chuyển STOP_PENDING vì cần operator xác nhận tình trạng bên ngoài. Operator dùng CONFIRM_STOP với evidenceRef để chuyển CANCELLED. START bị chặn sau thu hồi. Execution đã hoàn tất giữ nguyên. Cùng một khóa hàng Campaign tuần tự hóa schedule, action và revoke. Revision kế tiếp bị chặn khi còn STOP_PENDING.

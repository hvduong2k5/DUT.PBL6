# MS-14 · Lập lịch và thực thi thủ công

Mỗi kênh của revision APPROVED có tối đa một execution. Lệnh schedule dùng lịch của Channel Plan đã duyệt. Campaign revision tiếp tục biểu thị quyết định duyệt. Execution có state và version riêng để các kênh tiến triển độc lập.

| Lệnh | Chuyển trạng thái | Điều kiện |
| --- | --- | --- |
| Schedule | mới → SCHEDULED | Revision hiện hành, đã duyệt, kênh thuộc snapshot, kỳ kênh chưa kết thúc |
| Start | SCHEDULED hoặc PAUSED → ACTIVE | Trong kỳ kênh, operator xác nhận readiness bằng evidenceRef và actor có quyền thực thi |
| Pause | ACTIVE → PAUSED | Lý do bắt buộc |
| Complete | ACTIVE → COMPLETED | Tham chiếu bằng chứng bắt buộc |
| Cancel | SCHEDULED, ACTIVE hoặc PAUSED → CANCELLED | Lý do bắt buộc |

Operator thực hiện kiểm tra thủ công trước Start, bao gồm Product, Article, Promotion, Media và consent khi kênh cần. evidenceRef là mã hồ sơ nội bộ, chỉ nhận ký tự ASCII chữ/số và . _ : / -. Lệnh chỉ ghi nhận hành động thủ công. Worker, gọi provider, tự động chuyển trạng thái theo đồng hồ và thu hồi approval xuyên kênh thuộc phần tiếp theo.

Bảng campaign_executions lưu revision, approval request, kênh, cửa sổ thời gian, trạng thái và version. Bảng campaign_execution_history lưu actor, request ID, chuyển trạng thái, lý do và evidence. Mỗi command lấy lock Campaign trước, ghi execution, lịch sử và idempotency trong cùng transaction. Schedule và action dùng expectedVersion; schedule kiểm version của revision, action kiểm version của execution. Khi Campaign còn execution chưa kết thúc, tạo revision mới trả 409. Quyền đọc dùng MARKETING_CAMPAIGN_VIEW. Lệnh dùng MARKETING_EXECUTION_MANAGE theo DEPARTMENT hoặc GLOBAL.

Admin API draft nằm dưới revision, gồm GET/POST executions và POST executions/{executionId}/actions. Identity BFF chuyển tiếp cùng suffix. Nghiệm thu tập trung vào schedule lặp, thao tác ngoài kỳ, thay đổi đồng thời, trạng thái theo từng kênh và audit.

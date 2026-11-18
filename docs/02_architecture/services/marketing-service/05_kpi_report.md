# MS-14 · Báo cáo KPI target–actual

Endpoint GET dưới revision dùng MARKETING_KPI_VIEW theo phạm vi Campaign. Chỉ revision đã được phê duyệt có báo cáo. Kế hoạch được lấy từ snapshot duyệt, kể cả khi approval đã bị thu hồi. Mỗi kênh × ngày địa phương giao với kế hoạch kênh là một ô cần số liệu. Tính cả kênh chưa lên lịch để dữ liệu thiếu không bị che đi. Ngày hiện hành và ngày tương lai chưa được đưa vào actual. Mỗi ngày đã khép lại lấy version thủ công mới nhất của execution. Snapshot đọc nhất quán trong một transaction.

Coverage NOT_SYNCED khi chưa có ô báo cáo, PARTIAL khi đã có một phần và COMPLETE khi đủ tất cả ngày của kế hoạch. Actual trả tổng quan sát được, hoặc null khi không có số liệu. Chỉ tính variance (actual − target) và attainmentPercent (actual / target × 100, làm tròn 2 chữ số) khi coverage COMPLETE. Target 0 có variance nhưng attainmentPercent là null. Chỉ đối chiếu CLICKS và IMPRESSIONS với unit count. Target khác được giữ và đánh dấu UNSUPPORTED. Các số KPI và tỷ lệ trả dưới dạng chuỗi thập phân để giữ độ chính xác.

Báo cáo cộng số lần theo kênh. Provider sync, dedup người dùng, hiệu chỉnh định nghĩa từ nguồn khác và attribution thuộc các contract tiếp theo.

## Nghiệm thu qua Kong và Identity

Ngày 2026-11-17, Muse chạy task MS14-KPI-REPORT-233d568 trên stack Docker cô lập tại commit 233d568. Cả 8 ca đều đạt qua Kong và Identity BFF bằng phiên Keycloak thật. Campaign, submit và approve đi qua API. Dữ liệu execution và KPI cho những ngày đã kết thúc được seed trực tiếp vào PostgreSQL test.

- Revision chưa duyệt trả 409 và nhân viên sai DEPARTMENT nhận 404.
- Coverage đi qua NOT_SYNCED, PARTIAL và COMPLETE. Chỉ dùng sourceVersion mới nhất và bỏ số liệu ngày hiện hành theo Asia/Ho_Chi_Minh.
- Khi đủ dữ liệu, 8 clicks trên target 10 trả variance -2 và attainmentPercent 80.00. Target impressions bằng 0 trả variance 25 và attainmentPercent null.

Driver PowerShell tự động dừng tiến triển khi tạo Campaign. Cùng request chạy thủ công bằng curl trả 201, và các bước tiếp theo hoàn tất bằng curl với cookie jar. Nguyên nhân driver chưa được xác định. Dùng HTTP method viết hoa khi chạy lại.

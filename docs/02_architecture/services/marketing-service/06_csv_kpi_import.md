# MS-14 · Nhập KPI từ CSV

Pilot nhận file xuất đã chuẩn hóa, mỗi dòng gồm external_execution_key,date,source_version,impressions,clicks. Key định danh duy nhất trong pilot và cần bao gồm mã tài khoản nếu nguồn dùng ID cục bộ. Operator chọn execution đích. Lần nhập đầu thành công khóa mapping hai chiều. Một execution đã ghi manual giữ nguồn manual. Một execution đã bind CSV chỉ nhận CSV với key cũ.

PUT metrics/csv nhận text/csv UTF-8 với header đúng, 1–31 dòng, một key, mỗi ngày duy nhất, LF hoặc CRLF. Ngày theo timezone của execution, chỉ nhận ngày đã kết thúc và nằm trong kỳ. Số liệu là tổng ngày. Cùng version và giá trị được replay, version cao hơn là correction, version thấp hơn hoặc trùng version khác giá trị trả 409. Execution phải từng bắt đầu. Khóa Campaign tuần tự hóa với ghi thủ công. Mapping và toàn bộ dòng commit cùng transaction. Báo cáo hiện có đọc cùng bảng snapshot, giữ quy tắc coverage và version.

Giới hạn này dùng CSV cột đơn giản với key ASCII chữ/số và . _ : -. File từ provider phải được chuẩn hóa trước. Adapter tải file tự động và mapping nhiều provider là phần triển khai sau.

### Tra cứu nguồn và chẩn đoán xung đột

GET metrics/source trả UNBOUND khi chưa nhập, MANUAL khi execution đã ghi số liệu thủ công, hoặc CSV kèm externalExecutionKey. Người có MARKETING_KPI_VIEW đúng phạm vi mới xem được. PostgreSQL đọc binding và số liệu trong cùng snapshot. Lỗi ghi 409 phân biệt METRIC_SOURCE_CONFLICT, CSV_MAPPING_CONFLICT, METRIC_VERSION_CONFLICT và EXECUTION_NOT_STARTED. Lỗi mapping không tiết lộ execution đang sở hữu key ở nơi khác.

## Nghiệm thu CSV qua Kong và Identity

Ngày 2026-11-17, Muse chạy MS14-CSV-KPI-1f5b9ea trên commit 1f5b9ea trong stack Docker cô lập. PUT metrics/csv đi qua Kong và Identity BFF bằng phiên staff. Kết quả 11/11 ca pass.

- Nhập lần đầu và replay giữ đúng số dòng. Bản đính chính được chọn trong GET metrics và KPI report.
- Khi dòng sau xung đột version, dòng trước và toàn bộ lô rollback. Mã key xung đột và việc trộn nguồn manual/CSV bị chặn.
- Ngày hiện tại, ngày trùng, số thập phân và header sai bị từ chối. Staff sai phòng ban không thể nhập.
- Đối soát PostgreSQL xác nhận mapping và các bản ghi khớp các lô thành công.

Kết quả trên áp dụng cho CSV import tại 1f5b9ea. API tra cứu nguồn và mã lỗi chi tiết được nghiệm thu riêng.


## Demo backend qua API ngày 2026-11-18

Dùng stack Docker cô lập với Kong, Identity BFF, Keycloak và PostgreSQL. Campaign mẫu có một kênh Email và mục tiêu 100 lượt click. Tạo campaign, gửi duyệt và duyệt qua API với hai tài khoản staff. Trạng thái lần lượt là DRAFT, IN_REVIEW và APPROVED. Seed lịch sử execution ACTIVE cho hai ngày đã khép lại để demo ngay.

- GET kpi-report trước khi nhập trả NOT_SYNCED.
- PUT metrics/csv bản đầu trả 200. Báo cáo trả COMPLETE, 80 lượt click và mức hoàn thành 80%.
- PUT metrics/csv bản đính chính trả 200. Báo cáo trả COMPLETE, 110 lượt click và mức hoàn thành 110%.

Lượt demo dùng image từ lát CSV 1f5b9ea và dữ liệu giả lập. API tra cứu nguồn, runner định kỳ và Web Admin nằm ngoài phạm vi của lượt chạy này.

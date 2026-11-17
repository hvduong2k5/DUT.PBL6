# MS-14 · Nhập KPI từ CSV

Pilot nhận file xuất đã chuẩn hóa, mỗi dòng gồm external_execution_key,date,source_version,impressions,clicks. Key định danh duy nhất trong pilot và cần bao gồm mã tài khoản nếu nguồn dùng ID cục bộ. Operator chọn execution đích. Lần nhập đầu thành công khóa mapping hai chiều. Một execution đã ghi manual giữ nguồn manual. Một execution đã bind CSV chỉ nhận CSV với key cũ.

PUT metrics/csv nhận text/csv UTF-8 với header đúng, 1–31 dòng, một key, mỗi ngày duy nhất, LF hoặc CRLF. Ngày theo timezone của execution, chỉ nhận ngày đã kết thúc và nằm trong kỳ. Số liệu là tổng ngày. Cùng version và giá trị được replay, version cao hơn là correction, version thấp hơn hoặc trùng version khác giá trị trả 409. Execution phải từng bắt đầu. Khóa Campaign tuần tự hóa với ghi thủ công. Mapping và toàn bộ dòng commit cùng transaction. Báo cáo hiện có đọc cùng bảng snapshot, giữ quy tắc coverage và version.

Giới hạn này dùng CSV cột đơn giản với key ASCII chữ/số và . _ : -. File từ provider phải được chuẩn hóa trước. Adapter tải file tự động và mapping nhiều provider là phần triển khai sau.

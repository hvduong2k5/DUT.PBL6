# MS-14 · Đồng bộ KPI từ file export định kỳ

Pilot dùng export theo ngày của một tài khoản quảng cáo. Job phía nguồn tạo CSV UTF-8 với bốn cột campaign_id,date,impressions,clicks và metadata batch gồm timezone, phiên bản export và tài khoản. Phiên bản export phải tăng khi nguồn đính chính. Dùng timezone đúng như execution. Khi hai timezone khác nhau, yêu cầu nguồn xuất lại theo đúng timezone vì dữ liệu ngày không đủ để đổi ranh giới ngày.

Script normalize_kpi_export.py lọc campaign được chọn, tạo key theo nguồn, tài khoản và campaign ngoài, rồi xuất định dạng CSV mà MS-14 đang nhận. Script giữ tổng tuyệt đối của ngày, phát hiện ngày trùng trong campaign, chặn số không hợp lệ và chia đầu vào theo lô tối đa 31 ngày bằng cách yêu cầu job nguồn cấp file theo kỳ. Job có thể gọi script bằng lịch riêng và chuyển kết quả vào PUT metrics/csv qua Kong. Mỗi lô giữ nguyên version và bytes khi retry. Lần nhập thử đầu tiên do operator xác nhận execution đích. Lần chạy sau kiểm GET metrics/source để xác nhận mapping trước khi ghi.

Ở lát tiếp theo, cần thêm runner có danh tính IAM được cấp quyền theo DEPARTMENT và lịch do môi trường quản lý. Runner ghi mã batch và trạng thái theo từng execution, giữ checkpoint sau khi được 200, retry lỗi mạng/503 với backoff, và ghi nhận lỗi 409/422 để đối soát. Chạy lại file đã thành công dùng cùng source_version. Phiên bản thay thế phải do nguồn công bố. Khi xử lý nhiều bản export, runner chạy theo thứ tự version của từng execution. Tải export qua API provider và thông tin xác thực của tài khoản sẽ được triển khai khi có nguồn thật.

Ví dụ chạy cho export đã chuẩn hóa từ tài khoản demo. Trên Windows, dùng Python có dữ liệu timezone IANA và ghi stdout ra file bằng PowerShell 7.

```powershell
python services/marketing-service/scripts/normalize_kpi_export.py export.csv --source ads --account demo --campaign spring --version 2 --export-timezone Asia/Ho_Chi_Minh --execution-timezone Asia/Ho_Chi_Minh > ms14-kpi.csv
```

Chỉ gửi file sang API sau khi script thoát với mã 0.

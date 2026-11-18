# MS-14 · Đồng bộ KPI từ file export định kỳ

Pilot dùng export theo ngày của một tài khoản quảng cáo. Job phía nguồn tạo CSV UTF-8 với bốn cột campaign_id,date,impressions,clicks và metadata batch gồm timezone, phiên bản export và tài khoản. Phiên bản export phải tăng khi nguồn đính chính. Dùng timezone đúng như execution. Khi hai timezone khác nhau, yêu cầu nguồn xuất lại theo đúng timezone vì dữ liệu ngày không đủ để đổi ranh giới ngày.

Script normalize_kpi_export.py lọc campaign được chọn, tạo key theo nguồn, tài khoản và campaign ngoài, rồi xuất định dạng CSV mà MS-14 đang nhận. Script giữ tổng tuyệt đối của ngày, phát hiện ngày trùng trong campaign, chặn số không hợp lệ và chia đầu vào theo lô tối đa 31 ngày bằng cách yêu cầu job nguồn cấp file theo kỳ. Job có thể gọi script bằng lịch riêng và chuyển kết quả vào PUT metrics/csv qua Kong. Mỗi lô giữ nguyên version và bytes khi retry. Lần nhập thử đầu tiên do operator xác nhận execution đích. Lần chạy sau kiểm GET metrics/source để xác nhận mapping trước khi ghi.

Ở lát tiếp theo, cần thêm runner có danh tính IAM được cấp quyền theo DEPARTMENT và lịch do môi trường quản lý. Runner ghi mã batch và trạng thái theo từng execution, giữ checkpoint sau khi được 200, retry lỗi mạng/503 với backoff, và ghi nhận lỗi 409/422 để đối soát. Chạy lại file đã thành công dùng cùng source_version. Phiên bản thay thế phải do nguồn công bố. Khi xử lý nhiều bản export, runner chạy theo thứ tự version của từng execution. Tải export qua API provider và thông tin xác thực của tài khoản sẽ được triển khai khi có nguồn thật.

Ví dụ chạy cho export đã chuẩn hóa từ tài khoản demo. Trên Windows, dùng Python có dữ liệu timezone IANA và ghi stdout ra file bằng PowerShell 7.

```powershell
python services/marketing-service/scripts/normalize_kpi_export.py export.csv --source ads --account demo --campaign spring --version 2 --export-timezone Asia/Ho_Chi_Minh --execution-timezone Asia/Ho_Chi_Minh > ms14-kpi.csv
```

Chỉ gửi file sang API sau khi script thoát với mã 0.

## Chạy một lô qua Kong

Runner sync_kpi_export.py dùng bearer staff do môi trường cấp qua MARKETING_KPI_BEARER_TOKEN. Cần cả MARKETING_KPI_VIEW để kiểm nguồn và MARKETING_CHANNEL_MANAGE để nhập, theo scope execution. Token dùng trong bộ nhớ. Runner dùng route bearer của Marketing qua Kong, chỉ chấp nhận HTTPS hoặc HTTP loopback và từ chối redirect. Runner yêu cầu operator đã tạo mapping CSV đúng trước lần chạy định kỳ đầu tiên.

```powershell
python services/marketing-service/scripts/sync_kpi_export.py ms14-kpi.csv --gateway http://127.0.0.1:8000 --checkpoint kpi-sync.sqlite --campaign <uuid> --revision <uuid> --execution <uuid>
```

Runner giữ nguyên bytes trong tối đa ba lần thử khi gặp lỗi kết nối hoặc HTTP 502/503/504. Các lỗi còn lại trả mã thoát 2 với status và mã lỗi an toàn. Runner lưu checkpoint SQLite theo origin, Campaign, revision, execution và phiên bản export. Chỉ lưu IMPORTED khi HTTP 200 trả đúng các snapshot đã gửi. Replay cùng version và SHA-256 trả ALREADY_IMPORTED. Cùng version khác bytes và version cũ chưa nhập đều bị chặn. Lỗi mapping, lỗi 409/422 và lỗi mạng hết lượt retry lưu FAILED để chạy lại đúng lô hoặc đối soát. Export sửa bytes phải dùng version mới. Scheduler bên ngoài cấp token mới và cung cấp file theo thứ tự version. Provider API và scheduler cụ thể vẫn cần tích hợp riêng.

Giữ file checkpoint trên ổ cục bộ bền vững, ngoài thư mục Git, và dùng cùng đường dẫn qua mọi lần chạy cho cùng gateway. SQLite cấp một khóa ghi cho mỗi file trong lúc gọi API, vì vậy scheduler chạy một runner tại một thời điểm trên mỗi checkpoint. Nếu process dừng sau khi Kong đã ghi nhưng trước khi SQLite commit, chạy lại đúng file. Marketing API sẽ trả snapshot hiện tại mà không ghi trùng. Các lần chạy FAILED không làm tăng version đã nhập thành công.

Ca nghiệm thu cho Muse gồm lô đầu thành công, replay không gọi lại PUT, cùng version khác bytes, file cũ đến muộn, lỗi mapping được lưu FAILED, lỗi 503 thử lại cùng bytes, và mô phỏng dừng sau khi server ghi nhưng trước khi lưu checkpoint. Kiểm tra SQLite và số version trong PostgreSQL test ở từng ca.

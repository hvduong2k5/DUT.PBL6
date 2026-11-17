# MS-14 · KPI theo execution

Pilot hỗ trợ một nguồn MANUAL và một bản ghi mỗi ngày địa phương của execution. Ngày theo timezone của revision. Chỉ ghi ngày giao với cửa sổ execution, tối đa tới ngày hiện tại. Một execution phải từng bắt đầu để nhận dữ liệu. Số liệu là tổng tuyệt đối của ngày, bao gồm impressions và clicks, dùng số nguyên không âm. Hai số 0 là dữ liệu đã báo cáo.

PUT theo execution và ngày nhận sourceVersion dương. Bản gửi cùng version và cùng giá trị trả kết quả hiện tại. Version thấp hơn hoặc cùng version nhưng khác giá trị trả 409. Version cao hơn tạo bản ghi mới và giữ bản cũ để đối soát. Transaction khóa Campaign trước khi ghi. Mỗi phiên bản giữ actor, request ID và thời điểm nhập. Pilot chỉ ghi số liệu test hoặc đã được đối chiếu thủ công.

GET trả từng ngày theo khoảng from/to bao gồm hai đầu mút, tối đa 31 ngày và nằm trong kỳ execution. Ngày thiếu có status MISSING và hai giá trị null. Coverage là NOT_SYNCED khi chưa có dòng, PARTIAL khi thiếu ngày và COMPLETE khi đủ ngày. Ngày hiện tại có thể được đính chính sau. Quyền MARKETING_KPI_VIEW và MARKETING_CHANNEL_MANAGE áp dụng theo scope phòng ban. API đi qua Identity BFF.

Mô hình này tách số liệu execution với KPI target của kế hoạch. Nguồn provider, mapping tự động, công thức target–actual và Finance cần contract riêng trước khi nối.

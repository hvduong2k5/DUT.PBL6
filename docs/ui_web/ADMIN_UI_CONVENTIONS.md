# Quy chuẩn giao diện Web Admin

**Cập nhật:** 2026-10-03
**Ứng dụng:** `apps/web-admin`

Tài liệu này ghi lại quy chuẩn chính đã chốt cho giao diện nhân viên nội bộ. Epic và Product Backlog xác định phạm vi nghiệp vụ; `UI_SCREEN_SPECIFICATION_FOR_STITCH_AI.md` chỉ được dùng để tham khảo cách tổ chức màn hình. Khi cách trình bày khác nhau, quy chuẩn trong tài liệu này được ưu tiên cho Web Admin.

## 1. Kiểm soát phạm vi tính năng

- Trước khi triển khai bất kỳ màn hình, trường dữ liệu, trạng thái hoặc hành động nào, phải kiểm tra Epic liên quan và truy được về User Story, Acceptance Criteria, Business Rule hoặc Functional Requirement cụ thể.
- `UI_SCREEN_SPECIFICATION_FOR_STITCH_AI.md`, bản thiết kế hoặc mockup không phải nguồn để tự bổ sung nghiệp vụ. Chúng chỉ hướng dẫn bố cục cho những khả năng đã được Epic xác nhận.
- Nếu một nội dung chỉ xuất hiện trong UI specification nhưng không có trong Epic, không triển khai như tính năng đã chốt; ghi nhận thành đề xuất hoặc câu hỏi cần Product Owner xác nhận.
- Nội dung nằm trong mục “điểm cần chốt”, “OPEN”, giả định hoặc phụ thuộc Epic khác không được tự suy diễn thành workflow, trạng thái, permission hay API hoàn chỉnh.
- Tài liệu scope/data matrix của mỗi màn phải ghi Epic, User Story và tiêu chí nguồn. Việc phụ thuộc một Epic khác không đồng nghĩa màn hiện tại sở hữu toàn bộ chức năng của Epic đó.
- Khi Epic và UI specification mâu thuẫn, Epic/Product Backlog được ưu tiên cho phạm vi; quy chuẩn UI chung được ưu tiên cho cách trình bày.

## 2. Ranh giới ứng dụng

- Web Admin là ứng dụng riêng trong `apps/web-admin`; không ghép vai trò nhân viên vào `apps/web-user`.
- Nhân viên dùng một loại tài khoản nội bộ. `Role` và `Permission` là hai khái niệm riêng: Role gom nhiều Permission, còn giao diện và BFF kiểm tra Permission nguyên tử tại thời điểm xem hoặc thao tác.
- Không tách role chỉ để biểu diễn cấp quản lý và nhân viên thường trong cùng nghiệp vụ. Ví dụ cùng dùng role `PACKING`; quyền vào trang quản lý, Workbench và phạm vi `ALL`/`ASSIGNED_ONLY` do Permission + Scope quyết định.
- Mã role là dữ liệu do Access Management cung cấp và không được dùng làm union đóng trong frontend. Permission mà UI/BFF thực sự hỗ trợ vẫn được khai báo rõ để kiểm tra authorization.
- Profile Mockoon chỉ là bộ dữ liệu thử Permission/Scope, không phải role thật. Tên profile được chọn bằng `ADMIN_MOCK_PROFILE` trong môi trường local.
- Sidebar chỉ hiển thị module mà phiên hiện tại có quyền xem. Việc ẩn menu/nút không thay thế kiểm tra authorization tại BFF/API.
- Scope dữ liệu như `ALL` hoặc `ASSIGNED_ONLY` đi cùng permission và phải được upstream thật áp dụng khi truy vấn.

## 3. Hệ thống hình ảnh

- Dùng cùng ngôn ngữ màu của Web Customer: nền kem `#fcf9f4`, nâu chính `#563a26`, vàng nhấn `#d4a24c`, đỏ gạch `#8f2817` và các sắc độ trung tính tương ứng.
- Nút chính, nút phụ, trạng thái focus, lỗi và cảnh báo giữ cùng ý nghĩa thị giác với `web-user`.
- Màu không được là dấu hiệu duy nhất của trạng thái; luôn có nhãn chữ hoặc biểu tượng kèm mô tả.

## 4. Bố cục chung

- Header trên cùng chứa thương hiệu, tìm kiếm nhanh, thông báo và thông tin nhân viên.
- Sidebar trái gom chức năng theo nhóm và lọc theo permission.
- Phần nội dung bắt đầu bằng page header: breadcrumb/tên/mô tả ở trái; nút hành động theo ngữ cảnh ở phải.
- Sidebar chuyển thành menu đóng/mở trên màn hình hẹp; bảng cho phép cuộn ngang thay vì ép mất cột.

## 5. Mẫu trang quản lý

Mọi trang quản lý danh sách như danh mục, sản phẩm, người dùng hoặc yêu cầu nghiệp vụ dùng cùng thứ tự:

1. Page header và các hành động toàn trang như `Thêm...`, `Tạo...`, `Làm mới`.
2. Cụm KPI card nằm ngay dưới page header và trước khu vực danh sách. Thông thường dùng 3–4 chỉ số tổng quan quan trọng nhất của module; chỉ số phải truy được về Epic/API hoặc được upstream cung cấp, không tự suy diễn công thức nghiệp vụ ở frontend.
3. Thanh chọn trạng thái dạng tab, có số lượng khi dữ liệu hỗ trợ.
4. Thanh tìm kiếm và bộ lọc theo nghiệp vụ; luôn có thao tác đặt lại.
5. Bảng chỉ chứa các cột nhận diện/tóm tắt cần thiết, không nhồi toàn bộ chi tiết item.
6. Phân trang hoặc thông tin số bản ghi.

Cột cuối của bảng luôn là cột tùy chọn với nút `•••`. Nút mở dropdown các hành động mà nhân viên được phép thực hiện và trạng thái item cho phép.

## 6. Modal hành động

- Xem chi tiết, tạo, sửa, xóa, xác nhận và các thao tác nghiệp vụ đều thực hiện trong modal/popup trên trang hiện tại; không điều hướng sang màn thao tác riêng.
- Modal phải có tiêu đề, item đang tác động, nút đóng/hủy, loading, lỗi inline và chống gửi lặp khi đang xử lý.
- Thao tác gây ảnh hưởng cần xác nhận rõ hệ quả. Lý do bắt buộc với từ chối, hủy, ghi đè hoặc thay đổi nhạy cảm.
- Modal chi tiết được phép rộng hơn; vẫn đóng được bằng nút rõ ràng và click vùng nền.

## 7. Trạng thái bắt buộc

Mỗi trang cần chủ động thiết kế loading, empty, filtered-empty, API error, permission denied và action pending/error/success. Deep link hoặc request trực tiếp vẫn phải kiểm tra lại permission ở server.

## 8. Trang quản lý và Workbench

- Trang quản lý dành cho actor có quyền giám sát phạm vi `ALL` hoặc `TEAM`; dùng KPI, bộ lọc và bảng để theo dõi nhiều item. Permission mở trang quản lý phải tách khỏi permission làm việc cá nhân.
- Workbench là trang thực hiện quy trình nhiều bước của nhân viên, thường nằm trong nhóm `Công việc của tôi`. Workbench chỉ lấy item thuộc scope `ASSIGNED_ONLY`, trình bày hướng dẫn/dữ liệu cần thao tác và các bước nghiệp vụ theo Epic.
- Một nhân viên không có quyền quản lý không thấy menu quản lý và không được gọi API danh sách toàn bộ. Một lead không tự động có quyền thao tác thay nhân viên nếu chưa được cấp permission workbench và được phân công phù hợp.
- CRUD hoặc hành động đơn giản trên trang quản lý vẫn dùng modal. Quy trình nhiều bước có thể có Workbench riêng; các bước xác nhận có ảnh hưởng trong Workbench vẫn dùng popup xác nhận khi cần.
- Sidebar, page/BFF và upstream đều kiểm tra permission; BFF/upstream còn kiểm tra data scope và quan hệ phân công trên từng record.

## 9. Màn tham chiếu đầu tiên

`/b2b/quotes` là implementation tham chiếu ban đầu của quy chuẩn: header + sidebar, page header, status tabs, search/filter, bảng cơ bản, cột `•••`, dropdown theo quyền/trạng thái và toàn bộ hành động trong modal.

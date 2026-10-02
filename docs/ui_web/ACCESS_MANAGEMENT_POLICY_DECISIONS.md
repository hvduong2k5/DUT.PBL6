# Access Management Policy — Decision Register

**Cập nhật:** 2026-10-02  
**Nguồn chính:** `docs/01_requirements/epics/EPIC_22_Administration.md`  
**Phạm vi:** Employee System Access, Role, Permission, Role–Permission và Employee–Role Assignment  
**Trạng thái triển khai:** Tạm gác các mutation; giữ các màn hình hiện tại ở chế độ tra cứu/read-only

## 1. Mục đích

Tài liệu này ghi lại các quyết định còn thiếu trước khi triển khai tạo, cập nhật, khóa, archive hoặc gán/thu hồi quyền. Đây không phải API contract và không tự thay đổi Product Backlog.

Quy ước trạng thái:

- **Đã chốt:** đã được Epic 22 hoặc quyết định sản phẩm hiện tại xác nhận đủ rõ để triển khai.
- **Tạm áp dụng:** là ranh giới an toàn của giao diện hiện tại, có thể thay đổi sau quyết định chính thức.
- **Chưa chốt:** còn nhiều phương án hợp lệ; không triển khai mutation dựa trên suy đoán.

## 2. Mô hình phân quyền

| ID | Vấn đề cần quyết định | Các phương án chính | Đề xuất hiện tại | Trạng thái | Ảnh hưởng triển khai |
| --- | --- | --- | --- | --- | --- |
| `AUTH-01` | Quan hệ giữa Role và Permission | Gộp Role với Permission / tách hai khái niệm | Role là nhóm quyền; Permission là quyền nguyên tử | **Đã chốt** | Giữ hai danh mục và permission gate độc lập |
| `AUTH-02` | Cấp Permission trực tiếp cho Employee | Chỉ qua Role / cho phép Direct Permission | Chỉ cấp qua Employee–Role Assignment | **Tạm áp dụng** — Direct Permission không có trong backlog | Chưa tạo UI/API Direct Permission |
| `AUTH-03` | Một Employee có nhiều Role | Một Role / nhiều Role | Cho phép nhiều Role có nguồn và thời hạn rõ ràng | **Chưa chốt** — Epic có scenario nhiều Role nhưng Mục 8 vẫn đặt câu hỏi | Chặn mutation assignment đến khi thống nhất |
| `AUTH-04` | Hợp nhất quyền từ nhiều Role | Union / ưu tiên Role / chính sách khác | Union các Permission còn hiệu lực; giữ mọi nguồn cấp | **Tạm áp dụng** | Access Review hiện dùng cách này |
| `AUTH-05` | Role inheritance | Có kế thừa / không kế thừa | Không hỗ trợ trong MVP | **Chưa chốt** | Không thiết kế cây Role |
| `AUTH-06` | Explicit Deny | Có Deny / chỉ Allow | Không hỗ trợ trong MVP | **Chưa chốt** | Chưa có precedence Allow–Deny |
| `AUTH-07` | Taxonomy scope | `ALL`, đơn vị, kho, kênh, khu vực, assigned-only hoặc tổ hợp | Xây danh mục scope dùng chung trước khi mutation | **Chưa chốt** | Chặn editor Role/Permission/Assignment |
| `AUTH-08` | Permission và điều kiện nghiệp vụ động | Permission đủ để thao tác / vẫn kiểm tra trạng thái, assignment, hạn mức | Permission chỉ là điều kiện cần; Epic chuyên môn vẫn kiểm tra điều kiện động | **Đã chốt bởi Epic 22** | BFF/backend không được chỉ kiểm tra tên Role hoặc ẩn nút |

## 3. Employee System Access

| ID | Vấn đề cần quyết định | Các phương án chính | Đề xuất hiện tại | Trạng thái | Ảnh hưởng triển khai |
| --- | --- | --- | --- | --- | --- |
| `ACC-01` | Nguồn tạo Account | Phải có Employee Profile / tạo đồng thời | Bắt buộc chọn Employee Profile hợp lệ trước | **Chưa chốt** | Chặn popup tạo Account |
| `ACC-02` | Định danh đăng nhập duy nhất | Email / số điện thoại / username / nhiều loại | Dùng email công việc ở MVP, vẫn lưu loại định danh | **Chưa chốt** | Chặn validation và kiểm tra trùng |
| `ACC-03` | Trường dữ liệu được sửa tại Epic 22 | Cho sửa Profile / chỉ sửa System Access | Họ tên, phòng ban, hợp đồng, chức vụ thuộc Epic 02; Epic 22 chỉ tham chiếu | **Đã chốt bởi Epic 22** | Popup Account không được sửa Employee Profile |
| `ACC-04` | Luồng lời mời và kích hoạt | Link email, OTP, mật khẩu tạm hoặc SSO | Dùng lời mời một lần, có hạn; không hiển thị secret | **Chưa chốt** | Chặn create/resend invitation |
| `ACC-05` | MFA và reset credential | Bắt buộc/tùy chọn; kênh và recovery khác nhau | Dùng chính sách chung của Epic 01 | **Chưa chốt** | Chặn thao tác credential trong Admin |
| `ACC-06` | Bộ trạng thái Account | Giữ hoặc điều chỉnh `INVITED`, `PENDING_ACTIVATION`, `ACTIVE`, `LOCKED`, `DISABLED` | Giữ mô hình đề xuất của Epic 22 | **Tạm áp dụng** | UI hiện chỉ hiển thị các trạng thái này |
| `ACC-07` | Mở khóa/reactivate | Tự động, Admin thực hiện hoặc cần phê duyệt | Tách Unlock và Reactivate thành action có lý do | **Chưa chốt** | Chưa hiển thị action mở khóa |
| `ACC-08` | Khóa ngay hoặc theo lịch | Chỉ khóa ngay / hỗ trợ thời điểm hiệu lực | Hỗ trợ cả hai, bắt buộc lý do | **Đã chốt về kết quả; chưa chốt timezone/lịch chạy** | Chặn form khóa hoàn chỉnh |
| `ACC-09` | Phiên đang hoạt động khi khóa/thu hồi quyền | Thu hồi ngay / grace period theo action | Thu hồi ngay với quyền đặc biệt; timeout ngắn cho cache | **Chưa chốt** | Chặn mutation ảnh hưởng access |
| `ACC-10` | Bảo vệ Admin hợp lệ cuối cùng | Chỉ cảnh báo / chặn / phê duyệt đặc biệt | Backend chặn mặc định và yêu cầu quy trình đặc biệt | **Mục tiêu đã chốt; cơ chế chưa chốt** | Chặn lock/revoke Role đặc quyền |
| `ACC-11` | HR Manager | Thêm actor riêng / dùng permission-scoped Admin | Thêm actor và ma trận scope đơn vị rõ ràng | **Chưa chốt** | Chưa cấp mutation cho HR Manager |
| `ACC-12` | Cập nhật đồng thời | Last-write-wins / revision / ETag | Dùng `revision` hoặc ETag và trả `409` khi stale | **Kết quả chống ghi đè đã chốt; cơ chế chưa chốt** | Mutation phải có concurrency token |

## 4. Role

| ID | Vấn đề cần quyết định | Các phương án chính | Đề xuất hiện tại | Trạng thái | Ảnh hưởng triển khai |
| --- | --- | --- | --- | --- | --- |
| `ROLE-01` | Có cho tạo Custom Role | Chỉ Role hệ thống / cho Custom Role | Cho Custom Role; Role hệ thống do phiên bản ứng dụng quản lý | **Chưa chốt** | Chặn nút “Thêm Role” |
| `ROLE-02` | Danh sách Role hệ thống/đặc quyền | Cấu hình động / registry cố định có version | Registry cố định, có `system` và `privileged` riêng | **Chưa chốt** | Chặn policy bảo vệ chính xác |
| `ROLE-03` | Tính bất biến của `roleCode` | Cho đổi / không đổi | Không đổi sau khi tạo | **Chưa chốt** | Chặn form cập nhật code |
| `ROLE-04` | Quy tắc mã/tên duy nhất | Duy nhất toàn hệ thống / theo tenant hoặc scope | Code duy nhất toàn hệ thống; tên không phân biệt hoa thường | **Chưa chốt** | Chặn validation create/update |
| `ROLE-05` | Sửa Role đặc quyền | Chặn hoàn toàn / approval / break-glass | Approval bởi actor khác, không self-approve | **Chưa chốt** | Chặn editor Role đặc quyền |
| `ROLE-06` | Xóa Role có lịch sử | Xóa cứng / inactive / archive | Không xóa cứng; dùng `INACTIVE` rồi `ARCHIVED` | **Đã chốt bởi Epic 22** | Không tạo API `DELETE` xóa dữ liệu |
| `ROLE-07` | Role inactive đang còn assignment | Thu hồi ngay / giữ cấu hình nhưng mất hiệu lực / bắt buộc thay thế | Không còn tạo quyền hiệu lực; hiển thị impact và yêu cầu kế hoạch thay thế | **Chưa chốt** | Chặn action deactivate |
| `ROLE-08` | Clone Role | Cho clone / không hỗ trợ | Chỉ cân nhắc sau khi policy Role đặc quyền rõ ràng | **Chưa chốt** | Không có action clone |
| `ROLE-09` | Thay tập Permission của Role | Áp dụng ngay / theo lịch / approval | Preview Employee bị ảnh hưởng, concurrency check và Audit trước khi áp dụng | **Kết quả impact/Audit đã chốt; quy trình chưa chốt** | Chặn Permission matrix mutation |

## 5. Permission

| ID | Vấn đề cần quyết định | Các phương án chính | Đề xuất hiện tại | Trạng thái | Ảnh hưởng triển khai |
| --- | --- | --- | --- | --- | --- |
| `PERM-01` | Taxonomy mã Permission | Chuỗi tự do / `RESOURCE_ACTION[_SCOPE]` có registry | Registry có version gồm resource, action và scope | **Chưa chốt** | Chặn create/edit Permission |
| `PERM-02` | Custom Permission | Chỉ hệ thống định nghĩa / Admin được tạo | MVP chỉ dùng Permission hệ thống; chưa cho tạo Custom Permission | **Chưa chốt** | Trang Permission giữ read-only |
| `PERM-03` | Tính bất biến của `permissionCode` | Cho đổi / migration / bất biến | Bất biến; thay thế bằng code mới và deprecate code cũ | **Chưa chốt** | Không có trường sửa code |
| `PERM-04` | Permission đã được module thực thi | Cho sửa/xóa / bảo vệ | Permission hệ thống không được sửa hoặc xóa thủ công | **Mục tiêu bảo vệ đã chốt; danh sách code chưa chốt** | Chặn mutation Permission hệ thống |
| `PERM-05` | Vòng đời Permission | Active/Archived / Active/Deprecated/Archived | Dùng `ACTIVE → DEPRECATED → ARCHIVED`, có kiểm tra nơi sử dụng | **Chưa chốt** | UI hiện chỉ hiển thị trạng thái |
| `PERM-06` | Archive Permission đang thuộc Role | Tự gỡ / chặn / migration có kế hoạch | Chặn archive đến khi có replacement/migration | **Chưa chốt** | Chặn action archive |
| `PERM-07` | Thu hồi một nguồn quyền trùng | Xóa toàn bộ quyền / tính lại các nguồn còn lại | Chỉ mất quyền nếu không còn nguồn hợp lệ khác | **Đã chốt bởi Epic 22** | Backend phải tính effective permission theo nguồn |
| `PERM-08` | Quyền xem và quyền mutation | Quyền xem bao gồm thao tác / tách từng action | Tách `VIEW`, `CREATE`, `UPDATE`, `DELETE`, `APPROVE`, `EXPORT` | **Đã chốt bởi Epic 22** | UI và BFF kiểm tra từng Permission |

## 6. Assignment và quyền đặc biệt

| ID | Vấn đề cần quyết định | Các phương án chính | Đề xuất hiện tại | Trạng thái | Ảnh hưởng triển khai |
| --- | --- | --- | --- | --- | --- |
| `ASSIGN-01` | Ma trận ai được gán Role nào | Mọi System Admin / theo cấp và scope | Ma trận actor × Role × scope × action | **Chưa chốt** | Chặn popup gán/thu hồi Role |
| `ASSIGN-02` | Self-assignment và self-approval | Cho phép / cấm / giới hạn | Cấm với Role đặc quyền; thao tác thông thường cần policy riêng | **Chưa chốt** | Chặn mutation nhạy cảm |
| `ASSIGN-03` | Separation of Duties | Không kiểm tra / rule theo cặp Role | Registry các tổ hợp Role xung đột | **Chưa chốt** | Chưa thể validate assignment |
| `ASSIGN-04` | Approval quyền đặc biệt | Không approval / one-person / two-person | Người duyệt khác người yêu cầu; approval hết hạn | **Chưa chốt** | Chưa có workflow approval |
| `ASSIGN-05` | Assignment có thời hạn | Không thời hạn / tùy chọn / bắt buộc với Role đặc quyền | Hỗ trợ start/end; bắt buộc end với quyền tạm thời | **Khả năng có thời hạn đã chốt; chi tiết chưa chốt** | Chặn form thời gian hiệu lực |
| `ASSIGN-06` | Timezone và thời điểm hiệu lực | UTC / timezone đơn vị / timezone người dùng | Lưu UTC, hiển thị timezone tổ chức | **Chưa chốt** | Chặn scheduler chính xác |
| `ASSIGN-07` | Gán Role cho Account locked | Cấm / cho cấu hình nhưng chưa có access | Cho cập nhật cấu hình; Account vẫn không truy cập được | **Đã nêu trong Epic 22, cần xác nhận sản phẩm** | UI phải tách configured access và actual access |
| `ASSIGN-08` | Break-glass | Không có / tài khoản hoặc assignment khẩn cấp | Chỉ thiết kế khi có owner, expiry và Audit riêng | **Chưa chốt** | Không triển khai trong MVP hiện tại |
| `ASSIGN-09` | Lý do bắt buộc | Mọi mutation / chỉ mutation nhạy cảm | Bắt buộc khi khóa, thu hồi, giảm quyền, archive và override | **Chưa chốt** | Chặn validation dialog |

## 7. Audit, review, privacy và notification

| ID | Vấn đề cần quyết định | Các phương án chính | Đề xuất hiện tại | Trạng thái | Ảnh hưởng triển khai |
| --- | --- | --- | --- | --- | --- |
| `GOV-01` | Nội dung Audit tối thiểu | Event ngắn / actor, target, before-after, scope, reason | Ghi actor, action, target, before/after, scope, time và reason khi áp dụng | **Đã chốt ở mức nguyên tắc** | Schema chi tiết thuộc Epic 23 |
| `GOV-02` | Ghi thao tác denied/failed | Không ghi / ghi mọi lần / theo catalog | Dùng Audit Catalog theo độ nhạy | **Chưa chốt** | Chặn catalog event hoàn chỉnh |
| `GOV-03` | Retention và legal hold | Một thời hạn chung / theo loại event | Theo loại event và quy định, không mặc định vĩnh viễn | **Chưa chốt** | Không triển khai delete/export lịch sử |
| `GOV-04` | Chu kỳ Access Review | Không định kỳ / tháng / quý / theo rủi ro | Theo quý và review ngay khi thay đổi đặc quyền | **Chưa chốt** | Chưa có campaign/review workflow |
| `GOV-05` | Owner và bằng chứng review | System Admin / manager / owner dữ liệu | Manager xác nhận, Security/Auditor giám sát | **Chưa chốt** | Chưa có action xác nhận review |
| `GOV-06` | Export Access Review | Cho tải trực tiếp / job có approval | Job có scope, expiry và Audit tải file | **Chưa chốt** | Không có nút export |
| `GOV-07` | Dữ liệu Employee được hiển thị | Đầy đủ / tối thiểu theo mục đích | Chỉ identity tham chiếu và dữ liệu System Access cần thiết | **Nguyên tắc đã chốt; field cụ thể chưa chốt** | BFF tiếp tục projection và masking |
| `GOV-08` | Notification | Không gửi / gửi cho employee/manager theo action | Gửi khi mời, khóa, thay đổi quyền đặc biệt và sắp hết hạn | **Chưa chốt** | Phụ thuộc Epic 28 |

## 8. Baseline đang áp dụng trong giao diện

Cho đến khi bảng trên được duyệt:

1. `/administration/employees`, `/administration/roles` và `/administration/permissions` chỉ phục vụ tra cứu/read-only.
2. Không thêm `POST`, `PATCH`, `PUT` hoặc `DELETE` cho Access Management.
3. Không hiển thị nút tạo/sửa/khóa/archive/gán/thu hồi như thể chức năng đã sẵn sàng.
4. Permission được thực thi ở BFF/backend; ẩn menu chỉ là lớp trình bày.
5. Không có Direct Permission, Role inheritance hoặc explicit Deny trong projection hiện tại.
6. Không xóa cứng Account, Role, Permission hoặc Assignment đã có lịch sử.
7. Mọi Admin API tương lai vẫn được đặt trong `mocks/mockoon/admin_extensions.json` cho đến khi có upstream thật.

## 9. Thứ tự cần duyệt để mở khóa mutation

1. **Identity và Account lifecycle:** `ACC-01~11`.
2. **Mô hình RBAC và scope:** `AUTH-02~07`, `ROLE-01~09`, `PERM-01~06`.
3. **Assignment và privileged access:** `ASSIGN-01~09`, `ACC-10`.
4. **Session, Audit và privacy:** `ACC-09`, `GOV-01~08`.
5. Sau khi bốn nhóm trên được duyệt mới chia mutation thành các lát cắt tạo Account, khóa Account, quản lý Role–Permission và Role Assignment.


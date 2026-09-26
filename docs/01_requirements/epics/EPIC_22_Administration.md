# EPIC 22 — User / Role / Permission Administration

## 1. Mục tiêu Epic

Epic này quản lý quyền truy cập hệ thống của nhân viên: tạo/cập nhật/khóa Employee Account, quản lý Role và Permission, gán Role, đồng thời cung cấp khả năng xem quyền hiệu lực của từng nhân viên để kiểm tra phân quyền.

EPIC 22 sở hữu Employee System Access, Role, Permission, Role–Permission và Employee–Role Assignment. Epic không sở hữu hồ sơ nhân sự, đăng nhập của Customer, nội dung Audit Log hay chính sách nghiệp vụ chi tiết của từng module; các trách nhiệm đó thuộc EPIC 02, 01, 23 và Epic chuyên môn tương ứng.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 2nd** | `US-ADM-01` đến `US-ADM-07` | Doanh nghiệp cấp/thu hồi System Access cho nhân viên, quản lý Role/Permission, gán quyền và kiểm tra quyền hiệu lực có nguồn gốc rõ ràng. |

Toàn bộ User Story có mức ưu tiên **Must Have — 2nd** theo Product Backlog. `US-USER-04` của EPIC 02 đã được gộp vào `US-ADM-03`, không triển khai thành luồng khóa tài khoản thứ hai.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors

| Actor | Vai trò trong Epic |
| --- | --- |
| System Administrator (`ACT-17`) | Quản lý Employee Account, Role, Permission, assignment và access review theo quyền. |
| HR Manager (chưa có Actor ID) | Đề nghị/tạo hoặc khóa System Access ở phạm vi được ủy quyền; quản lý Employee Profile vẫn thuộc EPIC 02. |
| Employee nội bộ | Nhận System Access và chỉ sử dụng chức năng/dữ liệu theo quyền hiệu lực. |
| Auditor / Security Operator (`ACT-18`) | Sử dụng Audit Log của EPIC 23 để điều tra; không tự quản lý Role chỉ vì có quyền Audit. |

> `HR Manager` xuất hiện trong `US-ADM-01`, `US-ADM-03` và EPIC 02 nhưng chưa được định nghĩa trong Actor Registry của Product Backlog. Actor ID, phạm vi được tạo/khóa tài khoản và quan hệ với `ACT-17` cần Product Owner chốt.

### 3.2. Business Rules và NFR áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-03` | Tài khoản nội bộ phải được gán Role phù hợp trước khi truy cập chức năng quản trị. | Employee Account chưa có Role hợp lệ không được vào vùng quản trị. |
| `BR-AUTH-04` | Nhân viên chỉ thực hiện thao tác thuộc quyền được cấp. | Mọi chức năng và phạm vi dữ liệu phải kiểm tra quyền hiệu lực, không chỉ dựa trên tên Role. |
| `BR-PACK-01` | Chỉ Packing Staff được phân công mới được xác nhận đóng gói. | Permission chung không được bỏ qua điều kiện được phân công của nghiệp vụ Packing. |
| `BR-PACK-05` | Người không có quyền không được tùy ý xem/tải Packing Video. | Quyền xem media nhạy cảm phải là Permission/phạm vi riêng, không suy ra từ quyền xem Order. |
| `BR-REFUND-01` | Chỉ Sales Manager hoặc cấp cao hơn được duyệt Refund. | Role Assignment không được làm mờ thẩm quyền nghiệp vụ; Permission duyệt Refund cần kiểm soát riêng. |
| `BR-AUDIT-01` | Thao tác quan trọng phải được ghi Audit Log. | Tạo/khóa tài khoản, thay Role/Permission và assignment phải phát sinh Audit đầy đủ. |
| `BR-AUDIT-04` | Chỉ Auditor/System Administrator đặc quyền được tra cứu/trích xuất Audit. | Quyền xem Audit phải tách biệt và không tự cấp cho mọi Admin. |
| `NFR-04` | Dữ liệu Customer, Employee và nghiệp vụ nhạy cảm phải được bảo vệ. | Danh sách tài khoản/quyền và thao tác quản trị phải chống truy cập trái phép. |
| `NFR-05` | Người dùng chỉ truy cập dữ liệu/chức năng thuộc quyền. | Quyền phải được thực thi nhất quán tại mọi điểm truy cập, không chỉ ẩn nút giao diện. |
| `NFR-08` | Dữ liệu cá nhân chỉ được dùng theo mục đích được phép. | Access Review và Account Admin chỉ hiển thị dữ liệu nhân viên cần thiết cho quản trị. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 01 — Authentication & Identity`: cung cấp cơ chế xác thực/phiên; Employee identity và luồng kích hoạt cần dùng chính sách bảo mật chung nhưng quyền nhân viên thuộc EPIC 22.
- `EPIC 02 — Customer & Employee Profile`: sở hữu họ tên, phòng ban, hợp đồng, chức vụ; EPIC 22 chỉ tham chiếu Employee để cấp System Access.
- `EPIC 10, 14, 15, 16, 19–21, 24–28`: định nghĩa hành động/phạm vi nghiệp vụ cần kiểm soát; EPIC 22 quản lý Permission, không thay thế điều kiện nghiệp vụ.
- `EPIC 23 — Audit & Security`: lưu, bảo vệ và cho tra cứu Audit Log; EPIC 22 phát sinh sự kiện Audit và thực thi kết quả quản trị quyền.
- `EPIC 28 — Notification`: gửi lời mời/kích hoạt, thông báo khóa hoặc thay đổi quyền nếu chính sách yêu cầu.

## 4. User Stories chi tiết

### US-ADM-01 — Tạo Employee Account

**Actor:** System Administrator (`ACT-17`) hoặc HR Manager (chưa có Actor ID)
**Ưu tiên:** Must Have — 2nd
**Phát hành:** MVP

> Là System Administrator hoặc HR Manager, tôi muốn tạo tài khoản (System Access) cho nhân viên để cấp quyền sử dụng hệ thống.

**Giá trị nghiệp vụ:** Nhân viên được cấp định danh truy cập gắn đúng hồ sơ và chỉ có thể vào hệ thống quản trị sau khi đáp ứng điều kiện kích hoạt/Role.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo System Access cho Employee hợp lệ
  Given Employee Profile tồn tại và actor có quyền tạo tài khoản trong phạm vi
  When actor nhập định danh đăng nhập/kênh kích hoạt hợp lệ và xác nhận
  Then hệ thống tạo một Employee Account gắn đúng Employee Profile
  And tài khoản có trạng thái khởi tạo phù hợp
  And lưu người tạo cùng thời điểm

Scenario: Không tạo Account trùng cho cùng Employee
  Given Employee đã có System Access hiện hành
  When actor cố tạo thêm tài khoản trái quy tắc duy nhất
  Then hệ thống từ chối
  And hướng dẫn cập nhật/khôi phục tài khoản hiện có

Scenario: Định danh đăng nhập đã được sử dụng
  Given email, số điện thoại hoặc username đã thuộc tài khoản khác trong phạm vi duy nhất
  When actor tạo Employee Account
  Then hệ thống từ chối dữ liệu gây nhầm danh tính
  And không tiết lộ thông tin nhạy cảm của tài khoản hiện có

Scenario: Employee Profile không tồn tại hoặc không hợp lệ
  Given không có Employee phù hợp hoặc hồ sơ không đủ điều kiện cấp truy cập
  When actor tạo tài khoản
  Then hệ thống không tạo System Access mồ côi
  And hiển thị điều kiện cần xử lý tại EPIC 02

Scenario: Tài khoản chưa có Role
  Given Employee Account vừa được tạo nhưng chưa có Role hợp lệ
  When nhân viên cố truy cập chức năng quản trị
  Then hệ thống từ chối theo `BR-AUTH-03`
  And không tự gán Role mặc định có quyền rộng

Scenario: Gửi lời mời kích hoạt thất bại
  Given Account đã được tạo nhưng Notification Provider không gửi được lời mời
  When hệ thống nhận kết quả thất bại
  Then trạng thái Account/lời mời được thể hiện chính xác
  And không tạo lại Account chỉ để gửi lại thông báo

Scenario: Actor vượt phạm vi tạo tài khoản
  Given HR Manager/Admin không có quyền với đơn vị hoặc loại nhân viên mục tiêu
  When actor gửi yêu cầu tạo
  Then hệ thống từ chối
  And không tạo hoặc tiết lộ dữ liệu ngoài phạm vi
```

### US-ADM-02 — Cập nhật Employee Account

**Actor:** System Administrator (`ACT-17`)
**Ưu tiên:** Must Have — 2nd
**Phát hành:** MVP

> Là System Administrator, tôi muốn cập nhật thông tin tài khoản nhân viên để dữ liệu luôn chính xác.

**Giá trị nghiệp vụ:** Định danh/kênh truy cập và metadata tài khoản được hiệu chỉnh có kiểm soát mà không trộn với Employee Profile hoặc thay quyền ngoài ý muốn.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Cập nhật trường Account hợp lệ
  Given System Administrator có quyền và Account tồn tại
  When cập nhật trường thuộc System Access bằng dữ liệu hợp lệ
  Then hệ thống lưu thay đổi cho đúng Account
  And không tự sửa Employee Profile, Role hoặc Permission

Scenario: Cố sửa trường thuộc Employee Profile
  Given họ tên, phòng ban, hợp đồng hoặc chức vụ thuộc EPIC 02
  When Admin cố thay đổi qua chức năng Account
  Then hệ thống không cập nhật dữ liệu hồ sơ tại đây
  And hướng dẫn dùng quy trình đúng nếu có quyền

Scenario: Thay định danh đăng nhập cần xác minh
  Given chính sách yêu cầu xác minh email/số điện thoại mới
  When Admin cập nhật định danh
  Then hệ thống giữ trạng thái chờ xác minh theo chính sách
  And không coi kênh mới là đã xác minh trước khi có kết quả hợp lệ

Scenario: Giá trị mới bị trùng hoặc không hợp lệ
  Given Admin nhập định danh sai định dạng hoặc đã thuộc Account khác
  When lưu thay đổi
  Then hệ thống từ chối
  And giữ dữ liệu hiện tại không đổi

Scenario: Cập nhật đồng thời
  Given Account đã thay đổi sau khi Admin tải dữ liệu
  When Admin lưu trên phiên bản cũ
  Then hệ thống không ghi đè âm thầm thay đổi mới hơn
  And yêu cầu tải lại/đối chiếu

Scenario: Admin không có quyền với Account mục tiêu
  Given Account ngoài phạm vi quản trị của Admin
  When Admin xem hoặc cập nhật
  Then hệ thống từ chối
  And không tiết lộ dữ liệu tài khoản

Scenario: Không cho xem hoặc đặt mật khẩu đọc được
  Given Admin đang quản lý Account
  When mở chi tiết hoặc cập nhật thông tin
  Then hệ thống không hiển thị mật khẩu hiện tại dưới dạng đọc được
  And thao tác thiết lập lại thông tin xác thực phải theo quy trình bảo mật riêng
```

### US-ADM-03 — Khóa Employee Account

**Actor:** System Administrator (`ACT-17`) hoặc HR Manager (chưa có Actor ID)
**Ưu tiên:** Must Have — 2nd
**Phát hành:** MVP

> Là System Administrator hoặc HR Manager, tôi muốn khóa tài khoản nhân viên khi nhân viên nghỉ việc hoặc không còn được phép truy cập.

**Giá trị nghiệp vụ:** Quyền truy cập được thu hồi kịp thời, có lý do và không làm mất Employee Profile hoặc lịch sử nghiệp vụ của nhân viên.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Khóa Account hợp lệ
  Given Account đang `ACTIVE` và actor có quyền khóa
  When actor chọn khóa, nhập lý do và thời điểm hiệu lực hợp lệ
  Then Account chuyển `LOCKED` theo thời điểm áp dụng
  And không cho tạo phiên đăng nhập mới
  And lưu actor, thời điểm và lý do

Scenario: Xử lý phiên đang hoạt động khi khóa
  Given nhân viên đang có phiên hoạt động
  When khóa Account có hiệu lực
  Then các phiên/quyền truy cập tiếp theo bị thu hồi theo chính sách an toàn
  And không tiếp tục cho thao tác chỉ vì phiên được tạo trước lúc khóa

Scenario: Khóa không xóa hồ sơ hoặc lịch sử
  Given Employee có Profile và các bản ghi nghiệp vụ/Audit
  When Account bị khóa
  Then Employee Profile và lịch sử vẫn được bảo toàn
  And dữ liệu cũ vẫn quy được cho đúng actor

Scenario: Lên lịch khóa theo ngày nghỉ việc
  Given HR/Admin có ngày kết thúc quyền truy cập đã được xác nhận
  When cấu hình thời điểm khóa tương lai
  Then hệ thống lưu lịch và thực thi tại thời điểm phù hợp
  And hiển thị rõ trạng thái hiện tại với thời điểm dự kiến

Scenario: Yêu cầu khóa bị gửi lặp
  Given Account đã Locked bởi cùng quyết định
  When yêu cầu được gửi lại
  Then hệ thống không tạo nhiều chuyển trạng thái hoặc Audit nghiệp vụ trùng ngoài ý muốn
  And trả về trạng thái hiện tại

Scenario: Actor không có quyền khóa Account mục tiêu
  Given Account thuộc phạm vi/đặc quyền mà actor không được quản lý
  When actor cố khóa
  Then hệ thống từ chối
  And Account không thay đổi

Scenario: Khóa chính Account đang thao tác hoặc Admin cuối cùng
  Given thao tác có thể làm mất quyền quản trị cần thiết của hệ thống
  When actor yêu cầu khóa
  Then hệ thống áp dụng chính sách bảo vệ/duyệt đặc biệt đã được chốt
  And không để hệ thống rơi vào trạng thái không còn người quản trị hợp lệ ngoài ý muốn
```

### US-ADM-04 — Quản lý Role

**Actor:** System Administrator (`ACT-17`)
**Ưu tiên:** Must Have — 2nd
**Phát hành:** MVP

> Là System Administrator, tôi muốn tạo và quản lý Role để phân nhóm quyền truy cập.

**Giá trị nghiệp vụ:** Quyền được tổ chức thành nhóm có tên/mục đích rõ ràng, tái sử dụng cho nhiều nhân viên và quản lý vòng đời có kiểm soát.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo Role hợp lệ
  Given System Administrator có quyền quản lý Role
  When nhập mã/tên duy nhất, mô tả và phạm vi hợp lệ
  Then hệ thống tạo Role ở trạng thái cấu hình phù hợp
  And Role chưa cấp quyền hiệu lực cho ai chỉ vì vừa được tạo

Scenario: Role trùng hoặc dữ liệu không hợp lệ
  Given mã/tên Role đã tồn tại hoặc trường bắt buộc không hợp lệ
  When Admin lưu
  Then hệ thống từ chối
  And không ghi đè Role hiện có

Scenario: Cập nhật Role đang được sử dụng
  Given Role đã được gán cho Employee
  When Admin thay mô tả/trạng thái hoặc tập Permission theo quy trình
  Then hệ thống hiển thị phạm vi ảnh hưởng trước khi xác nhận
  And lưu phiên bản/thay đổi có thể truy vết

Scenario: Vô hiệu hóa Role đang được gán
  Given Role có assignment đang hiệu lực
  When Admin yêu cầu vô hiệu hóa
  Then hệ thống cảnh báo các Employee bị ảnh hưởng
  And áp dụng quy tắc thay thế/thu hồi đã được chốt
  And không để assignment trỏ tới Role hiệu lực mơ hồ

Scenario: Xóa Role có lịch sử
  Given Role đã từng được gán hoặc dùng làm căn cứ Audit
  When Admin yêu cầu xóa
  Then hệ thống không xóa cứng làm mất lịch sử
  And dùng trạng thái ngừng sử dụng/archive theo chính sách

Scenario: Bảo vệ Role hệ thống
  Given Role được đánh dấu là Role hệ thống/đặc quyền được bảo vệ
  When Admin cố sửa/xóa ngoài quy trình đặc biệt
  Then hệ thống từ chối
  And giữ cấu hình an toàn

Scenario: Người không có quyền quản lý Role
  Given nhân viên thiếu Permission quản lý Role
  When tạo, sửa hoặc vô hiệu hóa Role
  Then hệ thống từ chối
  And không thay đổi Role
```

### US-ADM-05 — Quản lý Permission

**Actor:** System Administrator (`ACT-17`)
**Ưu tiên:** Must Have — 2nd
**Phát hành:** MVP

> Là System Administrator, tôi muốn quản lý Permission để kiểm soát từng chức năng mà mỗi Role được phép sử dụng.

**Giá trị nghiệp vụ:** Mỗi hành động/phạm vi dữ liệu nhạy cảm có mã và mô tả rõ ràng, được gán cho Role mà không phụ thuộc tên hiển thị hoặc logic giao diện.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem danh mục Permission
  Given Admin có quyền quản lý phân quyền
  When mở danh mục Permission
  Then hệ thống hiển thị mã, tên, hành động, tài nguyên/phạm vi, trạng thái và nơi đang sử dụng
  And phân biệt Permission hệ thống với cấu hình có thể chỉnh sửa

Scenario: Tạo hoặc cập nhật Permission được phép cấu hình
  Given mô hình cho phép Permission tùy chỉnh và Admin có quyền
  When nhập mã duy nhất cùng hành động/phạm vi hợp lệ
  Then hệ thống lưu Permission
  And không tự gán Permission đó vào Role

Scenario: Permission trùng hoặc không xác định được phạm vi
  Given mã đã tồn tại hoặc action/resource/scope không hợp lệ
  When Admin lưu
  Then hệ thống từ chối
  And không tạo Permission gây hiểu mơ hồ

Scenario: Gán Permission vào Role
  Given Role và Permission đang hợp lệ
  When Admin thêm Permission vào Role và xác nhận ảnh hưởng
  Then quan hệ Role–Permission có hiệu lực theo chính sách
  And Access Review thể hiện nguồn quyền từ Role đó

Scenario: Thu hồi Permission khỏi Role
  Given Role đang cung cấp Permission cho Employee
  When Admin thu hồi
  Then quyền hiệu lực được tính lại theo mọi Role còn lại
  And không thu hồi sai nếu Employee vẫn có cùng Permission từ nguồn hợp lệ khác

Scenario: Sửa hoặc xóa mã Permission đã được thực thi
  Given Permission đã được module nghiệp vụ sử dụng
  When Admin cố đổi mã/xóa cứng
  Then hệ thống chặn hoặc yêu cầu quy trình migration/archive đã được chốt
  And không tạo lỗ hổng do module tiếp tục kiểm tra mã cũ

Scenario: Permission xem không bao gồm Permission thao tác
  Given Role chỉ có quyền xem một tài nguyên
  When Employee cố tạo/sửa/xóa/phê duyệt tài nguyên đó
  Then hệ thống từ chối nếu thiếu Permission hành động tương ứng
  And không suy rộng từ quyền xem
```

### US-ADM-06 — Gán Role cho Employee

**Actor:** System Administrator (`ACT-17`)
**Ưu tiên:** Must Have — 2nd
**Phát hành:** MVP

> Là System Administrator, tôi muốn gán Role cho nhân viên để nhân viên chỉ truy cập đúng phạm vi công việc.

**Giá trị nghiệp vụ:** Employee nhận đúng tập quyền từ Role có nguồn, hiệu lực và người phê duyệt rõ ràng; thay đổi được áp dụng kịp thời và không tạo leo thang ngoài ý muốn.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Gán Role hợp lệ
  Given Employee Account và Role đang hợp lệ
  And Admin có quyền gán Role đó trong phạm vi
  When Admin xác nhận assignment với thời điểm hiệu lực nếu áp dụng
  Then hệ thống tạo Employee–Role Assignment
  And quyền hiệu lực của Employee được tính lại
  And lưu actor cùng thời điểm

Scenario: Gán Role trùng đang hiệu lực
  Given Employee đã có cùng Role trong khoảng hiệu lực
  When Admin gán lại
  Then hệ thống không tạo assignment trùng ngoài ý muốn
  And trả về trạng thái hiện tại

Scenario: Gán Role đã vô hiệu hóa
  Given Role không còn Active
  When Admin gán cho Employee
  Then hệ thống từ chối
  And quyền hiệu lực không thay đổi

Scenario: Admin cố gán Role vượt thẩm quyền
  Given Role có đặc quyền hoặc phạm vi cao hơn quyền được ủy quyền của Admin
  When Admin gán Role
  Then hệ thống từ chối hoặc yêu cầu phê duyệt đặc biệt theo chính sách
  And không cấp quyền trước khi được phép

Scenario: Thu hồi Role
  Given Employee có assignment đang hiệu lực
  When Admin thu hồi với lý do nếu bắt buộc
  Then assignment kết thúc theo thời điểm hiệu lực
  And quyền được tính lại từ các Role còn lại
  And phiên đang hoạt động không tiếp tục dùng quyền đã mất

Scenario: Role có thời hạn
  Given assignment có ngày bắt đầu/kết thúc hợp lệ
  When đến mốc hiệu lực
  Then hệ thống kích hoạt/hết hạn assignment theo chính sách thời gian
  And Access Review thể hiện trạng thái và thời hạn

Scenario: Gán Role cho Account bị khóa
  Given Employee Account đang `LOCKED`
  When Admin cập nhật assignment
  Then hệ thống có thể lưu thay đổi theo quyền quản trị
  And Account vẫn không được truy cập cho đến khi có quy trình mở khóa hợp lệ

Scenario: Không để mất Admin hợp lệ cuối cùng
  Given thu hồi Role có thể làm hệ thống không còn người có quyền quản trị thiết yếu
  When Admin xác nhận thao tác
  Then hệ thống áp dụng cơ chế bảo vệ/phê duyệt đã chốt
  And không tạo trạng thái mất quyền quản trị ngoài ý muốn
```

### US-ADM-07 — Xem quyền của từng Employee

**Actor:** System Administrator (`ACT-17`)
**Ưu tiên:** Must Have — 2nd
**Phát hành:** MVP

> Là System Administrator, tôi muốn xem quyền của từng nhân viên để kiểm tra việc phân quyền.

**Giá trị nghiệp vụ:** Admin biết Employee đang có quyền gì, do Role/assignment nào, trong phạm vi và thời hạn nào để phát hiện quyền thiếu, thừa hoặc bất thường.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem quyền hiệu lực của Employee
  Given Admin có quyền Access Review và Employee thuộc phạm vi
  When mở chi tiết quyền
  Then hệ thống hiển thị Account status, Role assignment và danh sách Permission hiệu lực
  And mỗi Permission chỉ rõ nguồn Role, scope và thời hạn nếu có

Scenario: Employee có nhiều Role cùng Permission
  Given cùng Permission đến từ nhiều Role hợp lệ
  When Admin xem quyền hiệu lực
  Then hệ thống hiển thị Permission một cách không gây hiểu là nhiều quyền khác nhau
  And vẫn liệt kê đầy đủ các nguồn cấp quyền

Scenario: Role hết hạn hoặc bị vô hiệu hóa
  Given assignment đã hết hạn hoặc Role không còn Active
  When Admin xem quyền
  Then hệ thống không trình bày quyền đó như đang hiệu lực
  And vẫn cho xem lịch sử/trạng thái nguồn nếu được phép

Scenario: Account bị khóa
  Given Employee Account đang Locked nhưng còn Role Assignment
  When Admin xem Access Review
  Then hệ thống phân biệt quyền cấu hình với khả năng truy cập thực tế đang bị chặn
  And không mô tả Employee như đang sử dụng được quyền

Scenario: Tìm nhân viên có Permission nhạy cảm
  Given Admin có quyền review theo Permission
  When lọc theo một Permission hoặc Role
  Then hệ thống hiển thị các Employee có quyền hiệu lực trong phạm vi Admin
  And thể hiện nguồn assignment để xử lý

Scenario: Dữ liệu quyền thay đổi khi đang review
  Given Role/assignment thay đổi sau khi báo cáo được tải
  When Admin làm mới hoặc thực hiện hành động phụ thuộc dữ liệu
  Then hệ thống hiển thị trạng thái mới nhất cùng thời điểm tính quyền
  And không dựa vào snapshot cũ như quyền hiện tại

Scenario: Người dùng không có quyền Access Review
  Given nhân viên thiếu Permission xem phân quyền
  When truy cập danh sách hoặc chi tiết
  Then hệ thống từ chối
  And không tiết lộ Role/Permission của Employee khác
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-ADM-01` | Cho actor có quyền tạo một Employee Account duy nhất gắn với Employee Profile hợp lệ. | `US-ADM-01`, `BR-AUTH-03` |
| `FR-ADM-02` | Kiểm tra định danh đăng nhập duy nhất, ngăn Account mồ côi/trùng và không tiết lộ Account hiện có. | `US-ADM-01`, `NFR-04` |
| `FR-ADM-03` | Quản lý trạng thái lời mời/kích hoạt riêng với Account; lỗi gửi không tạo Account trùng. | `US-ADM-01`, `EPIC 28` |
| `FR-ADM-04` | Account chưa có Role hợp lệ không được truy cập vùng quản trị. | `US-ADM-01`, `US-ADM-06`, `BR-AUTH-03` |
| `FR-ADM-05` | Cho System Administrator cập nhật trường System Access nhưng không sửa Employee Profile, Role/Permission ngoài quy trình tương ứng. | `US-ADM-02`, `EPIC 02` |
| `FR-ADM-06` | Xử lý định danh mới cần xác minh theo chính sách, ngăn trùng và phát hiện cập nhật đồng thời. | `US-ADM-02` |
| `FR-ADM-07` | Không hiển thị mật khẩu đọc được; reset credential phải qua quy trình bảo mật riêng. | `US-ADM-02`, `NFR-04` |
| `FR-ADM-08` | Cho actor có quyền khóa Account ngay/lên lịch với lý do và chống xử lý lặp. | `US-ADM-03`, `BR-AUDIT-01` |
| `FR-ADM-09` | Khi khóa/thu hồi quyền có hiệu lực, chặn đăng nhập mới và xử lý phiên/quyền đang hoạt động theo chính sách an toàn. | `US-ADM-03`, `US-ADM-06`, `NFR-05` |
| `FR-ADM-10` | Khóa Account không xóa Employee Profile, assignment history hoặc dữ liệu nghiệp vụ/Audit. | `US-ADM-03`, `EPIC 02`, `EPIC 23` |
| `FR-ADM-11` | Bảo vệ thao tác có thể khóa/thu hồi Admin thiết yếu cuối cùng theo chính sách được chốt. | `US-ADM-03`, `US-ADM-06` |
| `FR-ADM-12` | Cho tạo Role có mã/tên duy nhất, mô tả, scope, trạng thái và dấu phân biệt Role hệ thống/tùy chỉnh. | `US-ADM-04` |
| `FR-ADM-13` | Khi sửa/vô hiệu hóa Role đang dùng, hiển thị ảnh hưởng, tính lại quyền và không xóa cứng lịch sử. | `US-ADM-04`, `BR-AUDIT-01` |
| `FR-ADM-14` | Bảo vệ Role hệ thống/đặc quyền khỏi sửa hoặc xóa ngoài quy trình đặc biệt. | `US-ADM-04`, `NFR-05` |
| `FR-ADM-15` | Cung cấp danh mục Permission có mã, action, resource, scope, trạng thái và nơi sử dụng. | `US-ADM-05` |
| `FR-ADM-16` | Kiểm tra Permission duy nhất/hợp lệ; bảo vệ mã đã được module thực thi khỏi đổi/xóa gây lỗ hổng. | `US-ADM-05` |
| `FR-ADM-17` | Cho thêm/thu hồi Permission khỏi Role và tính quyền hợp nhất từ mọi Role còn hiệu lực. | `US-ADM-05`, `US-ADM-06`, `BR-AUTH-04` |
| `FR-ADM-18` | Phân biệt các action xem/tạo/sửa/xóa/phê duyệt/xuất và scope; không suy rộng quyền xem thành quyền thao tác. | `US-ADM-05`, `BR-AUTH-04` |
| `FR-ADM-19` | Cho Admin có thẩm quyền tạo/thu hồi Employee–Role Assignment với hiệu lực/thời hạn và chống trùng. | `US-ADM-06` |
| `FR-ADM-20` | Ngăn gán Role inactive hoặc vượt phạm vi/đặc quyền của người gán; yêu cầu phê duyệt khi chính sách quy định. | `US-ADM-06`, `NFR-05` |
| `FR-ADM-21` | Role thay đổi/hết hạn/thu hồi phải được phản ánh vào quyền hiệu lực và phiên truy cập trong thời gian chính sách cho phép. | `US-ADM-04~06`, `NFR-05` |
| `FR-ADM-22` | Cung cấp Access Review theo Employee với Account status, Role/assignment, Permission hiệu lực, nguồn, scope và thời hạn. | `US-ADM-07` |
| `FR-ADM-23` | Cho lọc Employee theo Role/Permission hiệu lực trong phạm vi quyền và phân biệt cấu hình quyền với khả năng truy cập của Account Locked. | `US-ADM-07` |
| `FR-ADM-24` | Ghi rõ thời điểm tính quyền và yêu cầu làm mới trước hành động phụ thuộc nếu cấu hình đã thay đổi. | `US-ADM-07` |
| `FR-ADM-25` | Kiểm soát quyền quản trị theo phạm vi Employee/đơn vị, loại Role/Permission và hành động; không dựa chỉ vào giao diện. | `US-ADM-01~07`, `BR-AUTH-04`, `NFR-05` |
| `FR-ADM-26` | Phát sinh Audit đầy đủ cho Account status, Role, Permission, assignment và thay đổi nhạy cảm. | `US-ADM-01~07`, `BR-AUDIT-01`, `EPIC 23` |
| `FR-ADM-27` | Không cho Permission chung bỏ qua điều kiện nghiệp vụ như phân công Packing, quyền media hoặc thẩm quyền duyệt Refund. | `US-ADM-05~07`, `BR-PACK-01`, `BR-PACK-05`, `BR-REFUND-01` |

## 6. Vòng đời và mô hình phân quyền

### 6.1. Vòng đời Employee System Access đề xuất

```text
INVITED → PENDING_ACTIVATION → ACTIVE → LOCKED
                    │             │        │
                    └─────────────┴────► DISABLED
```

- `INVITED`: Account đã tạo và lời mời đang chờ gửi/tiếp nhận.
- `PENDING_ACTIVATION`: định danh/credential chưa hoàn tất kích hoạt.
- `ACTIVE`: Account có thể xác thực; vẫn phải có Role/Permission hợp lệ để vào chức năng quản trị.
- `LOCKED`: tạm hoặc vô thời hạn không được truy cập; mở khóa chưa có User Story riêng và cần chính sách.
- `DISABLED`: ngừng System Access theo quyết định dài hạn; không đồng nghĩa xóa Employee Profile/lịch sử.

### 6.2. Mô hình quyền hiệu lực

```text
Employee Account
  └── Employee–Role Assignment (scope, effective period)
        └── Role (status)
              └── Role–Permission
                    └── Permission (action, resource, scope)
```

- Quyền hiệu lực chỉ tồn tại khi Account có khả năng truy cập, assignment đang hiệu lực, Role active và Permission hợp lệ.
- Nếu nhiều Role cấp cùng Permission, thu hồi một nguồn không làm mất quyền nếu vẫn còn nguồn hợp lệ khác.
- Permission quyết định khả năng kỹ thuật/nghiệp vụ cơ bản; điều kiện động như “được phân công đơn”, trạng thái Case hoặc hạn mức phê duyệt vẫn phải được Epic chuyên môn kiểm tra.
- Direct Permission cho Employee chưa có trong Product Backlog; mặc định tài liệu chỉ mô tả quyền qua Role cho đến khi Product Owner quyết định khác.
- Chính sách Deny, ưu tiên khi nhiều scope giao nhau và kế thừa Role chưa được định nghĩa; không tự suy diễn trong triển khai.

### 6.3. Quy tắc tài khoản, khóa và session

- Employee Account và Employee Profile có định danh/liên kết nhưng vòng đời riêng; cập nhật phòng ban/chức vụ không tự đổi Role.
- Account status thay đổi phải có actor, thời điểm hiệu lực, lý do và trạng thái trước/sau.
- Khóa/thu hồi quyền phải ảnh hưởng tới yêu cầu truy cập mới và phiên hiện hữu theo thời hạn an toàn được chốt; cache/quyền cũ không được tồn tại vô hạn.
- Lời mời hoặc reset credential là dữ liệu nhạy cảm, có thời hạn, dùng một lần và không được hiển thị lại ở dạng bí mật đọc được.
- Không xóa cứng Account/Role/Permission/Assignment đã có lịch sử dùng làm căn cứ nghiệp vụ hoặc Audit.

### 6.4. Access Review và Audit

- Access Review phải giải thích “ai có quyền gì, trên phạm vi nào, từ Role/assignment nào, hiệu lực khi nào”.
- Snapshot/báo cáo quyền phải ghi thời điểm tính; quyền có thể thay đổi sau khi tải nên hành động nhạy cảm luôn kiểm tra lại trạng thái hiện tại.
- Access Review không thay thế Audit Log: review mô tả cấu hình/quyền hiện tại, EPIC 23 lưu lịch sử thay đổi/thao tác.
- Audit event từ EPIC 22 cần actor, hành động, target, trước/sau, scope, thời điểm và lý do khi áp dụng.
- Quyền xem Audit, quyền quản lý RBAC và quyền xem Employee Account là các Permission riêng, không mặc nhiên đi cùng nhau.

## 7. Traceability

| User Story | Business Rule / NFR | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-ADM-01` | `BR-AUTH-03`, `BR-AUTH-04`, `NFR-04` | EPIC 01, 02, 28 | Account đúng Employee, không trùng/mồ côi; chưa có Role không vào quản trị; lỗi invite không tạo Account mới. |
| `US-ADM-02` | `BR-AUTH-04`, `NFR-04` | EPIC 01, 02, 23 | Chỉ sửa System Access trong quyền, không sửa Profile/Role; định danh hợp lệ và không lộ credential. |
| `US-ADM-03` | `BR-AUTH-03`, `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 01, 02, 23 | Khóa đúng quyền/lý do/thời điểm, chặn phiên, chống lặp và giữ hồ sơ/lịch sử. |
| `US-ADM-04` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 23 | Role duy nhất, có trạng thái/phạm vi; thay đổi hiển thị ảnh hưởng và không xóa cứng lịch sử. |
| `US-ADM-05` | `BR-AUTH-04`, `BR-PACK-01`, `BR-PACK-05`, `BR-REFUND-01` | Các Epic nghiệp vụ | Permission có action/resource/scope; quyền xem không suy rộng; điều kiện nghiệp vụ vẫn được thực thi. |
| `US-ADM-06` | `BR-AUTH-03`, `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 01, 23 | Assignment đúng Role/phạm vi/hiệu lực, không trùng/vượt quyền; thu hồi cập nhật session và giữ Admin thiết yếu. |
| `US-ADM-07` | `BR-AUDIT-04`, `NFR-05`, `NFR-08` | EPIC 23 | Review hiển thị quyền hiệu lực, nguồn/scope/thời hạn và Account status; người không quyền không xem được. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Bổ sung `HR Manager` vào Actor Registry với ID/trách nhiệm; phạm vi tạo/khóa Account và khác biệt quyền với System Administrator.
- Nguồn tạo Employee Account: bắt buộc có Employee Profile trước hay cho phép tạo đồng thời; định danh duy nhất dùng email, số điện thoại hay username.
- Quy trình kích hoạt, đặt/reset credential, MFA cho nhân viên, thời hạn lời mời và kênh gửi ở MVP.
- Bộ trạng thái Account, quyền mở khóa/reactivate, trường hợp khóa tạm/vĩnh viễn và phê duyệt khi khóa actor đặc quyền.
- Thời gian tối đa để khóa/thu hồi Role có hiệu lực lên session/cache và cách xử lý phiên đang thực hiện giao dịch quan trọng.
- Taxonomy Permission: resource, action, scope; Permission do hệ thống định nghĩa hay Admin được tạo mới; quy trình version/migration mã.
- Một Employee được có nhiều Role không; scope theo đơn vị/kho/kênh/khu vực; cách hợp nhất và có hỗ trợ explicit Deny/direct Permission không.
- Danh sách Role hệ thống, Role đặc quyền, điều kiện sửa/clone/archive và cơ chế bảo vệ Admin hợp lệ cuối cùng.
- Ma trận thẩm quyền gán Role, phân tách nhiệm vụ, self-assignment/self-approval và quy trình phê duyệt quyền đặc biệt/break-glass.
- Assignment tạm thời, timezone/thời điểm hiệu lực, nhắc hết hạn và người chịu trách nhiệm gia hạn.
- Chu kỳ Access Review, owner phê duyệt/thu hồi, bằng chứng xác nhận, export và cách xử lý quyền thừa/orphaned Account.
- Dữ liệu Account/Employee nào hiển thị cho Admin/HR/Auditor; retention, masking và quyền riêng tư.
- Phạm vi Audit cụ thể, sự kiện thất bại/denied access cần ghi, thời hạn lưu và ranh giới với EPIC 23.

## 9. UI/UX Reference

- Employee Account form cần tìm/chọn đúng Employee Profile, hiển thị định danh, trạng thái kích hoạt/Role và không trộn trường HR Profile.
- Account detail cần tách Profile reference, System Access status, invitation/verification, Role assignment và lịch sử thay đổi.
- Lock dialog phải hiển thị Employee, phạm vi ảnh hưởng, session, thời điểm hiệu lực và yêu cầu lý do; cảnh báo Admin cuối cùng.
- Role editor cần mô tả/scope/trạng thái, permission matrix và số Employee bị ảnh hưởng trước khi lưu/vô hiệu hóa.
- Permission matrix cần nhóm theo module/resource/action/scope, phân biệt view với mutation/approval/export và Role hệ thống với tùy chỉnh.
- Assignment screen cần hiển thị Role đặc quyền, scope, thời hạn, nguồn phê duyệt và preview quyền hiệu lực sau thay đổi.
- Access Review cần trả lời rõ quyền hiệu lực, nguồn Role, scope, thời hạn, Account status và thời điểm tính; hỗ trợ lọc theo Role/Permission.
- Mọi trạng thái từ chối phải dễ hiểu nhưng không tiết lộ tài nguyên/dữ liệu mà actor không có quyền.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

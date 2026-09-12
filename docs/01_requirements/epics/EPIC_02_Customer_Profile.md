# EPIC 02 — Customer & Employee Profile

## 1. Mục tiêu Epic

Epic này quản lý dữ liệu hồ sơ phục vụ mua hàng của khách hàng và hồ sơ nhân sự phục vụ quản trị nội bộ. Khách có thể duy trì thông tin cá nhân cùng nhiều địa chỉ giao hàng để checkout nhanh hơn; HR Manager có thể quản lý dữ liệu hồ sơ nhân viên, tách biệt với tài khoản và quyền truy cập hệ thống.

Epic không xác thực danh tính, tạo phiên đăng nhập, quản lý mật khẩu, tạo tài khoản nhân viên, role/permission, khóa tài khoản nhân viên hoặc quản lý lịch sử đơn hàng. Các nội dung này lần lượt thuộc EPIC 01, EPIC 22 và EPIC 08.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP v1.0** | `US-USER-01`, `US-USER-02` | Khách đã xác thực có hồ sơ và nhiều địa chỉ để sử dụng khi checkout. |
| **Giai đoạn 2** | `US-USER-03`, `US-USER-05` | Doanh nghiệp quản lý được hồ sơ nhân sự và trạng thái tài khoản khách hàng theo thẩm quyền. |
| **Không triển khai độc lập** | `US-USER-04` | Đã gộp vào `US-ADM-03` của EPIC 22 để tránh trùng lặp. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Registered Customer (`ACT-02`) | Xem, cập nhật hồ sơ cá nhân và quản lý địa chỉ giao hàng của chính mình. |
| HR Manager | Quản lý hồ sơ nhân sự, không trực tiếp cấp hoặc khóa quyền truy cập hệ thống. |
| Nhân viên có quyền quản lý khách hàng | Xem và xử lý trạng thái tài khoản Customer theo chính sách doanh nghiệp. |
| System Administrator (`ACT-17`) | Cấp tài khoản, role, permission và khóa tài khoản nhân viên qua EPIC 22. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-03` | Tài khoản nội bộ phải được gán Role phù hợp trước khi truy cập chức năng quản trị. | Chỉ HR Manager hoặc người được cấp quyền mới thao tác hồ sơ nhân sự/trạng thái Customer. |
| `BR-AUTH-04` | Một nhân viên chỉ được thực hiện những thao tác thuộc quyền được cấp. | Customer chỉ quản lý dữ liệu của mình; nhân viên chỉ thao tác trong phạm vi quyền được cấp. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Thay đổi nhạy cảm đối với hồ sơ nhân sự hoặc trạng thái Customer cần được chuyển cho cơ chế Audit của EPIC 23. |

> Product Backlog chưa quy định cụ thể tập giá trị trạng thái Customer, chính sách xác minh khi đổi email/số điện thoại hoặc thời hạn lưu trữ dữ liệu. Các điểm này được liệt kê tại mục cần chốt, không tự suy diễn thành quy tắc bắt buộc.

### 3.3. Phụ thuộc giữa Epic

- `EPIC 01 — Authentication & Identity`: xác thực Customer trước khi cho phép truy cập/cập nhật hồ sơ.
- `EPIC 06 — Checkout`: dùng thông tin hồ sơ và địa chỉ đã lưu để giảm thao tác nhập lại.
- `EPIC 08 — Order Management`: chỉ hiển thị đơn hàng thuộc Customer sau khi đã xác thực.
- `EPIC 22 — User / Role / Permission Administration`: quản lý System Access, role, permission và trạng thái tài khoản nhân viên.
- `EPIC 23 — Audit & Security`: lưu lịch sử các thay đổi nhạy cảm theo chính sách Audit.

## 4. User Stories chi tiết

### US-USER-01 — Customer Profile

**Actor:** Registered Customer (`ACT-02`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là một khách hàng, tôi muốn quản lý họ tên, số điện thoại, email và thông tin cá nhân để sử dụng khi mua hàng.

**Giá trị nghiệp vụ:** Khách không cần nhập lại thông tin cơ bản ở mỗi lần mua; doanh nghiệp có thông tin liên hệ nhất quán để phục vụ đơn hàng và hỗ trợ khách.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem hồ sơ của chính mình
  Given khách đã đăng nhập
  When khách mở trang hồ sơ
  Then hệ thống hiển thị thông tin cá nhân thuộc về tài khoản của khách
  And không hiển thị dữ liệu hồ sơ của khách hàng khác

Scenario: Cập nhật hồ sơ hợp lệ
  Given khách đã đăng nhập và đang ở trang hồ sơ
  When khách cập nhật các trường thông tin hợp lệ và lưu thay đổi
  Then hệ thống lưu thông tin mới cho đúng tài khoản Customer
  And thông tin mới được sử dụng tại các bước mua hàng có liên quan

Scenario: Từ chối dữ liệu hồ sơ không hợp lệ
  Given khách nhập thiếu trường bắt buộc hoặc nhập sai định dạng một trường được kiểm tra
  When khách lưu hồ sơ
  Then hệ thống chỉ rõ trường cần chỉnh sửa
  And không ghi nhận dữ liệu không hợp lệ

Scenario: Ngăn cập nhật hồ sơ của khách khác
  Given khách A đã đăng nhập
  When khách A cố cập nhật hồ sơ thuộc khách B
  Then hệ thống từ chối thao tác
  And hồ sơ của khách B không bị thay đổi
```

### US-USER-02 — Customer Address

**Actor:** Registered Customer (`ACT-02`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là một khách hàng, tôi muốn lưu nhiều địa chỉ giao hàng để đặt hàng nhanh hơn.

**Giá trị nghiệp vụ:** Hỗ trợ khách gửi sản phẩm đến nhà, cơ quan hoặc người thân mà không phải nhập lại toàn bộ địa chỉ ở mỗi checkout.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Lưu thêm địa chỉ giao hàng
  Given khách đã đăng nhập
  When khách nhập thông tin địa chỉ giao hàng hợp lệ và lưu
  Then hệ thống thêm địa chỉ vào danh sách địa chỉ của khách
  And địa chỉ có thể được chọn ở checkout tiếp theo

Scenario: Quản lý nhiều địa chỉ
  Given khách đã có ít nhất hai địa chỉ giao hàng đã lưu
  When khách xem danh sách địa chỉ
  Then hệ thống hiển thị các địa chỉ thuộc về khách
  And khách có thể chọn, cập nhật hoặc xóa một địa chỉ theo chính sách nghiệp vụ

Scenario: Dùng địa chỉ đã lưu khi checkout
  Given khách đã đăng nhập và có địa chỉ giao hàng đã lưu
  When khách đến bước nhập thông tin nhận hàng
  Then hệ thống cho phép khách chọn một địa chỉ đã lưu
  And khách có thể kiểm tra hoặc chỉnh sửa thông tin đơn trước khi xác nhận đặt hàng

Scenario: Ngăn truy cập địa chỉ của khách khác
  Given khách A đã đăng nhập
  When khách A yêu cầu xem, sửa hoặc xóa địa chỉ của khách B
  Then hệ thống từ chối thao tác
  And địa chỉ của khách B không bị thay đổi
```

### US-USER-03 — Employee Profile

**Actor:** HR Manager  
**Ưu tiên:** Should Have  
**Phát hành:** Giai đoạn 2

> Là HR Manager, tôi muốn quản lý hồ sơ nhân sự (họ tên, phòng ban, hợp đồng, chức vụ) để quản lý thông tin nhân viên.

**Giá trị nghiệp vụ:** Tập trung thông tin nhân sự phục vụ vận hành, phân công và quản trị tổ chức; tránh dùng thông tin tài khoản đăng nhập làm hồ sơ nhân sự.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo hồ sơ nhân sự
  Given HR Manager có quyền quản lý hồ sơ nhân sự
  When HR Manager nhập thông tin hồ sơ nhân viên hợp lệ và lưu
  Then hệ thống tạo hồ sơ bao gồm họ tên, phòng ban, hợp đồng và chức vụ
  And hồ sơ có thể được tra cứu bởi người có quyền

Scenario: Cập nhật hồ sơ nhân sự
  Given hồ sơ nhân sự đã tồn tại
  When HR Manager cập nhật thông tin hợp lệ như phòng ban, hợp đồng hoặc chức vụ
  Then hệ thống lưu thông tin mới
  And việc thay đổi không tự động thay đổi role, permission hoặc trạng thái System Access

Scenario: Từ chối truy cập không có thẩm quyền
  Given nhân viên không có quyền quản lý hồ sơ nhân sự
  When nhân viên cố xem hoặc chỉnh sửa hồ sơ nhân sự ngoài phạm vi được cấp
  Then hệ thống từ chối thao tác
  And không tiết lộ dữ liệu nhân sự không cần thiết
```

### US-USER-04 — Employee Status

**Trạng thái:** Đã gộp vào `US-ADM-03 — Lock Employee Account` thuộc `EPIC 22` để tránh trùng lặp.

- Không triển khai hoặc viết Acceptance Criteria riêng trong Epic 02.
- Hồ sơ nhân sự (`US-USER-03`) và quyền truy cập hệ thống là hai dữ liệu khác nhau.
- Khi nhân viên nghỉ việc hoặc không còn quyền truy cập, thao tác khóa System Access được xử lý bởi EPIC 22; việc cập nhật thông tin hồ sơ nhân sự vẫn thuộc phạm vi HR theo chính sách doanh nghiệp.

### US-USER-05 — Customer Account Status

**Actor:** Nhân viên có quyền quản lý khách hàng  
**Ưu tiên:** Should Have  
**Phát hành:** Giai đoạn 2

> Là nhân viên có quyền quản lý khách hàng, tôi muốn xem và xử lý trạng thái tài khoản khách hàng khi cần thiết.

**Giá trị nghiệp vụ:** Doanh nghiệp có cơ chế xử lý các tình huống cần kiểm soát tài khoản Customer theo chính sách, mà không trao quyền đó cho mọi nhân viên.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem trạng thái tài khoản Customer
  Given nhân viên có quyền quản lý khách hàng
  When nhân viên tra cứu một Customer
  Then hệ thống hiển thị trạng thái tài khoản hiện tại và thông tin cần thiết để nhận diện Customer

Scenario: Cập nhật trạng thái Customer theo thẩm quyền
  Given nhân viên có quyền và Customer thuộc trường hợp được phép xử lý theo chính sách
  When nhân viên chọn trạng thái mới và cung cấp thông tin nghiệp vụ bắt buộc
  Then hệ thống cập nhật trạng thái Customer
  And ghi nhận thay đổi để có thể truy vết theo chính sách Audit

Scenario: Từ chối thay đổi trạng thái khi không có quyền
  Given nhân viên không có quyền quản lý trạng thái Customer
  When nhân viên cố cập nhật trạng thái một Customer
  Then hệ thống từ chối thao tác
  And trạng thái Customer không bị thay đổi

Scenario: Không ảnh hưởng dữ liệu mua hàng lịch sử
  Given Customer có đơn hàng đã phát sinh
  When trạng thái tài khoản Customer được cập nhật
  Then hệ thống vẫn bảo toàn dữ liệu đơn hàng lịch sử
  And chỉ áp dụng ảnh hưởng của trạng thái theo chính sách đã được phê duyệt
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-USER-01` | Hiển thị hồ sơ cá nhân cho đúng Customer đã xác thực. | `US-USER-01` |
| `FR-USER-02` | Cho phép Customer cập nhật họ tên, số điện thoại, email và thông tin cá nhân hợp lệ của chính mình. | `US-USER-01` |
| `FR-USER-03` | Kiểm tra dữ liệu hồ sơ trước khi lưu và thông báo trường cần sửa khi dữ liệu không hợp lệ. | `US-USER-01` |
| `FR-USER-04` | Cho phép Customer tạo, xem, chọn, cập nhật và xóa nhiều địa chỉ giao hàng thuộc về mình theo chính sách nghiệp vụ. | `US-USER-02` |
| `FR-USER-05` | Cung cấp địa chỉ Customer đã lưu cho Checkout; Checkout sử dụng bản thông tin của đơn đã xác nhận, không làm thay đổi địa chỉ lưu nếu Customer không yêu cầu. | `US-USER-02`, `EPIC 06` |
| `FR-USER-06` | Cho phép HR Manager tạo, xem và cập nhật hồ sơ nhân sự gồm họ tên, phòng ban, hợp đồng và chức vụ. | `US-USER-03` |
| `FR-USER-07` | Tách dữ liệu hồ sơ nhân sự khỏi tài khoản đăng nhập, role, permission và trạng thái System Access. | `US-USER-03`, `US-USER-04`, `EPIC 22` |
| `FR-USER-08` | Hiển thị trạng thái Customer cho nhân viên được cấp quyền quản lý khách hàng. | `US-USER-05` |
| `FR-USER-09` | Cho phép người được cấp quyền cập nhật trạng thái Customer theo chính sách đã phê duyệt và chuyển thông tin thay đổi cho Audit. | `US-USER-05`, `EPIC 23` |
| `FR-USER-10` | Từ chối mọi yêu cầu xem hoặc chỉnh sửa hồ sơ/địa chỉ/nhân sự ngoài phạm vi quyền truy cập. | `US-USER-01~05`, `BR-AUTH-04` |

## 6. Quy tắc dữ liệu, bảo mật và quyền riêng tư

- Customer chỉ xem và sửa dữ liệu hồ sơ, địa chỉ của chính mình; không được truy cập dữ liệu Customer khác qua giao diện hoặc thao tác nghiệp vụ.
- Thông tin hồ sơ nhân sự chỉ hiển thị cho người có thẩm quyền quản lý nhân sự.
- Hồ sơ nhân sự không tự cấp quyền đăng nhập, role hoặc permission; các quyền đó do EPIC 22 quản lý.
- Thay đổi dữ liệu nhạy cảm và trạng thái Customer phải có khả năng truy vết theo chính sách Audit.
- Chi tiết về mô hình dữ liệu, cơ chế phân quyền kỹ thuật, API endpoint, mã hóa và thời hạn lưu trữ thuộc tài liệu Architecture/Technical Design hoặc chính sách dữ liệu được phê duyệt.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-USER-01` | `BR-AUTH-04` | EPIC 01, 06, 08 | Customer chỉ quản lý được hồ sơ của mình; dữ liệu hợp lệ được sử dụng khi mua hàng. |
| `US-USER-02` | `BR-AUTH-04` | EPIC 01, 06 | Customer có nhiều địa chỉ, chọn được địa chỉ ở checkout và không xem/sửa địa chỉ người khác. |
| `US-USER-03` | `BR-AUTH-03`, `BR-AUTH-04` | EPIC 22, 23 | HR Manager quản lý hồ sơ nhân sự nhưng không tự thay đổi quyền System Access. |
| `US-USER-04` | `BR-AUTH-03`, `BR-AUTH-04` | EPIC 22 | Employee Status được xử lý duy nhất bởi `US-ADM-03`. |
| `US-USER-05` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 01, 22, 23 | Chỉ nhân viên được cấp quyền thay đổi trạng thái Customer; thay đổi có thể truy vết. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Trường nào là bắt buộc trong hồ sơ Customer và quy tắc định dạng/kiểm tra của từng trường.
- Khi Customer đổi email hoặc số điện thoại, có cần xác minh lại không và kênh xác minh nào được dùng.
- Chính sách địa chỉ: có hỗ trợ đặt một địa chỉ mặc định hay không; điều kiện xóa địa chỉ đang dùng trong đơn chờ xử lý.
- Danh sách trạng thái Customer, người được phép chuyển từng trạng thái, lý do bắt buộc và ảnh hưởng cụ thể của mỗi trạng thái đến đăng nhập/mua hàng.
- Chính sách bảo vệ, tra cứu và lưu trữ dữ liệu hồ sơ nhân sự; vai trò nào ngoài HR Manager được phép xem từng loại dữ liệu.
- Chính sách lưu trữ/xóa/ẩn danh dữ liệu cá nhân khi Customer yêu cầu hoặc khi tuân thủ quy định áp dụng.

## 9. UI/UX Reference

- Trang hồ sơ Customer với thông tin cá nhân và hành động lưu thay đổi.
- Trang sổ địa chỉ: danh sách, thêm, chỉnh sửa, xóa và chọn địa chỉ khi checkout.
- Màn hình quản lý hồ sơ nhân sự dành cho HR Manager.
- Màn hình tra cứu Customer và thao tác thay đổi trạng thái chỉ hiển thị cho nhân viên được phân quyền.

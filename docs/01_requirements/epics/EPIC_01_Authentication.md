# EPIC 01 — Authentication & Identity

## 1. Mục tiêu Epic

Epic này quản lý việc xác định danh tính khách hàng khi mua sắm trên kênh D2C. Hệ thống phải cho phép khách mua nhanh với tư cách Guest, đồng thời hỗ trợ khách đăng ký, đăng nhập và khôi phục quyền truy cập để quản lý thông tin, đơn hàng và lịch sử mua.

Epic không quản lý hồ sơ khách hàng, địa chỉ, lịch sử đơn chi tiết hay quyền nhân viên. Những nội dung đó thuộc các Epic Customer Profile, Order Management và User/Role/Permission Administration.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP v1.0** | `US-AUTH-01`, `US-AUTH-02`, `US-AUTH-03`, `US-AUTH-05` | Khách có thể mua không cần tài khoản hoặc dùng tài khoản để mua và theo dõi đơn. |
| **Giai đoạn 2** | `US-AUTH-04`, `US-AUTH-06` | Khách đăng nhập thuận tiện qua mạng xã hội và hợp nhất lịch sử đơn Guest. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Mua hàng không cần tạo tài khoản. |
| Registered Customer (`ACT-02`) | Đăng ký, đăng nhập, khôi phục quyền truy cập và quản lý đơn thuộc tài khoản. |
| Identity Provider (`EXT-05`) | Xác thực danh tính từ Google, Facebook hoặc Zalo khi tính năng Social Login được bật. |
| Notification Provider (`EXT-06`) | Gửi thông điệp xác minh/khôi phục khi quy trình yêu cầu. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-01` | Khách vãng lai được phép mua mà không bắt buộc tạo tài khoản. | Checkout không được chặn chỉ vì Guest chưa đăng nhập. |
| `BR-AUTH-02` | Khách có thể đăng ký bằng thông tin cá nhân hoặc Identity Provider được hỗ trợ. | Hỗ trợ đăng ký thường ở MVP; Social Login triển khai khi nhà cung cấp được hỗ trợ. |
| `BR-AUTH-03` | Tài khoản nội bộ phải được gán Role phù hợp trước khi vào quản trị. | Là ranh giới với Epic 22; không cấp quyền nhân viên trong Epic này. |
| `BR-AUTH-04` | Nhân viên chỉ thực hiện thao tác thuộc quyền được cấp. | Mọi màn hình/quy trình sau đăng nhập phải tôn trọng cơ chế phân quyền của Epic 22. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 02 — Customer & Employee Profile`: hồ sơ và địa chỉ để khách đã xác thực sử dụng khi checkout.
- `EPIC 06 — Checkout`: sử dụng trạng thái Guest hoặc Customer để cho phép đặt hàng.
- `EPIC 08 — Order Management`: hiển thị các đơn thuộc đúng khách hàng hoặc đơn Guest đã liên kết.
- `EPIC 22 — User / Role / Permission Administration`: quản lý tài khoản nhân viên, role và permission; không thuộc phạm vi Epic 01.
- `EPIC 28 — Omnichannel Notification`: gửi Email OTP, thông điệp khôi phục và các thông báo xác thực liên quan.

## 4. User Stories chi tiết

### US-AUTH-01 — Guest Checkout

**Actor:** Guest Customer (`ACT-01`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là một khách vãng lai, tôi muốn đặt hàng mà không cần tạo tài khoản để có thể mua sản phẩm nhanh chóng.

**Giá trị nghiệp vụ:** Giảm rào cản ở bước mua hàng đầu tiên; khách chưa sẵn sàng đăng ký vẫn có thể trở thành đơn hàng hợp lệ.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Guest bắt đầu checkout không cần đăng nhập
  Given khách chưa đăng nhập và giỏ hàng có sản phẩm hợp lệ
  When khách chọn tiếp tục đặt hàng với tư cách Guest
  Then hệ thống cho phép khách nhập thông tin cần thiết cho đơn hàng
  And không yêu cầu khách tạo tài khoản trước khi sang bước thanh toán

Scenario: Đơn Guest được tạo mà không tự tạo tài khoản
  Given Guest đã hoàn tất thông tin checkout và thanh toán theo quy trình hợp lệ
  When đơn hàng được tạo thành công
  Then đơn được ghi nhận với thông tin liên hệ của Guest theo quy định nghiệp vụ
  And hệ thống không tự động tạo tài khoản Customer nếu Guest không yêu cầu

Scenario: Guest vẫn tuân thủ điều kiện đặt hàng chung
  Given Guest đang checkout
  When sản phẩm không còn đủ tồn kho hoặc thông tin bắt buộc của đơn chưa hợp lệ
  Then hệ thống không xác nhận đơn hàng
  And hiển thị lý do để Guest có thể sửa hoặc chọn lại sản phẩm
```

### US-AUTH-02 — Customer Registration

**Actor:** Registered Customer (`ACT-02`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là một khách hàng, tôi muốn đăng ký tài khoản để lưu thông tin và theo dõi lịch sử mua hàng.

**Giá trị nghiệp vụ:** Chuyển khách mua một lần thành khách có thể quay lại, theo dõi đơn và sử dụng các dịch vụ cá nhân hóa ở các giai đoạn sau.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đăng ký tài khoản thành công
  Given khách cung cấp đầy đủ thông tin đăng ký hợp lệ
  When khách gửi yêu cầu đăng ký
  Then hệ thống tạo một tài khoản Customer mới
  And khách có thể dùng tài khoản đó để đăng nhập và truy cập thông tin/đơn hàng của mình

Scenario: Không tạo trùng tài khoản
  Given thông tin định danh đăng ký đã thuộc về một tài khoản Customer hiện có
  When khách gửi yêu cầu đăng ký
  Then hệ thống không tạo tài khoản thứ hai cho cùng thông tin định danh
  And hướng dẫn khách đăng nhập hoặc khôi phục quyền truy cập

Scenario: Từ chối thông tin đăng ký không hợp lệ
  Given khách bỏ trống hoặc nhập sai định dạng một trường bắt buộc
  When khách gửi yêu cầu đăng ký
  Then hệ thống chỉ rõ trường cần sửa
  And không tạo tài khoản khi dữ liệu chưa hợp lệ
```

### US-AUTH-03 — Login

**Actor:** Registered Customer (`ACT-02`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là một khách hàng, tôi muốn đăng nhập tài khoản để truy cập thông tin cá nhân và đơn hàng.

**Giá trị nghiệp vụ:** Khách có thể tiếp tục hành trình mua sắm, xem các dữ liệu thuộc về mình và tái sử dụng thông tin đã lưu.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đăng nhập bằng thông tin hợp lệ
  Given khách đã có tài khoản Customer hợp lệ
  When khách cung cấp đúng thông tin xác thực và gửi yêu cầu đăng nhập
  Then hệ thống xác thực thành công và thiết lập phiên đăng nhập
  And khách được truy cập hồ sơ, địa chỉ và các đơn hàng thuộc về chính mình

Scenario: Đăng nhập thất bại
  Given khách cung cấp thông tin xác thực không hợp lệ
  When khách gửi yêu cầu đăng nhập
  Then hệ thống không thiết lập phiên đăng nhập
  And hiển thị thông báo phù hợp cùng lựa chọn khôi phục quyền truy cập

Scenario: Bảo vệ dữ liệu của khách khác
  Given khách A đã đăng nhập
  When khách A yêu cầu truy cập hồ sơ hoặc đơn hàng chỉ thuộc về khách B
  Then hệ thống từ chối truy cập
  And không tiết lộ dữ liệu của khách B
```

### US-AUTH-04 — Social Login

**Actor:** Registered Customer (`ACT-02`), Identity Provider (`EXT-05`)  
**Ưu tiên:** Should Have  
**Phát hành:** Giai đoạn 2

> Là một khách hàng, tôi muốn đăng nhập bằng tài khoản Google, Facebook hoặc Zalo khi được hỗ trợ để không phải tạo mật khẩu mới.

**Giá trị nghiệp vụ:** Rút ngắn thời gian đăng nhập/đăng ký và giảm số khách bỏ dở vì không muốn nhớ thêm mật khẩu.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đăng nhập Social thành công
  Given một Identity Provider đang được hệ thống hỗ trợ
  When khách hoàn tất xác thực thành công tại Identity Provider
  Then hệ thống xác định hoặc tạo liên kết với đúng tài khoản Customer
  And thiết lập phiên đăng nhập cho khách

Scenario: Khách hủy hoặc xác thực thất bại ở Identity Provider
  Given khách đã chọn đăng nhập qua một Identity Provider được hỗ trợ
  When khách hủy hoặc Identity Provider trả về kết quả không thành công
  Then hệ thống không thiết lập phiên đăng nhập
  And không tạo tài khoản Customer chưa được xác thực

Scenario: Không tạo trùng tài khoản khi Social identity đã liên kết
  Given Social identity đã được liên kết với một Customer
  When khách đăng nhập lại bằng Social identity đó
  Then hệ thống đăng nhập vào Customer hiện có
  And không tạo Customer mới
```

### US-AUTH-05 — Password Recovery

**Actor:** Registered Customer (`ACT-02`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là một khách hàng, tôi muốn khôi phục quyền truy cập tài khoản khi quên mật khẩu.

**Giá trị nghiệp vụ:** Tránh mất khách và giảm nhu cầu hỗ trợ thủ công khi khách quên thông tin xác thực.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Khởi tạo khôi phục quyền truy cập
  Given khách cung cấp thông tin liên hệ đã được dùng cho tài khoản
  When khách yêu cầu khôi phục mật khẩu
  Then hệ thống gửi hướng dẫn hoặc mã xác minh dùng một lần qua kênh đã đăng ký
  And không làm lộ thông tin nhạy cảm của tài khoản

Scenario: Đặt lại mật khẩu với yêu cầu hợp lệ
  Given khách có yêu cầu khôi phục còn hiệu lực và đã xác minh thành công
  When khách đặt mật khẩu mới hợp lệ
  Then hệ thống cập nhật thông tin xác thực của tài khoản
  And khách có thể đăng nhập bằng mật khẩu mới

Scenario: Từ chối yêu cầu khôi phục không hợp lệ hoặc đã hết hạn
  Given mã hoặc liên kết khôi phục không hợp lệ, đã dùng hoặc đã hết hạn
  When khách gửi yêu cầu đặt lại mật khẩu
  Then hệ thống không thay đổi mật khẩu hiện tại
  And yêu cầu khách khởi tạo một lượt khôi phục mới
```

### US-AUTH-06 — Link Guest Order

**Actor:** Registered Customer (`ACT-02`)  
**Ưu tiên:** Should Have  
**Phát hành:** Giai đoạn 2

> Là một khách hàng đã từng mua với tư cách Guest, tôi muốn liên kết đơn hàng cũ với tài khoản mới để theo dõi lịch sử mua hàng.

**Giá trị nghiệp vụ:** Hợp nhất hành trình mua của khách; dữ liệu đơn Guest không bị tách rời khi khách quyết định đăng ký sau đó.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Liên kết đơn Guest thành công
  Given khách đã đăng nhập và đã xác minh quyền sở hữu thông tin liên hệ của đơn Guest
  When khách yêu cầu liên kết đơn Guest phù hợp với tài khoản
  Then hệ thống gắn đơn đó với tài khoản Customer
  And đơn xuất hiện trong lịch sử đơn hàng của khách

Scenario: Ngăn liên kết đơn không thuộc về khách
  Given khách không chứng minh được quyền sở hữu thông tin liên hệ của một đơn Guest
  When khách yêu cầu liên kết đơn đó
  Then hệ thống từ chối liên kết
  And đơn vẫn không xuất hiện trong lịch sử của khách

Scenario: Không tạo liên kết trùng lặp
  Given đơn Guest đã được liên kết với một Customer
  When có yêu cầu liên kết lại đơn đó
  Then hệ thống không tạo thêm liên kết hoặc lịch sử đơn trùng lặp
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-AUTH-01` | Cho phép khách chuyển từ giỏ hàng sang checkout mà không cần có phiên đăng nhập. | `US-AUTH-01` |
| `FR-AUTH-02` | Thu thập và kiểm tra các thông tin cần thiết để tạo đơn Guest theo quy định checkout. | `US-AUTH-01`, `EPIC 06` |
| `FR-AUTH-03` | Tạo tài khoản Customer từ thông tin đăng ký hợp lệ và ngăn tạo trùng theo thông tin định danh. | `US-AUTH-02` |
| `FR-AUTH-04` | Cho phép Customer đăng nhập, đăng xuất và duy trì phiên làm việc an toàn. | `US-AUTH-03` |
| `FR-AUTH-05` | Chỉ cho phép Customer đã xác thực truy cập hồ sơ, địa chỉ và đơn hàng thuộc về mình. | `US-AUTH-03` |
| `FR-AUTH-06` | Khởi tạo, xác minh và hoàn tất quy trình đặt lại mật khẩu; mã/liên kết khôi phục chỉ có hiệu lực theo chính sách bảo mật. | `US-AUTH-05` |
| `FR-AUTH-07` | Tích hợp đăng nhập với các Identity Provider đã được doanh nghiệp phê duyệt và liên kết identity ngoài với Customer. | `US-AUTH-04` |
| `FR-AUTH-08` | Cho phép Customer đã xác minh liên kết các đơn Guest thuộc về mình vào lịch sử đơn hàng. | `US-AUTH-06` |
| `FR-AUTH-09` | Không được tự động tạo Customer chỉ vì Guest hoàn tất đơn hàng. | `US-AUTH-01`, `BR-AUTH-01` |

## 6. Quy tắc bảo mật và dữ liệu

- Không lưu hoặc hiển thị mật khẩu ở dạng có thể đọc được.
- Không tiết lộ dữ liệu hồ sơ, đơn hàng hoặc trạng thái xác thực của khách cho người không sở hữu tài khoản/đơn đó.
- Thông điệp khôi phục quyền truy cập phải dùng kênh liên hệ đã được xác minh hoặc được chính sách nghiệp vụ cho phép.
- Mỗi yêu cầu đăng ký, đăng nhập, khôi phục và liên kết đơn phải được kiểm tra dữ liệu đầu vào, giới hạn thử lại và ghi nhận sự kiện phù hợp với chính sách Audit.
- Chi tiết về thuật toán, token, session, API endpoint, cơ sở dữ liệu và nhà cung cấp cụ thể thuộc tài liệu Architecture/Technical Design, không thuộc User Story Specification này.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-AUTH-01` | `BR-AUTH-01` | EPIC 05, 06, 08 | Guest đặt được đơn mà không tạo tài khoản và vẫn tuân thủ tồn kho/checkout. |
| `US-AUTH-02` | `BR-AUTH-02` | EPIC 02, 08, 28 | Tạo Customer hợp lệ, không trùng và có thể dùng tài khoản sau đăng ký. |
| `US-AUTH-03` | `BR-AUTH-02` | EPIC 02, 08, 22 | Đăng nhập đúng, không truy cập chéo dữ liệu khách khác. |
| `US-AUTH-04` | `BR-AUTH-02` | EXT-05, EPIC 02 | Social identity được liên kết chính xác, không tạo trùng Customer. |
| `US-AUTH-05` | `BR-AUTH-02` | EPIC 28 | Khôi phục hợp lệ cập nhật được mật khẩu; yêu cầu hết hạn bị từ chối. |
| `US-AUTH-06` | `BR-AUTH-01` | EPIC 08 | Chỉ liên kết đơn Guest khi khách xác minh được quyền sở hữu. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Thông tin định danh nào được dùng để đăng ký và khôi phục: email, số điện thoại hoặc cả hai.
- Chính sách mật khẩu, thời hạn hiệu lực và giới hạn thử lại của luồng khôi phục.
- Kênh gửi xác minh/khôi phục ở MVP và nhà cung cấp sẽ được phê duyệt.
- Danh sách Identity Provider được kích hoạt ở Giai đoạn 2 và quy tắc xử lý khi cùng một thông tin liên hệ xuất hiện ở nhiều provider.
- Cách Customer xác minh quyền sở hữu đơn Guest trước khi liên kết, bao gồm các trường hợp thay đổi email/số điện thoại.

## 9. UI/UX Reference

- Màn hình đăng ký, đăng nhập, quên mật khẩu và đặt lại mật khẩu.
- Lựa chọn “Tiếp tục với tư cách Guest” tại checkout.
- Nút đăng nhập qua Social Provider chỉ hiển thị khi provider đã được hỗ trợ.
- Màn hình/hành động liên kết đơn Guest sau khi Customer đăng nhập và hoàn tất xác minh.

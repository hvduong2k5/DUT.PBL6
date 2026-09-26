# EPIC 15 — Review & Rating

## 1. Mục tiêu Epic

Epic này cho phép khách hàng đã mua hàng chia sẻ đánh giá có thể xác minh theo đúng sản phẩm/SKU; đồng thời giúp CSKH phát hiện phản hồi tiêu cực, Sales Manager xử lý nội dung vi phạm và phân tích xu hướng chất lượng sản phẩm/dịch vụ.

Epic quản lý vòng đời Review, Rating, Review Media, nhãn Verified Review, phản hồi công khai của nhân viên và quyết định kiểm duyệt. Epic không quản lý dữ liệu Product/SKU gốc, trạng thái Order, quy trình Ticket hỗ trợ, hồ sơ đổi/trả hay phân tích hồ sơ khách hàng; các trách nhiệm đó lần lượt thuộc EPIC 04, 08, 16, 14 và 25.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 2** | `US-REV-01` đến `US-REV-06` | Khách đã mua hàng có thể đánh giá kèm hình ảnh; người xem nhận biết Verified Review; CSKH và Sales Manager theo dõi, xử lý, phản hồi và phân tích đánh giá theo quyền. |

Toàn bộ Epic có mức ưu tiên **Should Have** theo Product Backlog và không thuộc phạm vi MVP v1.0.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors

| Actor | Vai trò trong Epic |
| --- | --- |
| Registered Customer (`ACT-02`) | Tạo, xem và chỉnh sửa đánh giá cho sản phẩm/SKU đã mua trong phạm vi chính sách. |
| Guest Customer (`ACT-01`) | Xem Rating, Review và nhãn Verified đã được phép công khai. |
| Customer Service (`ACT-11`) | Theo dõi đánh giá tiêu cực, xem ngữ cảnh được phép và phản hồi/hỗ trợ khách hàng. |
| Sales Manager (`ACT-12`) | Kiểm duyệt nội dung vi phạm, phản hồi đánh giá và phân tích xu hướng đánh giá theo quyền. |
| Media / Object Storage (`EXT-09`) | Lưu trữ ảnh Review và cung cấp tài nguyên theo chính sách truy cập. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-REVIEW-01` | Chỉ khách hàng đã mua sản phẩm mới được tạo Verified Review. | Quyền tạo Review và nhãn Verified phải được xác định từ Order/Order Item hợp lệ, không dựa trên khai báo của khách. |
| `BR-REVIEW-02` | Review phải gắn với sản phẩm/Variant tương ứng. | Mỗi Review phải tham chiếu đúng Product/SKU trên dòng hàng đã mua. |
| `BR-REVIEW-03` | Nội dung vi phạm chính sách có thể bị ẩn hoặc xử lý bởi nhân viên có quyền. | Quyết định kiểm duyệt phải có lý do, phân quyền và khả năng truy vết. |
| `BR-REVIEW-04` | Rating bắt buộc nằm trong thang từ 1 đến 5 sao. | Không chấp nhận Rating thiếu hoặc ngoài miền hợp lệ. |
| `BR-REVIEW-05` | CSKH/Manager được phân quyền có thể Reply Review và phản hồi được hiển thị công khai. | Reply phải thể hiện nguồn nhân viên và tuân theo trạng thái công khai của Review. |
| `BR-REVIEW-06` | Khách hàng được chỉnh sửa Review trong một khoảng thời gian nhất định, ví dụ 30 ngày kể từ lúc đăng. | Cửa sổ chỉnh sửa phải được cấu hình và kiểm tra theo thời điểm đăng đầu tiên. |
| `BR-AUTH-04` | Nhân viên chỉ được thực hiện thao tác thuộc quyền được cấp. | Theo dõi, Reply, kiểm duyệt và xem phân tích phải được kiểm soát độc lập theo quyền. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Việc ẩn/khôi phục Review, Reply và thay đổi nội dung quan trọng phải có khả năng truy vết. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 04 — Product, Variant & SKU`: cung cấp Product/SKU và thông tin công khai; EPIC 15 chỉ gắn Review và số liệu Rating vào đúng đối tượng.
- `EPIC 08 — Order Management`: cung cấp Order Item, chủ sở hữu và trạng thái hoàn tất giao hàng để xác minh quyền đánh giá.
- `EPIC 14 — Return / Refund / Complaint`: quản lý đổi/trả, hoàn tiền và hồ sơ khiếu nại; Review không tự tạo quyết định đổi/trả.
- `EPIC 16 — Customer Service`: quản lý Ticket và quá trình hỗ trợ. EPIC 15 phát hiện/chuyển ngữ cảnh đánh giá tiêu cực nhưng không thay thế Ticket.
- `EPIC 22 — Administration` và `EPIC 23 — Audit & Security`: cung cấp Role/Permission và Audit Log cho Reply, kiểm duyệt và phân tích.
- `EPIC 25 — Customer Analytics`: có thể sử dụng dữ liệu Review đã được phép, nhưng phân tích hồ sơ/hành vi khách hàng không thuộc EPIC 15.
- `EPIC 28 — Notification`: gửi thông báo khi có Reply hoặc thay đổi trạng thái Review nếu chính sách yêu cầu.

## 4. User Stories chi tiết

### US-REV-01 — Đánh giá sản phẩm đã mua

**Actor:** Registered Customer (`ACT-02`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là khách hàng đã mua hàng, tôi muốn đánh giá sản phẩm để chia sẻ trải nghiệm.

**Giá trị nghiệp vụ:** Tạo nguồn phản hồi đáng tin cậy từ giao dịch thực tế, hỗ trợ người mua khác và giúp doanh nghiệp cải thiện chất lượng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo Review cho dòng hàng đủ điều kiện
  Given khách hàng đã đăng nhập và sở hữu một Order Item đủ điều kiện đánh giá
  When khách chọn Rating từ 1 đến 5 sao, nhập nội dung hợp lệ và gửi Review
  Then hệ thống tạo Review gắn với đúng Order Item, Product và SKU đã mua
  And Review được gắn nhãn Verified theo dữ liệu giao dịch
  And Rating tổng hợp được cập nhật theo Review đang được phép công khai

Scenario: Rating nằm ngoài thang hợp lệ
  Given khách hàng đang tạo hoặc chỉnh sửa Review
  When Rating bị thiếu hoặc nằm ngoài khoảng từ 1 đến 5 sao
  Then hệ thống từ chối lưu Review
  And hiển thị yêu cầu chọn Rating hợp lệ

Scenario: Khách không sở hữu sản phẩm
  Given khách hàng không có Order Item đủ điều kiện cho Product/SKU
  When khách cố tạo Review cho Product/SKU đó
  Then hệ thống từ chối tạo Review
  And không gắn nhãn Verified bằng dữ liệu do khách tự khai báo

Scenario: Tạo Review trùng cho cùng đơn vị mua hàng
  Given khách đã có Review cho Order Item theo quy tắc duy nhất đang áp dụng
  When khách gửi thêm một Review cho cùng đối tượng đó
  Then hệ thống không tạo Review trùng
  And hướng dẫn khách xem hoặc chỉnh sửa Review hiện có nếu còn trong thời hạn

Scenario: Chỉnh sửa Review trong thời hạn
  Given khách là chủ sở hữu Review và vẫn còn trong cửa sổ chỉnh sửa
  When khách cập nhật Rating hoặc nội dung bằng dữ liệu hợp lệ
  Then hệ thống lưu phiên bản mới của Review
  And cập nhật Rating tổng hợp nếu Rating công khai thay đổi
  And lưu thời điểm chỉnh sửa để truy vết

Scenario: Chỉnh sửa Review quá thời hạn
  Given cửa sổ chỉnh sửa Review đã kết thúc
  When khách yêu cầu thay đổi Rating hoặc nội dung
  Then hệ thống từ chối thay đổi trực tiếp
  And hiển thị hướng dẫn xử lý tiếp theo theo chính sách
```

### US-REV-02 — Đăng hình ảnh thực tế cùng Review

**Actor:** Registered Customer (`ACT-02`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn đăng hình ảnh thực tế cùng đánh giá.

**Giá trị nghiệp vụ:** Hình ảnh thực tế bổ sung bằng chứng trực quan cho trải nghiệm sử dụng và tăng giá trị tham khảo của Review.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đính kèm ảnh hợp lệ khi tạo Review
  Given khách đủ điều kiện tạo Review
  And ảnh đáp ứng loại tệp, dung lượng, số lượng và chính sách nội dung
  When khách tải ảnh lên và gửi Review hợp lệ
  Then hệ thống lưu ảnh qua Media Storage
  And liên kết ảnh với đúng Review
  And chỉ công khai ảnh khi ảnh ở trạng thái được phép hiển thị

Scenario: Tệp ảnh không hợp lệ
  Given khách đang tải ảnh cho Review
  When tệp sai định dạng, vượt dung lượng hoặc vượt số lượng được phép
  Then hệ thống từ chối tệp không hợp lệ
  And hiển thị lý do cùng giới hạn áp dụng

Scenario: Tải ảnh bị gián đoạn
  Given quá trình tải ảnh chưa hoàn tất
  When kết nối hoặc Media Storage gặp lỗi
  Then hệ thống không gắn tham chiếu ảnh hỏng vào Review công khai
  And cho phép khách thử lại mà không tạo Review trùng

Scenario: Thay đổi ảnh trong cửa sổ chỉnh sửa
  Given khách là chủ Review và vẫn còn trong thời hạn chỉnh sửa
  When khách thêm hoặc loại bỏ ảnh hợp lệ
  Then hệ thống cập nhật tập ảnh của đúng Review
  And tài nguyên không còn được sử dụng được xử lý theo chính sách lưu giữ

Scenario: Ảnh bị xác định vi phạm
  Given Review có ảnh đã được gửi
  When nhân viên có quyền xác định ảnh vi phạm chính sách
  Then hệ thống cho phép ẩn hoặc xử lý ảnh theo quyết định có lý do
  And không bắt buộc ẩn toàn bộ Review nếu phần nội dung còn lại không vi phạm
```

### US-REV-03 — Nhận biết Verified Review

**Actor:** Guest Customer, Registered Customer
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn biết đánh giá nào đến từ người đã mua hàng thực tế.

**Giá trị nghiệp vụ:** Người xem phân biệt được phản hồi đã được xác minh bằng giao dịch, tăng độ tin cậy khi lựa chọn sản phẩm.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị nhãn Verified cho Review hợp lệ
  Given Review được tạo từ Order Item đủ điều kiện của chính chủ
  When người xem mở danh sách hoặc chi tiết Review
  Then hệ thống hiển thị nhãn Verified Review
  And nhãn không tiết lộ mã đơn hoặc dữ liệu cá nhân của người đánh giá

Scenario: Không cho phép tự khai báo Verified
  Given một Review không có quan hệ xác minh hợp lệ với Order Item
  When Review được xử lý hoặc hiển thị
  Then hệ thống không gắn nhãn Verified chỉ từ nội dung do người dùng gửi

Scenario: Hiển thị đúng Product/SKU được đánh giá
  Given Review gắn với một SKU cụ thể đã mua
  When người xem xem Review trên trang sản phẩm
  Then hệ thống thể hiện SKU/Variant tương ứng theo dữ liệu được phép công khai
  And không gán Review sang SKU khác

Scenario: Bảo vệ danh tính người đánh giá
  Given Review được phép công khai
  When khách khác xem Review
  Then hệ thống chỉ hiển thị danh tính theo chính sách công khai
  And không hiển thị thông tin giao dịch hoặc liên hệ nhạy cảm
```

### US-REV-04 — Theo dõi đánh giá tiêu cực

**Actor:** Customer Service (`ACT-11`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là CSKH, tôi muốn xem các đánh giá tiêu cực để chủ động hỗ trợ khách hàng.

**Giá trị nghiệp vụ:** CSKH phát hiện sớm trải nghiệm không tốt và chuyển sang quy trình hỗ trợ phù hợp mà không bỏ sót hoặc xử lý trùng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem hàng đợi Review tiêu cực
  Given CSKH có quyền xem đánh giá tiêu cực
  When CSKH mở hàng đợi Review
  Then hệ thống hiển thị các Review đáp ứng ngưỡng tiêu cực đang áp dụng
  And cho phép lọc theo thời gian, Rating, Product/SKU và trạng thái xử lý

Scenario: Xem ngữ cảnh được phép của Review
  Given CSKH chọn một Review tiêu cực
  When hệ thống tải chi tiết
  Then CSKH thấy Review, Rating, Product/SKU, thời điểm và ngữ cảnh giao dịch cần thiết theo quyền
  And dữ liệu không cần thiết cho hỗ trợ không được hiển thị

Scenario: Chuyển sang quy trình hỗ trợ khách hàng
  Given một Review tiêu cực cần liên hệ hoặc theo dõi
  When CSKH khởi tạo hỗ trợ từ Review
  Then hệ thống chuyển tham chiếu Review và ngữ cảnh được phép sang EPIC 16
  And Review ghi nhận trạng thái đã chuyển hỗ trợ để hạn chế xử lý trùng

Scenario: Review tiêu cực đã có Ticket liên quan
  Given Review đã liên kết với một Ticket đang tồn tại
  When CSKH cố khởi tạo hỗ trợ lần nữa
  Then hệ thống cảnh báo về Ticket hiện có
  And không tạo Ticket trùng ngoài ý muốn

Scenario: Nhân viên không có quyền
  Given nhân viên không có quyền xem hàng đợi Review tiêu cực
  When nhân viên truy cập chức năng
  Then hệ thống từ chối truy cập
  And không tiết lộ nội dung hoặc ngữ cảnh khách hàng
```

### US-REV-05 — Quản lý Review vi phạm chính sách

**Actor:** Sales Manager (`ACT-12`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Sales Manager, tôi muốn quản lý và xử lý những đánh giá vi phạm chính sách.

**Giá trị nghiệp vụ:** Giữ môi trường đánh giá hữu ích và an toàn nhưng vẫn bảo toàn phản hồi trung thực, lý do xử lý và khả năng kiểm tra quyết định.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Ẩn Review vi phạm
  Given Sales Manager có quyền kiểm duyệt
  And Review vi phạm một chính sách hợp lệ
  When Sales Manager chọn ẩn Review và nhập lý do
  Then Review không còn hiển thị công khai
  And hệ thống lưu người xử lý, thời điểm, lý do và trạng thái trước/sau
  And Rating của Review bị ẩn không còn đóng góp vào Rating công khai

Scenario: Chỉ xử lý Media vi phạm
  Given chỉ một ảnh trong Review vi phạm chính sách
  When Sales Manager quyết định ẩn ảnh và giữ nội dung Review
  Then ảnh vi phạm không còn hiển thị công khai
  And phần Review hợp lệ vẫn được công khai theo trạng thái hiện tại

Scenario: Khôi phục Review sau khi xem xét
  Given Review đang bị ẩn và được xác định đủ điều kiện khôi phục
  When Sales Manager có quyền chọn khôi phục và ghi lý do
  Then Review được hiển thị lại theo chính sách
  And Rating tổng hợp được tính lại nhất quán
  And quyết định khôi phục được ghi nhận để truy vết

Scenario: Từ chối thao tác không có quyền
  Given nhân viên không có quyền kiểm duyệt Review
  When nhân viên cố ẩn hoặc khôi phục Review
  Then hệ thống từ chối thao tác
  And không thay đổi trạng thái Review

Scenario: Xử lý đồng thời trên Review đã thay đổi
  Given Review đã được người khác chỉnh sửa hoặc xử lý sau khi màn hình được tải
  When Sales Manager gửi quyết định dựa trên phiên bản cũ
  Then hệ thống không ghi đè âm thầm trạng thái mới hơn
  And yêu cầu tải lại dữ liệu trước khi quyết định

Scenario: Reply công khai vào Review
  Given CSKH hoặc Sales Manager có quyền Reply
  And Review đang được phép công khai
  When nhân viên gửi nội dung phản hồi hợp lệ
  Then hệ thống gắn Reply với đúng Review và danh tính vai trò nhân viên
  And Reply được hiển thị công khai theo chính sách
  And thao tác có khả năng truy vết

Scenario: Reply vào Review đang bị ẩn
  Given Review không còn được hiển thị công khai
  When nhân viên cố gửi Reply công khai
  Then hệ thống không công khai Reply tách rời Review
  And hiển thị trạng thái hiện tại để nhân viên xử lý phù hợp
```

### US-REV-06 — Phân tích đánh giá

**Actor:** Sales Manager (`ACT-12`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Manager, tôi muốn phân tích đánh giá để phát hiện vấn đề về chất lượng sản phẩm hoặc dịch vụ.

**Giá trị nghiệp vụ:** Tổng hợp phản hồi thành tín hiệu có thể hành động, giúp phát hiện Rating giảm, sản phẩm/SKU có vấn đề và xu hướng trải nghiệm theo thời gian.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem phân bố Rating
  Given Sales Manager có quyền xem phân tích Review
  When chọn khoảng thời gian và phạm vi sản phẩm
  Then hệ thống hiển thị số Review, Rating trung bình và phân bố từ 1 đến 5 sao
  And số liệu chỉ dùng Review thuộc phạm vi và trạng thái được quy định

Scenario: Phân tích theo Product và SKU
  Given Review được gắn đúng Product/SKU
  When Sales Manager lọc hoặc nhóm số liệu
  Then hệ thống cho phép so sánh kết quả theo Product và SKU
  And không gộp sai các SKU có trải nghiệm khác nhau

Scenario: Theo dõi xu hướng theo thời gian
  Given có dữ liệu Review trong nhiều kỳ
  When Sales Manager chọn kỳ phân tích
  Then hệ thống hiển thị xu hướng Rating và khối lượng Review theo thời gian
  And nêu rõ múi giờ, kỳ dữ liệu và thời điểm cập nhật số liệu

Scenario: Review bị ẩn sau khi báo cáo được tải
  Given một Review đóng góp vào số liệu trước đó nhưng sau đó bị ẩn
  When hệ thống tính lại báo cáo
  Then số liệu công khai loại Review đó theo quy tắc tổng hợp
  And kết quả nhất quán với Rating đang hiển thị cho khách

Scenario: Không có dữ liệu trong phạm vi lọc
  Given bộ lọc không có Review phù hợp
  When Sales Manager xem báo cáo
  Then hệ thống hiển thị trạng thái không có dữ liệu
  And không biểu diễn giá trị không có dữ liệu như Rating bằng 0 sao

Scenario: Người dùng không có quyền xem phân tích
  Given nhân viên không có quyền xem báo cáo Review
  When nhân viên truy cập chức năng
  Then hệ thống từ chối truy cập
  And không cung cấp dữ liệu tổng hợp ngoài phạm vi quyền
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-REV-01` | Xác định quyền tạo Review từ Registered Customer, Order Item thuộc chính chủ và điều kiện Order được cấu hình. | `US-REV-01`, `BR-REVIEW-01` |
| `FR-REV-02` | Gắn mỗi Review với đúng Order Item, Product và SKU/Variant đã mua; không nhận quan hệ do khách tự khai báo. | `US-REV-01`, `US-REV-03`, `BR-REVIEW-02` |
| `FR-REV-03` | Bắt buộc Rating là số nguyên thuộc thang 1–5 sao. | `US-REV-01`, `BR-REVIEW-04` |
| `FR-REV-04` | Ngăn tạo Review trùng theo đơn vị duy nhất được chốt và dẫn khách đến Review hiện có. | `US-REV-01` |
| `FR-REV-05` | Cho phép chủ sở hữu chỉnh sửa Review trong cửa sổ cấu hình tính từ lần đăng đầu tiên; lưu thời điểm và lịch sử cần thiết. | `US-REV-01`, `BR-REVIEW-06` |
| `FR-REV-06` | Tạo nhãn Verified Review từ quan hệ giao dịch hợp lệ và không tiết lộ dữ liệu Order nhạy cảm. | `US-REV-01`, `US-REV-03`, `BR-REVIEW-01` |
| `FR-REV-07` | Tiếp nhận ảnh Review theo giới hạn loại tệp, dung lượng và số lượng; chỉ liên kết tài nguyên tải lên hoàn tất. | `US-REV-02` |
| `FR-REV-08` | Quản lý trạng thái công khai của từng Review Media và cho phép xử lý Media vi phạm độc lập với phần Review còn lại. | `US-REV-02`, `US-REV-05`, `BR-REVIEW-03` |
| `FR-REV-09` | Hiển thị Review, Rating, nhãn Verified và SKU/Variant theo dữ liệu được phép công khai, đồng thời bảo vệ danh tính và dữ liệu giao dịch. | `US-REV-03` |
| `FR-REV-10` | Tính Rating tổng hợp từ tập Review hợp lệ theo quy tắc thống nhất và tính lại khi Review/Rating thay đổi trạng thái. | `US-REV-01`, `US-REV-05`, `US-REV-06` |
| `FR-REV-11` | Cung cấp hàng đợi Review tiêu cực cho CSKH với bộ lọc theo thời gian, Rating, Product/SKU và trạng thái xử lý. | `US-REV-04` |
| `FR-REV-12` | Cho phép chuyển Review tiêu cực cùng ngữ cảnh được phép sang quy trình Ticket của EPIC 16 và ngăn khởi tạo trùng ngoài ý muốn. | `US-REV-04`, `EPIC 16` |
| `FR-REV-13` | Cho phép nhân viên có quyền Reply Review; Reply gắn đúng Review, thể hiện vai trò nhân viên và tuân theo trạng thái công khai. | `US-REV-05`, `BR-REVIEW-05` |
| `FR-REV-14` | Cho phép người có quyền ẩn hoặc khôi phục Review/Media với lý do bắt buộc và không xóa cứng bằng chứng kiểm duyệt. | `US-REV-05`, `BR-REVIEW-03` |
| `FR-REV-15` | Ngăn ghi đè quyết định kiểm duyệt khi phiên bản Review đã thay đổi kể từ lúc nhân viên tải dữ liệu. | `US-REV-05` |
| `FR-REV-16` | Ghi Audit Log cho quyết định kiểm duyệt, khôi phục, Reply và các thay đổi quan trọng theo chính sách Audit. | `US-REV-01`, `US-REV-05`, `BR-AUDIT-01` |
| `FR-REV-17` | Cung cấp số Review, Rating trung bình, phân bố sao và xu hướng theo thời gian; hỗ trợ lọc/nhóm theo Product và SKU. | `US-REV-06` |
| `FR-REV-18` | Nêu rõ kỳ dữ liệu, múi giờ, thời điểm cập nhật và quy tắc trạng thái Review được đưa vào phân tích. | `US-REV-06` |
| `FR-REV-19` | Phân biệt trạng thái không có dữ liệu với Rating 0 và bảo đảm số liệu phân tích nhất quán với Rating công khai. | `US-REV-06` |
| `FR-REV-20` | Kiểm soát riêng quyền xem Review tiêu cực, Reply, kiểm duyệt và xem phân tích theo Role/Permission. | `US-REV-04~06`, `BR-AUTH-04` |

## 6. Vòng đời và quy tắc dữ liệu

### 6.1. Vòng đời Review

```text
ELIGIBLE ORDER ITEM
        │
        ▼
   PUBLISHED ───────► HIDDEN_POLICY
        ▲                    │
        └──── RESTORED ◄─────┘
```

- `PUBLISHED`: Review và các thành phần hợp lệ có thể hiển thị công khai và đóng góp vào Rating tổng hợp.
- `HIDDEN_POLICY`: Review bị ẩn khỏi công khai do quyết định kiểm duyệt có lý do; dữ liệu và lịch sử vẫn được bảo toàn.
- Chỉnh sửa trong thời hạn tạo phiên bản mới của cùng Review, không tạo Review độc lập và không đặt lại ngày bắt đầu cửa sổ chỉnh sửa.
- Nếu Review đang bị ẩn được khách chỉnh sửa, việc sửa nội dung không tự động khôi phục công khai; quy trình xét lại cần được chốt.

### 6.2. Quy tắc định danh và tính hợp lệ

- Review phải có định danh riêng và tham chiếu bất biến đến khách hàng, Order Item, Product và SKU/Variant tại thời điểm mua.
- Nhãn Verified là thuộc tính được suy ra từ giao dịch hợp lệ; không phải trường nội dung cho khách hoặc nhân viên tự bật.
- Điều kiện Order đủ để đánh giá phải được chốt rõ, thông thường dựa trên giao hàng hoàn tất. Không chỉ dựa vào việc Order từng được tạo hoặc thanh toán.
- Product/SKU ngừng bán không làm mất Review lịch sử; cách hiển thị Review trên catalog không còn công khai cần được xác định.
- Ảnh Review có vòng đời riêng để một tài nguyên vi phạm có thể bị ẩn mà không làm mất nội dung hợp lệ khác.

### 6.3. Rating tổng hợp và phân tích

- Rating công khai và báo cáo phải dùng cùng tập trạng thái Review được quy định; Review bị ẩn không được tiếp tục làm sai lệch Rating công khai.
- Rating trung bình phải đi kèm số lượng Review; không làm tròn hoặc hiển thị theo cách gây hiểu sai thang 1–5.
- Khi Rating được chỉnh sửa, Review bị ẩn hoặc được khôi phục, dữ liệu tổng hợp phải được cập nhật nhất quán và có cơ chế tránh đếm trùng.
- Phân tích theo SKU phải giữ khả năng tổng hợp lên Product nhưng không làm mất dấu SKU gây ra xu hướng bất thường.
- Nội dung phân tích nhằm đánh giá vấn đề sản phẩm/dịch vụ; việc suy luận hồ sơ hoặc phân khúc khách hàng thuộc EPIC 25 và phải tuân theo quyền riêng tư.

### 6.4. Kiểm duyệt, Reply và quyền riêng tư

- Quyết định ẩn/khôi phục phải lưu lý do, người thực hiện, thời điểm và trạng thái trước/sau; không dùng xóa cứng để che mất lịch sử.
- Reply của nhân viên phải tách biệt với nội dung Review, thể hiện rõ vai trò đại diện doanh nghiệp và không thể tồn tại công khai nếu Review gốc đang bị ẩn.
- Danh sách công khai không được để lộ Order ID, thông tin liên hệ, địa chỉ hoặc dữ liệu giao dịch nhạy cảm.
- CSKH chỉ nhận ngữ cảnh cần thiết để hỗ trợ; thao tác liên hệ, SLA và vòng đời Ticket do EPIC 16 quản lý.
- Các thay đổi đồng thời phải được phát hiện để tránh quyết định kiểm duyệt hoặc chỉnh sửa ghi đè lẫn nhau.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-REV-01` | `BR-REVIEW-01`, `BR-REVIEW-02`, `BR-REVIEW-04`, `BR-REVIEW-06` | EPIC 08, 23 | Chỉ chủ Order Item đủ điều kiện tạo Review 1–5 sao; gắn đúng SKU và chỉ sửa trong thời hạn. |
| `US-REV-02` | `BR-REVIEW-03`, `BR-REVIEW-06` | EXT-09, EPIC 23 | Ảnh hợp lệ gắn đúng Review; lỗi tải không tạo tham chiếu hỏng; Media vi phạm có thể bị xử lý riêng. |
| `US-REV-03` | `BR-REVIEW-01`, `BR-REVIEW-02` | EPIC 04, 08 | Nhãn Verified được suy ra từ giao dịch, đúng SKU và không làm lộ dữ liệu nhạy cảm. |
| `US-REV-04` | `BR-AUTH-04` | EPIC 16, 22 | CSKH có quyền xem đúng hàng đợi, nhận ngữ cảnh tối thiểu và không tạo Ticket trùng. |
| `US-REV-05` | `BR-REVIEW-03`, `BR-REVIEW-05`, `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 22, 23, 28 | Chỉ người có quyền được Reply/kiểm duyệt; quyết định có lý do, không ghi đè và có thể truy vết. |
| `US-REV-06` | `BR-AUTH-04` | EPIC 04, 22, 25 | Báo cáo đúng tập Review, nhất quán với công khai, phân tích được theo Product/SKU và thời gian. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Điều kiện Order Item đủ để đánh giá: `DELIVERED`, sau bao nhiêu thời gian và ảnh hưởng của hủy, hoàn tiền, đổi/trả hoặc giao một phần.
- Quy tắc duy nhất: một Review cho mỗi Order Item, mỗi SKU trong Order hay mỗi khách hàng–SKU; cách xử lý khách mua lại cùng SKU.
- Nội dung chữ là bắt buộc hay tùy chọn khi đã có Rating; độ dài tối thiểu/tối đa và chính sách ngôn ngữ/nội dung.
- Thời hạn chỉnh sửa chính thức thay cho ví dụ 30 ngày; khách có được xóa Review hay chỉ yêu cầu ẩn và quy tắc lưu giữ dữ liệu.
- Số lượng, định dạng, dung lượng, độ phân giải ảnh; có hỗ trợ video hay không dù Media Storage tổng thể có đề cập Review Images/Videos.
- Cơ chế kiểm duyệt: hậu kiểm hay tiền kiểm; danh mục lý do vi phạm, quy trình khiếu nại và quyền khôi phục.
- Ngưỡng xác định Review tiêu cực, SLA chuyển CSKH và điều kiện tự động tạo/gợi ý Ticket trong EPIC 16.
- Quy tắc Reply: số lượng Reply, quyền chỉnh sửa/ẩn, thời hạn và thông báo cho khách.
- Công thức Rating tổng hợp, độ chính xác/làm tròn, thời điểm cập nhật và cách xử lý Review bị ẩn hoặc Product/SKU ngừng bán.
- Phạm vi báo cáo của “Manager”, kỳ dữ liệu, quyền xuất dữ liệu và ngưỡng ẩn số liệu khi mẫu quá nhỏ.
- Chính sách hiển thị tên/avatar khách hàng và thời hạn lưu Review, Media, lịch sử chỉnh sửa, Audit Log.

## 9. UI/UX Reference

- Khu vực “Đánh giá đơn hàng” phải liệt kê đúng Order Item/SKU đủ điều kiện, trạng thái đã đánh giá và thời hạn còn có thể chỉnh sửa.
- Form Review cần hiển thị rõ thang 1–5 sao, giới hạn nội dung/ảnh, tiến trình tải lên và lỗi tại đúng trường.
- Danh sách công khai cần hiển thị Rating, nhãn Verified, SKU/Variant, thời điểm, Media và Reply của doanh nghiệp mà không lộ dữ liệu giao dịch.
- Hàng đợi CSKH cần có bộ lọc Rating, thời gian, Product/SKU, trạng thái hỗ trợ và chỉ báo Ticket đã liên kết.
- Màn hình kiểm duyệt cần hiển thị nội dung/Media, lịch sử phiên bản, lý do xử lý, trạng thái hiện tại và cảnh báo xung đột dữ liệu.
- Dashboard Review cần thể hiện Rating trung bình cùng số mẫu, phân bố sao, xu hướng và bộ lọc; trạng thái “không có dữ liệu” phải khác 0 sao.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

# EPIC 18 — B2B Sales & Wholesale

## 1. Mục tiêu Epic

Epic này hỗ trợ hành trình bán hàng doanh nghiệp từ yêu cầu báo giá theo số lượng, cung cấp logo/yêu cầu bao bì, trao đổi và phát hành phiên bản báo giá, đến khi khách chấp nhận báo giá để chuyển thành Order B2B. Thông tin pháp lý của doanh nghiệp được thu thập và chốt đúng thời điểm để phục vụ hóa đơn.

EPIC 18 sở hữu B2B Quote, Quote Version và quan hệ chuyển đổi sang Order. Epic không sở hữu Catalog/SKU, tồn kho, Payment, vòng đời Order, phát hành hóa đơn điện tử, Ticket CSKH hay lưu trữ vật lý của file; các trách nhiệm đó thuộc EPIC 04, 09, 07, 08, 24/EXT-11, 16 và EXT-09.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 2** | `US-B2B-01` đến `US-B2B-05` | Khách doanh nghiệp gửi yêu cầu, cung cấp logo/thông tin hóa đơn, nhận và chấp nhận báo giá; Sales Manager quản lý yêu cầu và chuyển báo giá hợp lệ thành Order B2B. |

Product Backlog chưa gán Priority riêng cho năm User Story của EPIC 18. Tài liệu giữ trạng thái **Chưa xác định — cần Product Owner chốt**, không kế thừa mức Should Have từ story cũ đã migrate.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và hệ thống ngoài

| Actor / hệ thống | Vai trò trong Epic |
| --- | --- |
| B2B Customer (`ACT-03`) | Đại diện doanh nghiệp gửi yêu cầu, bổ sung dữ liệu, nhận/chấp nhận báo giá và cung cấp thông tin hóa đơn. |
| Sales Manager (`ACT-12`) | Thẩm định, yêu cầu bổ sung, lập/phát hành phiên bản báo giá, từ chối hoặc chuyển đổi báo giá theo quyền. |
| Customer Service (`ACT-11`) | Giao tiếp/hỗ trợ qua Ticket nếu cần, không tự phê duyệt giá B2B ngoài thẩm quyền. |
| Accountant / Finance Staff (`ACT-19`) | Sử dụng thông tin doanh nghiệp và Order đã chốt để đối soát/phát hành hóa đơn theo phạm vi EPIC 24. |
| Notification Provider (`EXT-06`) | Gửi thông báo yêu cầu bổ sung, báo giá mới và kết quả chấp nhận qua EPIC 28. |
| Media / Object Storage (`EXT-09`) | Lưu file logo/tài liệu thiết kế và cung cấp theo quyền. |
| Tax / Invoice Provider (`EXT-11`) | Tiếp nhận dữ liệu hóa đơn từ luồng Finance/Order khi tích hợp. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-03` | Tài khoản nội bộ phải có Role phù hợp trước khi truy cập chức năng quản trị. | Sales Manager/Accountant phải có Role hợp lệ trước khi xử lý Quote hoặc dữ liệu hóa đơn. |
| `BR-AUTH-04` | Nhân viên chỉ thực hiện thao tác thuộc quyền được cấp. | Quyền xem yêu cầu, xem logo, lập/phát hành giá, chuyển đổi Order và xem dữ liệu thuế phải tách biệt. |
| `BR-PROD-01` | Một Product có thể có nhiều Variant/SKU. | Mỗi dòng yêu cầu/báo giá phải xác định đúng SKU hoặc ghi rõ đang chờ chốt SKU. |
| `BR-PROD-02` | Giá, khối lượng, quy cách và tồn kho được quản lý theo SKU. | Báo giá phải tham chiếu SKU và tách giá B2B đã duyệt khỏi giá catalog gốc. |
| `BR-PROD-04` | Thông tin sản phẩm hiển thị cho khách phải được quản lý/phê duyệt trước khi công khai. | Quote không được dùng dữ liệu Product/SKU nháp hoặc không được phép cung cấp cho khách. |
| `BR-BATCH-05` | Sản phẩm hết hạn không được bán. | Chấp nhận Quote không cho phép bỏ qua kiểm tra khả năng bán/hạn sử dụng khi tạo Order. |
| `BR-ORDER-01` | Order phải có trạng thái rõ ràng trong toàn bộ vòng đời. | B2B Order sau chuyển đổi do EPIC 08 quản lý; trạng thái Quote không thay thế trạng thái Order. |
| `BR-ORDER-02` | Không xác nhận Order vượt quá tồn khả dụng. | Quote không mặc nhiên bảo đảm tồn trừ khi có cơ chế reservation được chốt rõ. |
| `BR-ORDER-03` | Order chỉ chuyển Paid khi giao dịch được xác nhận hợp lệ. | Khách chấp nhận Quote không đồng nghĩa B2B Order đã thanh toán. |
| `BR-ORDER-05` | Thay đổi trạng thái quan trọng của Order phải được ghi nhận để truy vết. | Chuyển đổi Quote thành Order và snapshot điều khoản phải có khả năng đối soát. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Phát hành/sửa/thu hồi/chấp nhận Quote và truy cập dữ liệu nhạy cảm phải truy vết được. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 01, 02 — Authentication & Profile`: xác thực đại diện và hồ sơ liên hệ; việc xác minh pháp nhân/đại diện B2B cần được chốt.
- `EPIC 04 — Product, Variant & SKU`: cung cấp SKU và chính sách giá B2B cơ sở; EPIC 18 lưu mức giá/chiết khấu được duyệt trên từng Quote Version.
- `EPIC 06 — Checkout`: có thể được tái sử dụng để xác nhận dữ liệu giao nhận và tổng tiền theo Quote đã chấp nhận; không tự tính lại mất giá Quote hợp lệ.
- `EPIC 07, 08 — Payment & Order`: xử lý thanh toán và vòng đời Order sau khi Quote được chuyển đổi.
- `EPIC 09 — Inventory & Batch`: kiểm tra/giữ tồn theo chính sách; Quote không trực tiếp trừ tồn.
- `EPIC 10, 11 — Packing & Shipping`: thực hiện đóng gói tùy biến và giao đơn B2B sau khi Order hợp lệ.
- `EPIC 16 — Customer Service`: quản lý Ticket/hội thoại hỗ trợ; trạng thái và phiên bản Quote vẫn thuộc EPIC 18.
- `EPIC 17 — Promotion & Loyalty`: cung cấp ưu đãi chung nếu cho phép kết hợp; mức giá B2B đặc thù do Quote quản lý.
- `EPIC 24 — Finance` và `EXT-11`: quản lý nghĩa vụ thuế, phát hành/đối soát hóa đơn; EPIC 18 chỉ thu thập và chuyển snapshot dữ liệu cần thiết.
- `EPIC 28 — Notification`: gửi thông báo; EPIC 18 phát sinh sự kiện nghiệp vụ và không phụ thuộc việc lưu Quote vào kết quả gửi tin.

## 4. User Stories chi tiết

### US-B2B-01 — Gửi yêu cầu báo giá theo số lượng

**Actor:** B2B Customer (`ACT-03`)
**Ưu tiên:** Chưa xác định
**Phát hành:** Giai đoạn 2

> Là khách hàng doanh nghiệp, tôi muốn gửi yêu cầu báo giá theo số lượng.

**Giá trị nghiệp vụ:** Doanh nghiệp mô tả chính xác nhu cầu mua sỉ và nhận mã theo dõi để Sales Manager lập báo giá có căn cứ.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Gửi yêu cầu báo giá hợp lệ
  Given B2B Customer đã xác thực theo chính sách
  When khách chọn SKU, số lượng, thời gian mong muốn và nhập thông tin liên hệ bắt buộc
  Then hệ thống tạo B2B Quote Request có mã duy nhất và trạng thái `REQUESTED`
  And liên kết yêu cầu với đúng khách doanh nghiệp
  And hiển thị xác nhận tiếp nhận

Scenario: Yêu cầu có nhiều dòng sản phẩm
  Given khách cần báo giá nhiều SKU
  When khách gửi danh sách với số lượng hợp lệ cho từng dòng
  Then hệ thống lưu riêng từng SKU/số lượng trong cùng yêu cầu
  And không gộp các SKU khác nhau thành một dòng mơ hồ

Scenario: SKU hoặc số lượng không hợp lệ
  Given một dòng thiếu SKU, SKU không được phép công khai hoặc số lượng không hợp lệ
  When khách gửi yêu cầu
  Then hệ thống từ chối hoặc đánh dấu dòng cần sửa theo chính sách
  And không tự thay SKU khác

Scenario: Gửi lại yêu cầu do lỗi mạng
  Given yêu cầu trước đã được tiếp nhận nhưng khách chưa nhận phản hồi giao diện
  When cùng thao tác được gửi lại trong phạm vi chống trùng
  Then hệ thống không tạo nhiều Quote Request ngoài ý muốn
  And trả về mã yêu cầu đã tiếp nhận nếu xác định được an toàn

Scenario: Khách xem yêu cầu của chính mình
  Given khách đã đăng nhập và sở hữu Quote Request
  When khách mở chi tiết
  Then hệ thống hiển thị dữ liệu, trạng thái và trao đổi được phép
  And không hiển thị ghi chú nội bộ hoặc điều khoản chưa phát hành

Scenario: Truy cập yêu cầu của doanh nghiệp khác
  Given Quote Request không thuộc doanh nghiệp/đại diện hiện tại
  When người dùng mở bằng mã hoặc liên kết trực tiếp
  Then hệ thống từ chối
  And không tiết lộ sản phẩm, số lượng, giá hoặc thông tin doanh nghiệp
```

### US-B2B-02 — Tải logo cho yêu cầu thiết kế bao bì

**Actor:** B2B Customer (`ACT-03`)
**Ưu tiên:** Chưa xác định
**Phát hành:** Giai đoạn 2

> Là khách hàng doanh nghiệp, tôi muốn tải logo công ty để yêu cầu thiết kế bao bì.

**Giá trị nghiệp vụ:** Sales có đúng tài sản thương hiệu và yêu cầu tùy biến để đánh giá phạm vi, chi phí và thời gian trong Quote.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tải logo hợp lệ
  Given khách sở hữu Quote Request còn cho phép bổ sung
  And tệp đáp ứng loại, kích thước, số lượng và chính sách an toàn
  When khách tải logo cùng mô tả yêu cầu bao bì
  Then hệ thống lưu tệp qua Media Storage
  And liên kết đúng phiên bản tệp với Quote Request
  And thể hiện trạng thái tải thành công

Scenario: Tệp không hợp lệ
  Given khách đang tải logo hoặc tài liệu thiết kế
  When tệp sai định dạng, vượt dung lượng/số lượng hoặc không đạt kiểm tra an toàn
  Then hệ thống từ chối tệp
  And hiển thị lý do cùng giới hạn được hỗ trợ

Scenario: Tải tệp bị gián đoạn
  Given quá trình tải chưa hoàn tất
  When kết nối hoặc Media Storage gặp lỗi
  Then hệ thống không gắn tham chiếu hỏng như tài liệu đã nhận
  And cho phép thử lại mà không tạo bản ghi trùng

Scenario: Thay thế logo trước khi Quote được phát hành
  Given khách được phép cập nhật tài liệu và đã có logo trước đó
  When khách tải phiên bản mới
  Then hệ thống đánh dấu rõ phiên bản hiện hành
  And giữ lịch sử cần thiết để tránh Sales dùng nhầm tệp

Scenario: Thay logo sau khi đã có Quote
  Given Quote Version đã được phát hành dựa trên một phiên bản logo
  When khách thay logo hoặc yêu cầu thiết kế
  Then hệ thống không âm thầm sửa căn cứ của Quote đã phát hành
  And đánh dấu cần xem xét/phát hành Quote Version mới nếu thay đổi ảnh hưởng điều khoản

Scenario: Người không có quyền truy cập logo
  Given người dùng không thuộc doanh nghiệp hoặc nhân viên không có quyền
  When truy cập tệp trực tiếp
  Then hệ thống từ chối
  And không tiết lộ vị trí lưu trữ
```

### US-B2B-03 — Quản lý yêu cầu báo giá B2B

**Actor:** Sales Manager (`ACT-12`)
**Ưu tiên:** Chưa xác định
**Phát hành:** Giai đoạn 2

> Là Sales Manager, tôi muốn quản lý yêu cầu báo giá B2B để phản hồi khách hàng.

**Giá trị nghiệp vụ:** Sales thẩm định nhu cầu, lập các điều khoản giá/tùy biến/giao hàng có phiên bản rõ ràng và phản hồi đúng hạn.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem hàng đợi yêu cầu báo giá
  Given Sales Manager có quyền B2B Quote
  When mở danh sách yêu cầu
  Then hệ thống hiển thị trạng thái, khách doanh nghiệp, giá trị/số lượng dự kiến, người phụ trách và thời gian chờ
  And hỗ trợ lọc theo trạng thái, người phụ trách và thời gian

Scenario: Yêu cầu khách bổ sung thông tin
  Given Quote Request thiếu dữ liệu để thẩm định
  When Sales Manager ghi rõ nội dung cần bổ sung
  Then yêu cầu chuyển sang `NEEDS_INFO`
  And thông tin được chuyển qua kênh giao tiếp/Notification phù hợp
  And lịch sử yêu cầu được bảo toàn

Scenario: Lập Quote Version hợp lệ
  Given yêu cầu có đủ SKU, số lượng và dữ liệu cần thiết
  When Sales Manager nhập đơn giá B2B, chiết khấu, thuế/phí, giao hàng, tùy biến, hiệu lực và điều khoản hợp lệ
  Then hệ thống tạo Quote Version ở trạng thái `DRAFT`
  And tổng tiền được tính nhất quán từ các dòng và điều khoản

Scenario: Phát hành Quote cho khách
  Given Quote Version đầy đủ và Sales Manager có quyền phát hành
  When Sales Manager xác nhận phát hành
  Then phiên bản chuyển sang `SENT`
  And khách chỉ nhìn thấy phiên bản đã phát hành
  And hệ thống lưu actor cùng thời điểm

Scenario: Sửa Quote đã phát hành
  Given một Quote Version đã được gửi cho khách
  When Sales Manager thay giá hoặc điều khoản
  Then hệ thống tạo phiên bản mới thay vì sửa mất nội dung cũ
  And phiên bản cũ được đánh dấu `SUPERSEDED` khi phiên bản mới phát hành theo chính sách

Scenario: Từ chối yêu cầu báo giá
  Given yêu cầu không thể đáp ứng và Sales Manager có quyền
  When Sales Manager từ chối với lý do
  Then Quote Request chuyển `REJECTED`
  And lý do được lưu và phản hồi cho khách ở mức được phép

Scenario: Hai nhân viên cập nhật đồng thời
  Given Quote Request hoặc Draft đã thay đổi sau khi một Sales Manager tải dữ liệu
  When người đó lưu trên phiên bản cũ
  Then hệ thống không ghi đè âm thầm thay đổi mới hơn
  And yêu cầu tải lại/đối chiếu trước khi tiếp tục

Scenario: Nhân viên không có quyền phát hành
  Given nhân viên được xem nhưng không có quyền duyệt/phát hành giá
  When nhân viên cố gửi Quote cho khách
  Then hệ thống từ chối
  And Quote vẫn ở trạng thái hiện tại
```

### US-B2B-04 — Xác nhận báo giá để chuyển sang đặt hàng

**Actor:** B2B Customer (`ACT-03`)
**Ưu tiên:** Chưa xác định
**Phát hành:** Giai đoạn 2

> Là khách hàng doanh nghiệp, tôi muốn xác nhận báo giá để chuyển sang đặt hàng.

**Giá trị nghiệp vụ:** Ý chí chấp nhận của khách được gắn với đúng phiên bản/điều khoản, tạo Order B2B nhất quán và tránh chuyển đổi trùng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Chấp nhận Quote Version còn hiệu lực
  Given khách sở hữu Quote và phiên bản `SENT` còn hiệu lực
  When đại diện có quyền xem lại điều khoản và xác nhận chấp nhận
  Then hệ thống ghi nhận đúng người, thời điểm và Quote Version được chấp nhận
  And chuyển yêu cầu sang bước tạo/hoàn tất Order B2B

Scenario: Quote đã hết hạn
  Given Quote Version đã quá thời hạn hiệu lực
  When khách cố chấp nhận
  Then hệ thống từ chối chuyển đổi
  And hướng dẫn yêu cầu Sales phát hành báo giá mới

Scenario: Khách chấp nhận phiên bản đã bị thay thế
  Given Quote Version cũ đã `SUPERSEDED`
  When khách xác nhận từ liên kết cũ
  Then hệ thống không chấp nhận điều khoản cũ
  And hiển thị phiên bản hiện hành được phép

Scenario: Khả năng bán thay đổi trước khi tạo Order
  Given Quote còn hiệu lực nhưng SKU, tồn hoặc điều kiện bán đã thay đổi
  When hệ thống chuẩn bị tạo Order
  Then kiểm tra lại khả năng bán và tồn theo EPIC 04/09
  And không tạo Order vượt tồn hoặc chứa hàng không được phép bán
  And đưa Quote về luồng xử lý lại nếu điều kiện không còn đáp ứng

Scenario: Tạo Order từ Quote thành công
  Given Quote đã được chấp nhận và toàn bộ điều kiện tạo Order hợp lệ
  When hệ thống chuyển đổi
  Then tạo đúng một Order nguồn B2B tham chiếu Quote/Quote Version
  And Order lưu snapshot SKU, số lượng, giá, chiết khấu, thuế/phí, tùy biến và điều khoản cần thiết
  And Quote Request chuyển `CONVERTED`

Scenario: Gửi lặp yêu cầu chấp nhận
  Given Quote đã được chấp nhận hoặc chuyển đổi thành Order
  When khách gửi lại cùng yêu cầu
  Then hệ thống không tạo thêm Order
  And trả về kết quả chấp nhận/Order hiện có nếu khách có quyền

Scenario: Chấp nhận Quote không đồng nghĩa thanh toán
  Given Order đã được tạo từ Quote
  When chưa có kết quả Payment hợp lệ
  Then Order không được đánh dấu `PAID`
  And tiếp tục theo điều khoản thanh toán B2B đã chốt

Scenario: Người không có quyền chấp nhận
  Given người dùng không thuộc doanh nghiệp hoặc không có thẩm quyền đại diện theo chính sách
  When người dùng chấp nhận Quote
  Then hệ thống từ chối
  And không thay đổi Quote hoặc tạo Order
```

### US-B2B-05 — Cung cấp thông tin doanh nghiệp cho hóa đơn

**Actor:** B2B Customer (`ACT-03`)
**Ưu tiên:** Chưa xác định
**Phát hành:** Giai đoạn 2

> Là khách hàng doanh nghiệp, tôi muốn cung cấp thông tin doanh nghiệp để nhận hóa đơn phù hợp.

**Giá trị nghiệp vụ:** Order B2B có snapshot thông tin xuất hóa đơn đầy đủ, giảm sai sót và hỗ trợ Finance phát hành hóa đơn đúng đơn vị.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Cung cấp thông tin hóa đơn hợp lệ
  Given khách đang chuẩn bị chấp nhận Quote hoặc hoàn tất Order B2B
  When nhập tên pháp lý, mã số thuế, địa chỉ, người nhận và kênh nhận hóa đơn hợp lệ
  Then hệ thống lưu thông tin hóa đơn cho đúng doanh nghiệp/ngữ cảnh
  And cho khách kiểm tra lại trước khi xác nhận Order

Scenario: Thiếu dữ liệu bắt buộc
  Given chính sách hóa đơn yêu cầu các trường xác định
  When khách bỏ thiếu hoặc nhập dữ liệu sai định dạng
  Then hệ thống không cho hoàn tất bước yêu cầu hóa đơn
  And chỉ rõ trường cần sửa mà không tự suy đoán giá trị pháp lý

Scenario: Chọn hồ sơ doanh nghiệp đã lưu
  Given khách có quyền đối với nhiều hồ sơ xuất hóa đơn
  When khách chọn một hồ sơ cho Quote/Order
  Then hệ thống hiển thị đầy đủ dữ liệu để xác nhận
  And không tự chọn nhầm pháp nhân chỉ dựa trên hồ sơ dùng gần nhất

Scenario: Chốt snapshot thông tin hóa đơn trên Order
  Given Quote được chuyển thành Order với thông tin doanh nghiệp hợp lệ
  When Order được tạo
  Then hệ thống lưu snapshot thông tin hóa đơn gắn với Order
  And thay đổi hồ sơ doanh nghiệp sau đó không tự sửa Order cũ

Scenario: Yêu cầu sửa thông tin sau khi Order đã tạo
  Given Order đã tồn tại hoặc hóa đơn đã được phát hành
  When khách yêu cầu thay đổi thông tin hóa đơn
  Then hệ thống không âm thầm sửa snapshot/chứng từ đã phát sinh
  And chuyển sang quy trình điều chỉnh phù hợp với trạng thái hóa đơn

Scenario: Chuyển dữ liệu sang Finance/Invoice Provider
  Given Order đạt điều kiện phát hành hóa đơn
  When EPIC 24 hoặc luồng hóa đơn yêu cầu dữ liệu
  Then hệ thống cung cấp snapshot đúng Order cùng tham chiếu cần thiết
  And không đánh dấu đã phát hành chỉ vì đã gửi yêu cầu ra ngoài

Scenario: Invoice Provider thất bại
  Given yêu cầu phát hành đã được gửi nhưng Provider lỗi hoặc chưa xác nhận
  When hệ thống nhận kết quả
  Then trạng thái hóa đơn thể hiện đang chờ/thất bại theo Epic chịu trách nhiệm
  And dữ liệu Quote/Order đã chốt không bị mất hoặc tạo lại

Scenario: Người dùng truy cập hồ sơ doanh nghiệp ngoài quyền
  Given hồ sơ thuộc doanh nghiệp khác
  When người dùng tìm hoặc chọn hồ sơ
  Then hệ thống từ chối
  And không tiết lộ mã số thuế, địa chỉ hay thông tin liên hệ
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-B2B-01` | Cho B2B Customer hợp lệ tạo Quote Request có mã duy nhất với SKU/số lượng, nhu cầu giao, tùy biến và liên hệ bắt buộc. | `US-B2B-01`, `BR-PROD-01`, `BR-PROD-02` |
| `FR-B2B-02` | Hỗ trợ nhiều dòng SKU trong một yêu cầu; kiểm tra dữ liệu và chống tạo yêu cầu trùng khi gửi lại. | `US-B2B-01` |
| `FR-B2B-03` | Chỉ cho khách xem/cập nhật Quote Request thuộc doanh nghiệp/phạm vi đại diện của mình. | `US-B2B-01`, `BR-AUTH-04` |
| `FR-B2B-04` | Nhận logo/tài liệu thiết kế theo giới hạn loại, dung lượng, số lượng và trạng thái kiểm tra an toàn; chỉ liên kết tệp tải thành công. | `US-B2B-02` |
| `FR-B2B-05` | Quản lý phiên bản logo/tài liệu, đánh dấu bản hiện hành và không làm thay đổi căn cứ Quote đã phát hành. | `US-B2B-02`, `US-B2B-03` |
| `FR-B2B-06` | Kiểm soát quyền xem/tải file B2B và không tiết lộ vị trí lưu trữ trực tiếp cho người không có quyền. | `US-B2B-02`, `BR-AUTH-04` |
| `FR-B2B-07` | Cung cấp hàng đợi Quote Request với trạng thái, khách, giá trị/số lượng dự kiến, người phụ trách, thời gian và bộ lọc. | `US-B2B-03` |
| `FR-B2B-08` | Cho Sales Manager yêu cầu bổ sung hoặc từ chối với lý do và chuyển nội dung giao tiếp qua EPIC 16/28. | `US-B2B-03` |
| `FR-B2B-09` | Cho lập Quote Version với SKU/số lượng, giá/chiết khấu, thuế/phí, giao hàng, tùy biến, hiệu lực và điều khoản. | `US-B2B-03`, `BR-PROD-02` |
| `FR-B2B-10` | Chỉ phát hành Quote Version đầy đủ bởi actor có quyền; khách không thấy Draft/ghi chú nội bộ. | `US-B2B-03`, `BR-AUTH-04` |
| `FR-B2B-11` | Mọi thay đổi điều khoản đã phát hành tạo phiên bản mới; giữ phiên bản cũ và quan hệ supersede để đối soát. | `US-B2B-03`, `BR-AUDIT-01` |
| `FR-B2B-12` | Phát hiện cập nhật đồng thời để không ghi đè Quote Request/Draft mới hơn. | `US-B2B-03` |
| `FR-B2B-13` | Chỉ cho đúng đại diện có quyền chấp nhận Quote Version `SENT` còn hiệu lực và lưu bằng chứng chấp nhận. | `US-B2B-04`, `BR-AUTH-04` |
| `FR-B2B-14` | Từ chối chấp nhận phiên bản hết hạn, bị thu hồi/thay thế hoặc không còn thuộc khách hiện tại. | `US-B2B-04` |
| `FR-B2B-15` | Kiểm tra lại SKU, trạng thái bán và tồn trước chuyển đổi; Quote không tự vượt quy tắc khả dụng. | `US-B2B-04`, `BR-PROD-04`, `BR-BATCH-05`, `BR-ORDER-02` |
| `FR-B2B-16` | Tạo đúng một Order nguồn B2B tham chiếu Quote Version và lưu snapshot toàn bộ điều khoản thương mại cần thiết. | `US-B2B-04`, `BR-ORDER-01`, `BR-ORDER-05` |
| `FR-B2B-17` | Không đánh dấu Order `PAID` chỉ từ việc chấp nhận Quote; chuyển sang EPIC 07 theo điều khoản thanh toán. | `US-B2B-04`, `BR-ORDER-03` |
| `FR-B2B-18` | Thu thập và kiểm tra trường thông tin doanh nghiệp/hóa đơn bắt buộc theo chính sách áp dụng. | `US-B2B-05` |
| `FR-B2B-19` | Cho chọn hồ sơ hóa đơn trong phạm vi quyền và yêu cầu khách xác nhận, không tự suy đoán pháp nhân. | `US-B2B-05`, `BR-AUTH-04` |
| `FR-B2B-20` | Lưu snapshot thông tin hóa đơn trên Order; thay đổi hồ sơ sau đó không sửa hồi tố Order/chứng từ. | `US-B2B-05` |
| `FR-B2B-21` | Cung cấp snapshot đúng Order cho EPIC 24/EXT-11 và theo dõi kết quả riêng; gửi yêu cầu không đồng nghĩa phát hành thành công. | `US-B2B-05`, `EXT-11` |
| `FR-B2B-22` | Ghi Audit cho phát hành/thu hồi/thay thế/chấp nhận Quote, chuyển đổi Order và thao tác nhạy cảm theo chính sách. | `US-B2B-03~05`, `BR-AUDIT-01` |
| `FR-B2B-23` | Phát sinh sự kiện thông báo mà không phụ thuộc việc lưu/chuyển trạng thái Quote vào kết quả gửi tin. | `US-B2B-01~05`, `EPIC 28` |

## 6. Vòng đời và quy tắc dữ liệu

### 6.1. Vòng đời Quote Request

```text
REQUESTED → UNDER_REVIEW ⇄ NEEDS_INFO
                  │
                  ├──► REJECTED
                  └──► QUOTED → ACCEPTED → CONVERTED
                              ├──► EXPIRED
                              └──► WITHDRAWN
```

- `REQUESTED`: hệ thống đã tiếp nhận yêu cầu.
- `UNDER_REVIEW`: Sales đang thẩm định/lập báo giá.
- `NEEDS_INFO`: đang chờ khách bổ sung; lịch sử yêu cầu trước được giữ nguyên.
- `QUOTED`: có ít nhất một Quote Version `SENT` còn hiệu lực cho khách.
- `ACCEPTED`: khách đã chấp nhận đúng một phiên bản nhưng Order có thể chưa tạo xong.
- `CONVERTED`: đã liên kết thành công với một Order B2B duy nhất.
- `REJECTED`, `EXPIRED`, `WITHDRAWN`: kết thúc theo lý do tương ứng; mở lại hoặc báo giá mới phải theo chính sách rõ ràng.

### 6.2. Vòng đời Quote Version

```text
DRAFT → SENT → ACCEPTED
          ├──► SUPERSEDED
          ├──► EXPIRED
          └──► WITHDRAWN
```

- Chỉ một Quote Version hiện hành được phép chấp nhận tại một thời điểm.
- Phiên bản đã `SENT` là bằng chứng điều khoản đã cung cấp; không chỉnh sửa tại chỗ.
- Chấp nhận phải lưu version, actor/đại diện, doanh nghiệp, thời điểm, kênh và bằng chứng xác nhận được phép.
- Việc Quote hết hạn không tự hủy Order đã tạo hợp lệ từ Quote trước đó.

### 6.3. Quy tắc giá, tồn và chuyển đổi Order

- Dòng Quote phải gắn đúng SKU và lưu mô tả/snapshot cần thiết để giải thích báo giá khi catalog thay đổi.
- Tổng Quote phải tách tiền hàng, chiết khấu, chi phí tùy biến, vận chuyển dự kiến, thuế/phí và tổng cuối theo chính sách.
- Giá B2B được duyệt không sửa giá catalog; thứ tự giữa giá B2B, promotion và các chiết khấu khác phải được chốt.
- Phát hành Quote không mặc nhiên giữ tồn. Nếu doanh nghiệp cam kết tồn/năng lực sản xuất, cần trạng thái reservation và thời hạn riêng qua EPIC 09.
- Chuyển đổi phải chống trùng: một lần xác nhận hoặc retry không tạo nhiều Order cho cùng Quote Version.
- Order phải ghi nguồn `B2B`, Quote/Version, snapshot thương mại, yêu cầu tùy biến và thông tin hóa đơn; vòng đời sau đó thuộc EPIC 08.

### 6.4. File và dữ liệu doanh nghiệp

- File logo/tài liệu phải có mã, chủ sở hữu doanh nghiệp, Quote Request, phiên bản, tên/loại/kích thước, trạng thái xử lý và tham chiếu lưu trữ.
- Quyền sử dụng logo, phạm vi sử dụng, thời hạn lưu và trách nhiệm xác nhận bản quyền cần được chốt; việc tải lên không tự chứng minh quyền sở hữu trí tuệ.
- Thông tin hóa đơn cần phân biệt hồ sơ doanh nghiệp hiện tại với snapshot trên Quote/Order.
- Dữ liệu thuế, địa chỉ, liên hệ và file thương hiệu chỉ được dùng cho mục đích được phép và không hiển thị chéo doanh nghiệp.
- Phát hành hóa đơn có trạng thái/kết quả riêng; không gộp vào trạng thái `CONVERTED` của Quote.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-B2B-01` | `BR-PROD-01`, `BR-PROD-02`, `BR-AUTH-04` | EPIC 01, 02, 04 | Yêu cầu đúng SKU/số lượng, mã duy nhất, chống trùng và không truy cập chéo doanh nghiệp. |
| `US-B2B-02` | `BR-AUTH-04`, `BR-AUDIT-01` | EXT-09, EPIC 23 | File hợp lệ gắn đúng yêu cầu/phiên bản; lỗi tải không tạo tham chiếu hỏng; truy cập theo quyền. |
| `US-B2B-03` | `BR-PROD-02`, `BR-AUTH-03`, `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 04, 16, 22, 23, 28 | Manager quản lý đúng quyền; Quote có phiên bản, hiệu lực, điều khoản và không bị ghi đè đồng thời. |
| `US-B2B-04` | `BR-PROD-04`, `BR-BATCH-05`, `BR-ORDER-01~03`, `BR-ORDER-05` | EPIC 06–09 | Chỉ phiên bản hiện hành được chấp nhận; kiểm tra lại tồn; tạo một Order snapshot và chưa tự `PAID`. |
| `US-B2B-05` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 08, 24, EXT-11 | Dữ liệu hóa đơn hợp lệ, đúng doanh nghiệp, được chốt trên Order và kết quả phát hành theo dõi riêng. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Quy trình đăng ký/xác minh B2B Customer, pháp nhân và thẩm quyền của từng đại diện; một tài khoản có thể đại diện nhiều doanh nghiệp hay không.
- Các trường bắt buộc của Quote Request, ngưỡng “mua sỉ”, MOQ theo SKU và có cho yêu cầu SKU/chủng loại chưa xác định không.
- SLA phản hồi, ma trận phân công, quyền lập giá/phát hành/thu hồi và ngưỡng cần phê duyệt nhiều cấp.
- Công thức giá B2B/chiết khấu, biên lợi nhuận tối thiểu, đơn vị tiền tệ, làm tròn, thuế, vận chuyển và chi phí tùy biến.
- Quote có giữ tồn/năng lực sản xuất không; nếu có, thời điểm, số lượng, TTL và cách giải phóng khi hết hạn/từ chối.
- Điều khoản thanh toán B2B: trả trước, đặt cọc, công nợ, nhiều đợt; EPIC 07/24 hiện cần mở rộng gì để hỗ trợ.
- Điều kiện giao hàng nhiều đợt/địa chỉ, lead time sản xuất bao bì và tác động đến Packing/Shipping.
- Chính sách thay đổi Quote sau khi khách chấp nhận nhưng trước khi Order tạo; tranh chấp khi SKU/tồn/giá thay đổi.
- Loại, dung lượng, số lượng file; kiểm tra an toàn, preview file vector, phiên bản và thời hạn lưu/xóa logo.
- Xác nhận quyền sử dụng thương hiệu, quy trình duyệt thiết kế/mẫu in và bằng chứng khách chấp thuận thiết kế cuối.
- Bộ trường hóa đơn theo quy định áp dụng, nguồn xác minh mã số thuế và quy trình sửa/hủy/thay thế hóa đơn.
- Quan hệ giữa Quote và Ticket EPIC 16; trao đổi nào là bằng chứng chính thức thuộc Quote và trao đổi nào chỉ là hội thoại hỗ trợ.
- Có cho kết hợp giá Quote với coupon/Loyalty không; chính sách kênh B2B so với giá trong EPIC 04/17.
- Priority chính thức cho `US-B2B-01` đến `US-B2B-05` vì Product Backlog đang để trống.

## 9. UI/UX Reference

- Form Quote Request cần hỗ trợ nhiều dòng SKU/số lượng, thời gian giao, tùy biến, ghi chú và hiển thị mã sau khi gửi.
- Khu vực tải logo phải nêu định dạng/dung lượng, tiến trình, trạng thái kiểm tra và phiên bản đang dùng.
- Sales Workspace cần có hàng đợi, SLA/người phụ trách, ngữ cảnh doanh nghiệp, dòng hàng, tài liệu và lịch sử phiên bản Quote.
- Quote Builder phải tách đơn giá, chiết khấu, tùy biến, vận chuyển, thuế/phí, tổng và ngày hết hiệu lực; preview đúng nội dung khách nhận.
- Customer Quote View cần làm nổi bật version, hiệu lực, điều khoản, thông tin hóa đơn và hành động chấp nhận; liên kết cũ phải cảnh báo khi superseded.
- Màn hình chuyển đổi cần kiểm tra lại SKU/tồn và hiển thị lỗi theo từng dòng; chấp nhận không được trình bày như đã thanh toán.
- Form hóa đơn cần cho chọn hồ sơ hợp lệ, xem lại toàn bộ snapshot và cảnh báo rằng thay đổi sau phát hành cần quy trình điều chỉnh.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

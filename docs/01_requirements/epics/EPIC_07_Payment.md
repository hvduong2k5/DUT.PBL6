# EPIC 07 — Payment

## 1. Mục tiêu Epic

Epic này cho phép khách thanh toán Order bằng phương thức được hỗ trợ, theo dõi trạng thái thanh toán và giúp nhân viên có thẩm quyền kiểm tra, đối soát giao dịch với Order. Hệ thống phải ghi nhận kết quả thanh toán chính xác, không xử lý trùng một giao dịch và không chuyển Order sang `PAID` khi chưa có bằng chứng xác nhận hợp lệ.

Payment quản lý phương thức, lần thử thanh toán, tham chiếu giao dịch và trạng thái tài chính của giao dịch. Trạng thái Payment phải được phân biệt với trạng thái vòng đời Order: một Payment thành công có thể làm Order chuyển sang `PAID`, nhưng việc đóng gói, giao hàng, hủy hoặc hoàn tất Order thuộc các Epic khác.

Epic không tính lại tổng tiền của checkout, giữ tồn kho, quản lý vòng đời Order, gửi thông báo đa kênh, thực hiện quy trình duyệt hoàn tiền hoặc lập báo cáo tài chính tổng hợp. Các nội dung đó lần lượt thuộc EPIC 06, EPIC 09, EPIC 08, EPIC 28, EPIC 14 và EPIC 24.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 1st** | `US-PAY-01` đến `US-PAY-03` | Khách thanh toán bằng chuyển khoản/QR hoặc COD khi được hỗ trợ và biết trạng thái thanh toán của Order. |
| **MVP — ưu tiên 2nd** | `US-PAY-04`, `US-PAY-05` | Nhân viên kiểm tra được trạng thái và quản lý đối soát giao dịch với Order để phát hiện sai lệch. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Thanh toán Order Guest và xem trạng thái bằng cơ chế xác minh quyền sở hữu phù hợp. |
| Registered Customer (`ACT-02`) | Thanh toán và xem trạng thái Payment của Order thuộc tài khoản mình. |
| Sales Manager (`ACT-12`) | Kiểm tra trạng thái thanh toán phục vụ xử lý Order và xem kết quả đối soát theo quyền được cấp. |
| Accountant / Finance Staff (`ACT-19`) | Thực hiện nghiệp vụ đối soát và xử lý sai lệch tài chính theo thẩm quyền. |
| Payment Provider (`EXT-01`) | Tạo hoặc xử lý phiên thanh toán và cung cấp kết quả giao dịch. |
| Bank / Payment Confirmation (`EXT-02`) | Cung cấp bằng chứng xác nhận chuyển khoản hoặc thu tiền hợp lệ. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-ORDER-01` | Đơn hàng phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Payment phải cung cấp kết quả rõ ràng để EPIC 08 chuyển trạng thái Order đúng quy tắc. |
| `BR-ORDER-03` | Đơn hàng chỉ chuyển sang Paid khi giao dịch được xác nhận hợp lệ. | Trang quay lại từ cổng thanh toán hoặc thông báo chưa được xác minh không đủ để đánh dấu `PAID`. |
| `BR-ORDER-04` | Khách hàng chỉ được tự hủy đơn trong thời gian được phép. | Việc hủy Order đang có Payment phải tuân theo điều kiện Order và quy trình hoàn tiền nếu tiền đã được thu. |
| `BR-ORDER-05` | Mọi thay đổi trạng thái quan trọng của đơn hàng phải được ghi nhận để truy vết. | Kết quả Payment làm thay đổi Order phải lưu được nguồn, thời điểm và tham chiếu giao dịch. |
| `BR-REFUND-01` | Chỉ Sales Manager hoặc cấp cao hơn mới được duyệt yêu cầu hoàn tiền. | Payment không tự hoàn tiền chỉ vì Order bị hủy; phải nhận kết quả phê duyệt từ EPIC 14. |
| `BR-REFUND-02` | Hệ thống hỗ trợ hoàn tiền toàn phần hoặc một phần. | Payment phải có khả năng thực thi và ghi nhận số tiền hoàn đã được phê duyệt. |
| `BR-REFUND-03` | Tiền hoàn được trả theo phương thức gốc hoặc quy đổi theo thỏa thuận hợp lệ. | Kênh thực hiện hoàn tiền phải tuân theo quyết định của quy trình Refund. |
| `BR-REFUND-04` | Các thao tác hoàn tiền phải được lưu Audit Log. | Mọi lần thực hiện hoặc nhận kết quả hoàn tiền phải có khả năng truy vết. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Điều chỉnh hoặc xử lý ngoại lệ Payment/đối soát phải được ghi nhận. |

### 3.3. Phụ thuộc giữa Epic và hệ thống ngoài

- `EPIC 06 — Checkout`: cung cấp Order hợp lệ, tổng tiền đã chốt và phương thức thanh toán khách chọn.
- `EPIC 08 — Order Management`: là nguồn Order, quản lý điều kiện hiệu lực và thực hiện chuyển trạng thái dựa trên kết quả Payment.
- `EPIC 09 — Inventory & Batch`: quản lý reservation; Payment chỉ cung cấp kết quả để luồng Order quyết định giữ hoặc nhả tồn.
- `EPIC 14 — Return / Refund / Complaint`: phê duyệt và xác định số tiền/phương thức hoàn; Payment thực thi giao dịch hoàn khi được yêu cầu hợp lệ.
- `EPIC 23 — Audit & Security`: ghi nhận thao tác nhạy cảm, điều chỉnh và xử lý sai lệch.
- `EPIC 24 — Finance & Business Analytics`: sử dụng dữ liệu Payment đã đối soát để lập báo cáo tài chính.
- `EPIC 28 — Omnichannel Notification`: gửi thông báo thanh toán; Payment chỉ cung cấp trạng thái/sự kiện nghiệp vụ.
- `EXT-01 — Payment Provider` và `EXT-02 — Bank / Payment Confirmation`: cung cấp phiên, tham chiếu và kết quả giao dịch cần được xác minh.

## 4. User Stories chi tiết

### US-PAY-01 — Thanh toán bằng chuyển khoản hoặc QR

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP

> Là khách hàng, tôi muốn thanh toán bằng chuyển khoản/QR để hoàn tất đơn hàng thuận tiện.

**Giá trị nghiệp vụ:** Khách thanh toán nhanh với đúng số tiền và nội dung tham chiếu; doanh nghiệp có thể liên kết chính xác tiền nhận được với Order.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Khởi tạo thanh toán chuyển khoản hoặc QR
  Given Order hợp lệ đang chờ thanh toán
  And tổng tiền của Order đã được chốt
  When khách chọn phương thức chuyển khoản hoặc QR đang được hỗ trợ
  Then hệ thống tạo một lần thử thanh toán gắn với đúng Order
  And hiển thị hướng dẫn hoặc dữ liệu thanh toán gồm đúng số tiền và tham chiếu duy nhất cần thiết
  And không cho client tự thay đổi số tiền phải trả

Scenario: Xác nhận giao dịch thành công hợp lệ
  Given Payment đang chờ kết quả cho một Order còn hiệu lực
  When hệ thống nhận và xác minh được xác nhận thành công từ nguồn thanh toán tin cậy
  Then Payment được ghi nhận thành công đúng một lần
  And EPIC 08 nhận kết quả để chuyển đúng Order sang `PAID`
  And tham chiếu cùng thời điểm giao dịch được lưu để tra cứu và đối soát

Scenario: Không xác nhận Paid từ trang quay lại của khách
  Given khách được đưa trở lại hệ thống sau khi thao tác tại cổng thanh toán
  When chưa có kết quả giao dịch được xác minh từ nguồn tin cậy
  Then hệ thống không đánh dấu Payment thành công
  And không yêu cầu EPIC 08 chuyển Order sang `PAID`
  And hiển thị trạng thái đang chờ hoặc hướng dẫn kiểm tra lại phù hợp

Scenario: Xử lý thông báo giao dịch bị gửi lặp
  Given một giao dịch thành công đã được ghi nhận
  When nguồn thanh toán gửi lại cùng định danh giao dịch
  Then hệ thống không ghi nhận tiền lần thứ hai
  And không lặp lại thay đổi trạng thái Order gây sai lệch
  And vẫn duy trì kết quả nhất quán cho nguồn gửi xác nhận

Scenario: Số tiền hoặc Order tham chiếu không khớp
  Given hệ thống nhận được một xác nhận thanh toán
  When số tiền, đơn vị tiền tệ hoặc tham chiếu Order không khớp dữ liệu phải thu
  Then hệ thống không tự xác nhận Order là `PAID`
  And ghi nhận giao dịch vào diện cần đối soát
  And bảo toàn dữ liệu nhận được để nhân viên có quyền kiểm tra

Scenario: Thanh toán thất bại hoặc hết hạn
  Given Order đang chờ thanh toán bằng chuyển khoản hoặc QR
  When lần thử thanh toán thất bại, bị hủy hoặc hết thời hạn
  Then hệ thống hiển thị đúng trạng thái cho khách
  And chỉ cho tạo lần thử mới khi Order và reservation vẫn còn đủ điều kiện
```

### US-PAY-02 — Thanh toán COD

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP

> Là khách hàng, tôi muốn thanh toán COD nếu phương thức này được hỗ trợ.

**Giá trị nghiệp vụ:** Khách có thể đặt hàng khi chưa thuận tiện thanh toán trực tuyến, trong khi doanh nghiệp vẫn kiểm soát được nghĩa vụ thu tiền khi giao hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Chọn COD cho Order đủ điều kiện
  Given phương thức COD được hỗ trợ cho địa chỉ, giá trị Order và lựa chọn giao hàng hiện tại
  When khách chọn COD và xác nhận đặt hàng
  Then Order ghi nhận COD là phương thức thanh toán
  And Payment ghi nhận nghĩa vụ thu đúng tổng tiền khi giao
  And Order có thể chuyển sang fulfillment theo chính sách COD mà không bị đánh dấu `PAID`

Scenario: COD không khả dụng
  Given COD không được hỗ trợ cho địa chỉ, giá trị Order, sản phẩm hoặc lựa chọn giao hàng hiện tại
  When khách chọn hoặc tiếp tục với COD
  Then hệ thống thông báo COD không khả dụng
  And yêu cầu khách chọn phương thức thanh toán hợp lệ khác

Scenario: Xác nhận thu tiền COD thành công
  Given Order COD đã được giao và có nghĩa vụ thu tiền
  When hệ thống nhận được xác nhận thu tiền hợp lệ
  Then Payment được ghi nhận thành công với đúng số tiền đã thu
  And Order được cập nhật trạng thái thanh toán theo quy tắc đã phê duyệt
  And dữ liệu thu tiền sẵn sàng cho đối soát

Scenario: Giao COD không thành công
  Given Order COD chưa thu được tiền
  When giao hàng thất bại hoặc Order bị trả về
  Then Payment không được ghi nhận thành công
  And trạng thái Order/Return được xử lý bởi EPIC 08, EPIC 11 và EPIC 14
```

### US-PAY-03 — Xem trạng thái thanh toán

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP

> Là khách hàng, tôi muốn biết trạng thái thanh toán của đơn hàng.

**Giá trị nghiệp vụ:** Khách biết đã thanh toán thành công hay cần tiếp tục xử lý, giảm thanh toán lặp và giảm yêu cầu hỗ trợ không cần thiết.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Customer xem trạng thái Payment của Order
  Given Customer đã đăng nhập và sở hữu một Order
  When Customer mở thông tin thanh toán của Order đó
  Then hệ thống hiển thị trạng thái Payment hiện tại cùng phương thức và số tiền liên quan
  And không hiển thị dữ liệu nhạy cảm không cần thiết của phương thức thanh toán

Scenario: Guest xem trạng thái bằng thông tin xác minh hợp lệ
  Given Guest đã tạo một Order
  When Guest cung cấp thông tin tra cứu và bằng chứng sở hữu hợp lệ
  Then hệ thống hiển thị trạng thái Payment của đúng Order
  And không cho phép tra cứu Order Guest khác bằng cách đoán mã đơn

Scenario: Payment đang chờ xác nhận
  Given khách đã thực hiện thanh toán nhưng kết quả chưa được xác minh
  When khách xem trạng thái hoặc tải lại trang
  Then hệ thống hiển thị trạng thái đang chờ xác nhận
  And không trình bày giao dịch như đã thành công

Scenario: Order có nhiều lần thử thanh toán
  Given Order có một lần thử thất bại và một lần thử mới hơn
  When khách xem trạng thái thanh toán
  Then hệ thống hiển thị trạng thái tổng hợp nhất quán của Order
  And không làm khách hiểu nhầm lần thử thất bại cũ là nghĩa vụ thanh toán bổ sung
```

### US-PAY-04 — Nhân viên kiểm tra trạng thái thanh toán

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là nhân viên bán hàng, tôi muốn kiểm tra trạng thái thanh toán để xử lý đơn chính xác.

**Giá trị nghiệp vụ:** Nhân viên chỉ đưa Order sang bước vận hành phù hợp khi tình trạng thu tiền đáp ứng chính sách của phương thức thanh toán.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem chi tiết thanh toán của Order
  Given Sales Manager có quyền xem Order và Payment
  When Sales Manager mở thông tin thanh toán của một Order
  Then hệ thống hiển thị phương thức, số tiền phải thu, số tiền đã xác nhận, trạng thái và tham chiếu cần thiết
  And phân biệt rõ thanh toán trực tuyến, chuyển khoản và nghĩa vụ thu COD

Scenario: Cảnh báo trạng thái Payment không phù hợp với Order
  Given trạng thái hoặc số tiền Payment không khớp với Order
  When Sales Manager kiểm tra Order
  Then hệ thống hiển thị cảnh báo sai lệch
  And không trình bày Order như đã thanh toán đầy đủ
  And cho phép chuyển trường hợp đó sang quy trình đối soát

Scenario: Nhân viên không có quyền xem Payment
  Given một nhân viên không có quyền truy cập thông tin thanh toán
  When nhân viên cố xem Payment của Order
  Then hệ thống từ chối truy cập
  And không tiết lộ dữ liệu giao dịch

Scenario: Không tự ý đánh dấu Paid
  Given Sales Manager đang xem một Payment chưa được xác nhận hợp lệ
  When Sales Manager cố đưa Order sang `PAID` ngoài quy trình được phê duyệt
  Then hệ thống từ chối thao tác thông thường
  And yêu cầu xử lý theo quy trình ngoại lệ có thẩm quyền và Audit nếu chính sách cho phép
```

### US-PAY-05 — Đối soát giao dịch với Order

**Actor:** Sales Manager (`ACT-12`), Accountant / Finance Staff (`ACT-19`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là quản lý, tôi muốn đối soát giao dịch thanh toán với đơn hàng để phát hiện sai lệch.

**Giá trị nghiệp vụ:** Doanh nghiệp phát hiện khoản thu thiếu, thừa, trùng hoặc không xác định được Order trước khi số liệu được dùng cho vận hành và báo cáo tài chính.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đối soát giao dịch khớp Order
  Given có dữ liệu giao dịch từ nguồn thanh toán và dữ liệu Order tương ứng
  When nhân viên có quyền thực hiện đối soát
  Then hệ thống so sánh tham chiếu, số tiền, đơn vị tiền tệ và trạng thái liên quan
  And đánh dấu kết quả khớp khi các điều kiện bắt buộc đều phù hợp

Scenario: Phát hiện giao dịch sai lệch
  Given một giao dịch thiếu, thừa, trùng, không có Order hoặc có tham chiếu không khớp
  When hệ thống hoặc nhân viên thực hiện đối soát
  Then giao dịch được đưa vào danh sách sai lệch với lý do tương ứng
  And Order không bị tự động sửa thành trạng thái tài chính không có căn cứ

Scenario: Xử lý một sai lệch theo thẩm quyền
  Given một sai lệch đang chờ xử lý
  And nhân viên có quyền thực hiện hành động xử lý tương ứng
  When nhân viên ghi nhận kết quả và lý do nghiệp vụ
  Then hệ thống cập nhật trạng thái đối soát
  And lưu actor, thời điểm, dữ liệu trước/sau và lý do để Audit
  And không làm mất dữ liệu giao dịch gốc dùng làm bằng chứng

Scenario: Ngăn xử lý sai lệch ngoài thẩm quyền
  Given nhân viên chỉ có quyền xem kết quả đối soát
  When nhân viên cố điều chỉnh hoặc đóng một sai lệch
  Then hệ thống từ chối thao tác
  And dữ liệu Payment, Order và kết quả đối soát không bị thay đổi
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-PAY-01` | Cung cấp các phương thức thanh toán đang khả dụng cho Order theo chính sách áp dụng. | `US-PAY-01`, `US-PAY-02` |
| `FR-PAY-02` | Tạo lần thử thanh toán gắn với đúng Order, tổng tiền, đơn vị tiền tệ và tham chiếu duy nhất. | `US-PAY-01` |
| `FR-PAY-03` | Cung cấp hướng dẫn hoặc dữ liệu chuyển khoản/QR chính xác và không chấp nhận số tiền do client tự quyết định. | `US-PAY-01` |
| `FR-PAY-04` | Tiếp nhận và xác minh kết quả giao dịch từ nguồn thanh toán tin cậy trước khi ghi nhận thành công. | `US-PAY-01`, `BR-ORDER-03` |
| `FR-PAY-05` | Bảo đảm một giao dịch hoặc thông báo lặp chỉ được ghi nhận tác động tài chính một lần. | `US-PAY-01` |
| `FR-PAY-06` | Không xác nhận `PAID` khi số tiền, đơn vị tiền tệ, tham chiếu Order hoặc tính hợp lệ của nguồn xác nhận không khớp. | `US-PAY-01`, `BR-ORDER-03` |
| `FR-PAY-07` | Cho phép tạo lại lần thử thanh toán chỉ khi Order và reservation còn đủ điều kiện. | `US-PAY-01`, `EPIC 08`, `EPIC 09` |
| `FR-PAY-08` | Cho phép chọn COD khi Order đáp ứng chính sách và ghi nhận nghĩa vụ thu tiền mà không đánh dấu trước là `PAID`. | `US-PAY-02`, `BR-ORDER-03` |
| `FR-PAY-09` | Tiếp nhận kết quả thu COD hợp lệ và liên kết đúng khoản thu với Order để cập nhật và đối soát. | `US-PAY-02`, `EPIC 08`, `EPIC 11` |
| `FR-PAY-10` | Cho Customer hoặc Guest đã xác minh quyền sở hữu xem trạng thái Payment của đúng Order. | `US-PAY-03` |
| `FR-PAY-11` | Hiển thị trạng thái tổng hợp nhất quán khi Order có nhiều lần thử Payment. | `US-PAY-03` |
| `FR-PAY-12` | Cho nhân viên có quyền xem phương thức, số tiền, trạng thái và tham chiếu Payment cần thiết để xử lý Order. | `US-PAY-04` |
| `FR-PAY-13` | Phát hiện và cảnh báo sự không nhất quán giữa trạng thái hoặc số tiền Payment với Order. | `US-PAY-04`, `US-PAY-05` |
| `FR-PAY-14` | Đối soát giao dịch với Order theo tham chiếu, số tiền, đơn vị tiền tệ và trạng thái; phân loại các trường hợp sai lệch. | `US-PAY-05` |
| `FR-PAY-15` | Kiểm soát quyền xử lý sai lệch và ghi Audit cho mọi điều chỉnh, kết quả cùng lý do nghiệp vụ. | `US-PAY-04`, `US-PAY-05`, `BR-AUDIT-01` |
| `FR-PAY-16` | Cung cấp dữ liệu Payment đã xác nhận và kết quả đối soát cho Finance nhưng không thay thế nghiệp vụ báo cáo của EPIC 24. | `US-PAY-05`, `EPIC 24` |
| `FR-PAY-17` | Thực thi và ghi nhận giao dịch hoàn tiền toàn phần hoặc một phần khi nhận yêu cầu đã được phê duyệt hợp lệ từ EPIC 14. | `BR-REFUND-01~04`, `EPIC 14` |

## 6. Quy tắc dữ liệu và an toàn nghiệp vụ

- Payment Transaction là bản ghi một giao dịch tài chính với nguồn ngoài; Payment Attempt là một lần khách thử thanh toán. Một Order có thể có nhiều Attempt nhưng không được thu vượt nghĩa vụ thanh toán nếu chưa có quy trình ngoại lệ hợp lệ.
- Trạng thái Payment và trạng thái Order là hai khái niệm riêng. Payment chỉ cung cấp kết quả; EPIC 08 chịu trách nhiệm áp dụng chuyển trạng thái Order.
- Chỉ xác nhận Payment thành công từ bằng chứng đã được kiểm tra của `EXT-01` hoặc `EXT-02`. Kết quả hiển thị trên trình duyệt của khách không phải bằng chứng quyết định.
- Dữ liệu nhận từ nguồn thanh toán phải được kiểm tra tính xác thực, toàn vẹn, định danh giao dịch, Order tham chiếu, số tiền và đơn vị tiền tệ.
- Xử lý lặp phải cho cùng một kết quả nghiệp vụ và không được ghi nhận khoản thu, cập nhật Order hoặc kích hoạt tác động hạ nguồn nhiều lần.
- Không lưu hoặc hiển thị dư thừa thông tin tài chính nhạy cảm. Mỗi actor chỉ được xem dữ liệu cần thiết theo quyền và mục đích nghiệp vụ.
- Giao dịch gốc và bằng chứng đối soát phải được bảo toàn; thao tác điều chỉnh không được ghi đè làm mất dữ liệu ban đầu.
- Hoàn tiền chỉ được thực hiện sau quyết định hợp lệ từ EPIC 14 và phải liên kết với Payment gốc, số tiền được duyệt cùng kết quả từ nhà cung cấp.
- Chi tiết endpoint, chữ ký số cụ thể, cơ sở dữ liệu, hàng đợi, bộ nhớ khóa, thời gian phản hồi và cơ chế cập nhật realtime thuộc Architecture, Security Design hoặc NFR.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-PAY-01` | `BR-ORDER-01`, `BR-ORDER-03`, `BR-ORDER-05` | EPIC 06, 08, 09; EXT-01/02 | Tạo Payment đúng Order/tổng tiền; chỉ xác nhận từ nguồn hợp lệ; thông báo lặp không thu tiền hoặc cập nhật Order hai lần. |
| `US-PAY-02` | `BR-ORDER-01`, `BR-ORDER-03` | EPIC 06, 08, 11 | Chỉ hiện COD khi đủ điều kiện; COD chưa thu không phải `PAID`; kết quả thu tiền được liên kết đúng Order. |
| `US-PAY-03` | `BR-ORDER-01` | EPIC 01, 08 | Khách xem đúng trạng thái Payment của Order mình sở hữu; trạng thái chờ không hiển thị như thành công. |
| `US-PAY-04` | `BR-ORDER-03`, `BR-AUDIT-01` | EPIC 08, 22, 23 | Nhân viên có quyền xem được dữ liệu cần thiết; sai lệch được cảnh báo; không tự ý đánh dấu `PAID`. |
| `US-PAY-05` | `BR-ORDER-05`, `BR-AUDIT-01` | EPIC 08, 23, 24 | Giao dịch được đối soát và phân loại; xử lý theo đúng quyền, có lý do và Audit; dữ liệu gốc được bảo toàn. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Danh sách phương thức và nhà cung cấp được bật trong MVP: VietQR trực tiếp, VNPay, MoMo, ZaloPay hoặc phương án khác.
- Nguồn xác nhận có thẩm quyền cho từng phương thức và cách kiểm tra tính xác thực của kết quả giao dịch.
- Vòng đời chuẩn của Payment Attempt, Payment Transaction và Reconciliation Case; quy tắc tổng hợp khi một Order có nhiều Attempt.
- Thời hạn của phiên thanh toán và quan hệ với thời hạn reservation của EPIC 06/09.
- Cách xử lý tiền đến sau khi Order hoặc reservation hết hạn: đối soát thủ công, tái lập Order hay hoàn lại tiền.
- Điều kiện hỗ trợ COD theo khu vực, giá trị Order, nhóm sản phẩm, khách hàng và nhà vận chuyển.
- Luồng trạng thái Order dành cho COD vì Order phải được fulfillment trước khi thu tiền, khác với luồng trả trước `PENDING_PAYMENT → PAID` hiện tại.
- Nguồn xác nhận thu COD, thời điểm ghi nhận khoản thu và quy trình xử lý chênh lệch giữa số phải thu với số thực thu.
- Chính sách thanh toán thiếu, thừa, trùng, không rõ tham chiếu hoặc một Order có nhiều giao dịch thành công.
- Thẩm quyền và cơ chế phê duyệt khi cần điều chỉnh thủ công trạng thái Payment hoặc liên kết giao dịch với Order.
- Chu kỳ đối soát, dữ liệu đầu vào, thời hạn xử lý sai lệch và ranh giới trách nhiệm giữa Sales Manager với Accountant.
- Quy tắc làm tròn, đơn vị tiền tệ, giới hạn số tiền và số lần thử lại cho từng phương thức.
- Thời hạn lưu dữ liệu giao dịch, dữ liệu nhạy cảm được phép lưu và yêu cầu che/mã hóa theo tiêu chuẩn bảo mật áp dụng.

## 9. UI/UX Reference

- Hiển thị các phương thức thanh toán khả dụng cùng điều kiện hoặc chi phí liên quan trước khi khách chọn.
- Màn hình QR/chuyển khoản phải hiển thị rõ số tiền, nội dung tham chiếu và thời hạn còn hiệu lực nhưng không chứa dữ liệu tài khoản nhạy cảm không cần thiết.
- Phân biệt trực quan các trạng thái đang chờ, thành công, thất bại, hết hạn và cần đối soát; không dùng trang quay lại từ nhà cung cấp như bằng chứng thành công.
- Với COD, nêu rõ khách sẽ thanh toán khi nhận hàng và Order chưa được coi là đã thu tiền.
- Giao diện nhân viên cần hiển thị Payment và Order cạnh nhau, làm nổi bật sai lệch về tham chiếu, số tiền hoặc trạng thái.
- Thao tác xử lý ngoại lệ phải hiển thị phạm vi ảnh hưởng, yêu cầu lý do nghiệp vụ và xác nhận trước khi thực hiện.
- Chỉ tiêu thời gian phản hồi, cơ chế realtime, endpoint và công nghệ triển khai được đặc tả trong NFR hoặc Technical Design.

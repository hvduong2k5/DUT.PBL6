# EPIC 14 — Return, Refund & Complaint

## 1. Mục tiêu Epic

Epic này quản lý quy trình nghiệp vụ sau khi khách yêu cầu hủy, đổi/trả hoặc hoàn tiền: xác định điều kiện, thu thập bằng chứng, thẩm định, phê duyệt/từ chối, tiếp nhận hàng trả, quyết định tình trạng hàng và thực hiện nghĩa vụ hoàn tiền. Mỗi Case phải liên kết với đúng Order, dòng hàng, Payment, Shipment và bằng chứng liên quan.

Theo boundary của Product Backlog, EPIC 16 chịu trách nhiệm tiếp nhận/giao tiếp Ticket với khách; EPIC 14 chịu trách nhiệm state machine và quyết định nghiệp vụ Return/Refund. EPIC 08 vẫn là nguồn trạng thái Order, EPIC 07 thực thi giao dịch hoàn và EPIC 09 chỉ cập nhật tồn sau khi có kết quả kiểm tra hàng trả hợp lệ.

Epic không quản lý hội thoại CSKH tổng quát, tự sửa Order/Payment/Inventory, lưu Packing Video gốc hay xử lý vận chuyển giao hàng ban đầu. Các trách nhiệm này lần lượt thuộc EPIC 16, EPIC 08, EPIC 07, EPIC 09, EPIC 10 và EPIC 11.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 3rd** | `US-RET-01` đến `US-RET-06` | Khách tạo yêu cầu hợp lệ; CSKH thẩm định; Manager duyệt/từ chối và xử lý hoàn tiền/hàng trả có kiểm soát. |
| **Giai đoạn 2** | `US-RET-07` | CSKH được cấp quyền sử dụng Packing Video làm bằng chứng điều tra. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Tạo và theo dõi Case cho Order Guest sau khi xác minh quyền sở hữu. |
| Registered Customer (`ACT-02`) | Tạo yêu cầu hủy/đổi/trả, cung cấp bằng chứng và theo dõi kết quả cho Order của mình. |
| Warehouse Staff (`ACT-06`) | Tiếp nhận, kiểm tra và phân loại hàng trả để Inventory xử lý theo quyết định hợp lệ. |
| Customer Service (`ACT-11`) | Xem Order/bằng chứng, thẩm định ban đầu và giao tiếp với khách thông qua EPIC 16. |
| Sales Manager (`ACT-12`) | Duyệt/từ chối yêu cầu và phê duyệt nghĩa vụ hoàn tiền theo thẩm quyền. |
| Accountant / Finance Staff (`ACT-19`) | Theo dõi/đối soát kết quả Refund theo quyền, không thay thế quyền duyệt của Sales Manager. |
| Payment Provider (`EXT-01`) | Thực thi và trả kết quả giao dịch hoàn tiền qua EPIC 07. |
| Media / Object Storage (`EXT-09`) | Lưu ảnh/video khiếu nại và Packing Video theo miền media tương ứng. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-04` | Nhân viên chỉ được thực hiện thao tác thuộc quyền được cấp. | Thẩm định, duyệt, kiểm hàng và hoàn tiền phải kiểm tra quyền ở từng bước. |
| `BR-ORDER-01` | Order phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Return/Refund Case không tự sửa Order; kết quả hợp lệ được chuyển cho EPIC 08. |
| `BR-ORDER-03` | Order chỉ chuyển `PAID` khi giao dịch hợp lệ. | Chỉ khoản tiền thực sự đã xác nhận mới tạo nghĩa vụ có thể hoàn. |
| `BR-ORDER-04` | Khách chỉ được tự hủy Order trong thời gian cho phép. | Yêu cầu hủy phải kiểm tra trạng thái, thời hạn và hậu quả Payment/Shipment tại thời điểm xử lý. |
| `BR-ORDER-05` | Thay đổi trạng thái quan trọng phải được ghi nhận. | Kết quả hủy/Return phải liên kết lịch sử Order và nguồn quyết định. |
| `BR-PACK-04` | Packing Video liên kết với Order/Shipment tương ứng. | Chỉ video đúng Order/kiện mới được dùng làm bằng chứng điều tra. |
| `BR-PACK-05` | Người không có quyền không được tùy ý xem/tải Packing Video. | Quyền xử lý Case không mặc nhiên cho phép truy cập mọi video. |
| `BR-REFUND-01` | Chỉ Sales Manager hoặc cấp cao hơn được duyệt hoàn tiền. | CSKH/Accountant không tự phê duyệt Refund nếu không có quyền Manager tương ứng. |
| `BR-REFUND-02` | Hỗ trợ hoàn toàn phần hoặc một phần. | Số tiền duyệt phải xác định rõ phạm vi và không vượt nghĩa vụ có thể hoàn. |
| `BR-REFUND-03` | Hoàn về phương thức gốc hoặc loyalty khi khách đồng ý. | Phương thức hoàn phải bám Payment gốc hoặc lưu bằng chứng đồng ý đổi sang loyalty. |
| `BR-REFUND-04` | Duyệt/từ chối và thực hiện Refund phải có Audit Log. | Mọi quyết định, lần thực hiện và kết quả Refund phải truy vết được. |
| `BR-AUDIT-01` | Thao tác quan trọng phải được ghi Audit Log. | Thay bằng chứng, mở lại Case, kiểm hàng và điều chỉnh quyết định phải có lịch sử. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 01 — Authentication & Identity`: xác minh Customer/Guest và quyền sở hữu Order/Case.
- `EPIC 07 — Payment`: xác định số đã thu/có thể hoàn, thực thi Refund và trả kết quả giao dịch.
- `EPIC 08 — Order Management`: cung cấp Order/state và áp dụng kết quả hủy/Return hợp lệ.
- `EPIC 09 — Inventory & Batch`: nhận kết quả kiểm hàng để nhập lại tồn bán được, cách ly, ghi hỏng hoặc xử lý khác.
- `EPIC 10 — Packing & Fulfillment`: cung cấp Packing Video đúng Order/Package cho điều tra theo quyền.
- `EPIC 11 — Shipping & Delivery`: cung cấp trạng thái/bằng chứng giao và quản lý Shipment hoàn hàng nếu áp dụng.
- `EPIC 16 — Customer Service`: tiếp nhận Ticket, duy trì hội thoại và kích hoạt Case của EPIC 14.
- `EPIC 17 — Promotion & Loyalty`: nhận khoản hoàn sang loyalty chỉ khi khách đồng ý và chính sách hỗ trợ.
- `EPIC 22 — User / Role / Permission Administration`: quyết định quyền thẩm định, duyệt, kiểm hàng và truy cập media.
- `EPIC 23 — Audit & Security`: lưu dấu vết các quyết định và thao tác nhạy cảm.
- `EPIC 28 — Omnichannel Notification`: gửi thông báo kết quả Case; EPIC 14 chỉ cung cấp trạng thái/sự kiện nghiệp vụ.

## 4. User Stories chi tiết

### US-RET-01 — Yêu cầu hủy Order

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 3rd

**Phát hành:** MVP

> Là khách hàng, tôi muốn yêu cầu hủy đơn khi đơn còn đủ điều kiện.

**Giá trị nghiệp vụ:** Khách có thể dừng Order đúng chính sách và hệ thống xử lý nhất quán hậu quả đối với tồn, Payment và Shipment.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hủy trực tiếp Order chưa thanh toán đủ điều kiện
  Given khách đã xác minh quyền sở hữu Order
  And Order còn ở trạng thái cho phép tự hủy không cần phê duyệt
  When khách xác nhận hủy
  Then EPIC 08 áp dụng trạng thái `CANCELLED`
  And EPIC 09 giải phóng reservation còn hiệu lực đúng một lần
  And không tạo nghĩa vụ Refund khi chưa thu tiền

Scenario: Tạo Cancellation Case khi cần xử lý hậu quả
  Given Order đã thanh toán, đã tạo Shipment hoặc vượt phạm vi hủy trực tiếp
  When khách gửi yêu cầu hủy
  Then hệ thống tạo Case liên kết đúng Order và lý do
  And không tự hủy Order hoặc tự hoàn tiền trước quyết định hợp lệ

Scenario: Order không đủ điều kiện hủy
  Given trạng thái/thời hạn Order không cho phép hủy hoặc yêu cầu phải chuyển sang Return
  When khách gửi yêu cầu
  Then hệ thống từ chối hủy trực tiếp hoặc hướng dẫn quy trình phù hợp
  And Order không bị thay đổi trái phép

Scenario: Người không sở hữu Order yêu cầu hủy
  Given người dùng không chứng minh được quyền sở hữu Order
  When người dùng gửi yêu cầu hủy
  Then hệ thống từ chối thao tác
  And không tiết lộ dữ liệu Order

Scenario: Gửi lặp yêu cầu hủy
  Given yêu cầu hủy đã tạo Case hoặc đã được xử lý
  When cùng yêu cầu được gửi lại
  Then hệ thống trả kết quả nhất quán
  And không tạo nhiều Case, nhả tồn hoặc Refund trùng lặp
```

### US-RET-02 — Gửi yêu cầu đổi/trả

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 3rd

**Phát hành:** MVP

> Là khách hàng, tôi muốn gửi yêu cầu đổi/trả khi sản phẩm bị lỗi, hư hỏng hoặc không đúng đơn.

**Giá trị nghiệp vụ:** Khách có một quy trình minh bạch để yêu cầu xử lý sai sản phẩm/chất lượng trong phạm vi chính sách.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo Return Case hợp lệ
  Given khách đã xác minh quyền sở hữu Order và dòng hàng đủ điều kiện yêu cầu
  When khách chọn sản phẩm, số lượng, lý do và phương án mong muốn
  Then hệ thống tạo một Return Case liên kết Order và các dòng hàng cụ thể
  And đặt Case ở `RETURN_REQUESTED`
  And hiển thị mã Case cùng trạng thái cho khách

Scenario: Yêu cầu một phần Order
  Given Order có nhiều dòng hoặc số lượng lớn hơn một
  When khách yêu cầu trả một phần hợp lệ
  Then Case chỉ ghi nhận SKU/số lượng được chọn
  And phần còn lại của Order không tự bị trả hoặc hoàn tiền

Scenario: Quá thời hạn hoặc sai điều kiện
  Given dòng hàng không còn trong thời hạn hoặc lý do không thuộc chính sách đổi/trả
  When khách gửi yêu cầu
  Then hệ thống không tạo Case như yêu cầu hợp lệ
  And hiển thị điều kiện không đáp ứng hoặc hướng dẫn liên hệ CSKH

Scenario: Số lượng yêu cầu vượt lượng có thể trả
  Given một phần số lượng đã được trả/đang có Case khác
  When khách yêu cầu thêm làm tổng vượt số lượng đã mua còn lại
  Then hệ thống từ chối phần vượt
  And không tạo nghĩa vụ hoàn tiền vượt quá giao dịch gốc

Scenario: Yêu cầu trùng cho cùng dòng hàng
  Given đã có Case đang hoạt động cho cùng Order line và số lượng
  When khách gửi lại cùng yêu cầu
  Then hệ thống không tạo Case trùng
  And trả về Case hiện có hoặc hướng dẫn phù hợp
```

### US-RET-03 — Đính kèm ảnh/video khiếu nại

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 3rd

**Phát hành:** MVP

> Là khách hàng, tôi muốn đính kèm hình ảnh/video khiếu nại để chứng minh tình trạng sản phẩm.

**Giá trị nghiệp vụ:** Bằng chứng giúp CSKH/Manager thẩm định đúng tình trạng sản phẩm và giảm tranh chấp thiếu căn cứ.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đính kèm media hợp lệ
  Given khách sở hữu một Case cho phép bổ sung bằng chứng
  When khách gửi ảnh/video đáp ứng chính sách media
  Then hệ thống lưu bản ghi bằng chứng gắn với đúng Case, Order line và người gửi
  And thể hiện trạng thái media thành công trước khi dùng để thẩm định

Scenario: Media không hợp lệ
  Given tệp không đáp ứng loại, kích thước, số lượng hoặc chính sách an toàn
  When khách gửi bằng chứng
  Then hệ thống từ chối tệp không hợp lệ
  And giải thích điều kiện cần điều chỉnh

Scenario: Lưu media thất bại
  Given khách đang tải bằng chứng
  When việc lưu chưa hoàn tất
  Then hệ thống không trình bày media như bằng chứng đã sẵn sàng
  And cho phép thử lại mà không tạo bản hoàn chỉnh trùng

Scenario: Người khác bổ sung bằng chứng
  Given người dùng không sở hữu Case và không có quyền xử lý
  When người dùng cố thêm/xóa media
  Then hệ thống từ chối thao tác
  And bằng chứng của Case không bị thay đổi

Scenario: Bổ sung sau khi Case đã đóng
  Given Case đã có quyết định cuối cùng
  When khách cố thêm hoặc thay bằng chứng
  Then hệ thống áp dụng chính sách mở lại/khiếu nại quyết định nếu có
  And không ghi đè bằng chứng gốc tùy ý
```

### US-RET-04 — CSKH xem Order và bằng chứng

**Actor:** Customer Service (`ACT-11`)

**Ưu tiên:** Must Have — 3rd

**Phát hành:** MVP

> Là CSKH, tôi muốn xem thông tin đơn hàng và bằng chứng liên quan để xử lý khiếu nại.

**Giá trị nghiệp vụ:** CSKH có đủ bối cảnh để thẩm định ban đầu, yêu cầu bổ sung và chuyển Case cho người duyệt.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem hồ sơ Case đầy đủ
  Given CSKH có quyền và được phép xử lý Case
  When CSKH mở chi tiết
  Then hệ thống hiển thị snapshot Order line, Payment, Shipment, lịch sử trạng thái và bằng chứng khách gửi trong phạm vi quyền
  And phân biệt dữ liệu nguồn với nhận định xử lý

Scenario: Yêu cầu khách bổ sung thông tin
  Given Case thiếu dữ liệu cần thiết để thẩm định
  When CSKH ghi nhận yêu cầu bổ sung
  Then trạng thái/bước xử lý phản ánh đang chờ khách
  And nội dung giao tiếp được chuyển qua Ticket của EPIC 16

Scenario: Chuyển Case sang review
  Given dữ liệu bắt buộc đã đủ
  When CSKH hoàn tất thẩm định ban đầu
  Then Case chuyển sang `REVIEWING`
  And lưu nhận định, người xử lý và thời điểm
  And CSKH không tự phê duyệt Refund ngoài thẩm quyền

Scenario: CSKH không có quyền
  Given CSKH không được phép xem Case hoặc dữ liệu nhạy cảm tương ứng
  When CSKH truy cập
  Then hệ thống từ chối hoặc che dữ liệu ngoài quyền
  And không tiết lộ media/Payment không cần thiết

Scenario: Dữ liệu nguồn thay đổi
  Given trạng thái Order, Payment hoặc Shipment thay đổi khi Case đang mở
  When CSKH tải lại hồ sơ
  Then hệ thống hiển thị dữ liệu hiện tại cùng snapshot/lịch sử liên quan
  And không âm thầm sửa mất căn cứ tại thời điểm Case được tạo
```

### US-RET-05 — Duyệt hoặc từ chối yêu cầu đổi/trả

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Must Have — 3rd

**Phát hành:** MVP

> Là Sales Manager, tôi muốn duyệt hoặc từ chối yêu cầu đổi/trả theo chính sách.

**Giá trị nghiệp vụ:** Quyết định đổi/trả nhất quán, có căn cứ và kiểm soát được ảnh hưởng tới hàng hóa, Order và tài chính.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Duyệt Return Case hợp lệ
  Given Case đang `REVIEWING`, đủ bằng chứng và đáp ứng chính sách
  When Sales Manager chọn phê duyệt, phạm vi sản phẩm/số lượng và phương án xử lý
  Then Case chuyển sang `APPROVED`
  And lưu người duyệt, thời điểm, căn cứ và nghĩa vụ tiếp theo
  And không tự coi hàng đã được nhận hoặc tiền đã hoàn

Scenario: Từ chối Case
  Given Case đang `REVIEWING`
  When Sales Manager từ chối với lý do bắt buộc
  Then Case chuyển sang `REJECTED`
  And lý do được lưu và chuyển cho EPIC 16 để thông báo khách
  And không tạo nghĩa vụ Refund/nhập lại tồn

Scenario: Duyệt một phần
  Given Case có nhiều dòng hoặc số lượng
  When chính sách cho phép Manager chỉ duyệt một phần
  Then hệ thống lưu rõ phần được duyệt và phần bị từ chối
  And các nghĩa vụ hàng/tiền chỉ áp dụng cho phần được duyệt

Scenario: Người không có quyền duyệt
  Given nhân viên không có quyền Sales Manager tương ứng
  When nhân viên cố phê duyệt/từ chối
  Then hệ thống từ chối thao tác
  And trạng thái Case không bị thay đổi

Scenario: Case đã có quyết định
  Given Case đã `APPROVED` hoặc `REJECTED`
  When yêu cầu quyết định được gửi lại
  Then hệ thống không tạo quyết định hoặc nghĩa vụ trùng
  And việc thay đổi quyết định chỉ thực hiện qua quy trình mở lại có quyền/Audit
```

### US-RET-06 — Xử lý hoàn tiền

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Must Have — 3rd

**Phát hành:** MVP

> Là Manager, tôi muốn xử lý hoàn tiền cho khách hàng khi yêu cầu hợp lệ.

**Giá trị nghiệp vụ:** Khách nhận đúng số tiền/phương thức đã được duyệt và doanh nghiệp theo dõi được toàn bộ vòng đời Refund.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo Refund toàn phần hợp lệ
  Given Case đã được Sales Manager phê duyệt hoàn toàn phần
  And Payment gốc có số tiền đủ điều kiện hoàn
  When người có quyền xác nhận thực hiện Refund
  Then hệ thống tạo một Refund liên kết Case, Order và Payment gốc
  And EPIC 07 thực hiện đúng số tiền qua phương thức đã duyệt
  And Case chỉ chuyển `REFUNDED` sau khi có kết quả hoàn thành hợp lệ

Scenario: Tạo Refund một phần
  Given Case được duyệt hoàn cho một phần dòng hàng hoặc giá trị
  When thực hiện Refund
  Then số tiền hoàn không vượt số được duyệt và số còn có thể hoàn của Payment
  And lịch sử giữ được tổng đã hoàn cùng số còn lại

Scenario: Hoàn sang loyalty
  Given chính sách cho phép và khách đã đồng ý nhận giá trị qua loyalty
  When Manager chọn phương thức này
  Then hệ thống lưu bằng chứng đồng ý cùng giá trị quy đổi
  And chuyển yêu cầu sang EPIC 17
  And không đồng thời hoàn về phương thức gốc cho cùng giá trị

Scenario: Refund thất bại hoặc đang chờ
  Given yêu cầu Refund đã được gửi thực hiện
  When nhà cung cấp trả kết quả đang chờ hoặc thất bại
  Then hệ thống giữ trạng thái Refund tương ứng
  And không trình bày Case như đã hoàn tiền
  And cho phép xử lý tiếp theo theo chính sách mà không tạo hoàn trùng

Scenario: Yêu cầu Refund bị gửi lặp
  Given Refund đã được tạo hoặc hoàn thành
  When cùng yêu cầu thực hiện được gửi lại
  Then hệ thống không hoàn tiền lần thứ hai
  And trả kết quả gắn với Refund hiện có

Scenario: Người không có quyền phê duyệt/thực hiện
  Given người dùng không có quyền Sales Manager hoặc cấp cao hơn
  When người dùng cố duyệt hay thực hiện Refund
  Then hệ thống từ chối thao tác
  And không phát sinh giao dịch tài chính

Scenario: Hủy chưa thu tiền
  Given Order bị hủy trước khi có Payment thành công
  When hệ thống đánh giá nghĩa vụ tài chính
  Then không tạo Refund
  And Case/Order chỉ ghi nhận kết quả hủy phù hợp
```

### US-RET-07 — Tra cứu Packing Video phục vụ khiếu nại

**Actor:** Customer Service (`ACT-11`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là CSKH, tôi muốn tra cứu video đóng gói để kiểm tra nguyên nhân khi khách phản ánh thiếu/sai/hỏng sản phẩm.

**Giá trị nghiệp vụ:** CSKH đối chiếu phản ánh với bằng chứng đóng gói đúng Order/kiện hàng trong phạm vi quyền được cấp.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tra cứu video đúng Case và Order
  Given CSKH có quyền và đang xử lý Case liên quan đến một Order
  When CSKH yêu cầu xem Packing Video
  Then EPIC 10 cung cấp danh sách media liên kết đúng Order/Package/Shipment
  And hệ thống cho phép xem theo phạm vi quyền
  And lưu hoạt động truy cập theo chính sách Audit

Scenario: Không có video
  Given Order không yêu cầu hoặc không có Packing Video sẵn sàng
  When CSKH tra cứu
  Then hệ thống hiển thị trạng thái không có bằng chứng
  And không suy diễn việc thiếu video là bằng chứng chấp nhận/từ chối Case

Scenario: Video không thuộc Order
  Given một video thuộc Order hoặc kiện hàng khác
  When CSKH xử lý Case hiện tại
  Then hệ thống không hiển thị hoặc cho liên kết video đó vào Case

Scenario: CSKH không có quyền media
  Given CSKH có quyền xem Case nhưng không có quyền xem Packing Video
  When CSKH mở liên kết trực tiếp
  Then hệ thống từ chối truy cập
  And không tiết lộ vị trí lưu trữ

Scenario: CSKH cố thay đổi bằng chứng gốc
  Given CSKH chỉ có quyền tra cứu
  When CSKH cố xóa, thay thế hoặc liên kết lại Packing Video
  Then hệ thống từ chối thao tác
  And bằng chứng gốc của EPIC 10 được bảo toàn
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-RET-01` | Kiểm tra chủ sở hữu, trạng thái/thời hạn Order và hậu quả Payment/Shipment trước khi tiếp nhận yêu cầu hủy. | `US-RET-01`, `BR-ORDER-04` |
| `FR-RET-02` | Chuyển hủy trực tiếp đủ điều kiện sang EPIC 08/09; tạo Cancellation Case khi cần duyệt hoặc xử lý hậu quả liên miền. | `US-RET-01`, `EPIC 08`, `EPIC 09` |
| `FR-RET-03` | Tạo Return Case duy nhất gắn Order, dòng SKU, số lượng, lý do và phương án khách yêu cầu. | `US-RET-02` |
| `FR-RET-04` | Kiểm tra thời hạn, lý do, số lượng đã mua/đã trả và Case đang hoạt động trước khi tiếp nhận. | `US-RET-02` |
| `FR-RET-05` | Cho khách sở hữu Case đính kèm media hợp lệ và theo dõi trạng thái lưu của bằng chứng. | `US-RET-03` |
| `FR-RET-06` | Bảo toàn bằng chứng gốc; ngăn actor không quyền thêm/xóa/thay và kiểm soát bổ sung sau khi Case đóng. | `US-RET-03`, `BR-AUTH-04`, `BR-AUDIT-01` |
| `FR-RET-07` | Cho CSKH có quyền xem snapshot Order line, Payment, Shipment, lịch sử và bằng chứng cần thiết. | `US-RET-04` |
| `FR-RET-08` | Cho CSKH yêu cầu bổ sung qua EPIC 16 và chuyển Case đủ dữ liệu sang `REVIEWING`. | `US-RET-04`, `EPIC 16` |
| `FR-RET-09` | Cho Sales Manager duyệt, từ chối hoặc duyệt một phần Case đang review với phạm vi/lý do rõ ràng. | `US-RET-05`, `BR-REFUND-01` |
| `FR-RET-10` | Không coi `APPROVED` là đã nhận hàng hoặc đã hoàn tiền; tạo các nghĩa vụ tiếp theo độc lập. | `US-RET-05` |
| `FR-RET-11` | Quản lý việc tiếp nhận/kiểm tra hàng trả và chuyển kết quả sang EPIC 09 để nhập lại, cách ly, ghi hỏng hoặc xử lý khác. | `US-RET-02`, `US-RET-05`, `EPIC 09` |
| `FR-RET-12` | Tạo Refund toàn phần/một phần gắn đúng Case, Order, Payment gốc, số tiền và phương thức được duyệt. | `US-RET-06`, `BR-REFUND-02~03` |
| `FR-RET-13` | Không cho tổng Refund vượt số tiền Payment đã xác nhận còn có thể hoàn. | `US-RET-06`, `BR-ORDER-03`, `BR-REFUND-02` |
| `FR-RET-14` | Chỉ gửi yêu cầu Refund sang EPIC 07 sau phê duyệt hợp lệ và chỉ ghi `REFUNDED` khi có kết quả hoàn thành. | `US-RET-06`, `BR-REFUND-01`, `BR-REFUND-04` |
| `FR-RET-15` | Hỗ trợ hoàn sang loyalty chỉ khi chính sách cho phép, khách đồng ý và không hoàn trùng cùng giá trị qua phương thức gốc. | `US-RET-06`, `BR-REFUND-03`, `EPIC 17` |
| `FR-RET-16` | Chống tạo Case, quyết định, nhận hàng, cập nhật tồn và Refund trùng khi yêu cầu được gửi lại. | `US-RET-01~06` |
| `FR-RET-17` | Cho CSKH có quyền tra cứu Packing Video đúng Order/Package/Shipment thông qua EPIC 10. | `US-RET-07`, `BR-PACK-04~05` |
| `FR-RET-18` | Không cho quyền xem Case tự động trở thành quyền xem/sửa Packing Video. | `US-RET-07`, `BR-AUTH-04`, `EPIC 10`, `EPIC 22` |
| `FR-RET-19` | Chuyển trạng thái/kết quả cần giao tiếp với khách sang Ticket/Notification mà không phụ thuộc lưu Case vào kết quả gửi tin. | `US-RET-01~07`, `EPIC 16`, `EPIC 28` |
| `FR-RET-20` | Ghi Audit cho duyệt/từ chối, thay quyết định, kiểm hàng, phương thức/số tiền hoàn và từng kết quả Refund. | `US-RET-05`, `US-RET-06`, `BR-REFUND-04` |
| `FR-RET-21` | Kiểm soát quyền theo vai trò, phạm vi Case và hành động ở mỗi trạng thái. | `US-RET-04~07`, `BR-AUTH-04`, `EPIC 22` |
| `FR-RET-22` | Cung cấp dữ liệu Return/Refund đã xác nhận cho đối soát/báo cáo mà không thay thế nghiệp vụ EPIC 07/24. | `US-RET-06`, `EPIC 07`, `EPIC 24` |

## 6. State machine và quy tắc dữ liệu

### 6.1. State machine cơ sở trong Product Backlog

```text
DELIVERED
    ↓
RETURN_REQUESTED
    ↓
REVIEWING
    ├── REJECTED
    └── APPROVED
             ↓
        RETURNED
             ↓
         REFUNDED
```

State machine này là khung tối thiểu và đang gộp Return với Refund. Khi thiết kế cần tách ít nhất `Return Case status`, `Returned Goods status` và `Refund status`, vì hàng có thể đã nhận nhưng chưa hoàn tiền, Refund có thể thất bại/đang chờ, và một Case có thể chỉ đổi hàng hoặc không yêu cầu nhận lại hàng.

### 6.2. Quy tắc dữ liệu và an toàn nghiệp vụ

- Case phải có mã duy nhất, loại yêu cầu, chủ sở hữu, Order, các Order line/số lượng, lý do, trạng thái, kênh tiếp nhận và Ticket liên quan nếu có.
- Tổng số lượng đang yêu cầu/đã trả trên mọi Case không được vượt số lượng đã mua còn đủ điều kiện của từng Order line.
- Quyết định phải lưu phạm vi được duyệt/từ chối, căn cứ, actor và thời điểm; không chỉ lưu trạng thái cuối.
- Bằng chứng khách gửi và Packing Video là hai nguồn media khác nhau. Case tham chiếu bằng chứng nhưng không được ghi đè/xóa tài sản gốc tùy ý.
- Hàng trả về không tự động trở thành tồn bán được. Warehouse phải kiểm tra và phân loại trước khi EPIC 09 cập nhật số dư.
- Return status, Order status, Shipment status, Payment status và Refund status là các vòng đời riêng; ánh xạ phải có quy tắc rõ ràng.
- Số tiền có thể hoàn được tính từ Payment đã xác nhận, trừ các Refund đã hoàn/đang chiếm giữ hợp lệ; không chỉ dựa trên tổng Order ban đầu.
- Refund thành công không mặc nhiên đồng nghĩa hàng đã về kho; `RETURNED` không mặc nhiên đồng nghĩa tiền đã hoàn.
- Mọi xử lý lặp phải cho cùng kết quả và không tạo thêm Case, yêu cầu thu hồi, nhập tồn, loyalty credit hoặc Refund.
- Dữ liệu khiếu nại, Payment và media chỉ hiển thị theo quyền/mục đích; liên kết trực tiếp không được vượt kiểm soát truy cập.
- Chi tiết endpoint, object storage, quét tệp, workflow engine, hàng đợi, callback provider và retry thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-RET-01` | `BR-ORDER-04`, `BR-ORDER-05` | EPIC 08, 09, 11 | Chỉ chủ sở hữu hủy đúng điều kiện; Case phức tạp không tự hủy/hoàn; gửi lặp không gây tác động trùng. |
| `US-RET-02` | `BR-ORDER-01`, `BR-AUDIT-01` | EPIC 08, 09, 11 | Case đúng Order line/số lượng; quá hạn/vượt lượng/trùng bị chặn; hàng về chưa tự bán lại. |
| `US-RET-03` | `BR-AUTH-04`, `BR-AUDIT-01` | EXT-09 | Khách đúng Case thêm được media hợp lệ; lỗi lưu/không quyền không làm sai bằng chứng; bản gốc được giữ. |
| `US-RET-04` | `BR-AUTH-04` | EPIC 07, 08, 11, 16, 22 | CSKH thấy đủ dữ liệu trong quyền, yêu cầu bổ sung và chuyển review nhưng không tự duyệt Refund. |
| `US-RET-05` | `BR-REFUND-01`, `BR-REFUND-04` | EPIC 08, 09, 16, 23 | Manager duyệt/từ chối/toàn phần đúng quyền, có căn cứ; quyết định không đồng nghĩa đã nhận hàng/hoàn tiền. |
| `US-RET-06` | `BR-ORDER-03`, `BR-REFUND-01~04` | EPIC 07, 17, 23 | Refund đúng số/phương thức, không vượt/không trùng; chỉ hoàn thành khi có kết quả hợp lệ; mọi bước có Audit. |
| `US-RET-07` | `BR-PACK-04`, `BR-PACK-05` | EPIC 10, 16, 22, 23 | CSKH chỉ xem video đúng Case/Order khi có quyền và không thể sửa bằng chứng gốc. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Hợp nhất ranh giới `US-RET-01` với `US-ORD-04`: đề xuất EPIC 08 xử lý hủy tự phục vụ đơn giản, EPIC 14 xử lý Cancellation Case có duyệt/Refund/Shipment.
- Ma trận điều kiện hủy/đổi/trả theo trạng thái Order, thời gian, SKU, lý do, kênh bán và tình trạng Payment/Shipment.
- State machine riêng cho Cancellation Case, Return Case, Returned Goods, Exchange và Refund; tránh dùng một trạng thái `RETURNED` cho nhiều nghĩa.
- Có cho phép đổi hàng hay chỉ trả/hoàn trong MVP; nếu đổi, cách tạo Order thay thế, giữ tồn và xử lý chênh lệch giá.
- Trường hợp không cần khách gửi hàng lại nhưng vẫn hoàn/đổi; ai có quyền quyết định và ngưỡng giá trị.
- Quy trình vận chuyển hàng trả: khách tự gửi, đơn vị vận chuyển lấy, phí ai chịu và liên kết Shipment hoàn.
- Tiêu chí Warehouse kiểm tra hàng trả và kết quả: bán lại, cách ly, hỏng, trả nhà cung cấp hoặc tiêu hủy.
- Thời điểm cho phép Refund: ngay khi duyệt, khi khách gửi hàng, khi kho nhận hay sau kiểm tra chất lượng.
- Cách phân bổ số tiền hoàn cho giá hàng, giảm giá, phí vận chuyển, coupon, loyalty và thuế; quy tắc làm tròn.
- Trường hợp Payment có nhiều lần thu/hoàn, giao dịch đến trễ hoặc số tiền còn có thể hoàn không đủ.
- Quy trình hoàn loyalty, tỷ lệ quy đổi và bằng chứng đồng ý của khách.
- Có yêu cầu phân tách người duyệt Case và người thực hiện Refund hay không; ngưỡng cần duyệt hai cấp.
- Loại/số lượng/kích thước/thời hạn media khách được gửi và chính sách lưu giữ/legal hold.
- Dữ liệu nào CSKH, Warehouse, Manager, Accountant được xem; đặc biệt Payment, địa chỉ và Packing Video.
- SLA xử lý từng bước, cơ chế escalations và các mốc thông báo khách qua EPIC 16/28.
- Chính sách khiếu nại lại/mở lại Case đã `REJECTED` hoặc đã đóng và cách bảo toàn quyết định cũ.

## 9. UI/UX Reference

- Form khách chọn đúng Order line, số lượng, lý do, phương án mong muốn và xem điều kiện trước khi gửi.
- Khu vực media hiển thị tiến trình lưu và trạng thái thành công/thất bại; không coi preview cục bộ là bằng chứng đã lưu.
- Timeline Case tách rõ yêu cầu, review, quyết định, hàng hoàn, kiểm hàng và Refund; không dùng một nhãn “đã xử lý” mơ hồ.
- Workspace CSKH hiển thị Order/Payment/Shipment/bằng chứng cạnh Case nhưng che dữ liệu ngoài quyền và đưa giao tiếp sang EPIC 16.
- Màn hình Manager làm nổi bật phạm vi duyệt, số tiền dự kiến, phương thức hoàn và hậu quả Inventory/Shipping trước khi quyết định.
- Màn hình Warehouse hiển thị SKU/Batch/số lượng dự kiến, thực nhận, tình trạng và quyết định phân loại; không tự nhập lại tồn bán được.
- Trạng thái Refund phân biệt chưa tạo, đang chờ, thành công và thất bại; khách chỉ thấy “đã hoàn” sau kết quả hợp lệ.
- Chỉ tiêu hiệu năng, endpoint, upload media, callback Refund và công nghệ workflow được đặc tả trong NFR hoặc Technical Design.

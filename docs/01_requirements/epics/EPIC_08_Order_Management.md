# EPIC 08 — Order Management

## 1. Mục tiêu Epic

Epic này quản lý Order từ khi được tạo bởi một checkout hợp lệ cho đến khi hoàn tất, hủy, hết hạn, giao thất bại hoặc chuyển sang quy trình đổi trả. Khách có thể xem lịch sử, theo dõi trạng thái và tự hủy khi còn đủ điều kiện; nhân viên có thẩm quyền có thể tìm, xử lý và ưu tiên Order theo SLA.

Order là hồ sơ nghiệp vụ trung tâm liên kết dữ liệu thương mại đã chốt, Payment, reservation tồn kho, đóng gói, Shipment và lịch sử trạng thái. Mọi chuyển trạng thái phải hợp lệ, có nguồn phát sinh rõ ràng và có khả năng truy vết. Dữ liệu snapshot của Order không bị sửa hồi tố khi Product, giá, địa chỉ hoặc chương trình bán hàng thay đổi.

Epic không tự tính tổng checkout, xác nhận giao dịch tài chính, quyết định Batch xuất kho, thực hiện đóng gói, gọi nhà vận chuyển, gửi thông báo hoặc duyệt đổi trả/hoàn tiền. Các trách nhiệm này lần lượt thuộc EPIC 06, EPIC 07, EPIC 09, EPIC 10, EPIC 11, EPIC 28 và EPIC 14.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 1st** | `US-ORD-01`, `US-ORD-02`, `US-ORD-04` | Khách xem được đơn của mình, biết trạng thái hiện tại và tự hủy khi còn đủ điều kiện. |
| **MVP — ưu tiên 2nd** | `US-ORD-05`, `US-ORD-06` | Sales Manager và nhân viên vận hành xử lý danh sách Order theo đúng trạng thái và mức ưu tiên SLA. |
| **Giai đoạn 2** | `US-ORD-03` | Khách nhận thông báo thay đổi trạng thái thông qua EPIC 28. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Xem Order Guest bằng cơ chế xác minh quyền sở hữu và hủy khi đủ điều kiện. |
| Registered Customer (`ACT-02`) | Xem lịch sử, trạng thái và hủy Order thuộc tài khoản mình. |
| Warehouse Staff (`ACT-06`) | Xem hàng đợi Order cần xử lý kho theo quyền và SLA. |
| Packing Staff (`ACT-07`) | Nhận Order đủ điều kiện đóng gói và cập nhật kết quả thông qua EPIC 10. |
| Marketplace Operator (`ACT-09`) | Theo dõi Order nguồn Marketplace ở Giai đoạn 2; nghiệp vụ đồng bộ thuộc EPIC 12. |
| Sales Manager (`ACT-12`) | Xem, tìm kiếm, xử lý danh sách Order và giám sát SLA theo quyền được cấp. |
| Customer Service (`ACT-11`) | Tra cứu Order khi hỗ trợ khách hoặc xử lý yêu cầu đổi trả theo quyền. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-01` | Khách vãng lai được phép mua hàng mà không bắt buộc tạo tài khoản. | Order Guest phải tồn tại độc lập với Customer Account và vẫn tra cứu được bằng xác minh phù hợp. |
| `BR-ORDER-01` | Đơn hàng phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Mỗi Order có đúng trạng thái hiện tại và chỉ đi qua các chuyển trạng thái hợp lệ. |
| `BR-ORDER-02` | Không xác nhận đơn vượt quá tồn kho khả dụng. | Order chỉ chuyển từ kết quả Checkout sang bước tiếp theo khi reservation đã thành công. |
| `BR-ORDER-03` | Đơn hàng chỉ chuyển sang Paid khi giao dịch được xác nhận hợp lệ. | Chỉ kết quả Payment hợp lệ từ EPIC 07 mới được áp dụng trạng thái `PAID`. |
| `BR-ORDER-04` | Khách hàng chỉ được tự hủy đơn trong thời gian được phép. | Hệ thống phải kiểm tra trạng thái và điều kiện hủy tại thời điểm nhận yêu cầu. |
| `BR-ORDER-05` | Mọi thay đổi trạng thái quan trọng của đơn hàng phải được ghi nhận để truy vết. | Lịch sử phải lưu trạng thái trước/sau, nguồn, actor, thời điểm và lý do khi cần. |
| `BR-PACK-01` | Chỉ nhân viên đóng gói được phân công mới được xác nhận hoàn thành đóng gói. | Order chỉ nhận kết quả `PACKED` hợp lệ từ nghiệp vụ Packing có thẩm quyền. |
| `BR-MKTPLACE-01` | Order Marketplace phải xác định nguồn phát sinh. | Order đa kênh phải lưu nguồn và tham chiếu kênh khi phạm vi EPIC 12 được triển khai. |
| `BR-MKTPLACE-04` | Không tạo trùng Order nội bộ khi đồng bộ lại cùng một Order Marketplace. | Việc tiếp nhận Order từ sàn phải chống trùng theo tham chiếu nguồn. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Thao tác quản trị, hủy và xử lý ngoại lệ Order phải có khả năng kiểm tra. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 01 — Authentication & Identity`: xác minh Customer, Guest và quyền sở hữu Order; liên kết Order Guest với tài khoản khi đủ điều kiện.
- `EPIC 04 — Product, Variant & SKU`: cung cấp dữ liệu nguồn nhưng không được làm thay đổi snapshot Order đã tạo.
- `EPIC 06 — Checkout`: kiểm tra dữ liệu, giữ tồn và yêu cầu tạo Order với snapshot đã xác nhận.
- `EPIC 07 — Payment`: cung cấp kết quả Payment; Order Management quyết định chuyển trạng thái Order theo quy tắc.
- `EPIC 09 — Inventory & Batch`: quản lý reservation, nhả tồn và xuất kho; nhận kết quả vòng đời Order liên quan.
- `EPIC 10 — Packing & Fulfillment`: tiếp nhận Order đủ điều kiện và cung cấp kết quả đóng gói.
- `EPIC 11 — Shipping & Delivery`: tạo Shipment và cung cấp trạng thái giao hàng để cập nhật Order.
- `EPIC 14 — Return / Refund / Complaint`: tiếp nhận Order đủ điều kiện vào luồng đổi trả; Return có state machine riêng.
- `EPIC 22 — User / Role / Permission Administration`: quyết định quyền xem và xử lý Order của nhân viên.
- `EPIC 23 — Audit & Security`: lưu dấu vết các thao tác và chuyển trạng thái quan trọng.
- `EPIC 28 — Omnichannel Notification`: gửi thông báo dựa trên thay đổi trạng thái; Order chỉ cung cấp dữ liệu/sự kiện nghiệp vụ.
- `EPIC 12`, `EPIC 13`, `EPIC 18`: cung cấp Order từ Marketplace, Offline và B2B trong các giai đoạn sau, kèm nguồn và chính sách riêng.

## 4. User Stories chi tiết

### US-ORD-01 — Xem lịch sử đơn hàng

**Actor:** Registered Customer (`ACT-02`)

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP

> Là khách hàng, tôi muốn xem lịch sử đơn hàng để biết các lần mua trước.

**Giá trị nghiệp vụ:** Khách dễ dàng tra cứu các giao dịch đã phát sinh mà không cần liên hệ nhân viên hỗ trợ.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Customer xem danh sách Order của mình
  Given Customer đã đăng nhập và có các Order đã phát sinh
  When Customer mở lịch sử đơn hàng
  Then hệ thống chỉ hiển thị các Order thuộc Customer đó
  And mỗi Order có mã đơn, ngày đặt, tổng tiền và trạng thái hiện tại
  And danh sách được sắp xếp theo thời điểm phù hợp để dễ tra cứu

Scenario: Customer chưa có Order
  Given Customer đã đăng nhập nhưng chưa có Order nào
  When Customer mở lịch sử đơn hàng
  Then hệ thống hiển thị trạng thái chưa có đơn rõ ràng
  And không hiển thị dữ liệu Order của khách khác

Scenario: Xem chi tiết một Order lịch sử
  Given Customer sở hữu một Order
  When Customer chọn xem chi tiết Order đó
  Then hệ thống hiển thị snapshot sản phẩm, số lượng, giá, chi phí, thông tin nhận hàng và trạng thái liên quan
  And dữ liệu phản ánh Order tại thời điểm đặt thay vì dữ liệu Product hiện tại

Scenario: Order Guest đã được liên kết hợp lệ
  Given Customer đã xác minh và liên kết một Order Guest theo EPIC 01
  When Customer mở lịch sử đơn hàng
  Then Order đã liên kết xuất hiện trong lịch sử của Customer
  And Order không bị nhân bản do thao tác liên kết

Scenario: Ngăn xem Order của Customer khác
  Given Customer A đã đăng nhập
  When Customer A cố mở Order thuộc Customer B
  Then hệ thống từ chối truy cập
  And không tiết lộ nội dung hoặc trạng thái của Order thuộc Customer B
```

### US-ORD-02 — Xem trạng thái hiện tại của đơn hàng

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP

> Là khách hàng, tôi muốn xem trạng thái hiện tại của đơn hàng.

**Giá trị nghiệp vụ:** Khách biết Order đang chờ thanh toán, được xử lý, đang giao hay đã hoàn tất để chủ động theo dõi và phối hợp nhận hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Customer xem trạng thái Order của mình
  Given Customer đã đăng nhập và sở hữu một Order
  When Customer mở chi tiết Order
  Then hệ thống hiển thị trạng thái hiện tại cùng các mốc lịch sử được phép công khai
  And mô tả trạng thái bằng ngôn ngữ dễ hiểu cho khách

Scenario: Guest tra cứu Order hợp lệ
  Given Guest có mã Order và thông tin xác minh quyền sở hữu hợp lệ
  When Guest yêu cầu tra cứu
  Then hệ thống hiển thị thông tin và trạng thái của đúng Order
  And không yêu cầu Guest tạo tài khoản

Scenario: Guest không chứng minh được quyền sở hữu
  Given một người biết hoặc đoán được mã Order Guest
  When người đó không cung cấp đủ thông tin xác minh
  Then hệ thống từ chối tra cứu
  And không tiết lộ thông tin người nhận, sản phẩm hoặc trạng thái Order

Scenario: Trạng thái được cập nhật từ phân hệ nguồn
  Given Order nhận một kết quả hợp lệ từ Payment, Packing hoặc Shipping
  When Order Management áp dụng kết quả đó
  Then trạng thái hiện tại được cập nhật theo state machine
  And lịch sử ghi nguồn cùng thời điểm thay đổi

Scenario: Nhận kết quả lặp hoặc cũ
  Given một kết quả trạng thái đã được áp dụng hoặc không còn phù hợp với trạng thái hiện tại
  When cùng kết quả hoặc kết quả cũ được gửi lại
  Then hệ thống không tạo chuyển trạng thái trùng hoặc quay lùi trái phép
  And giữ trạng thái Order nhất quán
```

### US-ORD-03 — Nhận thông báo thay đổi trạng thái

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn nhận thông báo khi đơn hàng thay đổi trạng thái.

**Giá trị nghiệp vụ:** Khách được cập nhật kịp thời mà không phải liên tục tự kiểm tra Order.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Gửi yêu cầu thông báo cho thay đổi quan trọng
  Given Order chuyển sang một trạng thái được cấu hình cần thông báo
  When thay đổi trạng thái được lưu thành công
  Then hệ thống chuyển yêu cầu thông báo với đúng Order, người nhận và nội dung nghiệp vụ sang EPIC 28
  And việc gửi thông báo không làm thay đổi lại trạng thái Order

Scenario: Không phát thông báo trùng cho cùng một thay đổi
  Given một thay đổi trạng thái đã tạo yêu cầu thông báo
  When cùng kết quả trạng thái được xử lý lại
  Then hệ thống không tạo nhiều thông báo nghiệp vụ cho cùng một lần chuyển trạng thái

Scenario: Gửi thông báo thất bại
  Given trạng thái Order đã được cập nhật hợp lệ
  When nhà cung cấp thông báo chưa gửi được tin
  Then trạng thái Order vẫn được bảo toàn
  And EPIC 28 xử lý thử lại hoặc thất bại theo chính sách thông báo

Scenario: Nội dung không tiết lộ dữ liệu nhạy cảm
  Given hệ thống chuẩn bị thông báo trạng thái Order
  When tạo dữ liệu thông báo
  Then nội dung chỉ chứa thông tin cần thiết cho người nhận
  And không đưa dữ liệu Payment hoặc thông tin cá nhân nhạy cảm không cần thiết vào thông báo
```

### US-ORD-04 — Hủy đơn hàng khi còn đủ điều kiện

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP

> Là khách hàng, tôi muốn hủy đơn khi đơn còn đủ điều kiện hủy.

**Giá trị nghiệp vụ:** Khách chủ động dừng Order chưa đi quá xa trong fulfillment, đồng thời doanh nghiệp kiểm soát được tồn kho và khoản tiền liên quan.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hủy Order chưa thanh toán còn đủ điều kiện
  Given khách sở hữu một Order chưa thanh toán và còn trong trạng thái cho phép tự hủy
  When khách xác nhận hủy Order
  Then hệ thống chuyển Order sang `CANCELLED`
  And ghi nhận actor, thời điểm và lý do nếu được cung cấp
  And yêu cầu EPIC 09 giải phóng reservation còn hiệu lực

Scenario: Hủy Order đã thanh toán nhưng còn đủ điều kiện
  Given khách sở hữu một Order đã thanh toán và chính sách vẫn cho phép yêu cầu hủy
  When khách xác nhận hủy
  Then hệ thống ghi nhận kết quả hủy theo quy trình đã phê duyệt
  And chuyển nghĩa vụ hoàn tiền sang EPIC 14 thay vì tự coi tiền đã được hoàn
  And Order và Payment vẫn phản ánh đúng trạng thái riêng của từng nghiệp vụ

Scenario: Order không còn đủ điều kiện tự hủy
  Given Order đã đi qua mốc cho phép khách tự hủy
  When khách yêu cầu hủy
  Then hệ thống từ chối yêu cầu tự hủy
  And hiển thị lý do cùng hướng dẫn liên hệ hỗ trợ hoặc dùng quy trình đổi trả phù hợp

Scenario: Người không sở hữu Order yêu cầu hủy
  Given người dùng không chứng minh được quyền sở hữu Order
  When người dùng gửi yêu cầu hủy
  Then hệ thống từ chối thao tác
  And Order không bị thay đổi

Scenario: Gửi lặp yêu cầu hủy
  Given Order đã được khách hủy thành công
  When cùng yêu cầu hủy được gửi lại
  Then hệ thống giữ Order ở `CANCELLED`
  And không lặp lại việc nhả tồn hoặc tạo nhiều yêu cầu hoàn tiền
```

### US-ORD-05 — Xem và xử lý danh sách đơn hàng

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Sales Manager, tôi muốn xem và xử lý danh sách đơn hàng để đảm bảo đơn được thực hiện đúng.

**Giá trị nghiệp vụ:** Sales Manager có một nguồn thông tin thống nhất để giám sát Order và xử lý đúng bước thay vì cập nhật thủ công rời rạc.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem và lọc danh sách Order
  Given Sales Manager có quyền quản lý Order
  When Sales Manager mở danh sách và chọn điều kiện tìm kiếm hoặc lọc
  Then hệ thống hiển thị các Order phù hợp theo trạng thái, thời gian, nguồn hoặc thông tin tra cứu được hỗ trợ
  And hiển thị các dữ liệu tổng quan cần thiết để quyết định bước xử lý

Scenario: Xem chi tiết phục vụ xử lý
  Given Sales Manager có quyền xem một Order
  When Sales Manager mở chi tiết
  Then hệ thống hiển thị snapshot Order, Payment, reservation, Packing và Shipment liên quan theo dữ liệu sẵn có
  And làm rõ trạng thái hiện tại cùng hành động hợp lệ tiếp theo

Scenario: Thực hiện chuyển trạng thái hợp lệ
  Given Order đang ở trạng thái cho phép một hành động nghiệp vụ
  And Sales Manager có quyền thực hiện hành động đó
  When Sales Manager xác nhận xử lý
  Then hệ thống áp dụng đúng chuyển trạng thái
  And lưu lịch sử actor, thời điểm, trạng thái trước/sau và lý do nếu bắt buộc

Scenario: Từ chối chuyển trạng thái không hợp lệ
  Given hành động không hợp lệ với trạng thái hiện tại hoặc dữ liệu phụ thuộc chưa đáp ứng
  When Sales Manager yêu cầu xử lý Order
  Then hệ thống từ chối thay đổi
  And giải thích điều kiện còn thiếu hoặc trạng thái gây xung đột

Scenario: Nhân viên không có quyền quản lý Order
  Given nhân viên không có quyền xem hoặc xử lý Order tương ứng
  When nhân viên truy cập hoặc gửi hành động quản trị
  Then hệ thống từ chối thao tác
  And không tiết lộ hoặc thay đổi dữ liệu ngoài phạm vi quyền
```

### US-ORD-06 — Ưu tiên đơn cần xử lý theo SLA

**Actor:** Warehouse Staff (`ACT-06`), Packing Staff (`ACT-07`), Sales Manager (`ACT-12`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là nhân viên vận hành, tôi muốn biết đơn nào cần xử lý trước để đảm bảo SLA giao hàng.

**Giá trị nghiệp vụ:** Đội vận hành tập trung vào Order sắp hoặc đã quá hạn, giảm chậm đóng gói và giao hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị hàng đợi theo mức ưu tiên
  Given có nhiều Order đang chờ cùng một công đoạn
  When nhân viên vận hành mở hàng đợi công việc
  Then hệ thống sắp xếp hoặc đánh dấu Order theo quy tắc SLA đang áp dụng
  And hiển thị hạn xử lý và mức độ ưu tiên đủ để nhân viên chọn Order tiếp theo

Scenario: Cảnh báo Order sắp hoặc đã quá SLA
  Given một Order sắp đến hoặc đã vượt hạn xử lý của trạng thái hiện tại
  When hệ thống đánh giá SLA
  Then Order được đánh dấu cảnh báo tương ứng
  And Sales Manager có thể nhận biết mức độ chậm và công đoạn đang chịu trách nhiệm

Scenario: Chỉ hiển thị Order đủ điều kiện cho công đoạn
  Given một Order chưa đáp ứng điều kiện Payment, reservation hoặc trạng thái trước đó
  When nhân viên xem hàng đợi đóng gói hay giao hàng
  Then Order không được trình bày như công việc sẵn sàng xử lý
  And trạng thái chờ hoặc điều kiện còn thiếu được thể hiện phù hợp cho người có quyền

Scenario: SLA được tính lại sau chuyển trạng thái
  Given Order vừa chuyển sang một công đoạn mới
  When chuyển trạng thái được lưu thành công
  Then hệ thống xác định mốc SLA cho công đoạn mới theo chính sách
  And không tiếp tục dùng hạn xử lý của công đoạn trước như hạn hiện tại

Scenario: Thay đổi ưu tiên có kiểm soát
  Given chính sách cho phép người có quyền điều chỉnh mức ưu tiên của Order
  When Sales Manager thay đổi ưu tiên và cung cấp lý do bắt buộc
  Then hàng đợi phản ánh mức ưu tiên mới
  And thay đổi được ghi Audit mà không sửa mất dữ liệu SLA gốc
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-ORD-01` | Tạo và lưu một Order từ kết quả Checkout hợp lệ, gồm mã duy nhất, nguồn, chủ sở hữu/ngữ cảnh Guest và snapshot đã xác nhận. | `US-CHK-04`, `EPIC 06` |
| `FR-ORD-02` | Duy trì đúng một trạng thái hiện tại và lịch sử chuyển trạng thái của mỗi Order theo state machine được phê duyệt. | `US-ORD-02`, `US-ORD-05`, `BR-ORDER-01`, `BR-ORDER-05` |
| `FR-ORD-03` | Chỉ áp dụng chuyển trạng thái khi trạng thái nguồn, actor, dữ liệu phụ thuộc và điều kiện nghiệp vụ đều hợp lệ. | `US-ORD-02`, `US-ORD-04`, `US-ORD-05` |
| `FR-ORD-04` | Tiếp nhận kết quả từ Payment, Inventory, Packing và Shipping theo cách không tạo chuyển trạng thái trùng hoặc quay lùi trái phép. | `US-ORD-02`, `BR-ORDER-01~03` |
| `FR-ORD-05` | Cho Customer xem danh sách và chi tiết các Order thuộc tài khoản của mình. | `US-ORD-01` |
| `FR-ORD-06` | Hiển thị snapshot sản phẩm, giá, chi phí và thông tin giao hàng lịch sử mà không bị thay đổi bởi dữ liệu nguồn hiện tại. | `US-ORD-01`, `EPIC 04`, `EPIC 06` |
| `FR-ORD-07` | Cho phép Order Guest đã được xác minh liên kết với Customer mà không nhân bản Order. | `US-ORD-01`, `EPIC 01` |
| `FR-ORD-08` | Cho Customer hoặc Guest đã xác minh quyền sở hữu xem trạng thái hiện tại và các mốc lịch sử được phép công khai. | `US-ORD-02` |
| `FR-ORD-09` | Chuyển yêu cầu thông báo cho các thay đổi trạng thái được cấu hình sang EPIC 28 mà không làm phụ thuộc việc lưu trạng thái vào kết quả gửi tin. | `US-ORD-03`, `EPIC 28` |
| `FR-ORD-10` | Ngăn tạo yêu cầu thông báo nghiệp vụ trùng cho cùng một lần chuyển trạng thái. | `US-ORD-03` |
| `FR-ORD-11` | Cho phép khách hủy Order chỉ khi chứng minh được quyền sở hữu và Order còn thỏa chính sách hủy tại thời điểm xử lý. | `US-ORD-04`, `BR-ORDER-04` |
| `FR-ORD-12` | Khi hủy hoặc làm hết hạn Order, phối hợp nhả reservation và chuyển nghĩa vụ hoàn tiền nếu có mà không lặp tác động. | `US-ORD-04`, `EPIC 07`, `EPIC 09`, `EPIC 14` |
| `FR-ORD-13` | Cho Sales Manager có quyền tìm kiếm, lọc và xem danh sách/chi tiết Order theo dữ liệu nghiệp vụ cần thiết. | `US-ORD-05` |
| `FR-ORD-14` | Chỉ cung cấp các hành động hợp lệ với trạng thái hiện tại và từ chối hành động khi quyền hoặc dữ liệu phụ thuộc chưa đáp ứng. | `US-ORD-05`, `EPIC 22` |
| `FR-ORD-15` | Tính, hiển thị và cập nhật hạn SLA theo trạng thái/công đoạn cùng chính sách đang áp dụng. | `US-ORD-06` |
| `FR-ORD-16` | Cung cấp hàng đợi Order theo công đoạn, mức ưu tiên và tình trạng SLA; không coi Order chưa đủ điều kiện là sẵn sàng xử lý. | `US-ORD-06` |
| `FR-ORD-17` | Cảnh báo Order sắp hoặc đã quá SLA và xác định công đoạn đang chịu trách nhiệm. | `US-ORD-06` |
| `FR-ORD-18` | Ghi Audit cho chuyển trạng thái, hủy, xử lý ngoại lệ và thay đổi ưu tiên quan trọng, gồm actor, nguồn, thời điểm, trước/sau và lý do khi cần. | `US-ORD-04~06`, `BR-ORDER-05`, `BR-AUDIT-01` |
| `FR-ORD-19` | Lưu nguồn Order và tham chiếu ngoài; chống tạo trùng khi tiếp nhận lại cùng giao dịch từ kênh tích hợp. | `US-ORD-05`, `BR-MKTPLACE-01`, `BR-MKTPLACE-04`, `EPIC 12/13/18` |

## 6. State machine và quy tắc dữ liệu

### 6.1. State machine cơ sở trong Product Backlog

```text
PENDING_PAYMENT
       │
       ├── PAID
       │     │
       │     └── CONFIRMED
       │            │
       │            └── PROCESSING
       │                   │
       │                   └── PACKED
       │                          │
       │                          └── SHIPPED
       │                                 │
       │                                 ├── DELIVERED
       │                                 │      │
       │                                 │      └── COMPLETED
       │                                 │
       │                                 └── DELIVERY_FAILED
       │                                        │
       │                                        └── RETURNED
       │
       ├── EXPIRED
       └── CANCELLED
```

Đây là state machine cơ sở cho luồng trả trước. COD cần nhánh trạng thái riêng hoặc điều kiện chuyển trạng thái bổ sung vì Order phải được fulfillment trước khi Payment được xác nhận; quyết định này phải được cập nhật đồng bộ vào Product Backlog trước Technical Design.

Luồng Return sau `DELIVERED` thuộc EPIC 14 và không được trộn vào trạng thái fulfillment chính của Order nếu chưa có quy tắc ánh xạ rõ ràng.

### 6.2. Quy tắc dữ liệu và an toàn nghiệp vụ

- Order phải có mã nội bộ duy nhất. Order từ kênh ngoài phải lưu thêm nguồn và tham chiếu ngoài duy nhất trong phạm vi nguồn đó.
- Snapshot Order phải chứa tối thiểu SKU, mô tả/quy cách cần thiết, đơn giá, số lượng, các thành phần tổng tiền, người nhận, địa chỉ, phương thức giao và ưu đãi đã chốt nếu có.
- Payment status, reservation status, packing status, shipment status và return status là dữ liệu nghiệp vụ liên quan nhưng không được coi là một trường Order status duy nhất nếu chúng có vòng đời độc lập.
- Chuyển trạng thái phải tuân theo trạng thái nguồn và điều kiện đầu vào; không cho sửa trực tiếp trạng thái để bỏ qua Payment, Inventory, Packing hoặc Shipping.
- Kết quả tích hợp gửi lặp phải cho cùng kết quả, không tạo nhiều lần chuyển trạng thái, nhả tồn, hoàn tiền hoặc thông báo.
- Order hết hạn vì chưa thanh toán phải giải phóng reservation theo chính sách nhưng không được che giấu giao dịch đến trễ; giao dịch đến trễ được đưa vào đối soát của EPIC 07.
- Hủy Order đã thu tiền không đồng nghĩa tiền đã hoàn. Order cancellation và Refund phải được theo dõi riêng cho đến khi EPIC 14/07 xác nhận kết quả.
- Chỉ chủ sở hữu đã xác minh hoặc nhân viên có quyền mới được xem Order. Trường dữ liệu nhạy cảm phải được giới hạn theo actor và mục đích sử dụng.
- Chi tiết cơ sở dữ liệu, API endpoint, orchestration/choreography, hàng đợi, lịch chạy, khóa phân tán và cơ chế realtime thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-ORD-01` | `BR-AUTH-01`, `BR-ORDER-01` | EPIC 01, 04, 06 | Customer chỉ thấy lịch sử của mình; Order Guest liên kết không bị nhân bản; snapshot lịch sử không đổi theo catalog. |
| `US-ORD-02` | `BR-ORDER-01`, `BR-ORDER-03`, `BR-ORDER-05` | EPIC 07, 10, 11 | Khách xem đúng trạng thái; kết quả hợp lệ cập nhật state machine; kết quả lặp/cũ không làm trạng thái sai lệch. |
| `US-ORD-03` | `BR-ORDER-05` | EPIC 28 | Thay đổi quan trọng tạo đúng một yêu cầu thông báo; gửi tin thất bại không làm mất trạng thái Order. |
| `US-ORD-04` | `BR-ORDER-04`, `BR-ORDER-05` | EPIC 07, 09, 14, 23 | Chỉ chủ sở hữu hủy được Order đủ điều kiện; tồn và tiền được xử lý đúng; yêu cầu lặp không gây tác động trùng. |
| `US-ORD-05` | `BR-ORDER-01~05`, `BR-AUDIT-01` | EPIC 07, 09, 10, 11, 22, 23 | Sales Manager xem dữ liệu thống nhất, chỉ thực hiện hành động hợp lệ và mọi thay đổi quan trọng có lịch sử. |
| `US-ORD-06` | `BR-ORDER-01`, `BR-ORDER-05` | EPIC 09, 10, 11, 23 | Hàng đợi chỉ chứa Order đủ điều kiện, ưu tiên đúng SLA và cảnh báo quá hạn có thể truy vết. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Bổ sung state machine chính thức cho COD và xác định quan hệ giữa `PAID`, `CONFIRMED`, `PROCESSING`, giao thành công và thu tiền.
- Xác định trạng thái trước `PENDING_PAYMENT` nếu cần lưu Order trong lúc reservation/khởi tạo Payment chưa hoàn tất; `PENDING_INVENTORY` và `REJECTED_STOCK_UNAVAILABLE` hiện chỉ xuất hiện trong tài liệu legacy.
- Ma trận chuyển trạng thái đầy đủ: trạng thái nguồn, trạng thái đích, actor/nguồn được phép, điều kiện bắt buộc và tác động bù trừ.
- Chính sách tự hủy: trạng thái nào cho phép, thời hạn, lý do bắt buộc và cách xử lý Order đã thanh toán hoặc đã tạo Shipment.
- Thời hạn Order chờ thanh toán và quan hệ chính xác với thời hạn reservation; không mặc định cố định 15 phút nếu chưa có quyết định nghiệp vụ.
- Cách xử lý Payment thành công đến sau khi Order đã `EXPIRED` hoặc `CANCELLED`.
- Điểm chuyển từ `DELIVERED` sang `COMPLETED`: tự động theo thời gian, khách xác nhận hay nhân viên xác nhận.
- Phân biệt `DELIVERY_FAILED`, `RETURNED` của fulfillment với `RETURN_REQUESTED`, `RETURNED`, `REFUNDED` của EPIC 14 để tránh hai nghĩa cho cùng tên trạng thái.
- Công thức SLA theo kênh, phương thức thanh toán, vùng giao, ngày làm việc, ngày lễ và từng công đoạn.
- Quyền xem dữ liệu và quyền thực hiện từng chuyển trạng thái cho Sales Manager, Warehouse, Packing, Delivery, CSKH và operator từng kênh.
- Quy tắc nguồn Order, mã tham chiếu và chống trùng cho Website/App, Marketplace, Offline và B2B.
- Chính sách chỉnh sửa Order sau khi tạo. Mặc định không sửa snapshot trực tiếp; thay đổi quan trọng cần quy trình điều chỉnh/hủy có Audit.
- Dữ liệu nào trong lịch sử trạng thái được phép hiển thị cho khách và dữ liệu nào chỉ dành cho vận hành nội bộ.

## 9. UI/UX Reference

- Lịch sử của Customer hiển thị mã đơn, ngày đặt, tổng tiền, trạng thái và hành động xem chi tiết; có trạng thái rỗng rõ ràng.
- Chi tiết Order tách rõ trạng thái Order, Payment và giao hàng; tránh dùng một nhãn duy nhất gây hiểu nhầm toàn bộ giao dịch.
- Guest Order Lookup yêu cầu thêm bằng chứng sở hữu ngoài mã Order và không tiết lộ dữ liệu trước khi xác minh.
- Hành động hủy phải hiển thị điều kiện, hậu quả đối với tồn/tiền và yêu cầu xác nhận; khi không còn đủ điều kiện cần chỉ ra hướng xử lý tiếp theo.
- Danh sách quản trị hỗ trợ tìm kiếm/lọc theo trạng thái, thời gian, nguồn, Payment và SLA; hành động không hợp lệ không được hiển thị như có thể thực hiện.
- Hàng đợi vận hành làm nổi bật Order sắp quá hạn, đã quá hạn, bị chặn và công đoạn chịu trách nhiệm.
- Chỉ tiêu thời gian phản hồi, cơ chế realtime, endpoint và công nghệ điều phối được đặc tả trong NFR hoặc Technical Design.

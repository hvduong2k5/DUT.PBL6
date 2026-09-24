# EPIC 11 — Shipping & Delivery

## 1. Mục tiêu Epic

Epic này quản lý Shipment từ lúc Order/kiện hàng đủ điều kiện bàn giao cho đến khi giao thành công, giao thất bại, hủy hoặc chuyển sang luồng hoàn hàng. Nhân viên vận hành có thể tạo vận đơn, khách theo dõi hành trình, Delivery Staff xử lý các chuyến được giao và hệ thống đồng bộ trạng thái từ Logistics Provider.

Shipping là nguồn dữ liệu về phương thức giao, nhà vận chuyển, mã vận đơn, kiện hàng, trạng thái giao và các mốc tracking. Shipment status phải được phân biệt với Order status: tạo vận đơn chưa đồng nghĩa Order đã `SHIPPED`; chỉ khi kiện hàng thực sự được bàn giao hoặc được nhà vận chuyển xác nhận theo chính sách thì EPIC 08 mới áp dụng trạng thái tương ứng.

Epic cũng cung cấp báo phí vận chuyển cho Checkout theo yêu cầu tích hợp Logistics. Epic không quản lý snapshot tổng tiền của Order, Payment/COD, đóng gói, số dư Inventory, state machine tổng thể của Order, gửi thông báo đa kênh hoặc phê duyệt Return/Refund. Các trách nhiệm này lần lượt thuộc EPIC 06, EPIC 07, EPIC 10, EPIC 09, EPIC 08, EPIC 28 và EPIC 14.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 2nd** | `US-SHIP-01`, `US-SHIP-02`, `US-SHIP-06` | Nhân viên tạo được vận đơn; khách theo dõi được giao hàng; trạng thái từ 3PL được đồng bộ đúng Shipment và Order. |
| **Giai đoạn 2** | `US-SHIP-03` đến `US-SHIP-05` | Hỗ trợ đội giao nội bộ và báo cáo tỷ lệ giao thành công/thất bại. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Theo dõi Shipment của Order Guest sau khi xác minh quyền sở hữu. |
| Registered Customer (`ACT-02`) | Theo dõi Shipment của Order thuộc tài khoản mình. |
| Warehouse Staff (`ACT-06`) | Hỗ trợ bàn giao kiện hàng và xem thông tin vận đơn theo quyền. |
| Packing Staff (`ACT-07`) | Tạo/yêu cầu tạo vận đơn cho kiện đã đóng gói khi được phân quyền. |
| Delivery Staff (`ACT-08`) | Xem danh sách Shipment được giao và cập nhật kết quả giao nội bộ. |
| Sales Manager (`ACT-12`) | Giám sát Shipment, xử lý ngoại lệ và xem hiệu quả giao hàng theo quyền. |
| Logistics Provider (`EXT-03`) | Báo phí, nhận yêu cầu vận đơn và cung cấp cập nhật tracking/giao hàng. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-04` | Nhân viên chỉ được thực hiện thao tác thuộc quyền được cấp. | Tạo vận đơn, phân công, cập nhật giao và xử lý ngoại lệ đều phải kiểm tra quyền. |
| `BR-ORDER-01` | Order phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Kết quả Shipment chỉ được ánh xạ sang chuyển trạng thái Order hợp lệ. |
| `BR-ORDER-04` | Khách chỉ được tự hủy Order trong thời gian cho phép. | Khi đã tạo/bàn giao Shipment, yêu cầu hủy phải kiểm tra thêm khả năng hủy giao vận. |
| `BR-ORDER-05` | Thay đổi trạng thái quan trọng của Order phải được ghi nhận. | Mốc bàn giao, giao thành công/thất bại và nguồn cập nhật phải có lịch sử. |
| `BR-PACK-04` | Packing Video phải liên kết với Order/Shipment tương ứng. | Shipment/kiện hàng phải có định danh ổn định để liên kết bằng chứng Packing khi áp dụng. |
| `BR-REFUND-04` | Thao tác hoàn tiền phải có Audit Log. | Kết quả giao COD/hoàn hàng cung cấp dữ liệu cho Payment/Refund nhưng Shipping không tự xác nhận đã hoàn tiền. |
| `BR-AUDIT-01` | Thao tác quan trọng phải được ghi Audit Log. | Điều chỉnh tracking, phân công và cập nhật thủ công phải có khả năng truy vết. |

> Product Backlog hiện chưa có nhóm `BR-SHIPPING`; các quy tắc riêng về tạo vận đơn, bàn giao, tracking và giao thất bại cần được bổ sung ở backlog để traceability đầy đủ hơn.

### 3.3. Phụ thuộc giữa Epic và hệ thống ngoài

- `EPIC 06 — Checkout`: yêu cầu báo phí theo địa chỉ, khối lượng/quy cách và lựa chọn giao; chốt phí vào snapshot Order.
- `EPIC 07 — Payment`: quản lý nghĩa vụ thu COD và nhận kết quả thu tiền hợp lệ từ Shipping/3PL.
- `EPIC 08 — Order Management`: cung cấp Order đủ điều kiện, quản lý Order status và nhận kết quả Shipment để chuyển trạng thái.
- `EPIC 09 — Inventory & Batch`: nhận mốc nghiệp vụ phù hợp để hoàn tất xuất kho hoặc xử lý hàng hoàn.
- `EPIC 10 — Packing & Fulfillment`: cung cấp kiện hàng đã đóng gói, trọng lượng/kích thước và bằng chứng liên quan.
- `EPIC 14 — Return / Refund / Complaint`: tiếp nhận trường hợp hoàn hàng, giao thất bại hoặc yêu cầu đổi trả theo quy trình riêng.
- `EPIC 22 — User / Role / Permission Administration`: quyết định quyền tạo, phân công, cập nhật và xem Shipment.
- `EPIC 23 — Audit & Security`: lưu dấu vết cập nhật nhạy cảm và xử lý ngoại lệ.
- `EPIC 24 — Finance & Business Analytics`: nhận dữ liệu cước đã đối soát cho báo cáo tài chính ở giai đoạn sau.
- `EPIC 28 — Omnichannel Notification`: gửi thông báo tracking; Shipping chỉ cung cấp trạng thái/sự kiện nghiệp vụ.
- `EXT-03 — Logistics Provider`: cung cấp quote, mã vận đơn, nhãn giao hàng và các mốc tracking từ 3PL.

## 4. User Stories chi tiết

### US-SHIP-01 — Tạo vận đơn cho Order

**Actor:** Packing Staff (`ACT-07`), Warehouse Staff (`ACT-06`), Sales Manager (`ACT-12`) được cấp quyền

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là nhân viên vận hành, tôi muốn tạo vận đơn cho đơn hàng để bàn giao cho đơn vị giao hàng.

**Giá trị nghiệp vụ:** Kiện hàng được đăng ký đúng người nhận, dịch vụ giao và số tiền COD để có thể bàn giao, theo dõi và đối soát.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo vận đơn 3PL thành công
  Given Order và kiện hàng đã đủ điều kiện chuyển sang Shipping
  And người nhận, địa chỉ, trọng lượng, kích thước, dịch vụ giao và tiền COD nếu có đều hợp lệ
  When nhân viên có quyền yêu cầu tạo vận đơn
  Then hệ thống tạo Shipment gắn với đúng Order và kiện hàng
  And lưu nhà vận chuyển, mã vận đơn, dịch vụ, phí dự kiến và dữ liệu nhãn nhận được
  And trạng thái cho biết vận đơn đã tạo nhưng chưa mặc nhiên coi kiện hàng đã bàn giao

Scenario: Order chưa đủ điều kiện tạo vận đơn
  Given Order chưa `PACKED`, kiện hàng chưa hoàn tất hoặc dữ liệu giao bắt buộc còn thiếu
  When nhân viên yêu cầu tạo vận đơn
  Then hệ thống từ chối yêu cầu
  And chỉ rõ điều kiện chưa đáp ứng
  And không tạo Shipment không hoàn chỉnh ở trạng thái có thể bàn giao

Scenario: Nhà vận chuyển từ chối yêu cầu
  Given dữ liệu Shipment đã được gửi tới Logistics Provider
  When nhà vận chuyển trả về lỗi địa chỉ, dịch vụ, trọng lượng hoặc điều kiện khác
  Then Shipment không được trình bày như vận đơn đã tạo thành công
  And hệ thống lưu lý do phù hợp để nhân viên sửa hoặc chọn phương án khác

Scenario: Gửi lặp yêu cầu tạo vận đơn
  Given một yêu cầu đã tạo Shipment và mã vận đơn thành công
  When cùng yêu cầu đó được gửi lại
  Then hệ thống trả về Shipment đã tạo
  And không tạo thêm vận đơn hoặc nghĩa vụ cước trùng lặp

Scenario: Bàn giao kiện hàng cho nhà vận chuyển
  Given Shipment đã có vận đơn và kiện hàng thực tế sẵn sàng
  When việc bàn giao được xác nhận hợp lệ bởi actor hoặc nguồn có thẩm quyền
  Then Shipment ghi nhận mốc bàn giao cùng thời điểm và nguồn xác nhận
  And EPIC 08 nhận kết quả để xem xét chuyển Order sang `SHIPPED`
```

### US-SHIP-02 — Theo dõi trạng thái giao hàng

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là khách hàng, tôi muốn theo dõi trạng thái giao hàng.

**Giá trị nghiệp vụ:** Khách biết kiện hàng đã được bàn giao, đang vận chuyển, đang giao hay đã giao xong để chủ động nhận hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Customer xem tracking của Order
  Given Customer đã đăng nhập và sở hữu Order có Shipment
  When Customer mở theo dõi giao hàng
  Then hệ thống hiển thị nhà vận chuyển, mã vận đơn, trạng thái hiện tại và các mốc tracking được phép công khai
  And không hiển thị dữ liệu nội bộ hoặc dữ liệu cá nhân không cần thiết

Scenario: Guest theo dõi Shipment hợp lệ
  Given Guest có thông tin tra cứu và bằng chứng sở hữu Order hợp lệ
  When Guest yêu cầu theo dõi giao hàng
  Then hệ thống hiển thị tracking của đúng Shipment
  And không yêu cầu Guest tạo tài khoản

Scenario: Không chứng minh được quyền sở hữu
  Given người dùng chỉ biết hoặc đoán được mã Order/mã vận đơn
  When người dùng không cung cấp đủ thông tin xác minh
  Then hệ thống từ chối truy cập
  And không tiết lộ tên, địa chỉ, số điện thoại hoặc hành trình chi tiết

Scenario: Chưa có tracking
  Given Order chưa có Shipment hoặc nhà vận chuyển chưa cung cấp mốc hành trình
  When khách mở theo dõi
  Then hệ thống hiển thị trạng thái hiện tại một cách trung thực
  And không tạo mốc giao hàng giả định

Scenario: Order có nhiều Shipment
  Given một Order được tách thành nhiều kiện hoặc nhiều Shipment
  When khách xem theo dõi giao hàng
  Then hệ thống hiển thị riêng trạng thái của từng Shipment
  And trạng thái tổng hợp không làm khách hiểu nhầm toàn bộ Order đã giao khi vẫn còn kiện chưa hoàn tất
```

### US-SHIP-03 — Delivery Staff xem danh sách đơn được giao

**Actor:** Delivery Staff (`ACT-08`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Delivery Staff, tôi muốn xem danh sách đơn được giao cho mình để thực hiện giao hàng.

**Giá trị nghiệp vụ:** Nhân viên giao nội bộ biết đúng Shipment, thứ tự và thông tin liên hệ cần thiết trong phạm vi nhiệm vụ được phân công.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem danh sách Shipment được phân công
  Given Delivery Staff đã đăng nhập và có các Shipment được giao
  When nhân viên mở danh sách công việc
  Then hệ thống chỉ hiển thị Shipment đang được phân công cho nhân viên đó
  And hiển thị thông tin người nhận, địa chỉ, liên hệ, hạn giao và COD cần thu nếu có ở mức cần thiết

Scenario: Sắp xếp công việc giao hàng
  Given Delivery Staff có nhiều Shipment đang chờ giao
  When xem danh sách
  Then hệ thống hiển thị mức ưu tiên hoặc thứ tự theo SLA/chuyến giao đang áp dụng
  And làm rõ Shipment bị chặn hoặc chưa sẵn sàng

Scenario: Không xem Shipment của nhân viên khác
  Given Shipment được giao cho Delivery Staff B
  When Delivery Staff A cố truy cập ngoài phân công
  Then hệ thống từ chối truy cập
  And không tiết lộ thông tin người nhận

Scenario: Phân công thay đổi
  Given Shipment được người có quyền chuyển từ Delivery Staff A sang B
  When thay đổi phân công có hiệu lực
  Then danh sách của hai nhân viên được cập nhật phù hợp
  And lịch sử người giao trước/sau cùng lý do được bảo toàn
```

### US-SHIP-04 — Delivery Staff cập nhật trạng thái giao hàng

**Actor:** Delivery Staff (`ACT-08`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Delivery Staff, tôi muốn cập nhật trạng thái giao hàng để hệ thống phản ánh đúng tiến trình.

**Giá trị nghiệp vụ:** Khách và đội vận hành nhận được tiến trình giao thực tế; Payment COD và Order có dữ liệu đúng để xử lý bước tiếp theo.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Cập nhật một bước giao hợp lệ
  Given Delivery Staff được phân công Shipment đang ở trạng thái phù hợp
  When nhân viên chọn trạng thái tiếp theo và cung cấp dữ liệu bắt buộc
  Then Shipment chuyển sang trạng thái hợp lệ
  And lưu actor, thời điểm và ghi chú/bằng chứng nếu chính sách yêu cầu
  And EPIC 08 nhận kết quả cần thiết để cập nhật Order

Scenario: Xác nhận giao thành công
  Given Shipment đang được giao
  When Delivery Staff xác nhận giao thành công với bằng chứng bắt buộc nếu có
  Then Shipment được ghi nhận `DELIVERED`
  And kết quả thu COD được chuyển sang EPIC 07 nếu Shipment có nghĩa vụ thu tiền
  And việc giao thành công không tự suy diễn tiền COD đã thu nếu chưa có xác nhận hợp lệ

Scenario: Giao thất bại
  Given Shipment đang được giao
  When Delivery Staff xác nhận giao thất bại
  Then hệ thống yêu cầu lý do thất bại theo danh mục/chính sách
  And Shipment phản ánh kết quả thất bại cùng lần giao liên quan
  And Order/Return chỉ được cập nhật theo quy tắc của EPIC 08 và EPIC 14

Scenario: Chuyển trạng thái không hợp lệ
  Given trạng thái đích không hợp lệ với trạng thái Shipment hiện tại
  When Delivery Staff gửi cập nhật
  Then hệ thống từ chối thay đổi
  And giữ nguyên trạng thái hiện tại

Scenario: Cập nhật lặp
  Given một mốc giao đã được lưu thành công
  When cùng yêu cầu được gửi lại
  Then hệ thống không tạo mốc tracking hoặc tác động Order/Payment lần thứ hai
```

### US-SHIP-05 — Xem tỷ lệ giao thành công/thất bại

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là quản lý, tôi muốn xem tỷ lệ giao thành công/thất bại để đánh giá hiệu quả giao hàng.

**Giá trị nghiệp vụ:** Quản lý nhận biết chất lượng giao theo nhà vận chuyển, khu vực hoặc thời kỳ để điều chỉnh vận hành.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem tỷ lệ giao theo kỳ
  Given có Shipment với kết quả giao trong khoảng thời gian được chọn
  When Sales Manager mở báo cáo hiệu quả giao hàng
  Then hệ thống hiển thị số giao thành công, giao thất bại và tỷ lệ tương ứng
  And nêu rõ kỳ dữ liệu, mẫu số và thời điểm cập nhật

Scenario: Phân tích theo chiều nghiệp vụ
  Given dữ liệu có nhiều nhà vận chuyển, khu vực hoặc kênh Order
  When Sales Manager chọn chiều phân tích được hỗ trợ
  Then kết quả được nhóm/lọc đúng theo lựa chọn
  And tổng chi tiết nhất quán với tổng chung trong cùng phạm vi

Scenario: Phân biệt giao thất bại với hủy trước bàn giao
  Given có Shipment giao thất bại và Shipment bị hủy trước khi thực sự giao
  When hệ thống tính tỷ lệ
  Then hai loại kết quả được phân biệt theo định nghĩa chỉ số
  And không làm sai tỷ lệ bằng cách mặc định coi mọi Shipment hủy là giao thất bại

Scenario: Chưa đủ dữ liệu
  Given khoảng thời gian được chọn không có Shipment đủ điều kiện tính
  When Sales Manager xem báo cáo
  Then hệ thống hiển thị trạng thái chưa có dữ liệu
  And không hiển thị tỷ lệ gây hiểu nhầm

Scenario: Giới hạn dữ liệu chi tiết
  Given Sales Manager được xem chỉ số nhưng không có quyền xem một số dữ liệu cá nhân
  When mở phần chi tiết
  Then hệ thống vẫn áp dụng phạm vi quyền
  And không để báo cáo trở thành cách vượt quyền truy cập Shipment
```

### US-SHIP-06 — Đồng bộ trạng thái từ Logistics Provider

**Actor:** Hệ thống, Logistics Provider (`EXT-03`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là hệ thống, tôi muốn nhận cập nhật trạng thái từ đơn vị giao vận để đồng bộ trạng thái đơn hàng.

**Giá trị nghiệp vụ:** Shipment và Order phản ánh hành trình 3PL kịp thời, chính xác mà không phụ thuộc nhân viên cập nhật thủ công.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Nhận cập nhật tracking hợp lệ
  Given Shipment có mã vận đơn thuộc Logistics Provider
  When hệ thống nhận và xác minh cập nhật hợp lệ cho mã đó
  Then trạng thái ngoài được ánh xạ sang trạng thái Shipment nội bộ phù hợp
  And lưu trạng thái gốc, thời điểm sự kiện, thời điểm nhận và nguồn cập nhật
  And EPIC 08 nhận kết quả nếu cần chuyển trạng thái Order

Scenario: Cập nhật bị gửi lặp
  Given một mốc tracking đã được xử lý
  When Logistics Provider gửi lại cùng định danh/nội dung sự kiện
  Then hệ thống không tạo mốc hoặc tác động Order nhiều lần
  And phản ánh kết quả xử lý nhất quán

Scenario: Cập nhật đến không đúng thứ tự
  Given Shipment đã nhận một mốc mới hơn
  When mốc cũ hơn đến sau
  Then hệ thống bảo toàn dữ liệu gốc để đối soát
  And không làm trạng thái hiện tại quay lùi trái phép

Scenario: Mã vận đơn không xác định
  Given hệ thống nhận cập nhật cho mã vận đơn không liên kết được Shipment
  When xử lý cập nhật
  Then hệ thống không thay đổi Order bất kỳ
  And đưa trường hợp vào danh sách cần kiểm tra với dữ liệu nguồn cần thiết

Scenario: Nguồn cập nhật không hợp lệ
  Given yêu cầu cập nhật không xác minh được nguồn hoặc tính toàn vẹn
  When hệ thống tiếp nhận
  Then hệ thống từ chối áp dụng trạng thái
  And không tiết lộ dữ liệu Shipment trong phản hồi không cần thiết

Scenario: Trạng thái nhà vận chuyển chưa có ánh xạ
  Given Logistics Provider gửi một trạng thái mới hoặc không được hỗ trợ
  When hệ thống không tìm thấy quy tắc ánh xạ
  Then dữ liệu gốc được lưu vào diện cần xử lý
  And Shipment/Order không bị chuyển sang trạng thái suy đoán
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-SHIP-01` | Cung cấp lựa chọn dịch vụ và báo phí dựa trên địa chỉ, dữ liệu kiện hàng và điều kiện Logistics Provider cho EPIC 06. | `US-CHK-02`, yêu cầu hệ thống ngoài 36.7 |
| `FR-SHIP-02` | Chỉ tạo Shipment khi Order/Package đủ điều kiện và dữ liệu người nhận, dịch vụ, trọng lượng, kích thước, COD đều hợp lệ. | `US-SHIP-01`, `EPIC 08`, `EPIC 10` |
| `FR-SHIP-03` | Tạo Shipment với mã nội bộ duy nhất, liên kết Order/Package, nhà vận chuyển và mã vận đơn ngoài. | `US-SHIP-01` |
| `FR-SHIP-04` | Chống tạo nhiều Shipment/vận đơn hoặc nghĩa vụ cước khi cùng yêu cầu tạo vận đơn được gửi lặp. | `US-SHIP-01` |
| `FR-SHIP-05` | Phân biệt mốc tạo vận đơn với mốc bàn giao thực tế; chỉ cung cấp kết quả `SHIPPED` cho EPIC 08 khi điều kiện bàn giao đã đạt. | `US-SHIP-01`, `BR-ORDER-01` |
| `FR-SHIP-06` | Cho Customer hoặc Guest đã xác minh quyền sở hữu xem đúng Shipment và các mốc tracking được phép công khai. | `US-SHIP-02` |
| `FR-SHIP-07` | Hiển thị riêng từng Shipment/Package khi một Order được chia nhiều kiện và cung cấp trạng thái tổng hợp không gây hiểu nhầm. | `US-SHIP-02` |
| `FR-SHIP-08` | Cho Delivery Staff xem thông tin cần thiết của các Shipment đang được phân công cho mình. | `US-SHIP-03` |
| `FR-SHIP-09` | Quản lý phân công/reassignment Delivery Staff có quyền và lưu lịch sử thay đổi. | `US-SHIP-03`, `BR-AUDIT-01` |
| `FR-SHIP-10` | Cho Delivery Staff được phân công cập nhật các chuyển trạng thái hợp lệ với dữ liệu/lý do/bằng chứng bắt buộc. | `US-SHIP-04` |
| `FR-SHIP-11` | Chuyển kết quả giao COD hợp lệ sang EPIC 07 nhưng không tự xác nhận Payment khi chưa đủ bằng chứng thu tiền. | `US-SHIP-04`, `EPIC 07` |
| `FR-SHIP-12` | Chuyển kết quả giao thành công, giao thất bại hoặc hoàn hàng sang EPIC 08/14 mà không xử lý trùng. | `US-SHIP-04`, `EPIC 08`, `EPIC 14` |
| `FR-SHIP-13` | Tính tỷ lệ giao thành công/thất bại theo định nghĩa, kỳ và chiều phân tích được chọn; phân biệt hủy trước giao. | `US-SHIP-05` |
| `FR-SHIP-14` | Áp dụng quyền truy cập cho báo cáo và dữ liệu Shipment chi tiết. | `US-SHIP-05`, `BR-AUTH-04` |
| `FR-SHIP-15` | Tiếp nhận và xác minh nguồn/tính toàn vẹn của cập nhật tracking trước khi áp dụng. | `US-SHIP-06` |
| `FR-SHIP-16` | Ánh xạ trạng thái từng Logistics Provider sang Shipment status nội bộ có kiểm soát. | `US-SHIP-06` |
| `FR-SHIP-17` | Xử lý cập nhật lặp, đến sai thứ tự, không rõ mã vận đơn hoặc chưa có ánh xạ mà không làm sai Shipment/Order. | `US-SHIP-06` |
| `FR-SHIP-18` | Lưu trạng thái gốc, thời điểm sự kiện, thời điểm nhận, nguồn và kết quả ánh xạ để truy vết/đối soát. | `US-SHIP-06`, `BR-ORDER-05` |
| `FR-SHIP-19` | Chuyển yêu cầu thông báo cho các mốc tracking được cấu hình sang EPIC 28 mà không phụ thuộc việc lưu trạng thái vào kết quả gửi tin. | `US-SHIP-02`, `US-SHIP-06`, `EPIC 28` |
| `FR-SHIP-20` | Cung cấp dữ liệu phí dự kiến/thực tế và tham chiếu Shipment phục vụ đối soát cước, không thay thế báo cáo Finance của EPIC 24. | Yêu cầu hệ thống ngoài 36.7, `EPIC 24` |
| `FR-SHIP-21` | Ghi Audit cho điều chỉnh trạng thái, phân công, ánh xạ ngoại lệ và thao tác quản trị quan trọng. | `US-SHIP-03~06`, `BR-AUDIT-01` |

## 6. Vòng đời Shipment và quy tắc dữ liệu

### 6.1. Vòng đời nghiệp vụ đề xuất

```text
DRAFT / CREATING
       │
       ├── CREATED
       │      │
       │      ├── READY_FOR_HANDOVER
       │      │          │
       │      │          └── HANDED_OVER / IN_TRANSIT
       │      │                         │
       │      │                         ├── OUT_FOR_DELIVERY
       │      │                         │          ├── DELIVERED
       │      │                         │          └── DELIVERY_FAILED
       │      │                         └── RETURNING / RETURNED
       │      └── CANCELLED
       └── CREATION_FAILED
```

Tên trạng thái và ma trận chuyển chính thức phải được thống nhất trước Technical Design. Trạng thái ngoài của từng Logistics Provider được lưu nguyên bản và ánh xạ sang trạng thái nội bộ; không sử dụng trực tiếp chuỗi trạng thái ngoài làm Order status.

### 6.2. Quy tắc dữ liệu và an toàn nghiệp vụ

- Shipment phải có mã nội bộ duy nhất; mã vận đơn ngoài phải duy nhất trong phạm vi Logistics Provider phù hợp.
- Shipment liên kết với đúng Order và Package. Nếu một Order có nhiều kiện/Shipment, từng kiện phải theo dõi độc lập.
- Quote dùng ở Checkout và phí trên Shipment phải giữ tham chiếu/phiên bản cần thiết để giải thích chênh lệch giữa phí dự kiến, phí chốt và phí thực tế.
- Tạo vận đơn không đồng nghĩa đã xuất kho, bàn giao hoặc `SHIPPED`. Mỗi mốc cần bằng chứng/nguồn xác nhận riêng theo chính sách.
- Shipment status, Order status, Payment/COD status và Return status là các vòng đời riêng; việc ánh xạ giữa chúng phải qua quy tắc rõ ràng.
- Cập nhật tracking phải bảo toàn trạng thái gốc từ provider, trạng thái ánh xạ, thời gian sự kiện và thời gian hệ thống nhận để xử lý sự kiện đến trễ.
- Cùng một sự kiện không được tạo nhiều mốc, nhiều lần chuyển Order, xác nhận COD hoặc thông báo.
- Dữ liệu người nhận chỉ được hiển thị cho chủ sở hữu hoặc nhân viên có nhiệm vụ/quyền phù hợp; không để mã vận đơn đơn thuần trở thành chìa khóa truy cập dữ liệu cá nhân.
- Cập nhật thủ công không được ghi đè làm mất dữ liệu provider; mọi ngoại lệ phải lưu actor, lý do và Audit.
- Chi tiết endpoint, chữ ký xác thực, retry, hàng đợi, polling, định dạng label/mã vạch, bản đồ và tối ưu tuyến thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-SHIP-01` | `BR-AUTH-04`, `BR-ORDER-01`, `BR-ORDER-05` | EPIC 08, 10; EXT-03 | Chỉ tạo vận đơn cho kiện đủ điều kiện; yêu cầu lặp không tạo trùng; tạo vận đơn chưa chuyển `SHIPPED`. |
| `US-SHIP-02` | `BR-ORDER-01` | EPIC 01, 08, 28 | Chủ sở hữu xem đúng tracking; Guest phải xác minh; nhiều Shipment được hiển thị riêng và an toàn. |
| `US-SHIP-03` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 08, 22, 23 | Delivery Staff chỉ thấy việc được giao; reassignment có quyền và lịch sử. |
| `US-SHIP-04` | `BR-ORDER-01`, `BR-ORDER-05` | EPIC 07, 08, 14, 23 | Chỉ cập nhật hợp lệ; COD không tự suy diễn; giao thất bại có lý do; cập nhật lặp không gây tác động trùng. |
| `US-SHIP-05` | `BR-AUTH-04` | EPIC 08, 24 | Chỉ số có kỳ/mẫu số rõ, phân biệt hủy và thất bại, dữ liệu chi tiết tuân theo quyền. |
| `US-SHIP-06` | `BR-ORDER-01`, `BR-ORDER-05` | EPIC 08, 23, 28; EXT-03 | Nguồn được xác minh; trạng thái ánh xạ đúng; sự kiện lặp/cũ/không rõ không làm sai Shipment hoặc Order. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Bổ sung nhóm Business Rule `BR-SHIPPING` vào Product Backlog để quy định nguồn trạng thái, bàn giao, giao thất bại, chống trùng và bảo vệ tracking.
- Nhà vận chuyển/dịch vụ nào được bật trong MVP và cơ chế chọn provider khi Checkout báo phí khác provider thực tế.
- Một Order có được tách nhiều Package/Shipment, giao một phần hoặc đổi nhà vận chuyển sau khi tạo vận đơn hay không.
- Mốc chính thức để Order chuyển `SHIPPED`: quét bàn giao nội bộ, provider nhận hàng hay mốc tracking đầu tiên.
- Vòng đời Shipment chuẩn và bảng ánh xạ trạng thái riêng của GHN, GHTK, Viettel Post cùng quy tắc xử lý trạng thái mới.
- Quy tắc hủy vận đơn trước/sau bàn giao, phí phát sinh và quan hệ với hủy Order.
- Chính sách thử giao lại, số lần giao tối đa, lý do thất bại và thời điểm chuyển hoàn.
- Phân biệt `DELIVERY_FAILED`/`RETURNED` của Shipping với state machine Return/Refund của EPIC 14.
- Điều kiện và bằng chứng xác nhận giao thành công cho 3PL và Delivery Staff nội bộ.
- Nguồn xác nhận thu COD, xử lý thu thiếu/thừa, tiền chưa nộp và thời điểm chuyển dữ liệu sang Payment.
- Cách bảo vệ dữ liệu cá nhân trên nhãn giao hàng, màn hình Delivery Staff, tracking công khai và log.
- Công thức tỷ lệ giao thành công/thất bại, mẫu số, kỳ ghi nhận, giao lại và Order nhiều Shipment.
- Product Backlog yêu cầu đối soát cước nhưng chưa có User Story riêng; cần bổ sung actor, quy trình, chu kỳ và tiêu chí sai lệch nếu đưa vào phạm vi.
- Quan hệ giữa phí quote ở Checkout, phí chốt trên Order, phí provider dự kiến và phí thực tế khi đối soát.
- Chính sách lưu tracking, nhãn vận đơn, bằng chứng giao và dữ liệu vị trí nếu sau này hỗ trợ bản đồ realtime.
- Cơ chế thông báo cho khách ở những mốc nào thuộc EPIC 28; lỗi gửi tin không được làm lùi trạng thái Shipment.

## 9. UI/UX Reference

- Màn hình tạo vận đơn hiển thị Order/Package, người nhận, dịch vụ, trọng lượng/kích thước, COD và phí trước khi xác nhận.
- Phân biệt rõ “đã tạo vận đơn”, “sẵn sàng bàn giao”, “đã bàn giao”, “đang giao”, “giao thành công” và “giao thất bại”.
- Nhãn vận đơn/mã vạch chỉ được in hoặc tải bởi actor có quyền và phải dùng đúng Shipment/Package.
- Trang tracking của khách hiển thị timeline dễ hiểu, thời điểm cập nhật và từng kiện nếu Order có nhiều Shipment.
- Danh sách Delivery Staff chỉ hiển thị dữ liệu cần cho chuyến được phân công; số điện thoại/địa chỉ không được lộ ngoài nhiệm vụ.
- Màn hình ngoại lệ làm nổi bật sự kiện không rõ mã, sai thứ tự, chưa có ánh xạ hoặc bị provider từ chối.
- Báo cáo hiệu quả luôn hiển thị kỳ, mẫu số và bộ lọc; tỷ lệ không có dữ liệu không được trình bày như 0% thành công.
- Chỉ tiêu hiệu năng, realtime map, webhook, polling, endpoint và định dạng label được đặc tả trong NFR hoặc Technical Design.

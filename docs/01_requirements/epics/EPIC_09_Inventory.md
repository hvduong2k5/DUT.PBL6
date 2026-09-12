# EPIC 09 — Inventory & Batch

## 1. Mục tiêu Epic

Epic này quản lý tồn kho theo SKU và Batch/Lot, bao gồm nhập kho, xuất kho, giữ/nhả tồn, ngày sản xuất, hạn sử dụng và các điều chỉnh phản ánh chênh lệch thực tế. Hệ thống phải xác định được lượng tồn khả dụng chính xác, ngăn bán vượt mức và truy xuất được mọi biến động về lô hàng.

Inventory là nguồn sự thật về số lượng vật lý, số lượng đang giữ, số lượng không đủ điều kiện bán và số lượng khả dụng. Mọi biến động phải có loại nghiệp vụ, nguồn tham chiếu, thời điểm và actor phù hợp; không được sửa trực tiếp số dư mà không để lại lịch sử.

Epic không quản lý Product/SKU catalog, tạo Order, quy trình Checkout, thao tác đóng gói, chính sách mua hàng từ nhà cung cấp, gửi thông báo hay phân tích/dự báo nâng cao. Các trách nhiệm này lần lượt thuộc EPIC 04, EPIC 08, EPIC 06, EPIC 10, EPIC 27, EPIC 28 và EPIC 25.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 2nd** | `US-INV-01` đến `US-INV-06`, `US-INV-09` | Warehouse Staff quản lý được số lượng theo SKU/Batch, nhập xuất, HSD, cảnh báo sắp hết hạn và điều chỉnh tồn có kiểm soát. |
| **Giai đoạn 2** | `US-INV-07`, `US-INV-08`, `US-INV-10` | Supply/Sales theo dõi tiêu thụ, ưu tiên xuất gần hết hạn và phối hợp xử lý hàng có rủi ro HSD. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Warehouse Staff (`ACT-06`) | Xem tồn, ghi nhận nhập/xuất, quản lý Batch/HSD và khai báo hỏng, thất thoát hoặc điều chỉnh theo quyền. |
| Packing Staff (`ACT-07`) | Nhận Batch/SKU được phân bổ để lấy hàng; xác nhận đóng gói thuộc EPIC 10. |
| Sales Manager (`ACT-12`) | Xem tồn và cảnh báo sắp hết hạn để phối hợp bán hàng, không tự ý điều chỉnh kho nếu không có quyền. |
| Inventory/Supply Manager (`ACT-15`) | Theo dõi tồn, tốc độ tiêu thụ, rủi ro HSD và lập kế hoạch cung ứng. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-PROD-02` | Giá, khối lượng, quy cách đóng gói và tồn kho được quản lý theo SKU. | Mọi số dư và biến động tồn phải gắn với đúng SKU. |
| `BR-PROD-03` | Sản phẩm thực phẩm phải có ngày sản xuất và hạn sử dụng phù hợp. | Batch thực phẩm phải có dữ liệu ngày hợp lệ trước khi được coi là đủ điều kiện bán. |
| `BR-BATCH-01` | Tồn kho thực phẩm phải có thể truy xuất theo Batch/Lot. | Số lượng SKU phải phân rã và truy ngược được về từng Batch liên quan. |
| `BR-BATCH-02` | Mỗi Batch/Lot có mã lô, NSX, HSD, số lượng nhập/còn lại và nguồn nếu có. | Không cho hoàn tất nhập Batch khi thiếu dữ liệu bắt buộc áp dụng. |
| `BR-BATCH-03` | Hệ thống phải cảnh báo trước khi Batch/Lot đến HSD. | Batch đạt ngưỡng cảnh báo phải xuất hiện cho đúng actor và không tạo cảnh báo trùng không cần thiết. |
| `BR-BATCH-04` | Sản phẩm gần hết hạn phải được ưu tiên xử lý theo chính sách. | Dữ liệu rủi ro HSD phải hỗ trợ Sales/Supply quyết định phương án xử lý. |
| `BR-BATCH-05` | Sản phẩm hết hạn không được phép tiếp tục bán. | Số lượng hết hạn phải bị loại khỏi tồn khả dụng và không được reserve/xuất bán. |
| `BR-BATCH-06` | Nên ưu tiên Batch có HSD gần hơn khi phù hợp chính sách. | Đề xuất phân bổ xuất kho phải áp dụng FEFO có điều kiện và cho phép ngoại lệ được kiểm soát. |
| `BR-ORDER-02` | Không xác nhận đơn vượt quá tồn kho khả dụng. | Reservation phải nguyên tử và không làm tổng lượng giữ vượt lượng đủ điều kiện bán. |
| `BR-MKTPLACE-05` | Order Marketplace mới phải trừ/tạm giữ tồn khả dụng ngay. | EPIC 12 phải sử dụng cùng cơ chế reservation để tránh oversell đa kênh. |
| `BR-MKTPLACE-06` | Order Marketplace bị hủy phải hoàn trả tồn tương ứng. | Việc nhả tồn phải gắn với đúng reservation/Order và chống xử lý lặp. |
| `BR-OFFLINE-02` | Hàng cấp cho Offline Sales phải được ghi nhận. | Xuất/chuyển tồn sang kênh Offline phải có chứng từ và người nhận. |
| `BR-OFFLINE-03` | Offline Staff báo cáo bán, tồn, hỏng và trả theo kỳ. | Kết quả đối soát kênh Offline tạo biến động/điều chỉnh có nguồn rõ ràng. |
| `BR-OFFLINE-04` | Tồn Offline phải được đối soát với tồn hệ thống. | Chênh lệch chỉ được điều chỉnh theo quyền và phải giữ lịch sử. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Điều chỉnh, ghi nhận thất thoát và ngoại lệ xuất Batch phải có khả năng truy vết. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 04 — Product, Variant & SKU`: cung cấp SKU, khối lượng, quy cách và trạng thái bán; Inventory không tạo hoặc sửa catalog.
- `EPIC 05 — Shopping Cart`: đọc khả năng mua để cảnh báo; Cart không giữ tồn.
- `EPIC 06 — Checkout`: yêu cầu kiểm tra và giữ tồn nguyên tử cho các dòng SKU đã xác nhận.
- `EPIC 08 — Order Management`: cung cấp trạng thái Order để chuyển reservation thành xuất kho hoặc giải phóng đúng thời điểm.
- `EPIC 10 — Packing & Fulfillment`: nhận danh sách SKU/Batch cần lấy và cung cấp kết quả thực tế phục vụ xuất kho.
- `EPIC 12 — Marketplace` và `EPIC 13 — Offline Sales`: tạo biến động tồn đa kênh nhưng phải dùng cùng nguồn số dư Inventory.
- `EPIC 14 — Return / Refund / Complaint`: cung cấp kết quả kiểm tra hàng trả để quyết định nhập lại tồn bán được, cách ly hoặc ghi nhận hỏng.
- `EPIC 19 — OCOP Traceability`: sử dụng Batch/nguồn lô để truy xuất minh bạch sản phẩm.
- `EPIC 23 — Audit & Security`: lưu dấu vết thao tác nhạy cảm và thay đổi số lượng.
- `EPIC 25 — Customer Analytics & DSS`: phân tích tốc độ bán, dự báo và rủi ro tồn dựa trên dữ liệu Inventory.
- `EPIC 27 — Procurement & Supplier Management`: cung cấp PO/nguồn nhận để đối chiếu khi nhập kho.
- `EPIC 28 — Omnichannel Notification`: gửi cảnh báo đến actor; Inventory chỉ xác định điều kiện và dữ liệu cảnh báo.

## 4. User Stories chi tiết

### US-INV-01 — Xem tồn kho theo SKU

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Warehouse Staff, tôi muốn xem tồn kho theo SKU để biết số lượng sản phẩm khả dụng.

**Giá trị nghiệp vụ:** Nhân viên biết lượng vật lý, lượng đang giữ và lượng có thể bán để xử lý Order mà không gây thiếu hụt.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem số dư của một SKU
  Given Warehouse Staff có quyền xem tồn kho
  When nhân viên tra cứu một SKU
  Then hệ thống hiển thị số lượng hiện có, đang giữ, không đủ điều kiện bán và khả dụng
  And số lượng có thể được phân rã theo Batch và vị trí nếu nghiệp vụ có quản lý vị trí

Scenario: Tồn khả dụng phản ánh reservation
  Given một SKU có tồn đủ điều kiện bán và một phần đã được giữ cho Order
  When Warehouse Staff hoặc Checkout tra cứu tồn khả dụng
  Then lượng đã giữ không còn được tính là khả dụng cho reservation mới
  And tổng số lượng được trình bày nhất quán theo các thành phần số dư

Scenario: Hai yêu cầu giữ đơn vị cuối cùng
  Given một SKU chỉ còn một đơn vị khả dụng
  When hai yêu cầu hợp lệ cùng cố giữ đơn vị đó
  Then chỉ một yêu cầu giữ tồn thành công
  And yêu cầu còn lại nhận kết quả không đủ tồn
  And tổng reservation không vượt tồn đủ điều kiện bán

Scenario: SKU hoặc Batch hết hạn
  Given một SKU còn số lượng vật lý nhưng toàn bộ số lượng thuộc Batch đã hết hạn
  When hệ thống xác định tồn khả dụng để bán
  Then số lượng hết hạn không được tính vào tồn khả dụng
  And không thể được giữ cho Order bán hàng
```

### US-INV-02 — Ghi nhận nhập kho

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Warehouse Staff, tôi muốn ghi nhận nhập kho để cập nhật số lượng thực tế.

**Giá trị nghiệp vụ:** Hàng nhận thực tế được cộng đúng SKU/Batch và có nguồn tham chiếu để kiểm tra chênh lệch sau này.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Nhập kho một Batch hợp lệ
  Given SKU đã tồn tại và Warehouse Staff có quyền nhập kho
  When nhân viên ghi nhận số lượng nhận dương cùng thông tin Batch bắt buộc
  Then hệ thống tạo biến động nhập gắn với đúng SKU và Batch
  And cập nhật số dư sau khi giao dịch nhập hoàn tất
  And lưu actor, thời điểm và nguồn tham chiếu nếu có

Scenario: Nhập kho theo PO
  Given có PO từ EPIC 27 và hàng đã được nhận thực tế
  When Warehouse Staff đối chiếu rồi ghi nhận nhập kho
  Then biến động nhập liên kết với PO và số lượng thực nhận
  And chênh lệch với số lượng đặt được bảo toàn để xử lý

Scenario: Từ chối dữ liệu nhập không hợp lệ
  Given Warehouse Staff đang ghi nhận nhập kho
  When SKU không tồn tại, số lượng không dương hoặc Batch thiếu dữ liệu bắt buộc
  Then hệ thống không cập nhật tồn
  And chỉ rõ dữ liệu cần sửa

Scenario: Chứng từ nhập bị gửi lặp
  Given một nghiệp vụ nhập có tham chiếu duy nhất đã được ghi nhận
  When cùng yêu cầu được gửi lại
  Then hệ thống không cộng tồn lần thứ hai
  And trả về kết quả nhất quán của nghiệp vụ đã ghi nhận
```

### US-INV-03 — Ghi nhận xuất kho

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Warehouse Staff, tôi muốn ghi nhận xuất kho để theo dõi lượng hàng đã sử dụng.

**Giá trị nghiệp vụ:** Hệ thống phản ánh đúng hàng đã rời kho và truy được SKU/Batch cùng lý do xuất.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xuất kho cho Order hợp lệ
  Given Order đủ điều kiện xuất và có SKU/Batch được phân bổ
  When Warehouse Staff xác nhận lượng thực xuất
  Then hệ thống ghi biến động xuất cho đúng SKU, Batch và Order
  And giảm số dư tương ứng mà không làm tồn hợp lệ âm
  And không để reservation đã chuyển thành xuất tiếp tục chiếm tồn khả dụng

Scenario: Từ chối xuất vượt số lượng
  Given Batch không đủ số lượng có thể xuất
  When Warehouse Staff yêu cầu xuất số lượng lớn hơn số dư hợp lệ
  Then hệ thống từ chối biến động
  And không làm bất kỳ số dư nào trở thành âm

Scenario: Xuất kho ngoài Order
  Given có lý do nghiệp vụ được phê duyệt để xuất không gắn Order bán hàng
  When Warehouse Staff ghi nhận số lượng, Batch và tham chiếu bắt buộc
  Then hệ thống tạo biến động xuất đúng loại nghiệp vụ
  And lưu người thực hiện cùng lý do để truy vết

Scenario: Yêu cầu xuất bị gửi lặp
  Given một nghiệp vụ xuất đã hoàn tất với tham chiếu duy nhất
  When cùng yêu cầu được gửi lại
  Then hệ thống không trừ tồn lần thứ hai
```

### US-INV-04 — Quản lý Batch/Lot

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Warehouse Staff, tôi muốn quản lý Batch/Lot để truy xuất từng lô sản phẩm.

**Giá trị nghiệp vụ:** Mọi lượng hàng thực phẩm có thể truy ngược về lô sản xuất và nguồn liên quan khi kiểm kê, khiếu nại hoặc thu hồi.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo Batch với dữ liệu bắt buộc
  Given Warehouse Staff nhập kho cho một SKU
  When nhân viên cung cấp mã lô, NSX, HSD, số lượng nhập và nguồn nếu áp dụng
  Then hệ thống tạo Batch gắn với đúng SKU
  And Batch có số dư ban đầu khớp nghiệp vụ nhập đã hoàn tất

Scenario: Tra cứu lịch sử một Batch
  Given Batch đã có các nghiệp vụ nhập, giữ, xuất hoặc điều chỉnh
  When Warehouse Staff mở chi tiết Batch
  Then hệ thống hiển thị dữ liệu lô và toàn bộ biến động được phép xem
  And mỗi biến động có loại, số lượng, thời điểm và tham chiếu liên quan

Scenario: Ngăn Batch gây nhầm lẫn
  Given đã tồn tại Batch với định danh trong phạm vi duy nhất được quy định
  When Warehouse Staff cố tạo Batch trùng hoặc liên kết sai SKU
  Then hệ thống từ chối dữ liệu gây nhầm lẫn
  And không gộp số lượng của hai lô không cùng danh tính

Scenario: Bảo toàn Batch đã phát sinh giao dịch
  Given Batch đã có biến động tồn
  When người dùng cố xóa hoặc sửa thông tin định danh quan trọng
  Then hệ thống không cho làm mất khả năng truy xuất lịch sử
  And yêu cầu dùng quy trình hiệu chỉnh có Audit nếu dữ liệu sai
```

### US-INV-05 — Quản lý ngày sản xuất và hạn sử dụng

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Warehouse Staff, tôi muốn quản lý ngày sản xuất và hạn sử dụng của từng Batch.

**Giá trị nghiệp vụ:** Doanh nghiệp ngăn bán thực phẩm hết hạn và có dữ liệu đáng tin cậy để ưu tiên xuất hoặc xử lý lô rủi ro.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Lưu NSX và HSD hợp lệ
  Given Warehouse Staff đang tạo hoặc hiệu chỉnh Batch theo quyền
  When ngày sản xuất và hạn sử dụng đáp ứng quy tắc thời gian đã phê duyệt
  Then hệ thống lưu hai ngày cho đúng Batch
  And dùng HSD để xác định trạng thái đủ điều kiện bán

Scenario: Từ chối khoảng ngày không hợp lệ
  Given Warehouse Staff nhập ngày cho Batch
  When HSD không sau NSX hoặc dữ liệu ngày không đáp ứng chính sách sản phẩm
  Then hệ thống từ chối lưu dữ liệu không hợp lệ
  And không đưa Batch vào tồn khả dụng để bán

Scenario: Batch chuyển sang hết hạn
  Given Batch còn số lượng và HSD đã qua theo mốc thời gian nghiệp vụ
  When hệ thống đánh giá điều kiện bán
  Then Batch bị loại khỏi tồn khả dụng
  And không thể reserve hoặc xuất cho Order bán hàng
  And số lượng vật lý vẫn được giữ để xử lý theo quy trình kho

Scenario: Hiệu chỉnh ngày của Batch đã có giao dịch
  Given Batch đã phát sinh reservation hoặc xuất kho
  When người có quyền yêu cầu hiệu chỉnh NSX/HSD
  Then hệ thống yêu cầu lý do nghiệp vụ
  And lưu giá trị trước/sau để Audit
  And đánh giá lại ảnh hưởng đến tồn khả dụng và cảnh báo
```

### US-INV-06 — Cảnh báo Batch/Lot sắp hết hạn

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Warehouse Staff, tôi muốn nhận cảnh báo khi Batch/Lot sắp hết hạn để chủ động xử lý.

**Giá trị nghiệp vụ:** Kho có đủ thời gian kiểm tra, ưu tiên xuất, cách ly hoặc đề xuất phương án xử lý trước khi hàng hết hạn.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Batch đạt ngưỡng cảnh báo HSD
  Given Batch còn số lượng và đã đi vào khoảng cảnh báo theo chính sách
  When hệ thống đánh giá HSD định kỳ hoặc sau thay đổi liên quan
  Then Batch xuất hiện trong danh sách cảnh báo của Warehouse Staff
  And cảnh báo hiển thị SKU, mã Batch, HSD, số lượng còn lại và mức độ rủi ro

Scenario: Không tạo cảnh báo trùng không cần thiết
  Given Batch đã có cảnh báo đang hoạt động cho cùng ngưỡng HSD
  When hệ thống đánh giá lại mà mức cảnh báo không thay đổi
  Then hệ thống không tạo thêm bản cảnh báo nghiệp vụ trùng lặp

Scenario: Cảnh báo được cập nhật khi số lượng hết
  Given Batch đang có cảnh báo sắp hết hạn
  When số lượng còn lại về không hoặc Batch đã được xử lý hoàn toàn
  Then cảnh báo không còn được trình bày như rủi ro tồn đang hoạt động
  And lịch sử cảnh báo vẫn được bảo toàn

Scenario: Chuyển yêu cầu thông báo
  Given một cảnh báo mới hoặc tăng mức độ đã được xác định
  When actor được cấu hình cần nhận thông báo
  Then Inventory chuyển dữ liệu cảnh báo sang EPIC 28
  And việc gửi tin thất bại không làm mất trạng thái cảnh báo trong Inventory
```

### US-INV-07 — Xem tốc độ tiêu thụ và tồn kho

**Actor:** Inventory/Supply Manager (`ACT-15`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Supply Manager, tôi muốn xem tốc độ tiêu thụ và tồn kho để lập kế hoạch sản xuất.

**Giá trị nghiệp vụ:** Người quản lý có dữ liệu cơ sở để nhận biết SKU tiêu thụ nhanh hoặc chậm trước khi lập kế hoạch cung ứng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem tiêu thụ của SKU theo kỳ
  Given có dữ liệu xuất kho hợp lệ trong khoảng thời gian được chọn
  When Supply Manager xem tốc độ tiêu thụ của một SKU
  Then hệ thống hiển thị số lượng xuất/tiêu thụ và tồn hiện tại theo kỳ
  And nêu rõ khoảng thời gian cùng nguồn dữ liệu được sử dụng

Scenario: Phân biệt xuất bán và biến động khác
  Given SKU có xuất cho Order, hàng hỏng và điều chỉnh kiểm kê
  When hệ thống tính chỉ số tiêu thụ phục vụ kế hoạch
  Then các loại biến động được phân biệt theo quy tắc chỉ số
  And không mặc định coi mọi lần giảm tồn là bán hàng

Scenario: Không đủ dữ liệu
  Given SKU chưa có đủ dữ liệu trong kỳ được chọn
  When Supply Manager xem tốc độ tiêu thụ
  Then hệ thống hiển thị tình trạng dữ liệu chưa đủ
  And không trình bày một dự báo không có căn cứ như kết quả chắc chắn

Scenario: Chuyển sang phân tích nâng cao
  Given Supply Manager cần dự báo nhu cầu hoặc số ngày tồn kho tương lai
  When mở phân tích nâng cao
  Then hệ thống sử dụng chức năng của EPIC 25
  And Inventory vẫn là nguồn dữ liệu số dư và biến động đầu vào
```

### US-INV-08 — Ưu tiên xuất Batch gần hết hạn

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Warehouse Staff, tôi muốn hệ thống ưu tiên xuất các Batch gần hết hạn phù hợp với chính sách để giảm lãng phí.

**Giá trị nghiệp vụ:** Hàng đủ điều kiện bán có HSD gần hơn được sử dụng trước khi phù hợp, giảm tồn quá hạn mà không vi phạm yêu cầu chất lượng giao hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đề xuất Batch theo FEFO
  Given nhiều Batch của cùng SKU còn đủ điều kiện cho Order
  When hệ thống phân bổ số lượng xuất
  Then Batch có HSD gần hơn được đề xuất trước
  And vẫn đáp ứng ngưỡng HSD còn lại và điều kiện giao hàng theo chính sách

Scenario: Bỏ qua Batch không đủ điều kiện
  Given Batch gần HSD nhất đã hết hạn, bị cách ly hoặc không đủ thời gian sử dụng tối thiểu
  When hệ thống chọn Batch để xuất bán
  Then Batch đó không được phân bổ
  And hệ thống xét Batch đủ điều kiện tiếp theo

Scenario: Một Order cần nhiều Batch
  Given không Batch đơn lẻ nào đủ số lượng nhưng nhiều Batch hợp lệ có thể đáp ứng
  When chính sách cho phép tách số lượng qua nhiều Batch
  Then hệ thống phân bổ tổng số lượng theo thứ tự ưu tiên
  And giữ được liên kết từng lượng xuất với từng Batch

Scenario: Warehouse Staff chọn khác đề xuất
  Given chính sách cho phép ngoại lệ phân bổ Batch
  When Warehouse Staff có quyền chọn Batch khác và cung cấp lý do
  Then hệ thống kiểm tra Batch thay thế vẫn đủ điều kiện
  And lưu đề xuất ban đầu, lựa chọn thực tế và lý do để Audit
```

### US-INV-09 — Ghi nhận hỏng, thất thoát hoặc điều chỉnh tồn

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Warehouse Staff, tôi muốn ghi nhận hàng hỏng, thất thoát hoặc điều chỉnh tồn kho để số liệu phản ánh thực tế.

**Giá trị nghiệp vụ:** Số dư hệ thống khớp hàng thực tế mà vẫn bảo toàn nguyên nhân và trách nhiệm của mọi chênh lệch.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Ghi nhận hàng hỏng hoặc thất thoát
  Given Warehouse Staff có quyền và Batch còn đủ số lượng
  When nhân viên ghi nhận loại biến động, số lượng và lý do bắt buộc
  Then hệ thống giảm đúng số dư của SKU/Batch
  And lượng đó không còn được tính vào tồn khả dụng
  And lưu thông tin để truy vết

Scenario: Điều chỉnh sau kiểm kê
  Given số lượng kiểm đếm thực tế khác số dư hệ thống
  When người có quyền ghi nhận kết quả kiểm kê và lý do chênh lệch
  Then hệ thống tạo biến động điều chỉnh thay vì ghi đè lịch sử
  And số dư mới phản ánh kết quả đã được phê duyệt

Scenario: Từ chối điều chỉnh làm tồn âm
  Given số lượng điều chỉnh giảm lớn hơn số dư có thể điều chỉnh của Batch
  When Warehouse Staff gửi yêu cầu
  Then hệ thống từ chối thao tác
  And không làm số dư trở thành âm

Scenario: Điều chỉnh cần phê duyệt
  Given loại hoặc giá trị điều chỉnh vượt thẩm quyền của Warehouse Staff
  When nhân viên gửi yêu cầu điều chỉnh
  Then hệ thống không áp dụng vào số dư trước khi được phê duyệt
  And lưu trạng thái yêu cầu cùng người tạo để theo dõi

Scenario: Ngăn điều chỉnh lặp
  Given một chứng từ điều chỉnh đã được áp dụng
  When cùng tham chiếu được gửi lại
  Then hệ thống không thay đổi số dư lần thứ hai
```

### US-INV-10 — Sales nhận cảnh báo sản phẩm sắp hết hạn

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Sales Manager, tôi muốn nhận cảnh báo sản phẩm sắp hết hạn để phối hợp với Sales/CSKH triển khai chương trình bán phù hợp.

**Giá trị nghiệp vụ:** Bộ phận bán hàng biết sớm lượng hàng có rủi ro để đề xuất hành động phù hợp mà không bán hàng hết hạn.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Sales Manager xem danh sách rủi ro HSD
  Given có Batch còn số lượng và đạt ngưỡng cảnh báo dành cho Sales
  When Sales Manager mở danh sách cảnh báo
  Then hệ thống hiển thị SKU, Batch, HSD, số lượng còn lại và mức độ rủi ro được phép xem
  And cho phép lọc hoặc sắp xếp để ưu tiên xử lý

Scenario: Cảnh báo không cho phép bán hàng hết hạn
  Given một Batch đã chuyển từ sắp hết hạn sang hết hạn
  When Sales Manager xem cảnh báo hoặc lập chương trình bán
  Then hệ thống thể hiện Batch không còn đủ điều kiện bán
  And không đưa số lượng đó vào lượng có thể phân bổ cho bán hàng

Scenario: Chuyển yêu cầu thông báo cho Sales
  Given cảnh báo mới đáp ứng điều kiện gửi cho Sales Manager
  When Inventory tạo dữ liệu cảnh báo
  Then EPIC 28 nhận yêu cầu thông báo đúng actor
  And Inventory vẫn là nguồn trạng thái rủi ro HSD

Scenario: Dùng cảnh báo để phối hợp chương trình bán
  Given Sales Manager đã xem một cảnh báo sắp hết hạn
  When cần tạo khuyến mãi hoặc liên hệ khách hàng
  Then hệ thống điều hướng sang nghiệp vụ phù hợp của EPIC 17 hoặc EPIC 16
  And Inventory không tự tạo chương trình bán hoặc tự gửi nội dung CSKH
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-INV-01` | Duy trì và hiển thị tồn theo SKU, phân biệt tối thiểu số lượng hiện có, đang giữ, không đủ điều kiện bán và khả dụng. | `US-INV-01`, `BR-PROD-02` |
| `FR-INV-02` | Phân rã số dư SKU theo Batch/Lot và bảo đảm tổng số dư nhất quán với các Batch liên quan. | `US-INV-01`, `US-INV-04`, `BR-BATCH-01` |
| `FR-INV-03` | Giữ tồn nguyên tử cho yêu cầu hợp lệ và không cho tổng reservation vượt số lượng đủ điều kiện bán. | `US-INV-01`, `BR-ORDER-02`, `EPIC 06` |
| `FR-INV-04` | Chuyển, giải phóng hoặc hết hạn reservation theo kết quả Order mà không xử lý trùng. | `US-INV-01`, `EPIC 08` |
| `FR-INV-05` | Ghi nhận nhập kho bằng biến động gắn với đúng SKU, Batch, số lượng, actor và nguồn tham chiếu. | `US-INV-02` |
| `FR-INV-06` | Đối chiếu số lượng thực nhận với PO khi có và bảo toàn chênh lệch để xử lý. | `US-INV-02`, `EPIC 27` |
| `FR-INV-07` | Ghi nhận xuất kho cho Order hoặc lý do được phê duyệt, gắn với đúng SKU/Batch và không làm số dư âm. | `US-INV-03` |
| `FR-INV-08` | Chống cộng/trừ tồn nhiều lần khi cùng chứng từ hoặc yêu cầu nghiệp vụ được gửi lặp. | `US-INV-02`, `US-INV-03`, `US-INV-09` |
| `FR-INV-09` | Quản lý mã Batch, NSX, HSD, số lượng nhập/còn lại và nguồn sản xuất/nhà cung cấp nếu có. | `US-INV-04`, `US-INV-05`, `BR-BATCH-02` |
| `FR-INV-10` | Bảo toàn lịch sử Batch và ngăn xóa/sửa trực tiếp dữ liệu định danh đã có giao dịch; hiệu chỉnh phải có Audit. | `US-INV-04`, `BR-AUDIT-01` |
| `FR-INV-11` | Kiểm tra quan hệ NSX/HSD và loại Batch hết hạn hoặc không đủ điều kiện khỏi tồn khả dụng. | `US-INV-05`, `BR-PROD-03`, `BR-BATCH-05` |
| `FR-INV-12` | Xác định, duy trì và hiển thị cảnh báo Batch sắp hết hạn theo ngưỡng chính sách. | `US-INV-06`, `BR-BATCH-03` |
| `FR-INV-13` | Chuyển dữ liệu cảnh báo mới hoặc tăng mức độ sang EPIC 28 mà không tạo thông báo nghiệp vụ trùng. | `US-INV-06`, `US-INV-10`, `EPIC 28` |
| `FR-INV-14` | Tổng hợp tồn và lượng tiêu thụ theo SKU/kỳ, phân biệt xuất bán với hỏng, thất thoát và điều chỉnh. | `US-INV-07` |
| `FR-INV-15` | Cung cấp dữ liệu số dư/biến động cho phân tích và dự báo nhưng không tự thay thế nghiệp vụ DSS của EPIC 25. | `US-INV-07`, `EPIC 25` |
| `FR-INV-16` | Đề xuất phân bổ Batch theo FEFO có điều kiện, bỏ qua Batch hết hạn, cách ly hoặc không đáp ứng ngưỡng HSD còn lại. | `US-INV-08`, `BR-BATCH-04~06` |
| `FR-INV-17` | Cho phép ngoại lệ phân bổ Batch chỉ khi actor có quyền, Batch thay thế hợp lệ và lý do được ghi Audit. | `US-INV-08`, `BR-AUDIT-01` |
| `FR-INV-18` | Ghi nhận hỏng, thất thoát và chênh lệch kiểm kê bằng biến động có loại, số lượng, Batch, lý do và actor. | `US-INV-09` |
| `FR-INV-19` | Áp dụng ngưỡng thẩm quyền/phê duyệt cho điều chỉnh tồn và không cập nhật số dư trước khi yêu cầu bắt buộc được duyệt. | `US-INV-09`, `EPIC 22`, `EPIC 23` |
| `FR-INV-20` | Cung cấp danh sách rủi ro HSD cho Sales Manager nhưng không tự tạo Promotion hoặc quy trình CSKH. | `US-INV-10`, `EPIC 16`, `EPIC 17` |
| `FR-INV-21` | Tiếp nhận biến động từ Marketplace/Offline qua cùng cơ chế tồn trung tâm, giữ nguồn tham chiếu và chống xử lý lặp. | `BR-MKTPLACE-05~06`, `BR-OFFLINE-02~04`, `EPIC 12`, `EPIC 13` |
| `FR-INV-22` | Ghi Audit cho điều chỉnh, thất thoát, hiệu chỉnh Batch/HSD và ngoại lệ phân bổ có ảnh hưởng quan trọng. | `US-INV-04`, `US-INV-05`, `US-INV-08`, `US-INV-09`, `BR-AUDIT-01` |

## 6. Mô hình số dư và quy tắc dữ liệu

### 6.1. Các thành phần số dư nghiệp vụ

```text
Tồn đang được quản lý
   ├── Tồn vật lý đủ điều kiện bán
   │      ├── Đang giữ cho Order
   │      └── Khả dụng cho yêu cầu mới
   └── Tồn vật lý không đủ điều kiện bán
          ├── Hết hạn hoặc hỏng
          └── Cách ly hoặc chờ xử lý

Thất thoát đã xác nhận
   └── Ghi nhận bằng biến động giảm, không còn thuộc tồn vật lý
```

Công thức chính xác còn phụ thuộc chính sách vị trí kho, cách ly và hàng đang xử lý, nhưng số lượng khả dụng không được bao gồm hàng đã giữ hoặc không đủ điều kiện bán; lượng thất thoát đã xác nhận phải được loại khỏi tồn vật lý bằng biến động có truy vết.

### 6.2. Quy tắc dữ liệu và an toàn nghiệp vụ

- Mọi số lượng phải gắn với SKU; đối với thực phẩm, số lượng vật lý phải truy xuất được đến Batch/Lot.
- Số dư là kết quả của các biến động đã ghi nhận, không phải giá trị được sửa trực tiếp mà không có chứng từ hoặc lịch sử.
- Mỗi biến động có loại nghiệp vụ, số lượng có dấu/chiều rõ ràng, SKU, Batch khi bắt buộc, actor/nguồn, thời điểm và tham chiếu duy nhất khi có.
- Không cho tồn vật lý, số lượng Batch hoặc số lượng khả dụng trở thành âm do nhập, xuất, reserve, release hay điều chỉnh.
- Reservation, chuyển reservation thành xuất và release phải liên kết với đúng Order; xử lý lặp không được làm thay đổi số dư lần thứ hai.
- Batch hết hạn vẫn có thể còn tồn vật lý nhưng không thuộc tồn khả dụng và không được xuất cho bán hàng.
- FEFO là ưu tiên có điều kiện, không phải mệnh lệnh bỏ qua chất lượng giao hàng, ngưỡng HSD còn lại, cách ly hoặc yêu cầu truy xuất.
- Hàng trả chỉ được nhập lại tồn bán được sau khi có kết quả kiểm tra chất lượng hợp lệ từ EPIC 14; nếu không phải cách ly, ghi hỏng hoặc xử lý theo chính sách.
- Actor chỉ được xem hoặc thay đổi kho trong phạm vi quyền. Điều chỉnh nhạy cảm phải tuân theo ngưỡng phê duyệt và Audit.
- Chi tiết schema, endpoint, transaction/locking, message broker, scheduler, thuật toán dự báo và công nghệ thông báo thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-INV-01` | `BR-PROD-02`, `BR-BATCH-05`, `BR-ORDER-02` | EPIC 04, 06, 08 | Số dư SKU/Batch nhất quán; reservation làm giảm khả dụng; cạnh tranh đơn vị cuối không oversell. |
| `US-INV-02` | `BR-BATCH-01`, `BR-BATCH-02` | EPIC 04, 27 | Nhập đúng SKU/Batch và thực nhận; dữ liệu sai hoặc chứng từ lặp không cộng tồn. |
| `US-INV-03` | `BR-BATCH-01` | EPIC 08, 10 | Xuất đúng SKU/Batch/Order, không âm và yêu cầu lặp không trừ hai lần. |
| `US-INV-04` | `BR-BATCH-01`, `BR-BATCH-02` | EPIC 19, 23 | Batch đủ trường, tra được lịch sử và không bị xóa/sửa làm mất truy xuất. |
| `US-INV-05` | `BR-PROD-03`, `BR-BATCH-02`, `BR-BATCH-05` | EPIC 04, 23 | NSX/HSD hợp lệ; Batch hết hạn bị loại khỏi khả dụng; hiệu chỉnh có Audit. |
| `US-INV-06` | `BR-BATCH-03` | EPIC 28 | Cảnh báo đúng Batch/ngưỡng, không trùng và đóng khi rủi ro tồn không còn. |
| `US-INV-07` | `BR-PROD-02` | EPIC 25, 27 | Tiêu thụ theo kỳ phân biệt đúng loại biến động và không biến dữ liệu thiếu thành dự báo chắc chắn. |
| `US-INV-08` | `BR-BATCH-04`, `BR-BATCH-05`, `BR-BATCH-06` | EPIC 10, 23 | FEFO chỉ chọn Batch đủ điều kiện; ngoại lệ có quyền và Audit. |
| `US-INV-09` | `BR-BATCH-01`, `BR-AUDIT-01` | EPIC 22, 23 | Hỏng/thất thoát/kiểm kê tạo biến động đúng, không âm, không lặp và tuân thủ phê duyệt. |
| `US-INV-10` | `BR-BATCH-03~05` | EPIC 16, 17, 28 | Sales thấy đúng rủi ro HSD nhưng không thể dùng hàng hết hạn hoặc tự tạo chương trình ngoài Epic phụ trách. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Có quản lý nhiều kho, khu vực, kệ/bin và chuyển kho nội bộ trong MVP hay chỉ một kho logic.
- Công thức chính thức của `on_hand`, `reserved`, `allocated`, `quarantined`, `damaged` và `available`; thời điểm chuyển giữa các loại số dư.
- Vòng đời reservation, thời hạn, quan hệ với Payment/COD và cách xử lý Order hết hạn hoặc thanh toán đến trễ.
- Thời điểm trừ tồn vật lý: khi phân bổ, lấy hàng, đóng gói, bàn giao vận chuyển hay mốc khác.
- Phạm vi duy nhất của mã Batch/Lot: toàn hệ thống, theo SKU hay theo nhà cung cấp/nguồn sản xuất.
- Quy tắc nhập một Batch qua nhiều lần nhận và xử lý trường hợp cùng mã lô nhưng khác NSX/HSD hoặc nguồn.
- Ngưỡng HSD tối thiểu khi nhập kho và ngưỡng HSD còn lại khi phân bổ cho từng kênh/vùng giao.
- Các mức cảnh báo sắp hết hạn, lịch đánh giá, actor nhận cảnh báo và điều kiện đóng/mở lại cảnh báo.
- FEFO có bắt buộc ở MVP hay Giai đoạn 2; trường hợp cho phép ngoại lệ và cấp phê duyệt tương ứng.
- Ngưỡng số lượng/giá trị điều chỉnh cần phê duyệt, actor phê duyệt và bằng chứng bắt buộc cho hỏng/thất thoát.
- Quy trình kiểm kê toàn phần/chu kỳ, đóng băng giao dịch trong lúc đếm và xử lý chênh lệch đồng thời.
- Cách xử lý hàng khách trả: nhập lại bán được, cách ly, hỏng hoặc trả nhà cung cấp.
- Định nghĩa tốc độ tiêu thụ: dựa trên Order, xuất kho hay giao thành công; kỳ tính và cách xử lý trả hàng.
- Cảnh báo tồn tối thiểu từng xuất hiện trong tài liệu legacy nhưng chưa có User Story trong Product Backlog; cần bổ sung backlog nếu muốn đưa vào phạm vi chính thức.
- Chính sách lưu lịch sử biến động và khả năng sửa sai bằng bút toán đảo thay vì xóa/sửa chứng từ cũ.

## 9. UI/UX Reference

- Màn hình tồn theo SKU hiển thị rõ tổng vật lý, đang giữ, không đủ điều kiện và khả dụng; cho phép drill-down theo Batch.
- Chi tiết Batch hiển thị mã lô, SKU, NSX, HSD, nguồn, số dư, trạng thái HSD và lịch sử biến động.
- Biểu mẫu nhập/xuất/điều chỉnh yêu cầu chọn đúng SKU/Batch, loại nghiệp vụ, số lượng, tham chiếu và lý do khi bắt buộc.
- Cảnh báo HSD dùng mức độ trực quan nhưng luôn hiển thị ngày và số lượng cụ thể; hàng hết hạn phải khác rõ với hàng chỉ sắp hết hạn.
- Gợi ý FEFO hiển thị thứ tự Batch và lý do; khi chọn ngoại lệ phải cảnh báo, kiểm tra quyền và thu thập lý do.
- Hàng đợi phê duyệt điều chỉnh cho biết số dư trước, thay đổi đề nghị, số dư dự kiến, actor và bằng chứng liên quan.
- Chỉ tiêu hiệu năng, endpoint, cơ chế khóa đồng thời, scheduler và công nghệ gửi cảnh báo được đặc tả trong NFR hoặc Technical Design.

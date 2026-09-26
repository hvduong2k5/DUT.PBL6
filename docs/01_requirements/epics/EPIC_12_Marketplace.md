# EPIC 12 — Marketplace

## 1. Mục tiêu Epic

Epic này tích hợp các Marketplace như Shopee/TikTok Shop vào quy trình bán hàng nội bộ. Marketplace Operator có thể tiếp nhận và xử lý Order từ sàn, theo dõi lỗi đồng bộ, đối soát doanh thu/phí; Manager có thể đánh giá hiệu quả từng kênh. Hệ thống phải giữ định danh nguồn, chống tạo Order trùng và cập nhật tồn khả dụng kịp thời để giảm oversell đa kênh.

Marketplace Integration quản lý kết nối kênh, tham chiếu ngoài, ánh xạ dữ liệu, trạng thái đồng bộ và dữ liệu đối soát. Order, Inventory, Product, Packing, Shipping và Finance vẫn là các miền nghiệp vụ riêng; EPIC 12 điều phối việc trao đổi dữ liệu với sàn nhưng không thay thế các nguồn sự thật nội bộ đó.

Epic không trực tiếp quản lý catalog gốc, tự sửa số dư tồn, thực hiện Packing/Shipping, xác nhận Payment nội bộ hay lập báo cáo tài chính tổng hợp. Các trách nhiệm này lần lượt thuộc EPIC 04, EPIC 09, EPIC 10, EPIC 11, EPIC 07 và EPIC 24.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 2** | `US-MKT-01` đến `US-MKT-06` | Đơn Marketplace được tiếp nhận không trùng, đưa vào fulfillment, theo dõi đồng bộ, đối soát và tổng hợp doanh thu theo từng sàn. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Marketplace Customer (`ACT-04`) | Mua và tương tác với Order trên Marketplace; dữ liệu chỉ được đồng bộ trong phạm vi nền tảng và chính sách cho phép. |
| Marketplace Operator (`ACT-09`) | Theo dõi Order, lỗi đồng bộ, đưa Order vào fulfillment và thực hiện đối soát kênh. |
| Sales Manager (`ACT-12`) | Xem doanh thu/hiệu quả theo Marketplace và giám sát ngoại lệ theo quyền. |
| Accountant / Finance Staff (`ACT-19`) | Sử dụng dữ liệu settlement/đối soát cho nghiệp vụ tài chính theo quyền. |
| Marketplace Platform (`EXT-04`) | Cung cấp Product/Order/Customer/Settlement và nhận cập nhật tồn hoặc trạng thái được hỗ trợ. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-04` | Nhân viên chỉ được thực hiện thao tác thuộc quyền được cấp. | Xem dữ liệu kênh, thử lại đồng bộ và xử lý sai lệch phải kiểm tra quyền. |
| `BR-PROD-02` | Giá, quy cách và tồn được quản lý theo SKU. | Listing/line item ngoài phải ánh xạ được về đúng SKU nội bộ trước fulfillment. |
| `BR-BATCH-05` | Hàng hết hạn không được phép tiếp tục bán. | Tồn đẩy ra sàn không được bao gồm lượng không đủ điều kiện bán. |
| `BR-ORDER-01` | Order phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Trạng thái ngoài phải được ánh xạ có kiểm soát sang Order nội bộ. |
| `BR-ORDER-02` | Không xác nhận Order vượt tồn khả dụng. | Khi nhận Order mới, Inventory phải giữ/trừ khả dụng trước khi đưa vào fulfillment. |
| `BR-ORDER-05` | Thay đổi trạng thái quan trọng phải được ghi nhận. | Đồng bộ trạng thái phải lưu nguồn, trạng thái gốc, trước/sau và thời điểm. |
| `BR-MKTPLACE-01` | Order Marketplace phải xác định nguồn phát sinh. | Mọi Order phải lưu platform/shop và kênh nguồn. |
| `BR-MKTPLACE-02` | Order Marketplace phải có mã tham chiếu nền tảng gốc. | Không tạo Order nội bộ hợp lệ nếu thiếu định danh gốc bắt buộc. |
| `BR-MKTPLACE-03` | Order Marketplace phải được đối soát với Order nội bộ. | Dữ liệu Order, doanh thu và phí phải liên kết để phát hiện sai lệch. |
| `BR-MKTPLACE-04` | Không tạo trùng Order khi đồng bộ lại cùng Order Marketplace. | Khóa nghiệp vụ theo nguồn/shop/mã gốc phải chống trùng cả khi xử lý đồng thời. |
| `BR-MKTPLACE-05` | Order Marketplace mới phải trừ/tạm giữ tồn nội bộ ngay. | Order chỉ sẵn sàng fulfillment khi reservation hợp lệ; tác động lặp không được giữ tồn hai lần. |
| `BR-MKTPLACE-06` | Order bị hủy từ Marketplace phải cập nhật Hủy và hoàn trả tồn. | Hủy phải áp dụng đúng Order, nhả đúng reservation và chống xử lý lặp. |
| `BR-MKTPLACE-07` | Giá trị Order nội bộ tuân theo giá thanh toán thực tế trên sàn, gồm trợ giá và phí. | Snapshot phải phân tách các thành phần tiền để đối soát chính xác. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Thử lại, ánh xạ thủ công và xử lý sai lệch phải có khả năng truy vết. |

### 3.3. Phụ thuộc giữa Epic và hệ thống ngoài

- `EPIC 04 — Product, Variant & SKU`: là nguồn Product/SKU nội bộ, giá theo kênh và ánh xạ listing.
- `EPIC 08 — Order Management`: tạo/lưu Order nội bộ, snapshot và state machine; EPIC 12 cung cấp nguồn/tham chiếu ngoài.
- `EPIC 09 — Inventory & Batch`: quản lý tồn khả dụng, reservation/release và dữ liệu tồn được phép đồng bộ ra sàn.
- `EPIC 10 — Packing & Fulfillment`: tiếp nhận Order Marketplace đã đủ điều kiện cùng yêu cầu đóng gói theo kênh.
- `EPIC 11 — Shipping & Delivery`: quản lý Shipment; trách nhiệm tạo vận đơn có thể thuộc sàn hoặc nội bộ tùy mô hình fulfillment.
- `EPIC 22 — User / Role / Permission Administration`: kiểm soát quyền vận hành từng shop/kênh.
- `EPIC 23 — Audit & Security`: lưu dấu vết thao tác và xử lý ngoại lệ nhạy cảm.
- `EPIC 24 — Finance & Business Analytics`: dùng dữ liệu đã đối soát để lập báo cáo tài chính đa kênh.
- `EXT-04 — Marketplace Platform`: nguồn Order, trạng thái, settlement và dữ liệu nền tảng; nhận Product/tồn/trạng thái khi tích hợp hỗ trợ.

## 4. User Stories chi tiết

### US-MKT-01 — Xem Order phát sinh trên Marketplace

**Actor:** Marketplace Operator (`ACT-09`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Marketplace Operator, tôi muốn xem các đơn hàng phát sinh trên Shopee/TikTok để xử lý.

**Giá trị nghiệp vụ:** Operator có một danh sách thống nhất để nhận biết Order mới, nguồn sàn và tình trạng xử lý nội bộ.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem danh sách Order Marketplace
  Given hệ thống đã nhận các Order từ những shop Marketplace được kết nối
  When Marketplace Operator mở danh sách
  Then hệ thống hiển thị mã nội bộ, platform/shop, mã Order gốc, thời điểm đặt, tổng tiền và trạng thái đồng bộ
  And phân biệt được Order mới, đang xử lý, lỗi và đã liên kết nội bộ

Scenario: Lọc và tìm Order theo nguồn
  Given danh sách có Order từ nhiều platform/shop
  When Operator lọc theo sàn, shop, trạng thái, thời gian hoặc mã được hỗ trợ
  Then hệ thống trả về đúng Order trong phạm vi quyền của Operator
  And không thay đổi dữ liệu chỉ vì thao tác xem/lọc

Scenario: Hiển thị trạng thái nguồn và nội bộ
  Given một Order có trạng thái Marketplace khác cách biểu diễn trạng thái nội bộ
  When Operator xem chi tiết
  Then hệ thống hiển thị trạng thái gốc, trạng thái ánh xạ và trạng thái Order nội bộ
  And không che giấu lỗi hoặc trạng thái chưa ánh xạ

Scenario: Operator không có quyền với shop
  Given Operator không được cấp quyền truy cập một shop/kênh
  When Operator cố xem Order của shop đó
  Then hệ thống từ chối truy cập
  And không tiết lộ dữ liệu khách hàng hoặc Order
```

### US-MKT-02 — Đưa Order Marketplace vào fulfillment nội bộ

**Actor:** Marketplace Operator (`ACT-09`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Marketplace Operator, tôi muốn đưa đơn hàng từ Marketplace vào quy trình xử lý nội bộ để nhân viên kho và đóng gói thực hiện.

**Giá trị nghiệp vụ:** Order từ sàn sử dụng chung Inventory, Packing và Shipping mà vẫn giữ đầy đủ nguồn cùng điều kiện riêng của kênh.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tiếp nhận Order Marketplace hợp lệ
  Given platform/shop và mã Order gốc xác định được duy nhất
  And mọi line item ánh xạ được về SKU nội bộ
  And dữ liệu người nhận trong phạm vi được phép cùng giá trị Order đều hợp lệ
  When hệ thống tiếp nhận Order
  Then EPIC 08 tạo/liên kết một Order nội bộ với nguồn và snapshot Marketplace
  And EPIC 09 giữ đúng tồn khả dụng cho các SKU
  And Order được đưa vào fulfillment chỉ sau khi các điều kiện bắt buộc thành công

Scenario: Listing chưa ánh xạ SKU
  Given Order Marketplace có line item chưa ánh xạ được về SKU nội bộ
  When hệ thống tiếp nhận Order
  Then Order ngoài được giữ để Operator xử lý
  And không đưa line item không xác định vào Packing
  And không tự đoán SKU hoặc trừ tồn sai

Scenario: Không đủ tồn nội bộ
  Given Order Marketplace hợp lệ nhưng một SKU không còn đủ tồn khả dụng
  When hệ thống yêu cầu reservation
  Then Order không được trình bày như đã sẵn sàng fulfillment
  And Operator thấy ngoại lệ cần xử lý theo chính sách của sàn
  And không giữ tồn một phần như toàn bộ Order đã thành công nếu chính sách không cho phép

Scenario: Marketplace hủy Order
  Given Order đã được liên kết nội bộ và còn reservation
  When hệ thống nhận hủy hợp lệ từ Marketplace
  Then EPIC 08 cập nhật Order theo chuyển trạng thái hủy được phép
  And EPIC 09 giải phóng đúng reservation liên quan
  And xử lý lặp không nhả tồn nhiều lần

Scenario: Giá trị Order theo dữ liệu thanh toán trên sàn
  Given Order có giá niêm yết, giảm giá người bán, trợ giá sàn, phí và số khách thực trả
  When tạo snapshot nội bộ
  Then hệ thống lưu riêng các thành phần cần thiết
  And tổng dùng cho đối soát phản ánh giao dịch thực tế trên Marketplace
```

### US-MKT-03 — Theo dõi trạng thái đồng bộ

**Actor:** Marketplace Operator (`ACT-09`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Marketplace Operator, tôi muốn theo dõi trạng thái đồng bộ giữa Marketplace và hệ thống nội bộ để phát hiện lỗi.

**Giá trị nghiệp vụ:** Operator biết dữ liệu nào đã đồng bộ, đang chờ hay lỗi và có thể xử lý mà không tạo tác động trùng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem trạng thái đồng bộ của Order
  Given một Order có các lần đồng bộ từ/đến Marketplace
  When Operator mở chi tiết đồng bộ
  Then hệ thống hiển thị đối tượng, chiều đồng bộ, trạng thái, lần thành công gần nhất và lỗi gần nhất
  And liên kết được với platform/shop cùng mã tham chiếu ngoài

Scenario: Phát hiện đồng bộ thất bại
  Given một lần đồng bộ không hoàn tất
  When hệ thống ghi nhận lỗi
  Then đối tượng được đánh dấu cần xử lý với nhóm lỗi phù hợp
  And dữ liệu lỗi đủ để Operator quyết định thử lại, sửa ánh xạ hoặc chuyển xử lý khác

Scenario: Thử lại đồng bộ có kiểm soát
  Given lỗi thuộc loại cho phép thử lại và Operator có quyền
  When Operator yêu cầu thử lại
  Then hệ thống tạo lần xử lý mới liên kết lỗi trước
  And không nhân đôi Order, reservation hoặc thay đổi trạng thái đã áp dụng

Scenario: Cập nhật trạng thái đến sai thứ tự
  Given Order nội bộ đã áp dụng một trạng thái mới hơn
  When Marketplace gửi trạng thái cũ đến sau
  Then hệ thống giữ dữ liệu gốc để truy vết
  And không làm Order quay lùi trái phép

Scenario: Xung đột trạng thái giữa hai phía
  Given trạng thái Marketplace và Order nội bộ không thể tự ánh xạ an toàn
  When hệ thống phát hiện xung đột
  Then trường hợp được đưa vào hàng đợi ngoại lệ
  And không tự chọn một trạng thái có thể gây mất hàng hoặc sai tài chính
```

### US-MKT-04 — Đối soát Order, doanh thu và phí

**Actor:** Marketplace Operator (`ACT-09`), Accountant / Finance Staff (`ACT-19`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Marketplace Operator, tôi muốn đối soát đơn hàng, doanh thu và phí từ Marketplace.

**Giá trị nghiệp vụ:** Doanh nghiệp biết số phải thu, phí sàn, trợ giá và số thực nhận có khớp Order nội bộ hay không.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đối soát settlement khớp Order
  Given có dữ liệu settlement từ Marketplace và Order nội bộ tương ứng
  When người có quyền thực hiện đối soát
  Then hệ thống so sánh mã nguồn, giá trị hàng, giảm giá, trợ giá, phí, hoàn/điều chỉnh và số thực nhận
  And đánh dấu khớp khi mọi điều kiện bắt buộc phù hợp

Scenario: Phát hiện sai lệch
  Given settlement thiếu Order, trùng, sai số tiền, thiếu phí hoặc không khớp tham chiếu
  When hệ thống đối soát
  Then trường hợp được phân loại sai lệch với chênh lệch cụ thể
  And không tự sửa snapshot Order hoặc dữ liệu settlement gốc

Scenario: Xử lý sai lệch theo quyền
  Given một sai lệch đang mở
  And Operator/Accountant có quyền thực hiện hành động tương ứng
  When người dùng ghi nhận kết quả và lý do
  Then trạng thái đối soát được cập nhật
  And actor, thời điểm, trước/sau và lý do được lưu Audit
  And dữ liệu gốc vẫn được bảo toàn

Scenario: Settlement điều chỉnh ở kỳ sau
  Given Marketplace gửi khoản điều chỉnh liên quan Order đã đối soát kỳ trước
  When hệ thống tiếp nhận khoản điều chỉnh
  Then khoản mới được liên kết với Order và kỳ liên quan
  And không ghi đè làm mất kết quả đối soát trước

Scenario: Người chỉ có quyền xem
  Given người dùng được xem nhưng không được xử lý đối soát
  When người dùng cố đóng hoặc điều chỉnh sai lệch
  Then hệ thống từ chối thao tác
  And dữ liệu đối soát không bị thay đổi
```

### US-MKT-05 — Xem doanh thu theo Marketplace

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Manager, tôi muốn xem doanh thu theo từng Marketplace để đánh giá hiệu quả từng kênh.

**Giá trị nghiệp vụ:** Manager so sánh được giá trị bán và số thực nhận giữa các sàn trên cùng định nghĩa dữ liệu.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem doanh thu theo sàn và kỳ
  Given có Order/settlement Marketplace trong khoảng thời gian được chọn
  When Sales Manager mở báo cáo theo kênh
  Then hệ thống hiển thị doanh thu gộp, giảm giá/trợ giá, phí và doanh thu thuần theo định nghĩa được chọn
  And nêu rõ kỳ, nguồn dữ liệu và thời điểm cập nhật

Scenario: So sánh nhiều Marketplace
  Given có dữ liệu từ nhiều platform/shop
  When Manager chọn các kênh để so sánh
  Then hệ thống nhóm đúng dữ liệu theo platform/shop
  And tổng các nhóm nhất quán với tổng chung trong cùng phạm vi

Scenario: Phân biệt dữ liệu tạm tính và đã đối soát
  Given một phần Order chưa có settlement hoàn chỉnh
  When Manager xem báo cáo
  Then hệ thống phân biệt giá trị theo Order với giá trị đã đối soát/thực nhận
  And không trình bày dữ liệu chưa đối soát như số tài chính cuối cùng

Scenario: Không có dữ liệu
  Given kỳ được chọn không có giao dịch đủ điều kiện
  When Manager xem báo cáo
  Then hệ thống hiển thị trạng thái chưa có dữ liệu
  And không tạo tỷ lệ hoặc doanh thu gây hiểu nhầm

Scenario: Chuyển sang báo cáo tài chính chuyên sâu
  Given Manager cần báo cáo lợi nhuận, thuế hoặc tài chính đa kênh
  When mở phân tích chuyên sâu
  Then hệ thống sử dụng nghiệp vụ của EPIC 24
  And EPIC 12 vẫn cung cấp dữ liệu Marketplace đã đối soát làm đầu vào
```

### US-MKT-06 — Lưu mã Order gốc và chống tạo trùng

**Actor:** Hệ thống, Marketplace Platform (`EXT-04`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là hệ thống, tôi muốn lưu mã đơn hàng gốc của Marketplace để tránh tạo đơn trùng.

**Giá trị nghiệp vụ:** Mỗi Order ngoài chỉ tạo một Order nội bộ dù nền tảng gửi lại, hệ thống thử lại hoặc nhiều tiến trình nhận đồng thời.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Lưu định danh nguồn đầy đủ
  Given hệ thống nhận Order mới từ một shop Marketplace
  When Order được ghi nhận
  Then hệ thống lưu platform, shop/account và mã Order gốc
  And tổ hợp định danh đó liên kết duy nhất với một Order nội bộ

Scenario: Cùng Order được gửi lại
  Given Order nguồn đã liên kết với một Order nội bộ
  When Marketplace gửi lại cùng định danh
  Then hệ thống cập nhật/đối chiếu bản ghi hiện có theo quy tắc
  And không tạo Order hoặc reservation mới

Scenario: Hai yêu cầu đồng thời cho cùng Order nguồn
  Given hai tiến trình cùng nhận một Order Marketplace chưa có trong hệ thống
  When cả hai cố ghi nhận
  Then chỉ một Order nội bộ được tạo
  And cả hai nhận kết quả tham chiếu cùng Order đó

Scenario: Mã Order giống nhau ở hai nguồn khác nhau
  Given hai platform hoặc shop khác nhau sử dụng cùng chuỗi mã Order
  When hệ thống tiếp nhận cả hai
  Then mỗi Order được phân biệt bằng đầy đủ platform/shop/mã gốc
  And không gộp nhầm giao dịch của hai nguồn

Scenario: Thiếu mã tham chiếu bắt buộc
  Given payload Order không có platform, shop hoặc mã gốc cần thiết
  When hệ thống xử lý
  Then không tạo Order nội bộ có thể đi vào fulfillment
  And ghi nhận lỗi để Operator xử lý
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-MKT-01` | Cho Operator có quyền xem, tìm và lọc Order theo platform/shop, mã gốc, trạng thái và thời gian. | `US-MKT-01` |
| `FR-MKT-02` | Hiển thị đồng thời trạng thái gốc, trạng thái ánh xạ, Order nội bộ và trạng thái đồng bộ. | `US-MKT-01`, `US-MKT-03` |
| `FR-MKT-03` | Kiểm tra định danh nguồn, dữ liệu khách được phép, line item/SKU và giá trị trước khi tạo/liên kết Order nội bộ. | `US-MKT-02`, `BR-MKTPLACE-01~02`, `BR-MKTPLACE-07` |
| `FR-MKT-04` | Yêu cầu giữ tồn nguyên tử qua EPIC 09 ngay khi ghi nhận Order mới và không đưa vào fulfillment nếu điều kiện bắt buộc thất bại. | `US-MKT-02`, `BR-MKTPLACE-05` |
| `FR-MKT-05` | Khi nhận hủy hợp lệ, yêu cầu EPIC 08 hủy Order và EPIC 09 nhả đúng reservation mà không lặp tác động. | `US-MKT-02`, `BR-MKTPLACE-06` |
| `FR-MKT-06` | Lưu snapshot riêng các thành phần giá trị thực tế trên sàn gồm giảm giá người bán, trợ giá sàn, phí và số khách trả/thực nhận khi có. | `US-MKT-02`, `BR-MKTPLACE-07` |
| `FR-MKT-07` | Theo dõi mỗi lần đồng bộ theo đối tượng, chiều, trạng thái, thời điểm, lần thành công và lỗi gần nhất. | `US-MKT-03` |
| `FR-MKT-08` | Phân loại lỗi và cho phép thử lại có kiểm soát mà không nhân đôi tác động nghiệp vụ. | `US-MKT-03`, `BR-AUDIT-01` |
| `FR-MKT-09` | Xử lý trạng thái lặp, đến sai thứ tự, chưa ánh xạ hoặc xung đột mà không làm Order quay lùi trái phép. | `US-MKT-03`, `BR-ORDER-01`, `BR-ORDER-05` |
| `FR-MKT-10` | Đối soát settlement với Order theo nguồn, tham chiếu và các thành phần doanh thu/phí/điều chỉnh. | `US-MKT-04`, `BR-MKTPLACE-03`, `BR-MKTPLACE-07` |
| `FR-MKT-11` | Phân loại, theo dõi và xử lý sai lệch đối soát theo quyền, bảo toàn dữ liệu gốc và lịch sử. | `US-MKT-04`, `BR-AUTH-04`, `BR-AUDIT-01` |
| `FR-MKT-12` | Liên kết khoản điều chỉnh kỳ sau với đúng Order, settlement và kết quả đối soát trước. | `US-MKT-04` |
| `FR-MKT-13` | Tổng hợp doanh thu gộp, trợ giá/giảm giá, phí và thuần theo platform/shop/kỳ với định nghĩa rõ ràng. | `US-MKT-05` |
| `FR-MKT-14` | Phân biệt dữ liệu theo Order chưa đối soát với settlement đã đối soát/thực nhận. | `US-MKT-05` |
| `FR-MKT-15` | Cung cấp dữ liệu Marketplace đã đối soát cho EPIC 24 nhưng không thay thế báo cáo tài chính chuyên sâu. | `US-MKT-04`, `US-MKT-05`, `EPIC 24` |
| `FR-MKT-16` | Lưu tổ hợp platform/shop/mã Order gốc duy nhất và liên kết với đúng một Order nội bộ. | `US-MKT-06`, `BR-MKTPLACE-01~04` |
| `FR-MKT-17` | Chống tạo trùng Order/reservation khi cùng Order nguồn được gửi lại hoặc xử lý đồng thời. | `US-MKT-06`, `BR-MKTPLACE-04~05` |
| `FR-MKT-18` | Đồng bộ Product/listing theo ánh xạ SKU và chính sách giá/trạng thái được EPIC 04 cung cấp. | Yêu cầu hệ thống ngoài 36.2, `EPIC 04` |
| `FR-MKT-19` | Đồng bộ tồn bán được ra Marketplace từ tồn khả dụng EPIC 09; không bao gồm hàng đã giữ, hết hạn hoặc không đủ điều kiện. | Yêu cầu hệ thống ngoài 36.2, `BR-BATCH-05`, `EPIC 09` |
| `FR-MKT-20` | Đồng bộ dữ liệu khách hàng chỉ trong phạm vi nền tảng, pháp lý và mục đích fulfillment cho phép. | Yêu cầu hệ thống ngoài 36.2 |
| `FR-MKT-21` | Kiểm soát quyền theo platform/shop cho việc xem, thử lại, ánh xạ và đối soát. | `US-MKT-01~05`, `BR-AUTH-04` |
| `FR-MKT-22` | Ghi Audit cho ánh xạ thủ công, thử lại, xử lý xung đột và sai lệch quan trọng. | `US-MKT-03`, `US-MKT-04`, `BR-AUDIT-01` |

## 6. Luồng tích hợp và quy tắc dữ liệu

### 6.1. Luồng Order Marketplace

```text
Marketplace Platform
        │
        ▼
Xác minh nguồn + định danh platform/shop/order
        │
        ▼
Chống trùng + ánh xạ Listing → SKU
        │
        ├── Lỗi/thiếu ánh xạ ──► Hàng đợi ngoại lệ
        │
        ▼
Tạo/liên kết Order nội bộ + snapshot giá trị sàn
        │
        ▼
Giữ tồn qua EPIC 09
        │
        ├── Thất bại ──────────► Xử lý theo chính sách Marketplace
        │
        ▼
Packing / Shipping / cập nhật trạng thái
        │
        ▼
Settlement + đối soát doanh thu/phí
```

### 6.2. Quy tắc dữ liệu và an toàn nghiệp vụ

- Định danh Order ngoài phải gồm tối thiểu platform, shop/account và mã Order gốc; không giả định mã Order duy nhất trên mọi nguồn.
- Payload/sự kiện gốc cần được bảo toàn ở mức phù hợp để truy vết, nhưng dữ liệu cá nhân chỉ được lưu trong phạm vi được phép và cần thiết.
- Listing ngoài phải ánh xạ xác định về SKU nội bộ. Không dùng tên sản phẩm hoặc suy đoán gần đúng để tự động trừ tồn/fulfillment.
- Inventory nội bộ là nguồn tồn khả dụng. Số lượng đồng bộ ra sàn không được bao gồm reservation, hàng hết hạn, cách ly hoặc không đủ điều kiện bán.
- Order nội bộ lưu snapshot giá trị thực tế từ sàn; không tính lại bằng giá Website và không gộp mất trợ giá/phí cần cho đối soát.
- Trạng thái Marketplace và Order status là hai hệ trạng thái riêng. Mỗi platform cần bảng ánh xạ và quy tắc chiều đồng bộ rõ ràng.
- Mọi thao tác nhận/gửi/thử lại phải idempotent ở mức nghiệp vụ để không tạo trùng Order, reservation, hủy, hoàn tồn hoặc cập nhật trạng thái.
- Dữ liệu settlement/điều chỉnh là lịch sử tài chính; không ghi đè dữ liệu kỳ trước khi sàn gửi điều chỉnh mới.
- Thông tin xác thực kết nối Marketplace là dữ liệu nhạy cảm, chỉ actor/hệ thống có thẩm quyền được quản lý; không hiển thị trong log hoặc giao diện vận hành thông thường.
- Chi tiết API, webhook, chữ ký, token, rate limit, scheduler, queue, retry/backoff và lưu payload thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-MKT-01` | `BR-AUTH-04`, `BR-MKTPLACE-01~02` | EPIC 08, 22 | Operator chỉ thấy shop có quyền; Order hiển thị đủ nguồn/mã gốc và trạng thái ngoài/nội bộ. |
| `US-MKT-02` | `BR-MKTPLACE-01~07`, `BR-ORDER-02` | EPIC 08, 09, 10, 11 | Order hợp lệ vào fulfillment và giữ tồn; thiếu mapping/tồn bị chặn; hủy nhả tồn đúng một lần; giá trị theo sàn. |
| `US-MKT-03` | `BR-ORDER-01`, `BR-ORDER-05`, `BR-AUDIT-01` | EPIC 08, 23 | Lỗi có trạng thái rõ; retry không trùng; sự kiện cũ/xung đột không làm Order sai. |
| `US-MKT-04` | `BR-MKTPLACE-03`, `BR-MKTPLACE-07`, `BR-AUDIT-01` | EPIC 07, 08, 23, 24 | Settlement được đối soát đủ thành phần; sai lệch có quyền/lịch sử; điều chỉnh không ghi đè kỳ trước. |
| `US-MKT-05` | `BR-MKTPLACE-07` | EPIC 24 | Doanh thu theo sàn/kỳ nhất quán, phân biệt gộp/thuần và tạm tính/đã đối soát. |
| `US-MKT-06` | `BR-MKTPLACE-01`, `BR-MKTPLACE-02`, `BR-MKTPLACE-04` | EPIC 08, 09 | Tổ hợp nguồn duy nhất; gửi lặp/đồng thời chỉ tạo một Order; mã giống ở nguồn khác không bị gộp. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Marketplace và shop/account cụ thể được tích hợp ở Giai đoạn 2; phạm vi sandbox/production và quyền vận hành từng shop.
- Mô hình fulfillment: seller tự giao, sàn vận chuyển hay kết hợp; trách nhiệm tạo vận đơn và nguồn trạng thái Shipping.
- Bảng ánh xạ trạng thái Order/Payment/Shipping riêng cho từng platform và chiều nào là nguồn quyết định ở từng mốc.
- Quy tắc xử lý Order Marketplace khi không đủ tồn, listing chưa map, địa chỉ thiếu hoặc dữ liệu khách bị nền tảng che.
- Phạm vi duy nhất của Listing/SKU mapping và quy trình duyệt khi một listing chứa bundle/combo hoặc quà tặng.
- Tần suất/cơ chế đồng bộ tồn và buffer an toàn đa kênh; AC legacy yêu cầu “lập tức” nhưng chưa có SLA định lượng trong NFR.
- Quy tắc giá trị Order: gross merchandise value, seller discount, platform subsidy, shipping subsidy, commission, service fee, tax, refund và net settlement.
- Chu kỳ settlement, file/API đầu vào, thời hạn xử lý sai lệch và phân công giữa Marketplace Operator với Accountant.
- Quyền và quy trình chỉnh mapping/replay payload/retry sync; các thao tác có thể ảnh hưởng Order/tồn phải có phê duyệt phù hợp.
- Cách xử lý sự kiện đến muộn, mất thứ tự, thiếu sự kiện hoặc platform sửa Order sau khi đã vào Packing.
- Chính sách hủy một phần, trả một phần, hoàn tiền và điều chỉnh settlement của Marketplace.
- Quy tắc đồng bộ Customer trong giới hạn dữ liệu được sàn cho phép, thời hạn lưu và mục đích sử dụng; không tự hợp nhất với Customer nội bộ khi chưa xác minh.
- External Requirement 36.2 có đồng bộ Product và Customer nhưng chưa có User Story tương ứng; cần bổ sung backlog nếu đây là phạm vi triển khai chính thức.
- Có đồng bộ giá theo kênh hay chỉ Product/tồn; nguồn sự thật và phê duyệt giá thuộc EPIC 04.
- Các chỉ số doanh thu Marketplace nào nằm ở EPIC 12 và chỉ số tài chính nào chuyển sang EPIC 24 để tránh hai nguồn báo cáo.

## 9. UI/UX Reference

- Danh sách Order hiển thị rõ logo/tên platform, shop, mã gốc, mã nội bộ, trạng thái đồng bộ, fulfillment và cảnh báo lỗi.
- Chi tiết Order tách dữ liệu gốc Marketplace, snapshot nội bộ, mapping SKU, Inventory, Packing/Shipping và settlement.
- Hàng đợi lỗi nhóm theo thiếu mapping, không đủ tồn, trạng thái xung đột, lỗi gửi/nhận và sai lệch tài chính; hành động retry phải nêu tác động.
- Màn hình mapping Listing–SKU cho phép tìm/chọn SKU nhưng không tự áp dụng mapping mơ hồ; thay đổi quan trọng cần xác nhận và Audit.
- Đối soát hiển thị từng thành phần tiền của sàn cạnh giá trị nội bộ và làm nổi bật chênh lệch.
- Dashboard doanh thu phân biệt rõ gross, net, đã đối soát và chưa đối soát; luôn hiển thị kỳ cùng thời điểm cập nhật.
- Dữ liệu khách hàng bị che/giới hạn từ sàn phải giữ nguyên ý nghĩa, không hiển thị như dữ liệu đầy đủ.
- Chỉ tiêu realtime, API/webhook, rate limit, token, retry và công nghệ đồng bộ được đặc tả trong NFR hoặc Technical Design.

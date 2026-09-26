# EPIC 05 — Shopping Cart

## 1. Mục tiêu Epic

Epic này cho phép khách tạo danh sách sản phẩm/SKU dự định mua, thay đổi số lượng, xóa lựa chọn không cần thiết và xem tổng tiền tạm tính trước khi checkout. Giỏ hàng là bước chuẩn bị mua, không phải đơn hàng và không tự giữ tồn kho.

Epic không tạo đơn, thu thập địa chỉ, tính phí vận chuyển, áp khuyến mãi, tạm giữ tồn kho hay thanh toán. Các nội dung này lần lượt thuộc EPIC 06 — Checkout, EPIC 17 — Promotion & Loyalty và EPIC 07 — Payment.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP v1.0** | `US-CART-01` đến `US-CART-04` | Guest hoặc Customer chuẩn bị được giỏ hàng hợp lệ và biết tổng tiền hàng tạm tính trước khi checkout. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Thêm, sửa, xóa SKU trong giỏ và chuyển sang Guest Checkout mà không cần có tài khoản. |
| Registered Customer (`ACT-02`) | Quản lý giỏ hàng của mình trước khi checkout. |
| Sales Manager (`ACT-12`) | Cung cấp dữ liệu Product/SKU, giá và trạng thái bán qua EPIC 04; không trực tiếp quản lý giỏ của khách. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-01` | Khách vãng lai được phép mua hàng mà không bắt buộc tạo tài khoản. | Guest phải dùng được giỏ hàng và sang checkout mà không bị ép đăng nhập. |
| `BR-PROD-01` | Mỗi sản phẩm có thể có nhiều Variant/SKU. | Mỗi dòng giỏ hàng phải xác định chính xác SKU khách đã chọn, không chỉ Product chung. |
| `BR-PROD-02` | Giá, khối lượng, quy cách đóng gói và tồn kho được quản lý theo SKU. | Dòng giỏ và tổng tạm tính phải dùng giá/quy cách của SKU; khả năng mua cần dựa trên SKU. |
| `BR-BATCH-05` | Sản phẩm hết hạn không được phép tiếp tục bán. | Giỏ không được dẫn khách đến việc mua SKU chỉ còn hàng hết hạn. |
| `BR-ORDER-02` | Không xác nhận đơn vượt quá tồn kho khả dụng. | Giỏ có thể cảnh báo khả năng mua hiện tại; kiểm tra và tạm giữ mang tính quyết định diễn ra ở Checkout. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 01 — Authentication & Identity`: phân biệt Guest và Customer; Guest vẫn được dùng giỏ hàng.
- `EPIC 03 — Product Discovery` và `EPIC 04 — Product, Variant & SKU`: cung cấp Product/SKU được phép công khai, giá, quy cách và trạng thái bán.
- `EPIC 06 — Checkout`: nhận danh sách dòng giỏ đã chọn, kiểm tra lại dữ liệu và thực hiện giữ tồn trong thời gian thanh toán.
- `EPIC 09 — Inventory & Batch`: là nguồn tồn kho khả dụng và thông tin hàng hết hạn; Cart không tự quản lý Batch.
- `EPIC 17 — Promotion & Loyalty`: áp coupon/khuyến mãi sau MVP; không tính vào tổng tạm tính cơ bản nếu chưa có chính sách ưu đãi.

## 4. User Stories chi tiết

### US-CART-01 — Thêm sản phẩm vào giỏ hàng

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn thêm sản phẩm vào giỏ hàng để chuẩn bị mua.

**Giá trị nghiệp vụ:** Khách có thể gom nhiều SKU mong muốn trước khi quyết định mua, thay vì phải checkout ngay từng sản phẩm.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Thêm SKU hợp lệ vào giỏ
  Given khách đang xem một SKU được phép bán và có khả năng mua tại thời điểm kiểm tra
  When khách chọn số lượng hợp lệ và thêm SKU vào giỏ
  Then hệ thống thêm dòng giỏ tương ứng với đúng SKU và số lượng đã chọn
  And hiển thị tên, quy cách, giá hiện hành, số lượng và tạm tính của dòng đó

Scenario: Thêm lại SKU đã có trong giỏ
  Given giỏ của khách đã có một dòng cho SKU được chọn
  When khách thêm cùng SKU vào giỏ lần nữa với số lượng hợp lệ
  Then hệ thống cập nhật số lượng của dòng SKU đó theo chính sách giỏ hàng
  And không tạo các dòng trùng lặp gây khó hiểu cho cùng một SKU

Scenario: Không thêm SKU không thể mua
  Given SKU đang tạm ngừng bán, ngừng kinh doanh hoặc không còn khả năng mua theo dữ liệu hiện có
  When khách cố thêm SKU vào giỏ
  Then hệ thống không thêm SKU đó vào giỏ
  And hiển thị trạng thái phù hợp để khách chọn sản phẩm/SKU khác

Scenario: Guest thêm SKU vào giỏ
  Given khách chưa đăng nhập
  When Guest thêm SKU hợp lệ vào giỏ
  Then hệ thống lưu giỏ hàng theo ngữ cảnh Guest
  And không yêu cầu Guest tạo tài khoản chỉ để sử dụng giỏ hàng
```

### US-CART-02 — Thay đổi số lượng sản phẩm trong giỏ

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn thay đổi số lượng sản phẩm trong giỏ.

**Giá trị nghiệp vụ:** Khách tự điều chỉnh số lượng theo nhu cầu và thấy ngay ảnh hưởng đến tổng tiền trước khi checkout.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tăng hoặc giảm số lượng hợp lệ
  Given giỏ có một dòng SKU
  When khách thay đổi số lượng sang một giá trị hợp lệ
  Then hệ thống cập nhật số lượng của đúng dòng giỏ
  And tính lại tạm tính của dòng và tổng tiền tạm tính của giỏ

Scenario: Số lượng không hợp lệ
  Given khách đang thay đổi số lượng của một dòng giỏ
  When khách nhập số lượng bằng không, âm hoặc không hợp lệ theo chính sách bán hàng
  Then hệ thống không áp dụng giá trị không hợp lệ
  And thông báo để khách điều chỉnh hoặc dùng hành động xóa sản phẩm

Scenario: Số lượng vượt khả năng mua hiện tại
  Given số lượng khách yêu cầu lớn hơn khả năng mua được hệ thống ghi nhận tại thời điểm kiểm tra
  When khách cập nhật số lượng trong giỏ
  Then hệ thống thông báo số lượng không thể đáp ứng theo dữ liệu hiện có
  And không xác nhận số lượng vượt khả năng mua theo chính sách giỏ hàng

Scenario: Tồn kho thay đổi sau khi đã thêm giỏ
  Given SKU đã nằm trong giỏ của khách
  When khả năng mua của SKU thay đổi trước checkout
  Then hệ thống hiển thị cảnh báo hoặc cập nhật trạng thái phù hợp khi giỏ được kiểm tra lại
  And khách được yêu cầu điều chỉnh trước khi có thể tiếp tục checkout
```

### US-CART-03 — Xóa sản phẩm khỏi giỏ hàng

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn xóa sản phẩm khỏi giỏ.

**Giá trị nghiệp vụ:** Khách có thể bỏ các lựa chọn không còn phù hợp mà không phải tạo lại giỏ từ đầu.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xóa một SKU khỏi giỏ
  Given giỏ của khách có ít nhất một dòng SKU
  When khách chọn xóa một dòng giỏ
  Then hệ thống xóa đúng SKU đó khỏi giỏ
  And tính lại tổng tiền tạm tính của giỏ

Scenario: Xóa dòng cuối cùng trong giỏ
  Given giỏ chỉ còn một dòng SKU
  When khách xóa dòng đó
  Then hệ thống hiển thị trạng thái giỏ hàng trống
  And không cho chuyển sang checkout với giỏ trống

Scenario: Ngăn xóa giỏ của khách khác
  Given khách A đang sử dụng giỏ hàng của mình
  When khách A cố xóa dòng giỏ thuộc ngữ cảnh của khách B
  Then hệ thống từ chối thao tác
  And giỏ của khách B không bị thay đổi
```

### US-CART-04 — Xem tổng tiền tạm tính

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn xem tổng tiền tạm tính trước khi checkout.

**Giá trị nghiệp vụ:** Khách biết trước giá trị hàng hóa đã chọn, từ đó chủ động điều chỉnh giỏ trước khi sang checkout.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị tổng tiền của giỏ có sản phẩm
  Given giỏ có một hoặc nhiều dòng SKU hợp lệ
  When khách mở giỏ hoặc thay đổi dòng giỏ
  Then hệ thống hiển thị đơn giá hiện hành, số lượng và tạm tính của từng dòng
  And hiển thị tổng tiền tạm tính bằng tổng các dòng giỏ đang áp dụng

Scenario: Phân biệt tổng tạm tính và tổng thanh toán cuối cùng
  Given khách đang xem giỏ hàng
  When hệ thống hiển thị tổng tiền tạm tính
  Then hệ thống nêu rõ phí vận chuyển, ưu đãi và tổng thanh toán cuối cùng được xác định ở Checkout
  And không trình bày tổng tạm tính như giá trị đơn đã được xác nhận

Scenario: Giá SKU thay đổi trước checkout
  Given một SKU trong giỏ có giá hiện hành thay đổi theo chính sách giá
  When giỏ được tải lại hoặc kiểm tra trước checkout
  Then hệ thống cập nhật tổng tiền tạm tính theo giá hiện hành có hiệu lực
  And thông báo phù hợp để khách kiểm tra lại trước khi đặt hàng

Scenario: Giỏ trống
  Given giỏ không có dòng SKU nào
  When khách mở giỏ
  Then hệ thống hiển thị tổng tiền tạm tính bằng không hoặc trạng thái giỏ trống rõ ràng
  And không hiển thị thông tin thanh toán gây hiểu nhầm
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-CART-01` | Cung cấp giỏ hàng riêng cho Guest và Customer; chỉ chủ sở hữu/ngữ cảnh hợp lệ mới được xem hoặc thay đổi giỏ. | `US-CART-01~03`, `BR-AUTH-01` |
| `FR-CART-02` | Thêm SKU hợp lệ vào giỏ với số lượng hợp lệ và hiển thị thông tin SKU tương ứng. | `US-CART-01` |
| `FR-CART-03` | Gộp thao tác thêm lặp lại cùng SKU vào một dòng giỏ theo chính sách giỏ hàng. | `US-CART-01` |
| `FR-CART-04` | Kiểm tra trạng thái bán, khả năng mua và dữ liệu SKU trước khi thêm hoặc cập nhật dòng giỏ. | `US-CART-01`, `US-CART-02`, `BR-PROD-02`, `BR-BATCH-05` |
| `FR-CART-05` | Cho phép khách cập nhật số lượng một dòng giỏ hợp lệ và tính lại các giá trị tạm tính liên quan. | `US-CART-02` |
| `FR-CART-06` | Cho phép khách xóa một dòng SKU; cập nhật tổng tiền và hiển thị giỏ trống khi không còn dòng nào. | `US-CART-03` |
| `FR-CART-07` | Hiển thị đơn giá, số lượng, tạm tính mỗi dòng và tổng tiền tạm tính của cả giỏ theo giá hiện hành có hiệu lực. | `US-CART-04` |
| `FR-CART-08` | Phân biệt tổng tiền hàng tạm tính tại Cart với phí vận chuyển, ưu đãi, tổng thanh toán cuối cùng và việc giữ tồn ở Checkout. | `US-CART-04`, `EPIC 06`, `EPIC 17` |
| `FR-CART-09` | Cảnh báo hoặc ngăn tiếp tục khi SKU trong giỏ không còn có thể mua; quyết định xác nhận tồn và giữ tồn thuộc Checkout. | `US-CART-01`, `US-CART-02`, `BR-ORDER-02`, `EPIC 06` |

## 6. Quy tắc dữ liệu và trải nghiệm

- Mỗi dòng giỏ phải gắn với một SKU cụ thể; không dùng Product chung khi SKU có nhiều lựa chọn.
- Cart phản ánh lựa chọn dự định mua và không tự tạo đơn hoặc tự giữ tồn. Chỉ Checkout thực hiện việc tạm giữ tồn theo `US-CHK-03`.
- Giá và tổng tại Cart là **tạm tính**. Giá, tồn kho, phí vận chuyển, ưu đãi và tổng thanh toán phải được kiểm tra lại trước khi xác nhận đơn.
- Dòng giỏ không thể mua vì trạng thái bán, hết tồn hoặc hàng hết hạn phải được thông báo rõ; khách có thể sửa số lượng, xóa hoặc chọn SKU khác.
- Coupon, combo, loyalty, quy tắc gộp giỏ Guest sau đăng nhập và đồng bộ giỏ đa thiết bị chưa nằm trong User Story MVP hiện tại; phải được chốt riêng trước khi triển khai.
- Chi tiết về lưu trữ giỏ, session, đồng bộ thiết bị, API endpoint, cache và cơ sở dữ liệu thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-CART-01` | `BR-AUTH-01`, `BR-PROD-01`, `BR-PROD-02` | EPIC 01, 04, 09 | Guest/Customer thêm đúng SKU hợp lệ; SKU không thể mua bị từ chối. |
| `US-CART-02` | `BR-PROD-02`, `BR-BATCH-05`, `BR-ORDER-02` | EPIC 04, 06, 09 | Số lượng thay đổi đúng, tổng được tính lại và cảnh báo khi không đáp ứng khả năng mua. |
| `US-CART-03` | `BR-AUTH-01` | EPIC 01 | Xóa đúng dòng giỏ của chủ sở hữu và giỏ trống không checkout được. |
| `US-CART-04` | `BR-PROD-02` | EPIC 04, 06, 17 | Tổng tạm tính theo giá hiệu lực, phân biệt với phí/ưu đãi/tổng cuối ở Checkout. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Giới hạn số lượng tối đa của một SKU trong giỏ và cách xử lý khi số lượng vượt tồn khả dụng.
- Giỏ Guest được giữ trong bao lâu, có được gộp với giỏ Customer sau đăng nhập hay không và quy tắc xử lý SKU/số lượng trùng.
- Chính sách cập nhật giá trong giỏ khi giá SKU thay đổi, bao gồm thông báo cho khách và thời điểm buộc xác nhận lại.
- Cách hiển thị SKU tạm ngừng bán, hết hàng hoặc hết hạn trong giỏ: giữ lại để khách biết, tự xóa hay cho phép thay thế.
- Thứ tự áp dụng khuyến mãi/coupon/loyalty khi các tính năng Giai đoạn 2 được đưa vào checkout.
- Có hỗ trợ lưu giỏ/đồng bộ nhiều thiết bị cho Customer hay không; nếu có, xử lý xung đột thế nào.

## 9. UI/UX Reference

- Nút thêm giỏ xuất hiện sau khi khách chọn SKU; phản hồi rõ khi thêm thành công hoặc SKU không thể mua.
- Trang/mini-cart hiển thị ảnh/tên SKU, quy cách, đơn giá, điều khiển số lượng, xóa dòng và tổng tạm tính.
- Cảnh báo dễ hiểu khi giá hoặc khả năng mua thay đổi, kèm hành động sửa số lượng/xóa/chọn SKU khác.
- Giỏ trống có hành động quay lại danh mục/tìm kiếm; tổng tạm tính nêu rõ chưa bao gồm phí giao hàng và ưu đãi.

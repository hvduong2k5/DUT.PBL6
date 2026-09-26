# EPIC 06 — Checkout

## 1. Mục tiêu Epic

Epic này cho phép Guest Customer và Registered Customer chuyển một giỏ hàng hợp lệ thành yêu cầu đặt hàng: cung cấp thông tin nhận hàng, xem phí vận chuyển, kiểm tra lại toàn bộ chi phí và tạm giữ tồn kho trước khi chuyển sang thanh toán.

Checkout là ranh giới chốt dữ liệu thương mại của giao dịch. Giá, số lượng, địa chỉ nhận hàng, phí vận chuyển và ưu đãi phải được kiểm tra lại tại thời điểm khách xác nhận; dữ liệu đã chốt phải được chuyển sang Order dưới dạng snapshot để các thay đổi về sau không làm sai lệch đơn đã phát sinh.

Epic không quản lý nội dung giỏ hàng, chính sách giá, nghiệp vụ tính cước vận chuyển, xử lý giao dịch thanh toán, vòng đời Order sau khi tạo hoặc chính sách khuyến mãi. Các nội dung đó lần lượt thuộc EPIC 05, EPIC 04, EPIC 11, EPIC 07, EPIC 08 và EPIC 17.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP v1.0** | `US-CHK-01` đến `US-CHK-04` | Guest hoặc Customer cung cấp được thông tin nhận hàng, biết tổng chi phí cuối, giữ được tồn kho hợp lệ và xác nhận đặt hàng. |
| **Giai đoạn 2** | `US-CHK-05` | Khách áp dụng được mã ưu đãi hợp lệ do EPIC 17 cung cấp trong quá trình checkout. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Checkout và đặt hàng mà không bị bắt buộc tạo tài khoản. |
| Registered Customer (`ACT-02`) | Checkout bằng thông tin nhập mới hoặc địa chỉ đã lưu trong hồ sơ. |
| Logistics Provider (`EXT-03`) | Cung cấp lựa chọn giao hàng và dữ liệu tính phí thông qua nghiệp vụ của EPIC 11. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-01` | Khách vãng lai được phép mua hàng mà không bắt buộc tạo tài khoản. | Guest phải hoàn tất được checkout khi cung cấp đủ thông tin tạo đơn. |
| `BR-PROD-02` | Giá, khối lượng, quy cách đóng gói và tồn kho được quản lý theo SKU. | Checkout phải kiểm tra và chốt dữ liệu theo đúng SKU, không chỉ theo Product. |
| `BR-BATCH-05` | Sản phẩm hết hạn không được phép tiếp tục bán. | Không được giữ tồn hoặc tạo đơn cho SKU chỉ còn hàng không đủ điều kiện bán. |
| `BR-ORDER-01` | Đơn hàng phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Kết quả xác nhận checkout phải tạo Order ở trạng thái khởi đầu phù hợp với luồng thanh toán. |
| `BR-ORDER-02` | Không xác nhận đơn vượt quá tồn kho khả dụng. | Giữ tồn phải thành công trước khi Order được coi là đủ điều kiện chuyển sang thanh toán. |
| `BR-ORDER-03` | Đơn hàng chỉ chuyển sang Paid khi giao dịch được xác nhận hợp lệ. | Checkout không tự đánh dấu Order là `PAID`; trạng thái này thuộc kết quả của EPIC 07. |
| `BR-ORDER-05` | Mọi thay đổi trạng thái quan trọng của đơn hàng phải được ghi nhận để truy vết. | Việc tạo Order và các kết quả giữ tồn quan trọng phải có khả năng truy vết qua EPIC 08/23. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 01 — Authentication & Identity`: xác định Guest hoặc Customer; Guest không bị ép đăng nhập để checkout.
- `EPIC 02 — Customer & Employee Profile`: cung cấp hồ sơ và địa chỉ đã lưu của Customer; Checkout không tự thay đổi địa chỉ hồ sơ nếu khách không yêu cầu.
- `EPIC 04 — Product, Variant & SKU`: cung cấp SKU, giá hiện hành, quy cách và trạng thái bán.
- `EPIC 05 — Shopping Cart`: cung cấp giỏ và các dòng SKU đầu vào; Cart không giữ tồn và không tạo Order.
- `EPIC 07 — Payment`: tiếp nhận Order cùng phương thức thanh toán để xử lý QR/chuyển khoản, COD và trạng thái thanh toán.
- `EPIC 08 — Order Management`: tạo, lưu và quản lý Order cùng snapshot thương mại sau khi Checkout xác nhận hợp lệ.
- `EPIC 09 — Inventory & Batch`: kiểm tra tồn khả dụng, thực thi giữ/nhả tồn và ngăn bán vượt mức.
- `EPIC 11 — Shipping & Delivery`: cung cấp phương thức giao hàng, phí và điều kiện giao tương ứng.
- `EPIC 17 — Promotion & Loyalty`: kiểm tra và tính giá trị ưu đãi trong Giai đoạn 2.
- `EPIC 23 — Audit & Security`: truy vết các thay đổi nghiệp vụ quan trọng liên quan đến Order và reservation.

## 4. User Stories chi tiết

### US-CHK-01 — Nhập thông tin nhận hàng

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn nhập thông tin nhận hàng để hoàn tất đơn.

**Giá trị nghiệp vụ:** Doanh nghiệp có đủ dữ liệu chính xác để liên hệ và giao hàng, đồng thời Guest vẫn có thể mua mà không cần tạo tài khoản.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Guest nhập thông tin nhận hàng hợp lệ
  Given Guest có một giỏ hàng không rỗng thuộc ngữ cảnh của mình
  When Guest nhập đầy đủ họ tên người nhận, số điện thoại và địa chỉ giao hàng hợp lệ
  Then hệ thống chấp nhận thông tin nhận hàng cho phiên checkout
  And không yêu cầu Guest đăng ký hoặc đăng nhập tài khoản

Scenario: Customer chọn địa chỉ đã lưu
  Given Customer đã đăng nhập và có địa chỉ giao hàng đã lưu
  When Customer chọn một địa chỉ thuộc hồ sơ của mình
  Then hệ thống điền thông tin nhận hàng tương ứng vào checkout
  And cho phép Customer kiểm tra hoặc chỉnh sửa bản thông tin dùng cho đơn hiện tại
  And không tự thay đổi địa chỉ đã lưu nếu Customer không yêu cầu

Scenario: Thông tin nhận hàng không hợp lệ
  Given khách đang nhập thông tin nhận hàng
  When khách bỏ trống trường bắt buộc hoặc nhập số điện thoại hay địa chỉ không hợp lệ
  Then hệ thống chỉ rõ thông tin cần chỉnh sửa
  And không cho khách xác nhận đặt hàng cho đến khi dữ liệu hợp lệ

Scenario: Ngăn sử dụng địa chỉ của Customer khác
  Given Customer đang checkout bằng tài khoản của mình
  When Customer cố chọn một địa chỉ không thuộc hồ sơ của mình
  Then hệ thống từ chối yêu cầu
  And không tiết lộ thông tin địa chỉ của Customer khác
```

### US-CHK-02 — Xem phí vận chuyển

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn xem phí vận chuyển trước khi xác nhận đơn.

**Giá trị nghiệp vụ:** Khách biết chi phí giao hàng trước khi cam kết mua và có thể lựa chọn phương thức giao phù hợp nếu có nhiều lựa chọn.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị phí vận chuyển cho địa chỉ hợp lệ
  Given giỏ hàng có các SKU với dữ liệu khối lượng và quy cách hợp lệ
  And khách đã cung cấp địa chỉ giao hàng hợp lệ
  When hệ thống nhận được các lựa chọn giao hàng khả dụng từ nghiệp vụ Shipping
  Then hệ thống hiển thị phí và thông tin cần thiết của từng lựa chọn
  And tính lựa chọn được áp dụng vào tổng thanh toán

Scenario: Tính lại phí khi dữ liệu ảnh hưởng thay đổi
  Given checkout đã có một mức phí vận chuyển
  When khách thay đổi địa chỉ, lựa chọn giao hàng hoặc nội dung giỏ làm thay đổi dữ liệu tính cước
  Then hệ thống yêu cầu tính lại phí vận chuyển
  And cập nhật tổng thanh toán trước khi khách xác nhận

Scenario: Không có phương thức giao hàng phù hợp
  Given khách đã cung cấp địa chỉ nhận hàng
  When không có phương thức giao hàng nào hỗ trợ địa chỉ hoặc kiện hàng hiện tại
  Then hệ thống thông báo không thể giao đến địa chỉ đã chọn
  And không cho xác nhận đặt hàng với phương thức giao không hợp lệ

Scenario: Không thể lấy phí vận chuyển
  Given thông tin nhận hàng và giỏ hàng đều hợp lệ
  When hệ thống chưa thể xác định phí vận chuyển
  Then hệ thống thông báo để khách thử lại hoặc chọn phương án khả dụng khác
  And không hiển thị một tổng thanh toán chưa đầy đủ như tổng cuối cùng
```

### US-CHK-03 — Tạm giữ sản phẩm trong thời gian thanh toán

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn được tạm giữ sản phẩm trong thời gian thanh toán để tránh sản phẩm bị người khác mua mất.

**Giá trị nghiệp vụ:** Giảm tình trạng khách thanh toán cho hàng không còn khả dụng và ngăn hệ thống bán vượt tồn khi nhiều khách đặt đồng thời.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Giữ tồn thành công khi xác nhận đặt hàng
  Given checkout có giỏ hàng, thông tin nhận hàng, phí vận chuyển và tổng tiền hợp lệ
  And các SKU còn đủ tồn khả dụng
  When khách xác nhận đặt hàng
  Then hệ thống tạm giữ đúng SKU và số lượng cho giao dịch
  And liên kết reservation với Order tương ứng
  And chuyển Order sang bước thanh toán phù hợp

Scenario: Hai khách cùng đặt đơn vị tồn cuối cùng
  Given một SKU chỉ còn một đơn vị khả dụng
  When hai khách cùng xác nhận checkout cho SKU đó
  Then chỉ một yêu cầu được giữ tồn thành công
  And yêu cầu còn lại được thông báo rằng số lượng không còn đáp ứng
  And không tạo đơn được xác nhận vượt quá tồn kho khả dụng

Scenario: Một dòng SKU không đủ tồn
  Given checkout có nhiều dòng SKU
  And ít nhất một dòng không còn đủ số lượng khả dụng
  When khách xác nhận đặt hàng
  Then hệ thống không coi việc giữ tồn cho toàn bộ checkout là thành công
  And hiển thị đúng dòng cần được điều chỉnh
  And không chuyển checkout không hợp lệ sang Payment

Scenario: Reservation hết hạn trước khi thanh toán
  Given Order đang chờ thanh toán và reservation đã hết thời hạn theo chính sách
  When hệ thống xác nhận giao dịch chưa được thanh toán hợp lệ
  Then Order được chuyển sang trạng thái hết hạn theo EPIC 08
  And tồn đã giữ được giải phóng qua EPIC 09
  And giao dịch thanh toán cũ không còn được dùng để xác nhận Order như bình thường
```

### US-CHK-04 — Kiểm tra và xác nhận đơn

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Must Have — 1st

**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn kiểm tra lại sản phẩm, số lượng, địa chỉ, phí vận chuyển và tổng tiền trước khi đặt hàng.

**Giá trị nghiệp vụ:** Khách chủ động phát hiện sai sót trước khi cam kết mua; doanh nghiệp tạo Order từ dữ liệu đã được kiểm tra và chốt nhất quán.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị đầy đủ thông tin xác nhận
  Given checkout có dữ liệu hợp lệ
  When khách mở bước kiểm tra cuối
  Then hệ thống hiển thị đúng SKU, quy cách, đơn giá, số lượng và tạm tính từng dòng
  And hiển thị thông tin người nhận, địa chỉ và lựa chọn giao hàng
  And hiển thị riêng tiền hàng, phí vận chuyển, ưu đãi nếu có và tổng cần thanh toán

Scenario: Giá hoặc trạng thái SKU thay đổi trước khi xác nhận
  Given khách đang xem bước kiểm tra cuối
  When giá hoặc trạng thái bán của một SKU thay đổi trước lúc khách xác nhận
  Then hệ thống kiểm tra lại dữ liệu theo thông tin đang có hiệu lực
  And thông báo rõ thay đổi cho khách
  And yêu cầu khách xác nhận lại tổng mới hoặc điều chỉnh giỏ trước khi đặt hàng

Scenario: Tạo Order với snapshot đã xác nhận
  Given khách đã kiểm tra dữ liệu cuối cùng
  And việc kiểm tra và giữ tồn đều thành công
  When khách xác nhận đặt hàng
  Then hệ thống tạo một Order chứa snapshot SKU, đơn giá, số lượng, người nhận, địa chỉ, phí vận chuyển và ưu đãi áp dụng
  And thay đổi dữ liệu nguồn sau đó không tự sửa nội dung Order đã tạo

Scenario: Khách gửi lặp lại yêu cầu xác nhận
  Given một yêu cầu xác nhận checkout đã tạo Order thành công
  When cùng yêu cầu đó được gửi lại do nhấn nhiều lần hoặc thử lại kết nối
  Then hệ thống trả về kết quả gắn với Order đã tạo
  And không tạo thêm Order hoặc reservation trùng lặp

Scenario: Giỏ hàng không hợp lệ khi xác nhận
  Given giỏ hàng trống, không thuộc ngữ cảnh khách hiện tại hoặc có dòng không còn mua được
  When khách cố xác nhận đặt hàng
  Then hệ thống từ chối tạo Order
  And hướng dẫn khách quay lại giỏ để xử lý dữ liệu không hợp lệ
```

### US-CHK-05 — Sử dụng mã ưu đãi

**Actor:** Guest Customer, Registered Customer

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn sử dụng mã ưu đãi hợp lệ trong quá trình checkout.

**Giá trị nghiệp vụ:** Khách nhận đúng quyền lợi khuyến mãi và biết chính xác số tiền phải trả trước khi xác nhận đơn.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Áp dụng mã ưu đãi hợp lệ
  Given checkout đáp ứng điều kiện của một mã ưu đãi đang có hiệu lực
  When khách nhập và áp dụng mã
  Then hệ thống nhận kết quả kiểm tra ưu đãi từ EPIC 17
  And hiển thị giá trị giảm cùng tổng thanh toán mới

Scenario: Mã ưu đãi không hợp lệ
  Given khách nhập một mã không tồn tại, hết hạn, hết lượt hoặc không đủ điều kiện
  When khách yêu cầu áp dụng mã
  Then hệ thống không áp dụng ưu đãi
  And hiển thị lý do phù hợp để khách có thể tiếp tục checkout không dùng mã

Scenario: Điều kiện ưu đãi thay đổi trước khi xác nhận
  Given checkout đang có mã ưu đãi đã được chấp nhận
  When nội dung giỏ, địa chỉ, lựa chọn giao hàng hoặc điều kiện chương trình thay đổi
  Then hệ thống kiểm tra lại mã ưu đãi
  And tính lại tổng thanh toán trước khi cho phép xác nhận đặt hàng

Scenario: Lưu snapshot ưu đãi trên Order
  Given mã ưu đãi còn hợp lệ tại thời điểm xác nhận
  When Order được tạo thành công
  Then Order lưu mã, giá trị và dữ liệu ưu đãi cần thiết để đối soát
  And thay đổi chương trình sau đó không tự sửa tổng tiền của Order đã tạo
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-CHK-01` | Cho phép Guest và Customer bắt đầu checkout từ một giỏ hàng không rỗng thuộc đúng ngữ cảnh của mình. | `US-CHK-01`, `US-CHK-04`, `BR-AUTH-01` |
| `FR-CHK-02` | Thu thập và kiểm tra họ tên người nhận, số điện thoại và địa chỉ giao hàng bắt buộc trước khi xác nhận. | `US-CHK-01` |
| `FR-CHK-03` | Cho phép Customer chọn địa chỉ đã lưu; chỉnh sửa bản dùng cho checkout không tự cập nhật hồ sơ nếu khách không yêu cầu. | `US-CHK-01`, `EPIC 02` |
| `FR-CHK-04` | Kiểm tra lại SKU, trạng thái bán, giá, số lượng, điều kiện hạn sử dụng và tồn khả dụng khi vào checkout và trước lúc xác nhận. | `US-CHK-03`, `US-CHK-04`, `BR-PROD-02`, `BR-BATCH-05`, `BR-ORDER-02` |
| `FR-CHK-05` | Yêu cầu và hiển thị các lựa chọn giao hàng cùng phí áp dụng dựa trên địa chỉ và dữ liệu kiện hàng. | `US-CHK-02`, `EPIC 11` |
| `FR-CHK-06` | Yêu cầu tính lại phí vận chuyển khi địa chỉ, lựa chọn giao hàng hoặc nội dung giỏ ảnh hưởng đến dữ liệu tính cước. | `US-CHK-02` |
| `FR-CHK-07` | Hiển thị riêng tiền hàng, phí vận chuyển, giảm giá và tổng cần thanh toán trước khi khách xác nhận. | `US-CHK-02`, `US-CHK-04`, `US-CHK-05` |
| `FR-CHK-08` | Thực hiện yêu cầu giữ tồn nguyên tử cho đúng SKU và số lượng; chỉ tiếp tục khi toàn bộ yêu cầu giữ tồn thành công. | `US-CHK-03`, `BR-ORDER-02`, `EPIC 09` |
| `FR-CHK-09` | Liên kết reservation với Order và phối hợp giải phóng tồn khi Order hết hạn, bị hủy hoặc không thể tiếp tục theo chính sách. | `US-CHK-03`, `EPIC 08`, `EPIC 09` |
| `FR-CHK-10` | Tạo Order bằng snapshot dữ liệu thương mại và giao hàng đã được kiểm tra tại thời điểm xác nhận. | `US-CHK-04`, `EPIC 08` |
| `FR-CHK-11` | Bảo đảm cùng một yêu cầu xác nhận không tạo nhiều Order hoặc nhiều reservation do gửi lặp. | `US-CHK-03`, `US-CHK-04` |
| `FR-CHK-12` | Chuyển Order hợp lệ sang luồng Payment phù hợp nhưng không tự xác nhận trạng thái `PAID`. | `US-CHK-03`, `US-CHK-04`, `BR-ORDER-03`, `EPIC 07` |
| `FR-CHK-13` | Kiểm tra, áp dụng, tính lại và lưu snapshot mã ưu đãi thông qua nghiệp vụ của EPIC 17. | `US-CHK-05`, `EPIC 17` |

## 6. Quy tắc dữ liệu và an toàn nghiệp vụ

- Mọi phép tính giá trị checkout phải sử dụng dữ liệu do hệ thống kiểm tra; không tin đơn giá, phí, giảm giá hoặc tổng tiền do client tự gửi lên.
- Mỗi dòng checkout và Order phải tham chiếu đúng SKU. Snapshot phải đủ để hiển thị và đối soát đơn ngay cả khi catalog, địa chỉ hoặc chương trình ưu đãi thay đổi sau đó.
- Checkout chỉ được xác nhận khi giỏ, thông tin nhận hàng, lựa chọn giao hàng, tổng tiền và toàn bộ SKU đều hợp lệ tại cùng thời điểm kiểm tra cuối.
- Giữ tồn là kết quả nghiệp vụ của Checkout nhưng số lượng khả dụng và thao tác reservation/release do EPIC 09 chịu trách nhiệm; lưu trạng thái Order do EPIC 08 chịu trách nhiệm.
- Việc khách xác nhận checkout không đồng nghĩa đã thanh toán. Chỉ kết quả hợp lệ từ EPIC 07 mới được phép làm Order chuyển sang `PAID`.
- Yêu cầu xác nhận phải có khả năng chống xử lý trùng để thao tác thử lại hoặc nhấn nhiều lần không tạo nhiều Order.
- Dữ liệu cá nhân của Guest hoặc Customer chỉ được dùng trong phạm vi cần thiết để tạo và thực hiện đơn; không được cho phép truy cập chéo địa chỉ hoặc checkout của khách khác.
- Giá trị ưu đãi chỉ được chốt khi EPIC 17 xác nhận mã còn hợp lệ tại thời điểm tạo Order.
- Chi tiết API endpoint, cơ sở dữ liệu, khóa phân tán, TTL kỹ thuật, giao thức giữa dịch vụ và cơ chế realtime thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-CHK-01` | `BR-AUTH-01` | EPIC 01, 02 | Guest checkout không cần tài khoản; Customer dùng được địa chỉ của mình; dữ liệu nhận hàng không hợp lệ bị từ chối. |
| `US-CHK-02` | `BR-PROD-02` | EPIC 04, 11 | Phí đúng theo địa chỉ/kiện hàng, được tính lại khi đầu vào thay đổi và không tạo tổng cuối khi chưa có phí hợp lệ. |
| `US-CHK-03` | `BR-BATCH-05`, `BR-ORDER-02` | EPIC 08, 09 | Giữ tồn nguyên tử, không oversell, thất bại một dòng không chuyển sang Payment và reservation hết hạn được giải phóng. |
| `US-CHK-04` | `BR-PROD-02`, `BR-ORDER-01`, `BR-ORDER-03`, `BR-ORDER-05` | EPIC 04, 05, 07, 08, 23 | Tổng được kiểm tra lại, Order lưu snapshot chính xác và yêu cầu lặp không tạo đơn trùng. |
| `US-CHK-05` | Chính sách ưu đãi của EPIC 17 | EPIC 17 | Mã hợp lệ làm thay đổi đúng tổng tiền; mã không hợp lệ bị từ chối; Order lưu snapshot ưu đãi. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Bộ trường địa chỉ bắt buộc và nguồn danh mục hành chính được hỗ trợ, bao gồm cách xử lý thay đổi địa giới.
- Thời hạn hiệu lực của báo phí vận chuyển và cách xử lý khi nhà vận chuyển không phản hồi hoặc phí thay đổi.
- Thời hạn reservation cho từng phương thức thanh toán; thời điểm bắt đầu đếm và quy tắc gia hạn nếu có.
- Trình tự tạo Order và giữ tồn để vừa chống oversell vừa không để lại Order/reservation mồ côi khi một bước thất bại.
- Trạng thái khởi đầu và reservation đối với COD, vì COD không thể tuân theo cùng điều kiện trả trước như QR/chuyển khoản.
- Cách xử lý webhook thanh toán đến sau khi Order/reservation đã hết hạn; không tự chuyển `PAID` nếu chưa có quy tắc đối soát phù hợp.
- Chính sách khi chỉ một phần giỏ hết tồn: yêu cầu khách sửa giỏ hay cho phép tách đơn.
- Quy tắc làm tròn tiền, đơn vị tiền tệ, thuế/phí đã bao gồm và thứ tự áp dụng giảm giá sản phẩm, giảm giá đơn, miễn phí vận chuyển, loyalty.
- Khoá chống xử lý trùng của yêu cầu xác nhận và thời gian duy trì hiệu lực của kết quả checkout.
- Phạm vi hỗ trợ giao nhiều địa chỉ, nhận tại cửa hàng và ghi chú giao hàng; mặc định MVP là một địa chỉ cho một Order nếu chưa có quyết định khác.

## 9. UI/UX Reference

- Luồng checkout MVP nên thể hiện rõ ba nhóm thao tác: thông tin nhận hàng, giao hàng/thanh toán và kiểm tra xác nhận.
- Luôn hiển thị breakdown gồm tiền hàng, phí vận chuyển, giảm giá và tổng cần thanh toán; không dùng một con số tổng thiếu diễn giải.
- Cảnh báo thay đổi giá, phí hoặc tồn phải chỉ rõ SKU hay dữ liệu bị ảnh hưởng và cung cấp hành động quay lại chỉnh sửa.
- Nút xác nhận cần có trạng thái đang xử lý để hạn chế thao tác lặp, nhưng hệ thống vẫn phải chống tạo Order trùng ở phía nghiệp vụ.
- Khi reservation có thời hạn, giao diện Payment phải hiển thị thời gian còn lại và hành vi khi hết hạn theo chính sách đã chốt.
- Giao diện văn hóa Huế, chỉ tiêu tốc độ phản hồi, URL điều hướng và công nghệ realtime được đặc tả ở UI Design, NFR hoặc Technical Design thay vì User Story của Epic này.

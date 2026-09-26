# EPIC 26 — Gifting Experience

## 1. Mục tiêu Epic

Epic này cho phép Guest hoặc Registered Customer mua hàng làm quà: sử dụng thông tin người nhận khác người đặt, thêm lời chúc và yêu cầu ẩn giá trên tài liệu dành cho người nhận. Trải nghiệm phải giữ rõ ai là người mua/chủ Order, ai là người nhận, ai thanh toán và thông tin nào được phép xuất hiện ở từng tài liệu hoặc thông báo.

EPIC 26 sở hữu Gift Preference và snapshot quà tặng trên Order. Epic không sở hữu giỏ hàng, tính tiền, Order lifecycle, Payment, Packing, Shipping hay Notification; các phân hệ đó sử dụng Gift Preference đã chốt để thực hiện nghiệp vụ tương ứng.

Ẩn giá là quy tắc trình bày dành cho người nhận, không được sửa giá trị tài chính của Order, Payment, hóa đơn, Audit hoặc báo cáo nội bộ và không được che thông tin pháp luật/đơn vị vận chuyển bắt buộc phải hiển thị.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 2 — Should Have** | `US-GIFT-01` đến `US-GIFT-03` | Khách gửi quà cho người nhận khác, thêm lời chúc và ẩn giá trên phiếu quà tặng. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors

| Actor / bên liên quan | Vai trò |
| --- | --- |
| Guest Customer (`ACT-01`) | Mua quà không cần tài khoản, quản lý phiên checkout/Order bằng cơ chế xác minh phù hợp. |
| Registered Customer (`ACT-02`) | Mua quà, chọn thông tin giao nhận và theo dõi Order thuộc tài khoản. |
| Gift Recipient | Nhận hàng/thông báo được phép; không mặc nhiên là chủ Order hoặc được xem dữ liệu người mua. |
| Packing Staff (`ACT-07`) | Nhận chỉ dẫn đóng gói cần thiết và gắn đúng phiếu/lời chúc với đúng Order. |
| Delivery Staff / Logistics Provider (`ACT-08`, `EXT-03`) | Nhận dữ liệu giao hàng cần thiết; không cần truy cập nội dung tài chính bị hạn chế. |

### 3.2. Business Rules và NFR áp dụng

Product Backlog chưa định nghĩa nhóm `BR-GIFT`; Epic không tự tạo mã Business Rule mới.

| Mã | Ảnh hưởng đến Epic |
| --- | --- |
| `BR-AUTH-01` | Guest được phép mua quà mà không bắt buộc tạo tài khoản. |
| `BR-AUTH-04` | Nhân viên chỉ được xem/sửa dữ liệu quà tặng trong quyền và công đoạn được giao. |
| `BR-PROD-02` | Gift Preference không làm thay đổi SKU, giá hoặc quy cách đã chọn. |
| `BR-ORDER-01` | Gift Order vẫn theo Order lifecycle chuẩn, không tạo vòng đời song song. |
| `BR-ORDER-02`, `BR-ORDER-03` | Đánh dấu quà không bỏ qua kiểm tra tồn hoặc xác nhận Payment. |
| `BR-ORDER-04` | Gift Order chỉ được khách hủy trong điều kiện chung của Order. |
| `BR-ORDER-05` | Thay đổi Gift Preference quan trọng sau khi tạo Order phải truy vết được. |
| `BR-PACK-01`, `BR-PACK-02` | Chỉ Packing Staff được phân công thực hiện và vẫn phải kiểm tra SKU/số lượng. |
| `BR-AUDIT-01` | Truy cập/sửa dữ liệu người nhận hoặc override chỉ dẫn ẩn giá phải có Audit khi thuộc catalog nhạy cảm. |
| `NFR-04`, `NFR-05`, `NFR-08` | Bảo vệ thông tin người mua/người nhận, kiểm soát quyền và sử dụng đúng mục đích. |
| `NFR-10` | Dữ liệu quà tặng không được làm suy giảm khả năng xử lý Order quy mô lớn. |
| `NFR-11` | Theo dõi lỗi tạo phiếu/lời chúc và việc chuyển chỉ dẫn sang Packing/Notification. |

### 3.3. Phụ thuộc và ranh giới

- `EPIC 01`, `EPIC 02`: xác định Guest/Customer và địa chỉ đã lưu; Gift Recipient không tự trở thành Customer.
- `EPIC 05`, `EPIC 06`: giỏ/checkout thu thập và cho khách kiểm tra Gift Preference trước khi đặt.
- `EPIC 07`: xử lý Payment; đánh dấu quà không thay đổi số phải thanh toán.
- `EPIC 08`: sở hữu Order và lưu snapshot Gift Preference; người mua vẫn là chủ Order.
- `EPIC 10`: thực hiện hướng dẫn đóng gói, tạo/gắn phiếu quà hoặc lời chúc; EPIC 26 định nghĩa nội dung/kết quả mong muốn.
- `EPIC 11`: giao đến recipient snapshot và chỉ chia sẻ dữ liệu cần thiết cho vận chuyển.
- `EPIC 14`: Return/Refund theo chủ thể/chính sách được chốt; recipient không mặc nhiên có quyền yêu cầu Refund.
- `EPIC 23`: Audit truy cập và thay đổi nhạy cảm.
- `EPIC 28`: gửi thông báo đúng đối tượng/nội dung; không tự quyết định người mua hay recipient được nhận loại thông báo nào.

## 4. User Stories chi tiết

### US-GIFT-01 — Gửi sản phẩm đến người nhận khác

**Actor:** Guest Customer (`ACT-01`), Registered Customer (`ACT-02`)

**Ưu tiên:** Should Have — **Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn gửi sản phẩm đến một người nhận khác với người đặt hàng để mua quà.

**Giá trị nghiệp vụ:** Người mua hoàn tất Order quà tặng trong luồng checkout chuẩn mà người nhận, người mua và người thanh toán không bị nhập nhằng.

```gherkin
Scenario: Đánh dấu Order là quà và nhập recipient
  Given khách có giỏ hàng hợp lệ
  When chọn mua làm quà và nhập họ tên, số điện thoại, địa chỉ người nhận hợp lệ
  Then checkout lưu Gift Preference trong đúng phiên
  And hiển thị riêng người mua và người nhận để khách kiểm tra

Scenario: Người nhận giống người mua
  Given khách đã chọn mua làm quà
  When dùng cùng thông tin nhận hàng với người mua
  Then hệ thống vẫn cho phép nếu dữ liệu hợp lệ
  And không tự tắt lời chúc hoặc tùy chọn ẩn giá

Scenario: Recipient thiếu hoặc sai dữ liệu bắt buộc
  Given đơn quà cần giao tới người nhận khác
  When khách nhập thiếu tên, số điện thoại hoặc địa chỉ không hợp lệ
  Then hệ thống không cho xác nhận Order
  And chỉ rõ trường cần sửa mà không làm mất dữ liệu hợp lệ khác

Scenario: Tạo Gift Order thành công
  Given checkout, tồn và Payment condition đều đủ điều kiện
  When khách xác nhận đặt hàng
  Then EPIC 08 tạo đúng một Order với buyer và recipient snapshot riêng
  And Gift Preference được liên kết đúng Order

Scenario: Thay đổi recipient trước điểm khóa
  Given Order chưa qua điểm không còn cho sửa giao nhận
  When chủ Order cập nhật recipient hợp lệ
  Then hệ thống lưu snapshot mới và chuyển thay đổi tới Shipping/Packing liên quan
  And ghi lịch sử thay đổi theo chính sách

Scenario: Thay đổi recipient quá muộn
  Given Order đã đóng gói, bàn giao hoặc vượt cutoff
  When khách yêu cầu đổi người nhận/địa chỉ
  Then hệ thống không tự sửa Shipment hiện tại
  And hướng dẫn khách qua quy trình hỗ trợ phù hợp

Scenario: Recipient không phải chủ Order
  Given recipient biết mã Order hoặc nhận được liên kết thông báo
  When cố xem Payment, lịch sử người mua hoặc thay đổi Order
  Then hệ thống chỉ cho phép hành động recipient đã được xác minh và cấp rõ ràng
  And không tiết lộ dữ liệu buyer ngoài phạm vi

Scenario: Guest mua quà
  Given Guest có phiên checkout hợp lệ
  When hoàn tất Gift Order
  Then hệ thống không yêu cầu đăng ký tài khoản
  And cung cấp cơ chế tra cứu/quản lý Order phù hợp mà không lộ dữ liệu recipient
```

### US-GIFT-02 — Thêm lời chúc vào đơn quà

**Actor:** Guest Customer (`ACT-01`), Registered Customer (`ACT-02`)

**Ưu tiên:** Should Have — **Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn thêm lời chúc vào đơn quà tặng.

**Giá trị nghiệp vụ:** Recipient nhận đúng lời chúc của người mua trên phiếu/thiệp gắn với đúng Order mà nội dung không gây lỗi hiển thị hoặc làm lộ dữ liệu ngoài ý muốn.

```gherkin
Scenario: Thêm lời chúc hợp lệ
  Given checkout được đánh dấu là quà
  When khách nhập lời chúc trong giới hạn được hỗ trợ
  Then hệ thống lưu nội dung trong Gift Preference
  And hiển thị bản xem trước để khách xác nhận

Scenario: Không nhập lời chúc
  Given khách mua làm quà
  When bỏ trống lời chúc
  Then hệ thống vẫn cho tiếp tục
  And không tự tạo nội dung thay mặt khách nếu chưa được yêu cầu

Scenario: Nội dung vượt giới hạn
  Given có giới hạn độ dài/ký tự được công bố
  When khách nhập nội dung không hợp lệ
  Then hệ thống không cắt âm thầm
  And chỉ rõ giới hạn để khách chỉnh sửa

Scenario: Nội dung có markup hoặc mã nguy hiểm
  Given lời chúc chứa nội dung có thể được diễn giải như mã thực thi
  When hệ thống lưu hoặc hiển thị
  Then nội dung được kiểm tra và render an toàn dưới dạng được hỗ trợ
  And không thực thi mã hoặc làm hỏng tài liệu

Scenario: Sửa lời chúc trước điểm khóa
  Given Order chưa bắt đầu công đoạn tạo phiếu/đóng gói
  When chủ Order cập nhật lời chúc hợp lệ
  Then phiên bản mới được dùng cho Packing
  And thay đổi được liên kết đúng Order

Scenario: Sửa lời chúc sau khi đã tạo phiếu
  Given Packing đã tạo hoặc in artifact chứa lời chúc
  When khách yêu cầu thay đổi
  Then hệ thống không khẳng định thay đổi đã áp dụng cho kiện hàng
  And chuyển sang hỗ trợ/rework nếu chính sách cho phép

Scenario: Packing nhận đúng nội dung
  Given Gift Order sẵn sàng đóng gói
  When Packing Staff mở nhiệm vụ được phân công
  Then hệ thống hiển thị lời chúc và chỉ dẫn theo đúng phiên bản có hiệu lực
  And không trộn nội dung giữa các Order

Scenario: Không tạo được phiếu hoặc thiệp
  Given nội dung hợp lệ nhưng bước tạo artifact thất bại
  When Packing xử lý Order
  Then hệ thống cảnh báo lỗi có thể hành động
  And không đánh dấu đã gắn lời chúc khi chưa có bằng chứng hoàn tất
```

### US-GIFT-03 — Ẩn giá trên phiếu quà tặng

**Actor:** Guest Customer (`ACT-01`), Registered Customer (`ACT-02`)

**Ưu tiên:** Should Have — **Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn ẩn giá sản phẩm trên phiếu quà tặng.

**Giá trị nghiệp vụ:** Recipient nhận tài liệu quà tặng không có giá trong khi doanh nghiệp vẫn giữ đầy đủ dữ liệu tài chính và các chứng từ bắt buộc.

```gherkin
Scenario: Bật ẩn giá
  Given checkout được đánh dấu là quà
  When khách bật tùy chọn ẩn giá
  Then hệ thống lưu `hide_price_for_recipient` trong Gift Preference
  And bản xem trước phiếu quà không hiển thị đơn giá, giảm giá hoặc tổng tiền

Scenario: Không bật ẩn giá
  Given khách mua làm quà nhưng không chọn ẩn giá
  When tạo phiếu quà
  Then hệ thống áp dụng mẫu mặc định đã công bố
  And không tự suy diễn khách muốn ẩn giá

Scenario: Dữ liệu tài chính nội bộ vẫn đầy đủ
  Given Gift Order bật ẩn giá
  When buyer, Accountant hoặc nhân viên có quyền xem Order/Payment/hóa đơn
  Then số tiền vẫn đầy đủ theo quyền và mục đích
  And tùy chọn không thay đổi phép tính hoặc báo cáo tài chính

Scenario: Packing tạo phiếu dành cho recipient
  Given Order bật ẩn giá và sẵn sàng đóng gói
  When Packing tạo phiếu quà
  Then artifact không chứa trường giá bị hạn chế
  And Packing có chỉ báo đủ rõ để không dùng nhầm phiếu bán hàng có giá

Scenario: Tài liệu bắt buộc phải có giá
  Given pháp luật, thuế, hải quan, COD hoặc logistics yêu cầu hiển thị khoản tiền
  When hệ thống chuẩn bị tài liệu bắt buộc
  Then không che thông tin bắt buộc
  And thông báo cho buyer về giới hạn của tùy chọn trước khi xác nhận nếu có thể

Scenario: Thông báo cho recipient
  Given recipient được cấu hình nhận thông báo
  When EPIC 28 tạo nội dung dành cho recipient
  Then nội dung tuân thủ hide-price policy
  And không lộ Payment hoặc giá qua tiêu đề, preview hay attachment

Scenario: Thông báo/xác nhận cho buyer
  Given buyer cần xác nhận giao dịch
  When gửi Order confirmation hoặc chứng từ cho buyer
  Then hệ thống có thể hiển thị đầy đủ số tiền theo quyền
  And không gửi nhầm bản buyer cho recipient

Scenario: Thay đổi tùy chọn sau điểm khóa
  Given artifact đã được tạo hoặc kiện hàng đã qua cutoff
  When buyer đổi tùy chọn ẩn giá
  Then hệ thống không khẳng định tài liệu vật lý đã thay đổi
  And yêu cầu rework/hỗ trợ theo chính sách nếu còn khả thi
```

## 5. Functional Requirements

| Mã | Yêu cầu | Truy vết |
| --- | --- | --- |
| `FR-GIFT-01` | Cho Guest và Customer đánh dấu checkout/Order là quà. | `US-GIFT-01`, `BR-AUTH-01` |
| `FR-GIFT-02` | Thu thập, kiểm tra và hiển thị riêng buyer với recipient. | `US-GIFT-01` |
| `FR-GIFT-03` | Lưu buyer, recipient và Gift Preference thành snapshot trên đúng Order. | `US-GIFT-01`, `EPIC 08` |
| `FR-GIFT-04` | Không biến recipient thành Customer/chủ Order nếu chưa có quy trình và consent hợp lệ. | `US-GIFT-01`, `NFR-08` |
| `FR-GIFT-05` | Cho sửa recipient trước cutoff; sau cutoff chuyển hỗ trợ/rework thay vì sửa Shipment âm thầm. | `US-GIFT-01` |
| `FR-GIFT-06` | Kiểm soát riêng quyền buyer, recipient, Packing và Delivery đối với dữ liệu Gift Order. | `US-GIFT-01`, `NFR-05` |
| `FR-GIFT-07` | Đánh dấu quà không bỏ qua giá, tồn, Payment hoặc Order lifecycle. | `US-GIFT-01`, `BR-ORDER-01`–`03` |
| `FR-GIFT-08` | Cho thêm lời chúc tùy chọn với giới hạn được công bố. | `US-GIFT-02` |
| `FR-GIFT-09` | Hiển thị preview và không cắt/thay nội dung âm thầm. | `US-GIFT-02` |
| `FR-GIFT-10` | Kiểm tra, lưu và render lời chúc an toàn; không thực thi markup/mã nguy hiểm. | `US-GIFT-02`, `NFR-04` |
| `FR-GIFT-11` | Version hóa lời chúc và chỉ dẫn khi thay đổi trước cutoff. | `US-GIFT-02` |
| `FR-GIFT-12` | Chuyển đúng phiên bản lời chúc tới Packing và chống trộn Order. | `US-GIFT-02`, `EPIC 10` |
| `FR-GIFT-13` | Theo dõi kết quả tạo/gắn phiếu hoặc thiệp; không báo hoàn tất khi thất bại. | `US-GIFT-02`, `NFR-11` |
| `FR-GIFT-14` | Lưu tùy chọn ẩn giá dành cho recipient trên Gift Preference. | `US-GIFT-03` |
| `FR-GIFT-15` | Loại đơn giá, giảm giá và tổng tiền khỏi phiếu quà/bản recipient khi bật ẩn giá. | `US-GIFT-03`, `FR-28` |
| `FR-GIFT-16` | Không sửa hoặc xóa dữ liệu giá trong Order, Payment, Finance, Invoice hay Audit. | `US-GIFT-03` |
| `FR-GIFT-17` | Không che khoản tiền bắt buộc trên chứng từ pháp lý, thuế, COD hoặc logistics. | `US-GIFT-03` |
| `FR-GIFT-18` | Tách nội dung/tài liệu buyer và recipient để tránh gửi nhầm. | `US-GIFT-03`, `EPIC 28` |
| `FR-GIFT-19` | Cho Packing nhận biết rõ Order cần phiếu không giá và artifact đã tạo đúng mẫu. | `US-GIFT-03`, `EPIC 10` |
| `FR-GIFT-20` | Áp dụng cutoff/rework khi đổi lời chúc, recipient hoặc ẩn giá sau khi xử lý vật lý bắt đầu. | `US-GIFT-01`–`03` |
| `FR-GIFT-21` | Bảo vệ và tối thiểu hóa PII buyer/recipient trên UI, log, tài liệu và tích hợp. | `US-GIFT-01`–`03`, `NFR-08` |
| `FR-GIFT-22` | Audit thay đổi sau tạo Order, override và truy cập dữ liệu nhạy cảm theo catalog. | `US-GIFT-01`–`03`, `EPIC 23` |
| `FR-GIFT-23` | Các thao tác lặp/retry không tạo trùng Gift Preference hoặc artifact. | `US-GIFT-01`–`03` |
| `FR-GIFT-24` | Theo dõi lỗi truyền chỉ dẫn, tạo artifact và notification mà không dùng telemetry thay Audit. | `US-GIFT-01`–`03`, `NFR-11` |

## 6. Mô hình dữ liệu và vòng đời

```text
Gift Preference
  order_id + is_gift
  buyer_reference + recipient_snapshot
  message + message_version
  hide_price_for_recipient
  status + locked_at + created_at + updated_at

Gift Artifact
  order_id + preference_version + artifact_type
  audience + price_visibility + status + reference

Preference: DRAFT -> CONFIRMED -> LOCKED
Artifact: REQUESTED -> GENERATED -> ATTACHED | FAILED | VOIDED
```

- Một Order có đúng một Gift Preference hiện hành; lịch sử phiên bản được giữ khi cần truy vết.
- Gift Preference chỉ có hiệu lực với đúng Order, không tự áp dụng cho Order sau.
- Buyer/recipient snapshot không tự thay đổi khi Profile/Address Book thay đổi.
- `hide_price_for_recipient` là audience policy, không phải thao tác xóa giá.
- Artifact phải tham chiếu đúng `order_id` và `preference_version`; artifact cũ được vô hiệu hóa nếu rework hợp lệ.
- Chi tiết lưu trữ, render/in ấn và truyền sự kiện thuộc Technical Design.

## 7. Traceability

| User Story | FR chính | Epic liên quan |
| --- | --- | --- |
| `US-GIFT-01` | `FR-GIFT-01`–`07`, `20`–`24` | EPIC 01/02/06/08/11/14 |
| `US-GIFT-02` | `FR-GIFT-08`–`13`, `20`–`24` | EPIC 06/08/10/23 |
| `US-GIFT-03` | `FR-GIFT-14`–`24` | EPIC 07/08/10/24/28 |

`FR-28 — Gifting Experience` trong Functional Requirements cấp cao được chi tiết hóa bởi `FR-GIFT-01` đến `FR-GIFT-24`.

## 8. Quyết định còn mở

- Một Order hỗ trợ một hay nhiều recipient; nếu nhiều, cách tách Shipment, phí và Gift Preference.
- Gift Order có cho COD không; nếu có, ai trả và cách thông báo giới hạn ẩn giá.
- Cutoff cụ thể để sửa recipient, lời chúc, ẩn giá và quy trình rework/chi phí phát sinh.
- Mẫu phiếu/thiệp, ngôn ngữ, bộ ký tự, giới hạn lời chúc và chính sách nội dung bị cấm.
- Có hiển thị danh tính người tặng không; cho phép ẩn danh hay dùng tên tùy chọn ở mức nào.
- Recipient được nhận loại thông báo/tracking nào và cơ chế consent/xác minh.
- Recipient có thể từ chối nhận, yêu cầu Return hay đổi địa chỉ bằng quy trình nào.
- Tài liệu nào thuộc recipient-facing, buyer-facing, nội bộ, thuế, COD và logistics.
- Retention/masking cho PII recipient, lời chúc và artifact sau khi Order hoàn tất.

## 9. UI/UX Reference

- Checkout: công tắc “Đây là quà tặng”, thông tin recipient, lời chúc, preview và tùy chọn ẩn giá.
- Review Order: tách rõ buyer, recipient, thanh toán, giao hàng và Gift Preference.
- Buyer Order Detail: trạng thái/chỉnh sửa trước cutoff và giải thích giới hạn sau cutoff.
- Packing Workspace: badge Gift Order, phiên bản chỉ dẫn, preview artifact và xác nhận đã gắn.
- Recipient content: tối thiểu dữ liệu, không lộ giá/buyer theo policy và có trạng thái lỗi phù hợp.
- Mọi màn hình cần loading, validation error, permission denied, locked, artifact failed và integration unavailable.

Liên kết Figma sẽ được bổ sung khi thiết kế UI/UX được phê duyệt.

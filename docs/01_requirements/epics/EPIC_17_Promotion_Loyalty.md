# EPIC 17 — Promotion & Loyalty

## 1. Mục tiêu Epic

Epic này quản lý chính sách ưu đãi và quyền lợi khách hàng trong Giai đoạn 2: tạo/công khai chương trình, kiểm tra và áp dụng coupon, cấu hình Combo, tích/đổi điểm Loyalty và hỗ trợ đặt lại sản phẩm từ đơn cũ.

EPIC 17 là nguồn quyết định điều kiện, hạn mức và giá trị ưu đãi; Checkout chỉ yêu cầu đánh giá lại tại thời điểm xác nhận, còn Order lưu snapshot kết quả. Epic không sở hữu giá SKU gốc, tồn kho, thanh toán, hoàn tiền hay gửi thông báo; các trách nhiệm đó thuộc EPIC 04, 09, 07, 14 và 28.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 2** | `US-PROMO-01` đến `US-PROMO-04`, `US-LOY-01` đến `US-LOY-03` | Khách sử dụng/xem ưu đãi, mua Combo, tích và đổi điểm, đặt lại sản phẩm; Sales Manager quản lý chương trình theo điều kiện và hạn mức rõ ràng. |

Toàn bộ Epic có mức ưu tiên **Should Have** theo Product Backlog. Story cũ `US-08` được loại bỏ vì trùng `US-PROMO-01` và `US-CHK-05`; chi tiết endpoint thuộc Technical Design, không thuộc tài liệu Epic.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và hệ thống liên quan

| Actor / hệ thống | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Xem chương trình công khai và sử dụng coupon nếu chính sách cho phép khách vãng lai. |
| Registered Customer (`ACT-02`) | Sử dụng coupon, tích/đổi điểm, xem số dư/lịch sử và đặt lại sản phẩm. |
| Sales Manager (`ACT-12`) | Tạo, cấu hình, công khai, tạm dừng và theo dõi chương trình/Combo theo quyền. |
| Customer Service (`ACT-11`) | Xem giải thích kết quả ưu đãi/sổ điểm trong phạm vi quyền để hỗ trợ, không tự điều chỉnh nếu thiếu thẩm quyền. |
| Notification Provider (`EXT-06`) | Gửi thông tin chương trình và biến động điểm qua EPIC 28 khi được yêu cầu. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-01` | Guest được mua hàng mà không bắt buộc tạo tài khoản. | Coupon không được mặc định ép đăng ký nếu chương trình cho phép Guest; Loyalty cần định danh tài khoản để giữ sổ điểm. |
| `BR-AUTH-04` | Nhân viên chỉ thực hiện thao tác thuộc quyền được cấp. | Quyền tạo, công khai, tạm dừng, điều chỉnh và xem báo cáo ưu đãi phải tách biệt. |
| `BR-PROD-02` | Giá, khối lượng, quy cách và tồn được quản lý theo SKU. | Điều kiện/Combo phải áp dụng đúng SKU; ưu đãi không thay đổi giá gốc của SKU. |
| `BR-ORDER-02` | Không xác nhận Order vượt quá tồn khả dụng. | Coupon, Combo và reorder không tạo quyền mua nếu SKU không còn đủ tồn. |
| `BR-ORDER-05` | Thay đổi trạng thái quan trọng của Order phải được ghi nhận để truy vết. | Snapshot ưu đãi và các thay đổi liên quan Order phải có khả năng đối soát. |
| `BR-OFFLINE-06` | Offline Sales chỉ bán đúng giá hoặc mức giảm trong thẩm quyền. | Nếu khuyến mãi áp dụng Offline, kết quả phải tuân theo kênh và thẩm quyền của EPIC 13. |
| `BR-REFUND-03` | Refund về phương thức gốc hoặc đổi thành Loyalty khi khách đồng ý. | Khoản điểm từ Refund phải có nguồn riêng và chỉ ghi nhận sau quyết định hợp lệ của EPIC 14. |
| `BR-AUDIT-01` | Thao tác quan trọng phải được ghi Audit Log. | Công khai/tạm dừng chương trình, điều chỉnh hạn mức và sổ điểm phải truy vết được. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 04 — Product, Variant & SKU`: cung cấp SKU và giá gốc/giá theo kênh; EPIC 17 tính lợi ích khuyến mãi tách biệt.
- `EPIC 05, 06 — Cart & Checkout`: hiển thị ước tính và yêu cầu EPIC 17 kiểm tra lại coupon/đổi điểm trước khi tạo Order.
- `EPIC 07, 08 — Payment & Order`: cung cấp kết quả thanh toán/trạng thái Order để chốt hoặc hoàn tác quyền lợi; Order lưu snapshot thương mại.
- `EPIC 09 — Inventory`: quyết định tồn khả dụng cho Combo và reorder; Promotion không giữ hoặc trừ tồn trực tiếp.
- `EPIC 12, 13, 18 — Marketplace, Offline, B2B`: dùng chính sách theo kênh khi được cấu hình; tích hợp và thẩm quyền bán thuộc Epic tương ứng.
- `EPIC 14 — Return / Refund / Complaint`: quyết định hoàn/đảo điểm sau hủy, trả hàng hoặc Refund; chuyển Refund sang điểm chỉ khi khách đồng ý.
- `EPIC 21 — Marketing`: xây dựng/đánh giá chiến dịch; EPIC 17 cung cấp cơ chế ưu đãi và dữ liệu sử dụng, không quản lý toàn bộ campaign.
- `EPIC 28 — Notification`: gửi thông báo khuyến mãi/điểm; EPIC 17 chỉ phát sinh sự kiện nghiệp vụ.

## 4. User Stories chi tiết

### US-PROMO-01 — Sử dụng coupon hợp lệ

**Actor:** Guest Customer, Registered Customer
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn sử dụng coupon hợp lệ để nhận ưu đãi.

**Giá trị nghiệp vụ:** Khách nhận đúng quyền lợi và biết rõ lý do khi mã không áp dụng, trong khi doanh nghiệp kiểm soát thời hạn, phạm vi và lượt sử dụng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Áp dụng coupon hợp lệ
  Given giỏ/checkout đáp ứng mọi điều kiện của coupon đang có hiệu lực
  When khách nhập và áp dụng mã
  Then hệ thống trả về giá trị ưu đãi cùng phạm vi dòng hàng/phí được giảm
  And Checkout hiển thị tổng tiền mới và điều kiện đã áp dụng

Scenario: Coupon không tồn tại hoặc chưa đến kỳ
  Given mã không tồn tại, chưa bắt đầu hoặc đã kết thúc
  When khách áp dụng mã
  Then hệ thống không áp dụng ưu đãi
  And trả về lý do phù hợp mà không tiết lộ cấu hình nội bộ nhạy cảm

Scenario: Không đạt điều kiện đơn hàng
  Given coupon yêu cầu giá trị tối thiểu, SKU, danh mục, kênh hoặc nhóm khách cụ thể
  When checkout không đáp ứng một hay nhiều điều kiện
  Then hệ thống từ chối áp dụng
  And chỉ rõ điều kiện khách có thể hiểu và điều chỉnh nếu phù hợp

Scenario: Coupon hết lượt tổng hoặc lượt cá nhân
  Given coupon đã đạt hạn mức chương trình hoặc khách đã đạt hạn mức cá nhân
  When khách yêu cầu áp dụng
  Then hệ thống từ chối giữ thêm lượt sử dụng
  And không làm thay đổi hạn mức đã ghi nhận

Scenario: Điều kiện thay đổi trước khi xác nhận Order
  Given coupon đã được hiển thị là hợp lệ trong checkout
  When giỏ, địa chỉ, phí giao hàng, tài khoản hoặc trạng thái chương trình thay đổi
  Then hệ thống đánh giá lại coupon trên dữ liệu mới nhất
  And không cho tạo Order với giá trị ưu đãi cũ nếu không còn hợp lệ

Scenario: Hai checkout tranh lượt coupon cuối
  Given coupon chỉ còn một lượt có thể chốt
  When hai checkout đủ điều kiện xác nhận gần đồng thời
  Then hệ thống chỉ chốt tối đa một lượt hợp lệ
  And checkout còn lại nhận kết quả hết lượt và được tính lại tổng tiền

Scenario: Order thất bại sau khi tạm giữ lượt
  Given lượt coupon đã được giữ cho một checkout nhưng Order không được tạo hoặc hết thời hạn
  When hệ thống xử lý kết quả thất bại
  Then lượt tạm giữ được giải phóng theo chính sách
  And việc gửi lặp không giải phóng hoặc tiêu thụ nhiều lần
```

### US-PROMO-02 — Tạo coupon theo điều kiện

**Actor:** Sales Manager (`ACT-12`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Sales Manager, tôi muốn tạo coupon theo điều kiện để triển khai chương trình bán hàng.

**Giá trị nghiệp vụ:** Doanh nghiệp chủ động thiết kế ưu đãi có thời hạn, ngân sách và phạm vi rõ ràng mà không sửa giá SKU gốc.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo coupon hợp lệ
  Given Sales Manager có quyền quản lý coupon
  When nhập mã duy nhất, loại ưu đãi, giá trị, thời gian, phạm vi và hạn mức hợp lệ
  Then hệ thống tạo coupon ở trạng thái chưa công khai
  And không cho khách sử dụng trước khi chương trình đủ điều kiện hoạt động

Scenario: Mã coupon bị trùng
  Given đã có coupon dùng cùng mã theo quy tắc duy nhất
  When Sales Manager tạo coupon mới với mã đó
  Then hệ thống từ chối lưu
  And không tạo hai chính sách có thể bị hiểu là cùng một mã

Scenario: Cấu hình giá trị giảm không hợp lệ
  Given Sales Manager cấu hình phần trăm, số tiền, trần giảm hoặc giá trị đơn tối thiểu
  When giá trị âm, bằng không hoặc mâu thuẫn với loại ưu đãi
  Then hệ thống từ chối lưu/công khai
  And chỉ rõ trường cần sửa

Scenario: Công khai coupon đủ điều kiện
  Given coupon có cấu hình đầy đủ và Sales Manager có quyền công khai
  When Sales Manager xác nhận lịch hoạt động
  Then coupon chuyển sang trạng thái phù hợp với thời điểm hiệu lực
  And quyết định được ghi nhận để truy vết

Scenario: Sửa coupon đã có lượt sử dụng
  Given coupon đã được chốt trên ít nhất một Order
  When Sales Manager sửa điều kiện hoặc giá trị
  Then thay đổi không sửa hồi tố snapshot Order đã phát sinh
  And hệ thống yêu cầu phiên bản/thời điểm hiệu lực mới theo chính sách

Scenario: Tạm dừng coupon đang hoạt động
  Given coupon đang hoạt động và Sales Manager có quyền
  When Sales Manager tạm dừng kèm lý do
  Then yêu cầu áp dụng mới bị từ chối
  And Order đã chốt trước đó vẫn giữ snapshot ưu đãi

Scenario: Nhân viên không có quyền
  Given nhân viên thiếu quyền tạo hoặc công khai coupon
  When nhân viên thực hiện thao tác tương ứng
  Then hệ thống từ chối
  And không thay đổi chương trình
```

### US-PROMO-03 — Tạo Combo sản phẩm

**Actor:** Sales Manager (`ACT-12`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Sales Manager, tôi muốn tạo Combo sản phẩm để tăng giá trị đơn hàng.

**Giá trị nghiệp vụ:** Nhóm nhiều SKU thành một đề nghị mua hấp dẫn, đồng thời giữ chính xác thành phần, giá trị ưu đãi và tồn kho từng SKU.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo Combo hợp lệ
  Given các SKU thành phần tồn tại và Sales Manager có quyền
  When nhập tên, thành phần/số lượng, giá hoặc ưu đãi Combo và thời gian hiệu lực hợp lệ
  Then hệ thống tạo Combo ở trạng thái chưa công khai
  And lưu đúng từng SKU cùng số lượng bắt buộc

Scenario: Thành phần Combo không hợp lệ
  Given một SKU không tồn tại, ngừng bán hoặc số lượng thành phần không hợp lệ
  When Sales Manager lưu hoặc công khai Combo
  Then hệ thống từ chối trạng thái có thể bán
  And chỉ rõ thành phần cần xử lý

Scenario: Khách thêm Combo còn đủ tồn
  Given Combo đang hoạt động và tất cả SKU thành phần còn khả dụng
  When khách thêm một Combo vào giỏ
  Then giỏ nhận đúng SKU và số lượng thành phần
  And thể hiện riêng lợi ích/giá Combo theo chính sách

Scenario: Một thành phần không đủ tồn
  Given ít nhất một SKU thành phần không đủ số lượng
  When khách thêm hoặc xác nhận mua Combo
  Then hệ thống không xác nhận toàn bộ Combo như còn hàng
  And không tự bỏ thành phần hoặc thay SKU khác nếu chưa có chính sách

Scenario: Giá SKU thay đổi khi Combo đang hoạt động
  Given giá nguồn của một SKU thành phần thay đổi
  When Combo được xem hoặc checkout đánh giá lại
  Then hệ thống tính theo quy tắc giá Combo hiện hành
  And cảnh báo/ngăn bán nếu cấu hình trở nên không hợp lệ theo chính sách biên lợi nhuận

Scenario: Cập nhật Combo không làm đổi Order cũ
  Given Combo đã xuất hiện trong Order được tạo
  When Sales Manager thay thành phần, giá hoặc dừng Combo
  Then snapshot SKU/số lượng/ưu đãi trên Order cũ không thay đổi
  And yêu cầu mua mới dùng cấu hình hiện hành
```

### US-PROMO-04 — Xem chương trình khuyến mãi đang áp dụng

**Actor:** Guest Customer, Registered Customer
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn xem các chương trình khuyến mãi đang áp dụng.

**Giá trị nghiệp vụ:** Khách hiểu rõ chương trình, thời hạn và điều kiện trước khi mua, giảm kỳ vọng sai về quyền lợi.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem danh sách chương trình đang hoạt động
  Given có chương trình được phép công khai và đang trong thời gian hiệu lực
  When khách mở khu vực khuyến mãi
  Then hệ thống hiển thị tên, lợi ích, thời gian và điều kiện chính
  And chỉ hiển thị chương trình phù hợp kênh/đối tượng công khai

Scenario: Chương trình chưa bắt đầu hoặc đã kết thúc
  Given chương trình nằm ngoài thời gian hiệu lực
  When khách xem danh sách đang áp dụng
  Then chương trình không được trình bày như đang sử dụng được
  And có thể hiển thị sắp diễn ra/đã kết thúc nếu chính sách cho phép và ghi nhãn rõ

Scenario: Chương trình yêu cầu coupon hoặc đăng nhập
  Given ưu đãi chỉ áp dụng khi nhập mã hoặc thuộc nhóm khách đủ điều kiện
  When khách xem chi tiết
  Then hệ thống nêu rõ cách nhận và điều kiện
  And không cam kết khách chắc chắn đủ điều kiện trước bước kiểm tra

Scenario: Chương trình thay đổi trong lúc khách đang xem
  Given dữ liệu chương trình đã thay đổi hoặc bị tạm dừng
  When khách áp dụng hoặc đi đến checkout
  Then hệ thống dùng trạng thái hiện hành để đánh giá
  And không dựa riêng vào nội dung trang đã tải trước đó

Scenario: Không có chương trình phù hợp
  Given không có chương trình công khai phù hợp đối tượng/kênh
  When khách mở danh sách
  Then hệ thống hiển thị trạng thái không có chương trình
  And không hiển thị cấu hình nội bộ hoặc chương trình nháp
```

### US-LOY-01 — Tích điểm sau mua hàng

**Actor:** Registered Customer (`ACT-02`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn tích điểm sau mỗi lần mua hàng.

**Giá trị nghiệp vụ:** Khách nhận quyền lợi trung thành dựa trên giao dịch hợp lệ; doanh nghiệp có sổ điểm minh bạch, chống cộng trùng và có thể đảo khi Order thay đổi.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Ghi nhận điểm cho Order đủ điều kiện
  Given Registered Customer có Order đạt mốc tích điểm theo chính sách
  When hệ thống xử lý sự kiện đủ điều kiện
  Then tạo một giao dịch cộng điểm gắn với đúng khách và Order
  And số điểm được tính từ giá trị đủ điều kiện theo phiên bản chính sách áp dụng

Scenario: Không cộng điểm chỉ vì tạo hoặc thanh toán Order
  Given Order chưa đạt mốc hoàn tất được quy định
  When Order mới được tạo hoặc Payment vừa thành công
  Then hệ thống chưa đưa điểm vào số dư khả dụng nếu chính sách yêu cầu chờ hoàn tất
  And thể hiện điểm dự kiến/chờ xử lý tách biệt nếu có

Scenario: Nhận lặp sự kiện Order
  Given điểm cho mốc và Order đã được ghi nhận
  When cùng sự kiện được xử lý lại
  Then hệ thống không cộng điểm lần thứ hai
  And trả về kết quả giao dịch đã có để đối soát

Scenario: Order bị hủy hoặc trả hàng
  Given Order đã phát sinh điểm nhưng sau đó hủy/hoàn/trả làm giảm giá trị đủ điều kiện
  When EPIC 14/08 xác nhận kết quả hợp lệ
  Then hệ thống tạo giao dịch đảo/điều chỉnh theo chính sách
  And không sửa hoặc xóa giao dịch gốc khỏi sổ điểm

Scenario: Khách chưa có tài khoản Loyalty
  Given Order thuộc Guest hoặc chưa liên kết an toàn với Registered Customer
  When Order đạt mốc tích điểm
  Then hệ thống không cộng điểm vào một tài khoản suy đoán
  And áp dụng chính sách liên kết/nhận điểm sau đó nếu được hỗ trợ

Scenario: Xem số dư và lịch sử điểm
  Given khách đã đăng nhập
  When khách mở Loyalty
  Then hệ thống hiển thị số dư khả dụng, điểm đang chờ và lịch sử giao dịch thuộc chính khách
  And mỗi giao dịch có loại, số điểm, thời điểm, trạng thái và nguồn tham chiếu được phép
```

### US-LOY-02 — Đổi điểm thành ưu đãi

**Actor:** Registered Customer (`ACT-02`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn đổi điểm thành ưu đãi.

**Giá trị nghiệp vụ:** Khách sử dụng số điểm khả dụng cho quyền lợi rõ ràng, không chi vượt số dư hoặc bị trừ điểm khi Order không hoàn tất.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đổi điểm hợp lệ trong checkout
  Given khách có đủ điểm khả dụng và checkout đáp ứng điều kiện đổi
  When khách chọn số điểm/quyền lợi hợp lệ
  Then hệ thống giữ số điểm cần dùng cho checkout
  And hiển thị giá trị quy đổi cùng tổng tiền được tính lại

Scenario: Không đủ điểm khả dụng
  Given số điểm khả dụng thấp hơn yêu cầu
  When khách xác nhận đổi điểm
  Then hệ thống từ chối
  And không làm số dư trở thành âm

Scenario: Hai checkout dùng cùng số điểm
  Given khách mở nhiều checkout với cùng số dư
  When các checkout đồng thời yêu cầu giữ điểm vượt tổng khả dụng
  Then hệ thống chỉ chấp nhận tổng mức giữ không vượt số dư
  And yêu cầu còn lại nhận số dư mới để điều chỉnh

Scenario: Order được tạo thành công
  Given điểm đã được giữ và Order được tạo với snapshot quy đổi hợp lệ
  When kết quả chốt được xác nhận
  Then hệ thống chuyển giao dịch giữ thành giao dịch sử dụng đúng một lần
  And Order lưu số điểm cùng giá trị tiền/ưu đãi tương ứng

Scenario: Checkout hoặc Order không hoàn tất
  Given điểm đang được giữ nhưng checkout hết hạn, tạo Order thất bại hoặc Order bị hủy đủ điều kiện hoàn điểm
  When hệ thống nhận kết quả hợp lệ
  Then điểm được giải phóng/hoàn theo chính sách đúng một lần
  And lịch sử thể hiện quan hệ với giao dịch giữ/sử dụng ban đầu

Scenario: Điểm hết hạn hoặc chưa khả dụng
  Given điểm thuộc trạng thái đang chờ, đã hết hạn hoặc đã được giữ
  When khách yêu cầu đổi
  Then hệ thống không tính số điểm đó vào khả dụng
  And hiển thị phân loại số dư rõ ràng

Scenario: Refund được quy đổi thành điểm
  Given EPIC 14 đã phê duyệt Refund sang Loyalty và lưu sự đồng ý hợp lệ của khách
  When EPIC 17 nhận yêu cầu ghi có
  Then tạo giao dịch cộng điểm có nguồn `REFUND_CONVERSION`
  And không coi khoản đó là điểm thưởng mua hàng
  And chống ghi có trùng cho cùng nghĩa vụ Refund
```

### US-LOY-03 — Đặt lại sản phẩm từ Order cũ

**Actor:** Registered Customer (`ACT-02`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn đặt lại sản phẩm từ đơn hàng cũ để tiết kiệm thời gian.

**Giá trị nghiệp vụ:** Khách nhanh chóng tạo lại giỏ từ nhu cầu cũ nhưng vẫn mua theo SKU, giá, tồn và chính sách hiện tại.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Thêm lại toàn bộ SKU còn bán được
  Given khách sở hữu Order cũ và các SKU vẫn có thể mua
  When khách chọn đặt lại
  Then hệ thống thêm đúng SKU và số lượng phù hợp vào giỏ hiện tại
  And dùng giá cùng khuyến mãi hiện hành thay vì snapshot giá cũ

Scenario: Một SKU đã ngừng bán hoặc không tồn tại
  Given Order cũ có SKU không còn có thể mua
  When khách đặt lại
  Then hệ thống không thêm SKU đó như mặt hàng hợp lệ
  And chỉ rõ dòng bị bỏ qua hoặc cần lựa chọn thay thế

Scenario: Số lượng hiện tại không đủ
  Given SKU còn bán nhưng tồn khả dụng thấp hơn số lượng trong Order cũ
  When khách đặt lại
  Then hệ thống không cam kết số lượng cũ
  And hiển thị số lượng có thể chọn theo dữ liệu hiện tại nếu chính sách cho phép

Scenario: Coupon hoặc Combo cũ không còn hiệu lực
  Given Order cũ từng dùng coupon, điểm hoặc cấu hình Combo
  When khách đặt lại
  Then hệ thống không tự sao chép quyền lợi đã dùng/hết hạn
  And đánh giá giỏ mới theo chính sách hiện hành

Scenario: Giỏ hiện tại đã có sản phẩm
  Given khách có giỏ đang chứa SKU
  When khách đặt lại Order cũ
  Then hệ thống hợp nhất theo quy tắc giỏ hiện hành và giới hạn số lượng
  And không tạo dòng trùng hoặc vượt giới hạn ngoài ý muốn

Scenario: Khách cố đặt lại Order không thuộc mình
  Given Order không thuộc tài khoản khách
  When khách yêu cầu đặt lại bằng mã hoặc liên kết trực tiếp
  Then hệ thống từ chối
  And không tiết lộ SKU, giá hoặc thông tin Order
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-PL-01` | Đánh giá coupon theo mã, trạng thái, thời gian, kênh, khách, SKU/danh mục, giá trị đơn, phí và các điều kiện cấu hình. | `US-PROMO-01` |
| `FR-PL-02` | Trả về giá trị/phạm vi ưu đãi và lý do không hợp lệ có thể hiểu được mà không lộ cấu hình nhạy cảm. | `US-PROMO-01`, `US-PROMO-04` |
| `FR-PL-03` | Kiểm tra lại coupon khi đầu vào thay đổi và ngay trước khi tạo Order; không tin kết quả cũ do client gửi. | `US-PROMO-01`, `US-CHK-05` |
| `FR-PL-04` | Giữ, chốt và giải phóng lượt coupon nhất quán; ngăn vượt hạn mức khi xử lý đồng thời hoặc gửi lặp. | `US-PROMO-01` |
| `FR-PL-05` | Cho Sales Manager có quyền tạo coupon với mã duy nhất, loại/giá trị giảm, thời gian, phạm vi, điều kiện và hạn mức hợp lệ. | `US-PROMO-02`, `BR-AUTH-04` |
| `FR-PL-06` | Quản lý phiên bản/hiệu lực khi sửa coupon đã sử dụng; không thay đổi hồi tố snapshot Order. | `US-PROMO-02`, `BR-ORDER-05` |
| `FR-PL-07` | Cho công khai, lên lịch, tạm dừng và kết thúc coupon theo quyền; lưu lý do/thời điểm thay đổi quan trọng. | `US-PROMO-02`, `BR-AUDIT-01` |
| `FR-PL-08` | Cho tạo Combo từ đúng SKU/số lượng và cấu hình giá/ưu đãi/thời gian hợp lệ mà không sửa giá SKU gốc. | `US-PROMO-03`, `BR-PROD-02` |
| `FR-PL-09` | Đánh giá khả năng bán của toàn bộ thành phần Combo; không tự bỏ/thay thành phần hoặc xác nhận khi thiếu tồn. | `US-PROMO-03`, `BR-ORDER-02` |
| `FR-PL-10` | Bảo toàn snapshot thành phần, số lượng và ưu đãi Combo trên Order đã tạo khi cấu hình nguồn thay đổi. | `US-PROMO-03`, `BR-ORDER-05` |
| `FR-PL-11` | Hiển thị chương trình công khai đúng trạng thái, thời gian, kênh/đối tượng cùng điều kiện chính; không lộ chương trình nháp. | `US-PROMO-04` |
| `FR-PL-12` | Ghi điểm từ Order đủ điều kiện theo phiên bản chính sách và tạo duy nhất một giao dịch cho mỗi nguồn/mốc. | `US-LOY-01` |
| `FR-PL-13` | Tách điểm đang chờ, khả dụng, đang giữ, đã dùng, hết hạn và đã đảo; không cộng điểm suy đoán cho Guest. | `US-LOY-01`, `US-LOY-02` |
| `FR-PL-14` | Duy trì sổ điểm theo giao dịch bù trừ; không sửa/xóa giao dịch gốc khi hủy, trả hàng hoặc điều chỉnh. | `US-LOY-01`, `BR-AUDIT-01` |
| `FR-PL-15` | Cho khách xem số dư và lịch sử điểm của chính mình với loại, giá trị, thời điểm, trạng thái và nguồn được phép. | `US-LOY-01` |
| `FR-PL-16` | Giữ/chốt/giải phóng điểm đổi trong checkout một cách nhất quán; ngăn số dư âm và chi tiêu đồng thời vượt khả dụng. | `US-LOY-02` |
| `FR-PL-17` | Lưu snapshot số điểm và giá trị quy đổi trên Order; hoàn/đảo đúng một lần khi kết quả Order đủ điều kiện. | `US-LOY-02`, `EPIC 08`, `EPIC 14` |
| `FR-PL-18` | Ghi điểm từ Refund với nguồn riêng chỉ sau quyết định hợp lệ và sự đồng ý của khách; chống ghi trùng. | `US-LOY-02`, `BR-REFUND-03` |
| `FR-PL-19` | Cho chủ Order cũ yêu cầu thêm lại SKU vào giỏ nhưng dùng giá, tồn và chính sách ưu đãi hiện hành. | `US-LOY-03` |
| `FR-PL-20` | Xử lý từng dòng không còn bán/thiếu tồn, hợp nhất với giỏ hiện tại và không sao chép coupon/điểm đã dùng. | `US-LOY-03`, `BR-ORDER-02` |
| `FR-PL-21` | Lưu snapshot mã, phiên bản, phạm vi, giá trị ưu đãi/điểm và căn cứ cần thiết trên Order để đối soát. | `US-PROMO-01~03`, `US-LOY-02`, `EPIC 06`, `EPIC 08` |
| `FR-PL-22` | Kiểm soát quyền tạo/công khai/tạm dừng/điều chỉnh và ghi Audit cho thay đổi chương trình hoặc sổ điểm quan trọng. | `US-PROMO-02~03`, `US-LOY-01~02`, `BR-AUTH-04`, `BR-AUDIT-01` |
| `FR-PL-23` | Phát sinh sự kiện cho thông báo chương trình/biến động điểm mà không phụ thuộc giao dịch nghiệp vụ vào kết quả gửi tin. | `US-PROMO-04`, `US-LOY-01~02`, `EPIC 28` |

## 6. Vòng đời và quy tắc dữ liệu

### 6.1. Vòng đời Promotion/Coupon/Combo

```text
DRAFT → SCHEDULED → ACTIVE → ENDED
             │          │
             └──────► PAUSED
                         │
                         └────► ACTIVE hoặc ENDED
```

- `DRAFT`: cấu hình chưa được phép áp dụng/công khai.
- `SCHEDULED`: đã đủ điều kiện công khai nhưng chưa tới thời điểm bắt đầu.
- `ACTIVE`: có thể được xem/xét áp dụng, vẫn phải kiểm tra điều kiện và hạn mức tại thời điểm giao dịch.
- `PAUSED`: ngăn áp dụng mới; không sửa hồi tố Order đã chốt.
- `ENDED`: kết thúc do hết thời gian, hết ngân sách/lượt hoặc quyết định hợp lệ; không được kích hoạt lại âm thầm trên cùng phiên bản.

### 6.2. Vòng đời điểm Loyalty

```text
PENDING → AVAILABLE → HELD → REDEEMED
    │          │         │
    └──────► REVERSED ◄──┘
               ▲
AVAILABLE ──► EXPIRED
```

- Số dư là kết quả tổng hợp từ Loyalty Ledger, không phải giá trị được sửa trực tiếp không có giao dịch nguồn.
- Mỗi giao dịch phải có mã duy nhất, khách hàng, loại, số điểm có dấu, trạng thái, nguồn Order/Refund/điều chỉnh, phiên bản chính sách và thời điểm.
- `PENDING` không được chi tiêu; `HELD` không được dùng cho checkout khác; `REDEEMED`, `EXPIRED`, `REVERSED` không thuộc số dư khả dụng.
- Điều chỉnh thủ công, nếu được phép, phải dùng giao dịch bù trừ có lý do và thẩm quyền; không thay đổi/xóa bản ghi gốc.

### 6.3. Quy tắc tính tiền và snapshot

- Mọi giá trị tiền/điểm dùng đơn vị và độ chính xác thống nhất; quy tắc làm tròn phải được chốt trước khi triển khai.
- Phải xác định thứ tự áp dụng giữa giá theo kênh, Combo, coupon, điểm, phí vận chuyển và các giảm giá khác. Không cộng dồn ngầm định.
- Giá trị giảm không được làm số tiền của phạm vi áp dụng nhỏ hơn 0; phần vượt trần/giá trị dòng hàng không được chuyển sang phạm vi khác nếu chính sách không cho phép.
- Client chỉ gửi lựa chọn/mã; hệ thống tự tính lại trên dữ liệu đáng tin cậy tại thời điểm chốt.
- Order phải giữ snapshot đủ để giải thích tổng tiền ngay cả khi chương trình, giá, tài khoản Loyalty hoặc catalog thay đổi sau đó.

### 6.4. Hạn mức, đồng thời và hoàn tác

- Hạn mức toàn chương trình, theo khách, theo Order và theo kỳ phải được kiểm tra nguyên tử tại bước chốt.
- Reservation coupon/điểm cần thời hạn và quan hệ với checkout; hết hạn hoặc thất bại phải giải phóng đúng một lần.
- Hủy/return/refund không tự suy ra hoàn coupon hoặc điểm. EPIC 14/08 cung cấp kết quả, EPIC 17 áp dụng chính sách hoàn/đảo tương ứng.
- Thay đổi chương trình phải có hiệu lực theo thời điểm/phiên bản, không làm thay đổi quyền lợi đã chốt hợp lệ trên Order cũ.
- Dữ liệu hiển thị có thể là ước tính; quyết định cuối cùng luôn dựa trên lần kiểm tra tại Checkout/Order.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-PROMO-01` | `BR-AUTH-01`, `BR-PROD-02`, `BR-ORDER-02` | EPIC 04, 06, 08 | Coupon đúng điều kiện/hạn mức; kiểm tra lại khi chốt; tranh lượt/gửi lặp không vượt hạn mức. |
| `US-PROMO-02` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 04, 06, 08, 23 | Chỉ Manager có quyền quản lý; cấu hình hợp lệ, có phiên bản và không sửa hồi tố Order. |
| `US-PROMO-03` | `BR-PROD-02`, `BR-ORDER-02`, `BR-ORDER-05` | EPIC 04, 06, 09 | Combo đúng SKU/số lượng; thiếu một thành phần không xác nhận; Order cũ giữ snapshot. |
| `US-PROMO-04` | `BR-AUTH-01` | EPIC 03, 21, 28 | Chỉ chương trình công khai/đúng kỳ/kênh được hiển thị; điều kiện không gây hiểu nhầm. |
| `US-LOY-01` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 07, 08, 14, 23 | Tích điểm đúng Order/mốc/chính sách, không trùng; hủy/trả dùng giao dịch đảo và giữ lịch sử. |
| `US-LOY-02` | `BR-REFUND-03`, `BR-AUDIT-01` | EPIC 06, 08, 14 | Không chi vượt số dư; giữ/chốt/hoàn điểm đúng một lần; Refund-to-Loyalty cần đồng ý và nguồn riêng. |
| `US-LOY-03` | `BR-PROD-02`, `BR-ORDER-02` | EPIC 04, 05, 09 | Chỉ chủ Order đặt lại; SKU/giá/tồn hiện tại được kiểm tra; không sao chép ưu đãi cũ. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Guest có được dùng coupon không; cách áp hạn mức cá nhân khi chưa có tài khoản và chống lạm dụng theo chính sách riêng tư.
- Các loại ưu đãi hỗ trợ: giảm phần trăm, số tiền, phí vận chuyển, quà tặng; trần giảm và giá trị tối thiểu.
- Thứ tự/khả năng cộng dồn giữa giá kênh, Combo, coupon, Loyalty, B2B và khuyến mãi Marketplace/Offline.
- Phạm vi coupon: Product/SKU/danh mục/kênh/khu vực/nhóm khách; quy tắc bao gồm và loại trừ khi giao nhau.
- Hạn mức tổng, theo khách, Order, ngày; thời điểm giữ/chốt lượt và TTL reservation.
- Quy trình duyệt chương trình, phân tách người tạo/người công khai, ngưỡng ngân sách và điều kiện tự động tạm dừng.
- Combo là cấu trúc khuyến mãi hay SKU bán độc lập; quy tắc giá, thuế, vận chuyển, return từng thành phần và thay thế SKU.
- Mốc tích điểm (`PAID`, `DELIVERED` hay hết cửa sổ return), tỷ lệ/quy tắc làm tròn, trần điểm và sản phẩm/giá trị bị loại trừ.
- Thời hạn điểm, thứ tự tiêu điểm sắp hết hạn, thời gian `PENDING`, chính sách thông báo và xử lý timezone.
- Chính sách hoàn coupon/điểm khi hủy, trả một phần/toàn phần, thanh toán thất bại hoặc chỉnh sửa Order.
- Tỷ lệ Refund-to-Loyalty, bằng chứng đồng ý, khả năng rút lại và cách đối soát giá trị tiền với điểm.
- Quyền điều chỉnh điểm thủ công, phê duyệt nhiều cấp, giới hạn và báo cáo phát hiện bất thường/lạm dụng.
- Reorder có giữ số lượng cũ khi thiếu tồn, đề xuất SKU thay thế ra sao và xử lý Combo đã thay đổi thế nào.
- Dữ liệu khuyến mãi áp dụng theo từng kênh và nguồn sự thật khi Marketplace có promotion riêng.

## 9. UI/UX Reference

- Form coupon cho khách cần hiển thị mã, kết quả, phạm vi giảm và lý do không hợp lệ; tổng tiền phải được cập nhật rõ.
- Màn hình Sales Manager cần tách cấu hình, điều kiện, hạn mức, lịch hiệu lực và preview; cảnh báo tác động khi sửa chương trình đã dùng.
- Combo phải hiển thị từng SKU/số lượng, giá gốc/giá Combo hoặc lợi ích, trạng thái thiếu thành phần và điều kiện áp dụng.
- Trang khuyến mãi công khai cần phân biệt đang diễn ra, sắp diễn ra và kết thúc; nêu rõ coupon/đăng nhập/kênh nếu cần.
- Loyalty Wallet cần tách điểm khả dụng, đang chờ, đang giữ và sắp hết hạn; lịch sử thể hiện nguồn và giao dịch đảo.
- Checkout phải hiển thị thứ tự các ưu đãi, điểm đã dùng, tổng mới và cảnh báo khi kết quả hết hiệu lực trước lúc xác nhận.
- Reorder preview cần liệt kê dòng thêm được, dòng bị bỏ qua/thay đổi, giá hiện tại và thông báo không giữ ưu đãi cũ trước khi cập nhật giỏ.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

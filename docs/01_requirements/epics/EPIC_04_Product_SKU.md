# EPIC 04 — Product, Variant & SKU

## 1. Mục tiêu Epic

Epic này là nguồn dữ liệu thương mại trung tâm cho catalog: sản phẩm, Variant/SKU, giá, thông tin thực phẩm và trạng thái bán. Sales Manager có thể chuẩn bị dữ liệu chính xác để khách khám phá, chọn đúng quy cách và mua đúng giá trên từng kênh.

Epic không quản lý số lượng nhập/xuất, Batch/Lot, hạn sử dụng thực tế của từng lô, giữ tồn, đơn hàng, giỏ hàng hay đồng bộ marketplace. Các nội dung đó thuộc EPIC 09, 05–08, 10–13. Epic cũng không thay thế quy trình quản lý nội dung/SEO thuộc EPIC 20 hoặc truy xuất nguồn gốc OCOP thuộc EPIC 19.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP v1.0** | `US-PROD-01` đến `US-PROD-06` | Doanh nghiệp quản lý được sản phẩm/SKU, giá, thông tin thực phẩm và trạng thái bán; khách chọn được đúng SKU. |
| **Giai đoạn 2** | `US-PROD-07` | Doanh nghiệp quản lý chính sách giá theo Website, Marketplace, Offline và B2B. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Sales Manager (`ACT-12`) | Tạo/cập nhật sản phẩm, quản lý SKU, giá và trạng thái bán theo quyền được cấp. |
| Guest Customer (`ACT-01`) | Xem thông tin công khai và chọn SKU trước khi mua. |
| Registered Customer (`ACT-02`) | Có khả năng xem/chọn SKU như Guest và tiếp tục mua hàng qua các Epic sau. |
| Marketplace Operator (`ACT-09`) | Sử dụng dữ liệu giá/kênh được phê duyệt khi vận hành marketplace; đồng bộ thực tế thuộc EPIC 12. |
| Offline Sales Staff (`ACT-10`) | Áp dụng giá Offline hợp lệ; nghiệp vụ bán offline thuộc EPIC 13. |
| B2B Customer (`ACT-03`) | Nhận giá/chính sách B2B hợp lệ ở các luồng mua sỉ của EPIC 18. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-PROD-01` | Mỗi sản phẩm có thể có nhiều Variant/SKU. | Phải quản lý SKU độc lập dưới sản phẩm tương ứng. |
| `BR-PROD-02` | Giá, khối lượng, quy cách đóng gói và tồn kho được quản lý theo SKU. | Giá/thông tin quy cách hiển thị và khả năng mua phải xác định ở SKU được chọn. |
| `BR-PROD-03` | Sản phẩm thực phẩm phải có thông tin ngày sản xuất và hạn sử dụng phù hợp. | Thông tin thực phẩm cần được quản lý và công khai chính xác theo chính sách. |
| `BR-PROD-04` | Thông tin sản phẩm hiển thị cho khách phải được quản lý và phê duyệt trước khi công khai. | Tạo/cập nhật không đồng nghĩa tự động được hiển thị công khai cho khách. |
| `BR-BATCH-05` | Sản phẩm hết hạn không được tiếp tục bán. | Trạng thái bán của catalog không được làm hàng hết hạn trở thành lựa chọn có thể mua; EPIC 09 kiểm soát Batch thực tế. |
| `BR-ORDER-02` | Không xác nhận đơn vượt quá tồn kho khả dụng. | SKU được chọn phải được đối chiếu tồn khả dụng ở Cart/Checkout, không chỉ dựa vào trạng thái sản phẩm. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Thay đổi giá, SKU và trạng thái bán cần có khả năng truy vết theo EPIC 23. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 03 — Product Discovery`: hiển thị sản phẩm/SKU, giá và trạng thái đã được phép công khai.
- `EPIC 05 — Shopping Cart` và `EPIC 06 — Checkout`: nhận SKU cụ thể khách chọn, dùng giá phù hợp và kiểm tra khả năng bán.
- `EPIC 09 — Inventory & Batch`: quản lý tồn kho, Batch/Lot, ngày sản xuất/hạn sử dụng thực tế và chặn hàng hết hạn.
- `EPIC 12 — Marketplace`, `EPIC 13 — Offline Sales`, `EPIC 18 — B2B Sales`: dùng chính sách giá theo kênh nhưng quản lý đơn/kết nối từng kênh trong Epic của chúng.
- `EPIC 19 — OCOP Traceability`: quản lý dữ liệu nguồn gốc/chứng nhận, không thay bằng thuộc tính catalog.
- `EPIC 20 — Content & SEO`: quản lý nội dung marketing, bài viết và SEO; Epic 04 chỉ cung cấp dữ liệu sản phẩm cơ bản.
- `EPIC 22 — User / Role / Permission Administration` và `EPIC 23 — Audit & Security`: kiểm soát quyền Sales Manager và lưu dấu vết thay đổi quan trọng.

## 4. User Stories chi tiết

### US-PROD-01 — Tạo và cập nhật sản phẩm

**Actor:** Sales Manager (`ACT-12`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là Sales Manager, tôi muốn tạo và cập nhật sản phẩm để đưa sản phẩm lên hệ thống bán hàng.

**Giá trị nghiệp vụ:** Doanh nghiệp chủ động đưa các dòng mè xửng/đặc sản mới lên catalog và bảo đảm thông tin bán hàng luôn chính xác.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo sản phẩm với thông tin hợp lệ
  Given Sales Manager có quyền quản lý catalog
  When Sales Manager nhập đầy đủ thông tin sản phẩm bắt buộc và lưu
  Then hệ thống tạo một sản phẩm mới
  And sản phẩm sẵn sàng để được cấu hình SKU, giá và trạng thái bán

Scenario: Cập nhật sản phẩm hiện có
  Given một sản phẩm đã tồn tại
  When Sales Manager cập nhật thông tin hợp lệ và lưu thay đổi
  Then hệ thống lưu thông tin mới cho đúng sản phẩm
  And việc hiển thị công khai tuân theo quy trình quản lý/phê duyệt sản phẩm

Scenario: Từ chối dữ liệu sản phẩm không hợp lệ
  Given Sales Manager bỏ trống trường bắt buộc hoặc nhập thông tin không hợp lệ
  When Sales Manager lưu sản phẩm
  Then hệ thống chỉ rõ thông tin cần chỉnh sửa
  And không tạo hoặc cập nhật dữ liệu không hợp lệ
```

### US-PROD-02 — Quản lý Variant/SKU

**Actor:** Sales Manager (`ACT-12`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là Sales Manager, tôi muốn quản lý Variant/SKU theo khối lượng, hương vị và quy cách đóng gói.

**Giá trị nghiệp vụ:** Một sản phẩm có thể được bán đúng theo các lựa chọn thực tế như hộp 250g/500g, hương vị hoặc quy cách đóng gói khác nhau.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Thêm nhiều SKU cho một sản phẩm
  Given Sales Manager đang quản lý một sản phẩm
  When Sales Manager thêm các Variant/SKU hợp lệ với khối lượng, hương vị hoặc quy cách đóng gói tương ứng
  Then hệ thống liên kết các SKU với đúng sản phẩm
  And mỗi SKU có thể được quản lý giá và khả năng bán độc lập

Scenario: Cập nhật thông tin SKU
  Given SKU đã tồn tại dưới một sản phẩm
  When Sales Manager cập nhật thuộc tính hợp lệ của SKU và lưu
  Then hệ thống lưu thông tin mới cho đúng SKU
  And thông tin SKU hiển thị cho khách được cập nhật theo quy trình công khai đã phê duyệt

Scenario: Ngăn SKU bị nhầm lẫn hoặc trùng lặp
  Given Sales Manager thêm hoặc cập nhật SKU
  When mã/thuộc tính nhận diện SKU trùng hoặc không hợp lệ theo chính sách catalog
  Then hệ thống từ chối lưu dữ liệu gây trùng lặp hoặc nhầm lẫn
  And hiển thị lý do để Sales Manager điều chỉnh
```

### US-PROD-03 — Chọn Variant/SKU trước khi mua

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn chọn Variant/SKU trước khi mua để nhận đúng sản phẩm.

**Giá trị nghiệp vụ:** Hạn chế đơn hàng sai quy cách và bảo đảm khách biết rõ sản phẩm cụ thể sẽ được đưa vào giỏ hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Chọn SKU hợp lệ
  Given sản phẩm có từ hai SKU đang được phép bán trở lên
  When khách chọn một SKU
  Then hệ thống hiển thị đúng khối lượng, hương vị, quy cách và giá của SKU đó
  And SKU được chọn là SKU được chuyển sang giỏ hàng khi khách quyết định mua

Scenario: Yêu cầu chọn SKU khi cần thiết
  Given sản phẩm có nhiều SKU và khách chưa chọn SKU
  When khách thực hiện hành động mua
  Then hệ thống yêu cầu khách chọn một SKU hợp lệ trước khi tiếp tục
  And không tự thêm một SKU không rõ lựa chọn vào giỏ hàng

Scenario: SKU không còn có thể mua
  Given một SKU đang tạm ngừng bán hoặc không còn khả năng bán theo dữ liệu tồn kho/hạn sử dụng
  When khách xem hoặc cố chọn SKU đó
  Then hệ thống không cho khách hoàn tất mua SKU đó
  And hiển thị trạng thái phù hợp để khách chọn SKU khác nếu có
```

### US-PROD-04 — Xem thông tin thực phẩm của sản phẩm

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn xem thành phần, khối lượng, ngày sản xuất, hạn sử dụng và hướng dẫn bảo quản.

**Giá trị nghiệp vụ:** Khách có đủ thông tin cần thiết để lựa chọn thực phẩm phù hợp và sử dụng/bảo quản sản phẩm an toàn.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem thông tin sản phẩm và SKU được chọn
  Given sản phẩm/SKU đã được phép công khai
  When khách mở trang chi tiết và chọn một SKU
  Then hệ thống hiển thị thành phần, khối lượng, ngày sản xuất, hạn sử dụng và hướng dẫn bảo quản theo dữ liệu hợp lệ
  And khách có thể đối chiếu thông tin trước khi thêm SKU vào giỏ hàng

Scenario: Thông tin bắt buộc chưa sẵn sàng công khai
  Given thông tin thực phẩm bắt buộc của sản phẩm/SKU chưa đầy đủ hoặc chưa được phê duyệt
  When Sales Manager cố công khai sản phẩm/SKU đó
  Then hệ thống không cho công khai cho đến khi thông tin đáp ứng quy định nghiệp vụ

Scenario: Thông tin Batch thực tế được quản lý ở kho
  Given một đơn hàng cần được xử lý theo Batch/Lot thực tế
  When hệ thống cần xác định ngày sản xuất hoặc hạn sử dụng của lô xuất hàng
  Then hệ thống sử dụng dữ liệu Batch/Lot từ EPIC 09 theo chính sách tồn kho
  And không suy diễn Batch thực tế chỉ từ mô tả catalog chung
```

### US-PROD-05 — Quản lý giá bán theo SKU

**Actor:** Sales Manager (`ACT-12`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là Sales Manager, tôi muốn quản lý giá bán theo từng SKU để đảm bảo giá hiển thị chính xác.

**Giá trị nghiệp vụ:** Các quy cách khác nhau có giá chính xác, giảm sai lệch giữa giá hiển thị, giỏ hàng và đơn đã đặt.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Thiết lập giá cho một SKU
  Given Sales Manager có quyền quản lý giá và SKU đã tồn tại
  When Sales Manager nhập giá bán hợp lệ cho SKU và lưu
  Then hệ thống lưu giá gắn với đúng SKU
  And khách chọn SKU đó sẽ thấy giá đang áp dụng theo kênh bán phù hợp

Scenario: Giá của các SKU được quản lý độc lập
  Given một sản phẩm có nhiều SKU
  When Sales Manager cập nhật giá của một SKU
  Then giá SKU đó được cập nhật theo chính sách hiệu lực
  And giá của SKU khác không bị thay đổi ngoài ý muốn

Scenario: Từ chối giá không hợp lệ
  Given Sales Manager nhập giá không hợp lệ theo chính sách doanh nghiệp
  When Sales Manager lưu giá
  Then hệ thống không áp dụng giá đó
  And hiển thị thông báo để Sales Manager chỉnh sửa
```

### US-PROD-06 — Quản lý trạng thái sản phẩm

**Actor:** Sales Manager (`ACT-12`)  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là Sales Manager, tôi muốn quản lý trạng thái sản phẩm như đang bán, tạm ngừng bán hoặc ngừng kinh doanh.

**Giá trị nghiệp vụ:** Doanh nghiệp có thể chủ động mở bán, tạm dừng hoặc ngừng một sản phẩm mà vẫn bảo toàn dữ liệu kinh doanh đã phát sinh.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đưa sản phẩm vào trạng thái đang bán
  Given sản phẩm và SKU đáp ứng điều kiện công khai theo chính sách
  When Sales Manager đặt trạng thái là đang bán
  Then sản phẩm/SKU có thể xuất hiện cho khách theo quy tắc Product Discovery
  And khách chỉ có thể mua khi còn đáp ứng các điều kiện tồn kho/hạn sử dụng

Scenario: Tạm ngừng bán sản phẩm hoặc SKU
  Given sản phẩm/SKU đang được bán
  When Sales Manager chuyển trạng thái sang tạm ngừng bán
  Then khách không thể thêm mới sản phẩm/SKU đó vào giỏ hàng hoặc checkout
  And dữ liệu sản phẩm cùng đơn hàng lịch sử vẫn được bảo toàn

Scenario: Ngừng kinh doanh sản phẩm
  Given Sales Manager xác định sản phẩm không còn kinh doanh
  When Sales Manager chuyển trạng thái sang ngừng kinh doanh theo quyền được cấp
  Then sản phẩm không còn được coi là lựa chọn có thể mua trên các kênh áp dụng
  And các đơn hàng đã phát sinh không bị sửa đổi hồi tố
```

### US-PROD-07 — Omnichannel Pricing

**Actor:** Sales Manager (`ACT-12`)  
**Ưu tiên:** Should Have  
**Phát hành:** Giai đoạn 2

> Là Sales Manager, tôi muốn quản lý chính sách giá khác nhau cho từng kênh (Website, Shopee, Offline, B2B) để tối ưu chiến lược bán hàng.

**Giá trị nghiệp vụ:** Doanh nghiệp có thể vận hành nhiều kênh mà vẫn kiểm soát giá theo chiến lược, chi phí và thỏa thuận của từng kênh.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Thiết lập giá theo kênh cho SKU
  Given SKU đã có giá cơ bản và Sales Manager có quyền quản lý giá đa kênh
  When Sales Manager cấu hình chính sách giá hợp lệ cho một hoặc nhiều kênh
  Then hệ thống lưu chính sách giá theo đúng SKU và kênh áp dụng
  And chỉ kênh được chỉ định mới dùng giá theo chính sách đó

Scenario: Hiển thị đúng giá ở từng kênh
  Given một SKU có chính sách giá hiệu lực cho các kênh khác nhau
  When khách hoặc nhân viên truy cập SKU từ một kênh cụ thể
  Then hệ thống xác định và áp dụng giá hiệu lực của đúng kênh
  And không hiển thị nhầm giá của kênh khác

Scenario: Thay đổi giá không làm sai đơn đã phát sinh
  Given đơn hàng đã được tạo với giá hợp lệ tại thời điểm đặt hàng
  When Sales Manager cập nhật chính sách giá sau đó
  Then giá trên đơn hàng lịch sử vẫn được giữ nguyên
  And giá mới chỉ áp dụng cho giao dịch thỏa điều kiện hiệu lực theo chính sách

Scenario: Ngăn người không có quyền thay đổi giá đa kênh
  Given nhân viên không có quyền quản lý giá đa kênh
  When nhân viên cố tạo hoặc cập nhật chính sách giá
  Then hệ thống từ chối thao tác
  And không thay đổi giá đang hiệu lực
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-PROD-01` | Cho phép Sales Manager tạo và cập nhật dữ liệu sản phẩm hợp lệ. | `US-PROD-01` |
| `FR-PROD-02` | Kiểm tra thông tin sản phẩm bắt buộc trước khi lưu hoặc công khai. | `US-PROD-01`, `US-PROD-04`, `BR-PROD-03`, `BR-PROD-04` |
| `FR-PROD-03` | Cho phép tạo và cập nhật nhiều Variant/SKU dưới một sản phẩm. | `US-PROD-02`, `BR-PROD-01` |
| `FR-PROD-04` | Duy trì thuộc tính nhận diện SKU, khối lượng, hương vị và quy cách đóng gói để các SKU không bị nhầm lẫn. | `US-PROD-02` |
| `FR-PROD-05` | Hiển thị SKU để khách lựa chọn; yêu cầu chọn SKU trước khi thêm vào giỏ nếu sản phẩm có nhiều lựa chọn. | `US-PROD-03` |
| `FR-PROD-06` | Hiển thị thành phần, khối lượng, ngày sản xuất, hạn sử dụng và hướng dẫn bảo quản theo dữ liệu được phép công khai. | `US-PROD-04`, `BR-PROD-03` |
| `FR-PROD-07` | Quản lý giá bán độc lập theo từng SKU và bảo đảm giá SKU được chọn được dùng xuyên suốt các luồng mua liên quan. | `US-PROD-05`, `BR-PROD-02` |
| `FR-PROD-08` | Cho phép đặt trạng thái đang bán, tạm ngừng bán hoặc ngừng kinh doanh; trạng thái kiểm soát khả năng thêm mới vào giỏ/checkout. | `US-PROD-06` |
| `FR-PROD-09` | Không coi SKU có tồn kho không khả dụng hoặc liên quan Batch hết hạn là SKU có thể mua; nguồn quyết định tồn/hạn thuộc EPIC 09. | `US-PROD-03`, `US-PROD-06`, `BR-BATCH-05`, `BR-ORDER-02` |
| `FR-PROD-10` | Cho phép cấu hình, tra cứu và áp dụng chính sách giá theo SKU và kênh bán phù hợp. | `US-PROD-07` |
| `FR-PROD-11` | Bảo toàn giá đã chốt trên đơn hàng lịch sử khi giá SKU hoặc giá kênh được thay đổi. | `US-PROD-05`, `US-PROD-07`, `EPIC 08` |
| `FR-PROD-12` | Kiểm soát quyền thay đổi sản phẩm, SKU, giá và trạng thái; các thay đổi quan trọng có khả năng truy vết. | `US-PROD-01`, `US-PROD-02`, `US-PROD-05~07`, `EPIC 22`, `EPIC 23` |

## 6. Quy tắc dữ liệu, công khai và an toàn nghiệp vụ

- Product là thực thể catalog; SKU là đơn vị được khách chọn mua và là cấp quản lý của giá, khối lượng, quy cách đóng gói và tồn kho.
- Tạo/cập nhật dữ liệu catalog không tự động đồng nghĩa được công khai. Chỉ dữ liệu đáp ứng điều kiện quản lý/phê duyệt mới hiển thị cho khách.
- Dữ liệu Batch/Lot, hạn sử dụng thực tế và phương án xuất kho do EPIC 09 quản lý. Catalog không được dùng để bỏ qua quy tắc hàng hết hạn.
- Thay đổi giá hoặc trạng thái không được sửa hồi tố giá/sản phẩm trong đơn hàng đã phát sinh.
- Giá theo kênh chỉ được áp dụng khi có chính sách hiệu lực, kênh được xác định rõ và người thao tác có quyền phù hợp.
- Chi tiết về cơ sở dữ liệu, API endpoint, cơ chế lưu phiên bản giá, đồng bộ với sàn, lưu media hoặc công nghệ triển khai thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-PROD-01` | `BR-PROD-04` | EPIC 03, 22, 23 | Sales Manager tạo/cập nhật sản phẩm hợp lệ; sản phẩm chưa đủ điều kiện không công khai. |
| `US-PROD-02` | `BR-PROD-01`, `BR-PROD-02` | EPIC 03, 05, 09 | Nhiều SKU thuộc đúng Product và có thuộc tính thương mại độc lập. |
| `US-PROD-03` | `BR-PROD-01`, `BR-PROD-02`, `BR-ORDER-02` | EPIC 03, 05, 06, 09 | Khách chọn đúng SKU; SKU không có khả năng bán không thể được mua. |
| `US-PROD-04` | `BR-PROD-03`, `BR-PROD-04` | EPIC 03, 09 | Thông tin thực phẩm được hiển thị đúng dữ liệu công khai; Batch thực tế không bị suy diễn. |
| `US-PROD-05` | `BR-PROD-02`, `BR-AUDIT-01` | EPIC 05, 06, 08, 23 | Giá theo SKU chính xác và thay đổi giá có thể truy vết. |
| `US-PROD-06` | `BR-PROD-04`, `BR-BATCH-05` | EPIC 03, 05, 06, 09 | Trạng thái bán kiểm soát khả năng mua nhưng không làm bán hàng hết hạn. |
| `US-PROD-07` | `BR-PROD-02`, `BR-AUDIT-01` | EPIC 12, 13, 18, 23 | Giá đúng theo kênh; giá đơn đã phát sinh được bảo toàn. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Trường bắt buộc của Product và SKU; quy tắc đặt/mã hóa SKU để ngăn trùng lặp.
- Quy trình duyệt/công khai sản phẩm: ai duyệt, các trạng thái nội bộ và thời điểm sản phẩm được phép xuất hiện cho khách.
- Thông tin ngày sản xuất/hạn sử dụng công khai là thông tin theo sản phẩm/SKU, theo Batch thực tế hay cả hai; cách tránh hiển thị thông tin gây hiểu nhầm.
- Chính sách xử lý SKU có tồn kho bằng 0, SKU tạm ngừng bán hoặc Product ngừng kinh doanh trên catalog và giỏ hàng đang tồn tại.
- Chính sách giá: đơn vị tiền tệ, thuế/phí đã bao gồm hay chưa, thời điểm giá có hiệu lực và quy tắc thay đổi giá khi khách đang giữ giỏ.
- Thứ tự ưu tiên khi một SKU vừa có giá cơ bản, giá theo kênh, khuyến mãi hoặc báo giá B2B.
- Phạm vi kênh áp dụng giá đa kênh và trách nhiệm đồng bộ giá thực tế với Marketplace/Offline/B2B.

## 9. UI/UX Reference

- Màn hình Sales Manager tạo/cập nhật Product; quản lý Variant/SKU, giá và trạng thái bán.
- Giao diện khách xem chi tiết sản phẩm, chọn SKU và xem thông tin thực phẩm trước khi thêm giỏ.
- Trạng thái không thể mua phải phân biệt rõ: tạm ngừng bán, hết hàng, SKU không còn khả dụng.
- Màn hình cấu hình giá theo kênh, có chỉ báo kênh áp dụng, thời điểm hiệu lực và cảnh báo ảnh hưởng đến giá hiện tại.

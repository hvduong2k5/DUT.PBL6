# EPIC 27 — Procurement & Supplier Management

## 1. Mục tiêu Epic

Epic này quản lý chuỗi cung ứng thượng nguồn cho sản phẩm OCOP: hồ sơ nhà cung cấp/hộ nông dân/hợp tác xã, kế hoạch sản xuất hoặc nhập hàng, Purchase Order (PO) và đối chiếu hàng thực nhận. Supply Manager biết mua gì, từ ai, khi nào và theo điều kiện nào; Warehouse Staff nhận hàng có căn cứ và bảo toàn mọi chênh lệch.

EPIC 27 sở hữu Supplier Profile, Supplier Agreement/Price Policy, Supply Plan, PO và PO Receipt/Variance. Epic không sở hữu số dư Inventory, Batch, dữ liệu traceability công khai, dự báo nhu cầu, Payment/Accounts Payable hay báo cáo tài chính; các miền đó thuộc EPIC 09, 19, 25, 07/miền tài chính cần mở rộng và EPIC 24.

PO là cam kết mua hàng, không phải bằng chứng hàng đã nhập kho hoặc đã thanh toán. Việc nhận PO không được tự cộng tồn trước khi EPIC 09 chấp nhận biến động nhập hợp lệ.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 3 — Could Have** | `US-PROC-01` đến `US-PROC-04` | Quản lý Supplier, lập kế hoạch, phát hành PO và đối chiếu nhận hàng với PO. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và bên liên quan

| Actor / bên liên quan | Vai trò |
| --- | --- |
| Inventory/Supply Manager (`ACT-15`) | Quản lý Supplier, lập/duyệt theo quyền, phát hành và theo dõi PO/Supply Plan. |
| Warehouse Staff (`ACT-06`) | Tiếp nhận, kiểm đếm, ghi nhận chất lượng và đối chiếu hàng thực tế với PO. |
| Accountant / Finance Staff (`ACT-19`) | Xem cam kết/receipt đã được phép để đối chiếu tài chính; không sửa PO hoặc tồn nếu thiếu quyền. |
| Supplier / Farmer / Cooperative | Đối tác cung ứng được quản lý như business entity; chưa mặc nhiên là tài khoản hệ thống hay portal actor. |

### 3.2. Business Rules và NFR áp dụng

Product Backlog chưa định nghĩa nhóm `BR-PROC`; Epic không tự tạo mã Business Rule mới.

| Mã | Ảnh hưởng đến Epic |
| --- | --- |
| `BR-AUTH-03`, `BR-AUTH-04` | Tạo/sửa Supplier, duyệt/phát hành/hủy PO, nhận hàng và xử lý variance phải đúng Role/Permission. |
| `BR-PROD-01`, `BR-PROD-02` | Dòng PO thành phẩm phải xác định đúng Product/SKU, đơn vị/quy cách và snapshot giá. |
| `BR-PROD-04` | Thông tin nguồn gốc/chứng nhận hiển thị cho khách phải chính xác; Supplier/PO chỉ cung cấp dữ liệu nguồn để EPIC 19 duyệt. |
| `BR-BATCH-01`, `BR-BATCH-02` | Hàng thực phẩm nhận vào phải truy xuất theo Batch/Lot và NSX/HSD khi áp dụng. |
| `BR-BATCH-05` | Batch không đủ điều kiện HSD không được nhập vào tồn khả dụng như hàng bình thường. |
| `BR-AUDIT-01` | Thay đổi Supplier/Agreement, duyệt/phát hành/hủy PO, override và xử lý variance phải Audit theo catalog. |
| `NFR-04`, `NFR-05`, `NFR-08` | Bảo vệ hợp đồng, giá mua, thông tin liên hệ và giới hạn quyền theo phạm vi. |
| `NFR-06` | Receipt, Batch và Inventory movement phải nhất quán, chống nhận/cộng tồn trùng. |
| `NFR-10` | Danh sách Supplier/PO/Receipt và tính kế hoạch phải mở rộng, phân trang/xử lý nền khi cần. |
| `NFR-11` | Theo dõi lỗi chuyển PO, nhận hàng, cập nhật Inventory và dữ liệu kế hoạch. |

### 3.3. Phụ thuộc và ranh giới

- `EPIC 04`: cung cấp Product/SKU bán; danh mục nguyên liệu/vật tư mua vào cần mô hình riêng được chốt.
- `EPIC 09`: sở hữu Batch và Inventory movement; EPIC 27 cung cấp PO/Receipt/variance làm nguồn tham chiếu.
- `EPIC 19`: dùng Supplier/PO/snapshot nguồn cho traceability; chỉ EPIC 19 quyết định dữ liệu nào được công khai.
- `EPIC 22`, `EPIC 23`: cấp quyền và Audit thay đổi nhạy cảm.
- `EPIC 24`: dùng cam kết mua, receipt và chi phí đã xác nhận cho Finance; Procurement không tự ghi nhận Payment/thuế.
- `EPIC 25`: cung cấp forecast/velocity/expiry insight; Procurement lưu snapshot đầu vào và vẫn yêu cầu quyết định con người.
- `EPIC 28`: gửi PO/cảnh báo/phê duyệt khi được cấu hình; trạng thái gửi không thay trạng thái nghiệp vụ PO nếu chưa có xác nhận phù hợp.

## 4. User Stories chi tiết

### US-PROC-01 — Quản lý hồ sơ nhà cung cấp

**Actor:** Inventory/Supply Manager (`ACT-15`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Supply Manager, tôi muốn quản lý thông tin nhà cung cấp, hộ nông dân, hợp tác xã gồm tên, địa chỉ, hợp đồng và chính sách giá.

```gherkin
Scenario: Tạo Supplier Profile hợp lệ
  Given Supply Manager có quyền
  When nhập loại Supplier, tên, địa chỉ và thông tin bắt buộc hợp lệ
  Then hệ thống tạo mã Supplier duy nhất ở trạng thái phù hợp
  And lưu actor và thời điểm tạo

Scenario: Phát hiện hồ sơ có khả năng trùng
  Given đã có Supplier cùng định danh pháp lý hoặc khóa nhận diện được duyệt
  When người dùng tạo hồ sơ mới
  Then hệ thống chặn hoặc cảnh báo theo chính sách
  And không tự hợp nhất hai pháp nhân

Scenario: Quản lý hợp đồng có thời hạn
  Given Supplier tồn tại
  When thêm hợp đồng với số, hiệu lực, phạm vi và tài liệu hợp lệ
  Then hệ thống lưu phiên bản và khoảng hiệu lực
  And không ghi đè hợp đồng lịch sử

Scenario: Quản lý chính sách giá
  Given Supplier cung cấp item theo điều kiện xác định
  When thêm giá, đơn vị, tiền tệ, MOQ, lead time và hiệu lực
  Then hệ thống kiểm tra khoảng hiệu lực/xung đột theo chính sách
  And PO sau này dùng snapshot thay vì tự đổi theo giá mới

Scenario: Tạm ngừng Supplier
  Given Supplier có vấn đề hoặc không còn hợp tác
  When người có quyền đổi trạng thái cùng lý do
  Then hệ thống ngăn dùng cho PO mới theo chính sách
  And không xóa PO/Receipt lịch sử

Scenario: Supplier đang có PO mở
  Given Supplier còn PO chưa hoàn tất
  When yêu cầu vô hiệu hóa
  Then hệ thống cảnh báo tác động và áp dụng quy trình được duyệt
  And không tự hủy PO mở

Scenario: Cập nhật thông tin dùng cho traceability
  Given hồ sơ hiện tại thay đổi
  When lưu phiên bản mới
  Then EPIC 19 có thể nhận dữ liệu hiện hành
  And snapshot nguồn của Batch cũ không bị sửa hồi tố

Scenario: Không có quyền xem giá/hợp đồng
  Given người dùng chỉ được xem thông tin Supplier cơ bản
  When mở giá mua hoặc tài liệu hợp đồng
  Then hệ thống ẩn/từ chối dữ liệu hạn chế
  And không để file hoặc export vượt quyền
```

### US-PROC-02 — Tạo và gửi Purchase Order

**Actor:** Inventory/Supply Manager (`ACT-15`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Supply Manager, tôi muốn tạo PO gửi nhà cung cấp để chuẩn bị nguồn hàng.

```gherkin
Scenario: Tạo PO nháp hợp lệ
  Given Supplier đang hoạt động và item có thể mua
  When nhập số lượng, đơn vị, giá, thuế/phí, ngày cần và địa điểm nhận
  Then hệ thống tạo PO Draft với tổng tiền nhất quán
  And lưu snapshot Supplier, item, giá và điều khoản

Scenario: Dòng PO không hợp lệ
  Given số lượng âm/bằng không, đơn vị không tương thích hoặc thiếu item
  When lưu hoặc gửi duyệt
  Then hệ thống từ chối và chỉ rõ dòng cần sửa

Scenario: Gửi PO qua phê duyệt
  Given PO Draft đủ dữ liệu và vượt/thuộc ngưỡng cần duyệt
  When Supply Manager gửi duyệt
  Then PO chuyển đúng trạng thái và khóa trường theo policy
  And không được gửi Supplier trước khi đủ phê duyệt

Scenario: Phát hành PO
  Given PO đã được duyệt và Supplier còn hợp lệ
  When người có quyền phát hành
  Then hệ thống tạo phiên bản phát hành bất biến và trạng thái gửi
  And retry không tạo PO hoặc phiên bản trùng

Scenario: Supplier xác nhận hoặc đề nghị thay đổi
  Given PO đã gửi
  When ghi nhận phản hồi Supplier
  Then hệ thống lưu xác nhận hoặc đề nghị thay đổi riêng
  And thay đổi điều khoản phải tạo revision và duyệt lại khi cần

Scenario: Hủy PO chưa nhận hàng
  Given PO còn đủ điều kiện hủy
  When người có quyền hủy với lý do
  Then hệ thống chuyển Cancelled và lưu Audit
  And không tạo biến động Inventory

Scenario: Hủy PO đã nhận một phần
  Given PO có Receipt hợp lệ
  When yêu cầu hủy phần còn lại
  Then hệ thống giữ lượng đã nhận và đóng/hủy open quantity theo chính sách
  And không xóa Receipt

Scenario: Supplier bị vô hiệu trước khi phát hành
  Given Supplier không còn đủ điều kiện
  When người dùng phát hành PO
  Then hệ thống chặn và yêu cầu xử lý Supplier/PO
```

### US-PROC-03 — Lập kế hoạch sản xuất và cung ứng

**Actor:** Inventory/Supply Manager (`ACT-15`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Supply Manager, tôi muốn lập kế hoạch sản xuất/nhập hàng dựa trên dự báo nhu cầu để tránh thiếu hụt hoặc tồn quá mức.

```gherkin
Scenario: Tạo Supply Plan từ forecast
  Given forecast còn hiệu lực và Inventory/open PO khả dụng
  When chọn horizon, phạm vi và chính sách tồn
  Then hệ thống tính nhu cầu ròng theo công thức được duyệt
  And lưu snapshot forecast, tồn, open PO, lead time và policy version

Scenario: Xem phương án theo kịch bản
  Given có baseline và các giả định khác nhau
  When Supply Manager thay đổi forecast, safety stock hoặc lead time
  Then hệ thống hiển thị tác động thiếu/dư và ngày cần đặt
  And không sửa dữ liệu nguồn

Scenario: Thiếu forecast hoặc dữ liệu chất lượng thấp
  Given đầu vào thiếu, stale hoặc không đủ tin cậy
  When tạo kế hoạch
  Then hệ thống cảnh báo phạm vi ảnh hưởng
  And không trình bày đề xuất như kế hoạch đã phê duyệt

Scenario: Áp dụng MOQ và quy cách đặt
  Given Supplier có MOQ, pack size hoặc capacity hợp lệ
  When tính lượng đề xuất
  Then hệ thống làm tròn/theo ràng buộc đã duyệt
  And hiển thị chênh lệch so với nhu cầu ròng

Scenario: Phát hiện nguy cơ thiếu hoặc tồn quá mức
  Given projected inventory vượt ngưỡng trên/dưới
  When chạy kế hoạch
  Then hệ thống đánh dấu item/kỳ cùng nguyên nhân và giả định

Scenario: Điều chỉnh thủ công
  Given Supply Manager có quyền và đề xuất cần thay đổi
  When nhập lượng/ngày mới cùng lý do
  Then hệ thống giữ giá trị hệ thống và giá trị override riêng
  And lưu actor, thời điểm và Audit

Scenario: Phê duyệt Supply Plan
  Given kế hoạch đủ dữ liệu
  When người có thẩm quyền phê duyệt
  Then phiên bản được khóa làm baseline
  And sửa tiếp phải tạo revision mới

Scenario: Chuyển dòng kế hoạch thành PO
  Given plan line được duyệt và có Supplier phù hợp
  When tạo PO
  Then hệ thống tạo đúng một Draft PO tham chiếu plan/version
  And vẫn yêu cầu validation/phê duyệt PO độc lập
```

### US-PROC-04 — Đối chiếu nhận hàng với PO

**Actor:** Warehouse Staff (`ACT-06`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Warehouse Staff, tôi muốn đối chiếu hàng thực nhận với PO khi nhập kho để phát hiện chênh lệch.

```gherkin
Scenario: Nhận đủ theo PO
  Given PO đã phát hành còn open quantity
  When Warehouse Staff xác nhận đúng item, số lượng, đơn vị, Batch và chất lượng
  Then hệ thống tạo Receipt liên kết PO
  And yêu cầu EPIC 09 ghi nhận đúng lượng được chấp nhận

Scenario: Nhận một phần
  Given lượng thực nhận nhỏ hơn open quantity và được phép nhận phần
  When xác nhận Receipt
  Then hệ thống cập nhật received/open quantity
  And PO giữ trạng thái partially received

Scenario: Nhận thừa
  Given lượng thực nhận vượt open quantity hoặc tolerance
  When đối chiếu
  Then hệ thống tạo variance và chặn/đưa phần thừa chờ duyệt theo policy
  And không tự cộng toàn bộ vào tồn khả dụng

Scenario: Nhận thiếu, sai item hoặc sai đơn vị
  Given hàng thực tế không khớp PO
  When Warehouse Staff ghi nhận
  Then hệ thống phân loại variance với expected/actual
  And giữ bằng chứng, ghi chú và trạng thái xử lý

Scenario: Hàng hỏng hoặc không đạt chất lượng/HSD
  Given hàng có lỗi hoặc Batch không đạt điều kiện
  When kiểm tra nhận
  Then lượng đó được rejected/quarantined theo policy
  And không nhập như tồn khả dụng bình thường

Scenario: Nhận qua nhiều đợt
  Given PO cho phép nhiều Receipt
  When xác nhận đợt tiếp theo
  Then tổng received không vượt quy tắc được duyệt
  And mỗi Receipt giữ actor, thời điểm, Batch và chứng từ riêng

Scenario: Retry cùng Receipt
  Given cùng yêu cầu/chứng từ nhận đã được xử lý
  When hệ thống nhận lại
  Then không tạo Receipt hoặc Inventory movement trùng
  And trả kết quả nhất quán của lần xử lý trước

Scenario: PO không hợp lệ hoặc đã đóng
  Given PO bị hủy, đóng hoặc không thuộc địa điểm nhận
  When Warehouse Staff cố nhận
  Then hệ thống chặn nhận theo PO đó
  And hướng dẫn quy trình nhận ngoại lệ nếu chính sách cho phép

Scenario: Hoàn tất PO
  Given mọi dòng đã nhận/đóng phần còn lại và variance đã xử lý
  When người có quyền hoàn tất
  Then PO chuyển Received/Closed theo policy
  And dữ liệu Receipt sẵn sàng cho Inventory, Finance và traceability
```

## 5. Functional Requirements

| Mã | Yêu cầu | Truy vết |
| --- | --- | --- |
| `FR-PROC-01` | Tạo Supplier với loại, mã, tên, địa chỉ, liên hệ và định danh hợp lệ. | `US-PROC-01` |
| `FR-PROC-02` | Phát hiện khả năng trùng nhưng không tự hợp nhất Supplier. | `US-PROC-01` |
| `FR-PROC-03` | Quản lý phiên bản hợp đồng, tài liệu, phạm vi và khoảng hiệu lực. | `US-PROC-01` |
| `FR-PROC-04` | Quản lý chính sách giá, đơn vị, tiền tệ, MOQ, lead time và hiệu lực. | `US-PROC-01` |
| `FR-PROC-05` | Quản lý trạng thái Supplier và tác động tới PO mới mà không xóa lịch sử. | `US-PROC-01` |
| `FR-PROC-06` | Bảo vệ giá/hợp đồng/PII theo quyền và phạm vi. | `US-PROC-01`, `NFR-05` |
| `FR-PROC-07` | Cung cấp Supplier hiện hành và snapshot lịch sử cho EPIC 19. | `US-PROC-01` |
| `FR-PROC-08` | Tạo PO Draft với Supplier, dòng hàng, lượng, đơn vị, giá, thuế/phí, ngày/địa điểm. | `US-PROC-02` |
| `FR-PROC-09` | Lưu snapshot Supplier, item, giá và điều khoản trên PO revision. | `US-PROC-02` |
| `FR-PROC-10` | Kiểm tra dòng PO, đơn vị, tổng tiền và điều kiện Supplier. | `US-PROC-02` |
| `FR-PROC-11` | Hỗ trợ submit/approve/reject/revise/issue/acknowledge/cancel/close theo quyền. | `US-PROC-02` |
| `FR-PROC-12` | Phiên bản phát hành bất biến; sửa điều khoản tạo revision phù hợp. | `US-PROC-02` |
| `FR-PROC-13` | Chống phát hành/gửi PO trùng khi retry. | `US-PROC-02` |
| `FR-PROC-14` | Hủy/đóng phần còn lại không xóa Receipt hoặc biến động đã có. | `US-PROC-02`, `04` |
| `FR-PROC-15` | Tạo Supply Plan từ snapshot forecast, tồn, open PO, lead time và policy. | `US-PROC-03` |
| `FR-PROC-16` | Tính nhu cầu ròng và projected inventory theo công thức có phiên bản. | `US-PROC-03` |
| `FR-PROC-17` | Hỗ trợ scenario planning mà không sửa nguồn. | `US-PROC-03` |
| `FR-PROC-18` | Áp dụng MOQ, pack size, capacity và hiển thị tác động làm tròn. | `US-PROC-03` |
| `FR-PROC-19` | Cảnh báo input stale/thiếu và rủi ro thiếu/dư tồn. | `US-PROC-03` |
| `FR-PROC-20` | Lưu system proposal và manual override riêng, kèm lý do/Audit. | `US-PROC-03` |
| `FR-PROC-21` | Version hóa/phê duyệt Supply Plan và tạo Draft PO idempotent từ plan line. | `US-PROC-03` |
| `FR-PROC-22` | Tạo Receipt liên kết đúng PO line, item, lượng, đơn vị, Batch, địa điểm. | `US-PROC-04` |
| `FR-PROC-23` | Hỗ trợ nhận nhiều phần và tính received/open quantity nhất quán. | `US-PROC-04` |
| `FR-PROC-24` | Phân loại variance thừa/thiếu/sai item/sai đơn vị/hỏng/chất lượng/HSD. | `US-PROC-04` |
| `FR-PROC-25` | Chặn hoặc quarantine lượng ngoài tolerance/không đạt điều kiện. | `US-PROC-04` |
| `FR-PROC-26` | Gửi đúng lượng accepted sang EPIC 09 và chống Inventory movement trùng. | `US-PROC-04`, `NFR-06` |
| `FR-PROC-27` | Không nhận theo PO hủy/đóng/sai địa điểm; hỗ trợ quy trình ngoại lệ có quyền. | `US-PROC-04` |
| `FR-PROC-28` | Chỉ đóng PO khi line/variance thỏa điều kiện được duyệt. | `US-PROC-04` |
| `FR-PROC-29` | Audit thay đổi Supplier, Agreement, Plan, PO, Receipt, variance và override quan trọng. | `US-PROC-01`–`04`, `EPIC 23` |
| `FR-PROC-30` | Theo dõi lỗi gửi PO, kế hoạch, Receipt và tích hợp Inventory/Traceability/Finance. | `US-PROC-01`–`04`, `NFR-11` |

## 6. Mô hình dữ liệu và vòng đời

```text
Supplier -> Agreement / Price Policy
Supply Plan -> Plan Version -> Plan Line -> PO Draft
PO -> PO Revision -> PO Line -> Receipt -> Receipt Line -> Variance

Supplier: DRAFT -> ACTIVE -> SUSPENDED | INACTIVE
PO: DRAFT -> PENDING_APPROVAL -> APPROVED -> ISSUED -> ACKNOWLEDGED
    -> PARTIALLY_RECEIVED -> RECEIVED -> CLOSED
    -> REJECTED | CANCELLED
Receipt: DRAFT -> CONFIRMED -> POSTED | PARTIALLY_POSTED | FAILED | VOIDED
Variance: OPEN -> IN_REVIEW -> ACCEPTED | REJECTED | RESOLVED
```

- Supplier code và PO number phải duy nhất theo phạm vi được chốt; định danh pháp lý không được sửa mất lịch sử.
- PO revision đã phát hành là snapshot bất biến; revision mới không sửa Receipt của revision cũ.
- `ordered`, `received`, `accepted`, `rejected/quarantined` và `open` là các lượng riêng, cùng đơn vị quy đổi có căn cứ.
- Receipt và Inventory movement dùng khóa idempotency/tham chiếu hai chiều để chống cộng tồn hai lần.
- Variance giữ expected, actual, difference, tolerance, reason, decision và evidence.
- Công nghệ lưu trữ, workflow engine, ký/gửi tài liệu và tích hợp thuộc Technical Design.

## 7. Traceability

| User Story | FR chính | Epic liên quan |
| --- | --- | --- |
| `US-PROC-01` | `FR-PROC-01`–`07`, `29`, `30` | EPIC 19/22/23 |
| `US-PROC-02` | `FR-PROC-08`–`14`, `29`, `30` | EPIC 09/24/28 |
| `US-PROC-03` | `FR-PROC-15`–`21`, `29`, `30` | EPIC 09/25/27 |
| `US-PROC-04` | `FR-PROC-22`–`30` | EPIC 09/19/24 |

`FR-29 — Procurement` trong Functional Requirements cấp cao được chi tiết hóa bởi `FR-PROC-01` đến `FR-PROC-30`.

## 8. Quyết định còn mở

- Danh mục item mua vào: Product/SKU bán, nguyên liệu, bao bì, dịch vụ; mã và đơn vị chuẩn.
- Định danh pháp lý bắt buộc theo loại Supplier và quy tắc phát hiện/hợp nhất hồ sơ trùng.
- Workflow/ngưỡng phê duyệt PO, separation of duties và quyền hủy/đóng/mở lại.
- Cách gửi/xác nhận PO với Supplier khi chưa có Supplier Portal actor/integration trong backlog.
- Tiền tệ, thuế, tỷ giá, điều khoản thanh toán, Incoterm/điều kiện giao nếu áp dụng.
- MOQ, pack size, lead time, safety stock, capacity, tolerance nhận hàng và quy tắc làm tròn.
- Chính sách nhận một phần/thừa/thiếu, substitution, kiểm định chất lượng, quarantine và trả Supplier.
- Batch/NSX/HSD bắt buộc theo loại item và ngưỡng HSD tối thiểu lúc nhận.
- Khi nào PO/Receipt tạo cam kết/chi phí/công nợ trong EPIC 24; Accounts Payable chưa có story riêng.
- SLA forecast/freshness, horizon kế hoạch và điều kiện cho phép dùng baseline/manual plan.
- Retention, quyền tải hợp đồng/chứng từ và dữ liệu Supplier nào được phép công khai qua EPIC 19.

## 9. UI/UX Reference

- Supplier Directory/Profile: trạng thái, liên hệ, hợp đồng, giá, hiệu lực và dữ liệu hạn chế theo quyền.
- Supply Planning: forecast/tồn/open PO, scenario, nhu cầu ròng, cảnh báo và override có lý do.
- PO Workspace: Draft/revision/approval/issue/acknowledgement/receipt timeline.
- Receiving: quét/chọn PO line, expected–actual, Batch/NSX/HSD, chất lượng, variance và evidence.
- Variance Queue: loại, mức chênh lệch, owner, trạng thái và quyết định.
- Mọi màn hình cần loading, empty, stale input, permission denied, conflict, partial receipt và integration failed.

Liên kết Figma sẽ được bổ sung khi thiết kế UI/UX được phê duyệt.

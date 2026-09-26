# EPIC 19 — OCOP Traceability

## 1. Mục tiêu Epic

Epic này cung cấp hồ sơ truy xuất nguồn gốc công khai cho sản phẩm OCOP theo Product/SKU và Batch/Lot. Khách có thể quét QR để xem vùng nguyên liệu, nhà sản xuất, quy trình sản xuất và chứng nhận đã được kiểm tra; Manager quản lý, xác nhận và công khai dữ liệu có phiên bản, nguồn và thời điểm rõ ràng.

EPIC 19 sở hữu Traceability Profile, liên kết QR công khai, bản ghi nguồn gốc, chứng nhận và quy trình xác nhận/công khai. Epic không sở hữu dữ liệu Catalog, số dư/NSX/HSD của Batch, hồ sơ Supplier gốc, bài viết SEO hay tệp vật lý; các trách nhiệm đó thuộc EPIC 04, 09, 27, 20 và EXT-09.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 3rd** | `US-OCOP-01` đến `US-OCOP-04` | Khách quét QR và xem được nguồn gốc/chứng nhận đã công khai; Manager quản lý hồ sơ truy xuất theo sản phẩm/lô. |
| **Giai đoạn 2** | `US-OCOP-05`, `US-OCOP-06` | Dữ liệu nhà cung cấp/nguyên liệu được quản lý có hiệu lực và trải qua bước kiểm tra/xác nhận trước khi công khai. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và hệ thống ngoài

| Actor / hệ thống | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Quét QR và xem hồ sơ nguồn gốc/chứng nhận đã công khai mà không cần đăng nhập. |
| Registered Customer (`ACT-02`) | Có quyền xem công khai tương tự Guest và tiếp tục hành trình mua hàng ở Epic khác. |
| Inventory/Supply Manager (`ACT-15`) | Quản lý dữ liệu nguồn nguyên liệu/nhà cung cấp và liên kết hồ sơ với Batch theo quyền. |
| Sales Manager (`ACT-12`) | Quản lý hoặc xác nhận nội dung nguồn gốc công khai nếu được phân quyền. |
| Content Manager (`ACT-13`) | Phối hợp trình bày nội dung công khai; không tự xác nhận dữ liệu nguồn gốc nếu thiếu quyền. |
| Warehouse Staff (`ACT-06`) | Cung cấp Batch/NSX/HSD từ EPIC 09; không tự công khai hồ sơ truy xuất chỉ nhờ quyền kho. |
| Media / Object Storage (`EXT-09`) | Lưu tài liệu chứng nhận, hình ảnh/video nguồn gốc theo chính sách truy cập. |
| Origin / Traceability Provider (`EXT-10`) | Xác thực chứng nhận OCOP và cung cấp dữ liệu nguồn gốc bên ngoài khi tích hợp. |

> Product Backlog dùng actor chung “Manager” cho `US-OCOP-04~06` nhưng chưa ánh xạ dứt khoát sang `ACT-12` hay `ACT-15`. Tài liệu tạm phân chia theo trách nhiệm ở bảng trên; quyền sở hữu và phê duyệt cuối cùng cần Product Owner chốt.

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-03` | Tài khoản nội bộ phải có Role phù hợp trước khi truy cập chức năng quản trị. | Người quản lý/duyệt hồ sơ phải có Role nội bộ hợp lệ. |
| `BR-AUTH-04` | Nhân viên chỉ thực hiện thao tác thuộc quyền được cấp. | Quyền tạo, sửa, gửi duyệt, xác nhận, công khai và xem nguồn nội bộ phải được kiểm soát riêng. |
| `BR-PROD-01` | Một Product có thể có nhiều Variant/SKU. | Hồ sơ phải phân biệt dữ liệu cấp Product với dữ liệu chỉ áp dụng cho SKU cụ thể. |
| `BR-PROD-03` | Sản phẩm thực phẩm phải có thông tin NSX/HSD phù hợp. | Trang theo Batch chỉ dùng NSX/HSD từ Batch hợp lệ, không nhập bản sao mâu thuẫn. |
| `BR-PROD-04` | Thông tin sản phẩm hiển thị cho khách phải được quản lý/phê duyệt trước khi công khai. | Dữ liệu nguồn gốc/chứng nhận nháp hoặc chưa xác nhận không được hiển thị như thông tin chính thức. |
| `BR-BATCH-01` | Tồn kho thực phẩm phải truy xuất được theo Batch/Lot. | QR theo lô phải liên kết đúng Batch và không gộp các lô khác danh tính. |
| `BR-BATCH-02` | Batch có mã lô, NSX, HSD, số lượng và nguồn nếu có. | Hồ sơ truy xuất dùng định danh/NSX/HSD/nguồn từ EPIC 09 và nêu rõ trường còn thiếu. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Sửa dữ liệu, xác nhận, từ chối, công khai, đình chỉ và đổi liên kết QR phải truy vết được. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 04 — Product, Variant & SKU`: cung cấp Product/SKU và dữ liệu công khai cơ bản; EPIC 19 không sửa Catalog.
- `EPIC 09 — Inventory & Batch`: nguồn sự thật cho Batch/Lot, NSX/HSD và nguồn nhập; EPIC 19 không sửa số lượng hoặc ngày của Batch.
- `EPIC 20 — Content & SEO`: quản lý bài viết/landing page/SEO; EPIC 19 quản lý dữ kiện truy xuất và trạng thái xác nhận.
- `EPIC 22, 23 — Administration, Audit & Security`: cung cấp Role/Permission và Audit Log cho quy trình quản lý/phê duyệt.
- `EPIC 27 — Procurement & Supplier`: sở hữu Supplier Profile, hợp đồng và quan hệ cung ứng; EPIC 19 lưu tham chiếu/snapshot nguồn gốc cần công khai.
- `EXT-10 — Origin / Traceability Provider`: xác thực dữ liệu bên ngoài; kết quả Provider phải có trạng thái/thời điểm riêng, không tự đồng nghĩa đã được doanh nghiệp công khai.
- `EXT-09 — Media Storage`: lưu tài liệu/media; quyền xem hồ sơ không mặc nhiên cho phép tải bản chứng nhận nội bộ gốc.

## 4. User Stories chi tiết

### US-OCOP-01 — Quét QR để kiểm tra nguồn gốc

**Actor:** Guest Customer, Registered Customer
**Ưu tiên:** Must Have — 3rd
**Phát hành:** MVP

> Là khách hàng, tôi muốn quét QR trên sản phẩm để kiểm tra nguồn gốc.

**Giá trị nghiệp vụ:** Khách truy cập nhanh đúng hồ sơ của sản phẩm/lô đang cầm và nhận biết rõ dữ liệu đã được xác nhận.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Quét QR hợp lệ theo Batch
  Given QR đang hoạt động và liên kết với một Traceability Profile đã công khai của Batch
  When khách mở liên kết từ QR
  Then hệ thống hiển thị hồ sơ đúng Product/SKU và Batch
  And hiển thị mã lô, NSX/HSD cùng trạng thái/thời điểm xác nhận được phép
  And không yêu cầu khách đăng nhập chỉ để xem dữ liệu công khai

Scenario: QR hợp lệ ở cấp Product
  Given QR chỉ liên kết hồ sơ nguồn gốc chung của Product
  When khách truy cập
  Then hệ thống hiển thị rõ đây là thông tin cấp Product
  And không suy diễn mã Batch, NSX/HSD cụ thể khi chưa có liên kết lô

Scenario: Mã QR không tồn tại hoặc sai
  Given mã công khai không ánh xạ tới hồ sơ hợp lệ
  When khách truy cập
  Then hệ thống hiển thị trạng thái không tìm thấy/không hợp lệ
  And không để lộ định danh nội bộ hay dữ liệu nháp

Scenario: Hồ sơ bị đình chỉ sau khi QR đã in
  Given QR từng hợp lệ nhưng hồ sơ hiện bị `SUSPENDED`
  When khách quét mã
  Then hệ thống không hiển thị nội dung cũ như đang được xác nhận
  And hiển thị thông báo phù hợp cùng kênh hỗ trợ nếu chính sách yêu cầu

Scenario: QR bị gắn nhầm Batch
  Given mã QR thuộc Batch khác với hồ sơ được yêu cầu
  When hệ thống đối chiếu liên kết
  Then hệ thống không gộp hoặc hiển thị dữ liệu của hai Batch như cùng một lô
  And đưa sai lệch vào quy trình xử lý nội bộ

Scenario: Trang nguồn gốc tạm không tải được dữ liệu ngoài
  Given hồ sơ nội bộ đã công khai nhưng EXT-10 tạm không khả dụng
  When khách quét QR
  Then hệ thống hiển thị dữ liệu đã xác nhận gần nhất nếu chính sách cho phép
  And nêu rõ thời điểm kiểm tra gần nhất/trạng thái chưa thể cập nhật
  And không biến lỗi Provider thành xác nhận mới
```

### US-OCOP-02 — Xem vùng nguyên liệu, nhà sản xuất và quy trình

**Actor:** Guest Customer, Registered Customer
**Ưu tiên:** Must Have — 3rd
**Phát hành:** MVP

> Là khách hàng, tôi muốn xem thông tin vùng nguyên liệu, nhà sản xuất và quy trình sản xuất để tăng niềm tin.

**Giá trị nghiệp vụ:** Khách hiểu nguồn hình thành sản phẩm từ các dữ kiện đã được kiểm tra thay vì nội dung quảng bá không có phạm vi hoặc nguồn rõ ràng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem hồ sơ nguồn gốc đầy đủ
  Given Traceability Profile đã công khai có vùng nguyên liệu, nhà sản xuất và các bước quy trình
  When khách mở chi tiết
  Then hệ thống hiển thị từng nhóm thông tin với nguồn/phạm vi áp dụng
  And phân biệt dữ liệu cấp Product với dữ liệu riêng của Batch

Scenario: Sản phẩm có nhiều nguồn nguyên liệu
  Given sản phẩm có nhiều nguyên liệu hoặc nhiều vùng cung cấp
  When khách xem hồ sơ
  Then hệ thống hiển thị từng nguyên liệu cùng nguồn tương ứng được phép công khai
  And không rút gọn thành một nguồn duy nhất gây hiểu sai

Scenario: Một phần dữ liệu chưa được xác nhận
  Given hồ sơ có trường còn thiếu hoặc chưa được duyệt
  When khách xem bản công khai
  Then hệ thống không hiển thị trường đó như dữ kiện đã xác nhận
  And hiển thị trạng thái chưa có dữ liệu nếu cần thay vì tự suy diễn

Scenario: Dữ liệu nguồn thay đổi sau khi Batch được sản xuất
  Given thông tin Supplier/vùng nguyên liệu hiện tại đã thay đổi
  When khách xem Batch lịch sử
  Then hệ thống dùng snapshot nguồn gốc áp dụng cho Batch đó
  And không thay hồi tố Batch cũ chỉ vì hồ sơ nguồn hiện tại đổi

Scenario: Bảo vệ dữ liệu nội bộ
  Given bản ghi nguồn chứa hợp đồng, giá mua, liên hệ riêng hoặc ghi chú nội bộ
  When khách xem hồ sơ công khai
  Then hệ thống chỉ hiển thị trường đã được duyệt cho công khai
  And không cung cấp tài liệu/dữ liệu nội bộ qua liên kết trực tiếp

Scenario: Media quy trình không còn khả dụng
  Given hồ sơ tham chiếu ảnh/video đã bị lỗi hoặc không được phép công khai
  When khách xem quy trình
  Then hệ thống không hiển thị liên kết hỏng như bằng chứng hợp lệ
  And phần dữ liệu nguồn gốc còn lại vẫn được hiển thị nếu đủ điều kiện
```

### US-OCOP-03 — Xem chứng nhận OCOP và chứng nhận liên quan

**Actor:** Guest Customer, Registered Customer
**Ưu tiên:** Must Have — 3rd
**Phát hành:** MVP

> Là khách hàng, tôi muốn xem chứng nhận OCOP và các chứng nhận liên quan của sản phẩm.

**Giá trị nghiệp vụ:** Khách kiểm tra loại chứng nhận, đơn vị cấp, phạm vi và hiệu lực thay vì chỉ nhìn thấy một biểu tượng không có căn cứ.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị chứng nhận còn hiệu lực
  Given chứng nhận đã được xác nhận, còn hiệu lực và thuộc đúng Product/SKU/Batch
  When khách mở mục chứng nhận
  Then hệ thống hiển thị loại, số/mã, đơn vị cấp, mức/xếp hạng nếu có, thời gian và phạm vi
  And thể hiện nguồn cùng thời điểm xác minh gần nhất

Scenario: Chứng nhận đã hết hạn
  Given chứng nhận từng hợp lệ nhưng đã qua ngày hết hiệu lực
  When khách xem hồ sơ
  Then hệ thống không trình bày chứng nhận như đang còn hiệu lực
  And hiển thị trạng thái hết hạn hoặc lưu lịch sử theo chính sách công khai

Scenario: Chứng nhận không đúng phạm vi
  Given chứng nhận thuộc Product, cơ sở hoặc Batch khác
  When Manager cố liên kết hoặc công khai
  Then hệ thống từ chối liên kết không có căn cứ
  And không hiển thị chứng nhận trên hồ sơ hiện tại

Scenario: Provider xác thực thành công
  Given chứng nhận có mã và EXT-10 khả dụng
  When hệ thống nhận kết quả xác thực hợp lệ
  Then lưu kết quả, nguồn và thời điểm xác thực
  And vẫn tuân theo bước phê duyệt công khai nội bộ

Scenario: Provider không tìm thấy hoặc trả dữ liệu mâu thuẫn
  Given EXT-10 không xác nhận mã hoặc dữ liệu khác với hồ sơ nội bộ
  When Manager kiểm tra
  Then chứng nhận được đánh dấu cần xem xét/không xác minh
  And không tự công khai như chứng nhận hợp lệ

Scenario: Tài liệu chứng nhận có dữ liệu nhạy cảm
  Given bản tài liệu gốc chứa trường không được phép công khai
  When khách xem chứng nhận
  Then hệ thống chỉ cung cấp bản/thuộc tính đã được duyệt
  And không mặc nhiên cho tải tài liệu nội bộ gốc
```

### US-OCOP-04 — Quản lý nguồn gốc theo sản phẩm/lô

**Actor:** Manager được cấp quyền (`ACT-12`/`ACT-15` — cần chốt)
**Ưu tiên:** Must Have — 3rd
**Phát hành:** MVP

> Là Manager, tôi muốn quản lý thông tin nguồn gốc của từng sản phẩm/lô hàng.

**Giá trị nghiệp vụ:** Doanh nghiệp duy trì hồ sơ có cấu trúc, đúng đối tượng và có phiên bản để xuất bản thông tin đáng tin cậy cho khách.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo hồ sơ truy xuất cấp Product
  Given Product/SKU tồn tại và Manager có quyền
  When Manager nhập dữ liệu nguồn gốc hợp lệ rồi lưu
  Then hệ thống tạo Traceability Profile ở trạng thái `DRAFT`
  And liên kết đúng Product/SKU mà không tự công khai

Scenario: Tạo hồ sơ truy xuất theo Batch
  Given Batch tồn tại trong EPIC 09 và Manager có quyền
  When Manager tạo hồ sơ cho Batch
  Then hệ thống liên kết đúng Batch, SKU và Product
  And dùng mã lô/NSX/HSD từ nguồn Batch thay vì tạo bản sao tùy ý

Scenario: Ngăn liên kết sai Product, SKU hoặc Batch
  Given Batch không thuộc SKU/Product được chọn
  When Manager lưu hồ sơ
  Then hệ thống từ chối quan hệ không nhất quán
  And không tạo hồ sơ có thể gây truy xuất nhầm lô

Scenario: Tránh hồ sơ công khai trùng phạm vi
  Given đã có hồ sơ hiện hành cho cùng đối tượng/phạm vi
  When Manager tạo thêm hồ sơ chồng lấn
  Then hệ thống cảnh báo hoặc từ chối theo quy tắc phiên bản
  And không để hai hồ sơ cùng được hiểu là nguồn hiện hành

Scenario: Sửa hồ sơ đã công khai
  Given hồ sơ đang `PUBLISHED`
  When Manager thay dữ liệu nguồn gốc quan trọng
  Then hệ thống tạo revision mới ở trạng thái `DRAFT`
  And bản công khai cũ giữ nguyên cho đến khi revision mới được duyệt

Scenario: Hiệu chỉnh liên kết QR
  Given QR đã được phát hành/in trên sản phẩm
  When Manager yêu cầu đổi hồ sơ đích
  Then hệ thống yêu cầu quyền và lý do nghiệp vụ
  And không cho liên kết mã sang Batch khác một cách âm thầm
  And lưu trước/sau để Audit

Scenario: Nhân viên không có quyền
  Given nhân viên không có quyền quản lý traceability
  When nhân viên tạo, sửa hoặc liên kết QR
  Then hệ thống từ chối
  And không thay đổi hồ sơ
```

### US-OCOP-05 — Cập nhật nhà cung cấp và nguồn nguyên liệu

**Actor:** Manager được cấp quyền (`ACT-15` là vai trò đề xuất)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Manager, tôi muốn cập nhật thông tin nhà cung cấp/nguyên liệu để đảm bảo dữ liệu truy xuất chính xác.

**Giá trị nghiệp vụ:** Mỗi nguyên liệu/lô có nguồn, khoảng hiệu lực và bằng chứng rõ ràng, đồng thời không sửa hồi tố nguồn của Batch lịch sử.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Liên kết nguồn nguyên liệu hợp lệ
  Given Supplier/nguồn và nguyên liệu tồn tại hoặc đã được xác minh theo chính sách
  When Manager liên kết nguồn với Product/SKU/Batch cùng khoảng hiệu lực
  Then hệ thống lưu bản ghi nguồn gốc ở trạng thái chưa công khai
  And ghi rõ nguyên liệu, nguồn, phạm vi và thời gian áp dụng

Scenario: Một nguyên liệu có nhiều nguồn theo thời kỳ
  Given nguồn cung thay đổi theo mùa hoặc thời gian
  When Manager thêm nguồn mới với khoảng hiệu lực hợp lệ
  Then hệ thống giữ được lịch sử từng nguồn
  And xác định được nguồn áp dụng cho Batch tương ứng

Scenario: Khoảng hiệu lực mâu thuẫn
  Given các bản ghi cùng nguyên liệu/phạm vi có khoảng hiệu lực chồng lấn không được phép
  When Manager lưu thay đổi
  Then hệ thống từ chối hoặc yêu cầu giải quyết xung đột
  And không tự chọn một nguồn ngẫu nhiên để công khai

Scenario: Supplier Profile thay đổi
  Given EPIC 27 cập nhật tên, địa chỉ hoặc trạng thái Supplier
  When hồ sơ nguồn gốc hiện tại được làm mới
  Then hệ thống phân biệt dữ liệu Supplier hiện tại với snapshot của Batch lịch sử
  And không làm mất căn cứ đã dùng cho hồ sơ cũ

Scenario: Nguồn bị đình chỉ hoặc không còn hợp lệ
  Given nguồn/Supplier bị đánh dấu không còn hợp lệ
  When hệ thống đánh giá các hồ sơ liên quan
  Then hồ sơ chưa công khai bị chặn duyệt theo chính sách
  And hồ sơ đã công khai được đưa vào quy trình xem xét/đình chỉ thay vì âm thầm tiếp tục

Scenario: Người dùng không có quyền
  Given nhân viên không có quyền quản lý nguồn nguyên liệu
  When nhân viên thêm hoặc sửa dữ liệu
  Then hệ thống từ chối
  And không tiết lộ thông tin Supplier nội bộ ngoài quyền
```

### US-OCOP-06 — Kiểm tra và xác nhận trước khi công khai

**Actor:** Manager được cấp quyền (`ACT-12`/`ACT-15` — cần chốt)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Manager, tôi muốn kiểm tra và xác nhận nguồn dữ liệu nguồn gốc trước khi công khai cho khách hàng.

**Giá trị nghiệp vụ:** Chỉ hồ sơ đủ dữ liệu, đúng phạm vi và có bằng chứng được đưa ra công khai; mọi quyết định có người chịu trách nhiệm và lịch sử.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Gửi hồ sơ đủ dữ liệu để duyệt
  Given revision `DRAFT` có đầy đủ trường và nguồn bắt buộc
  When Manager có quyền gửi duyệt
  Then revision chuyển `IN_REVIEW`
  And bị khóa khỏi chỉnh sửa âm thầm trong lúc review

Scenario: Phê duyệt và công khai hồ sơ hợp lệ
  Given revision đang `IN_REVIEW` và mọi kiểm tra bắt buộc đạt
  When người có quyền phê duyệt xác nhận
  Then revision chuyển `APPROVED` rồi `PUBLISHED` theo lịch công khai
  And QR/trang công khai trỏ tới đúng revision hiện hành
  And lưu actor, thời điểm và căn cứ

Scenario: Từ chối hồ sơ thiếu hoặc sai dữ liệu
  Given revision không đáp ứng chính sách
  When người duyệt từ chối với lý do
  Then revision chuyển `REJECTED`
  And lý do được lưu để người soạn sửa trên revision phù hợp
  And bản công khai trước đó không bị thay đổi ngoài ý muốn

Scenario: Kết quả Provider mâu thuẫn
  Given dữ liệu nội bộ khác kết quả xác thực từ EXT-10
  When người duyệt đánh giá hồ sơ
  Then hệ thống thể hiện rõ sai lệch và nguồn/thời điểm
  And không cho công khai như đã xác nhận cho đến khi được giải quyết theo chính sách

Scenario: Hồ sơ thay đổi khi đang được duyệt
  Given revision đã thay đổi sau khi người duyệt tải dữ liệu
  When người duyệt gửi quyết định trên phiên bản cũ
  Then hệ thống không áp dụng quyết định lên dữ liệu mới hơn
  And yêu cầu tải lại/đối chiếu

Scenario: Đình chỉ hồ sơ đã công khai
  Given phát hiện chứng nhận hết hiệu lực, dữ liệu sai hoặc nguồn bị thu hồi
  When người có quyền đình chỉ với lý do
  Then hồ sơ chuyển `SUSPENDED` và không tiếp tục hiển thị như hợp lệ
  And QR vẫn dẫn tới thông báo an toàn thay vì tái sử dụng cho hồ sơ khác

Scenario: Người không có quyền phê duyệt
  Given nhân viên chỉ có quyền tạo/sửa nhưng không có quyền duyệt
  When nhân viên cố xác nhận hoặc công khai
  Then hệ thống từ chối
  And revision giữ nguyên trạng thái
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-OCOP-01` | Cung cấp mã/liên kết QR công khai ổn định, ánh xạ đúng Traceability Profile ở cấp Product/SKU hoặc Batch. | `US-OCOP-01`, `BR-BATCH-01` |
| `FR-OCOP-02` | Cho Guest/Customer xem hồ sơ đã công khai mà không cần đăng nhập; không lộ ID nội bộ hoặc dữ liệu nháp. | `US-OCOP-01` |
| `FR-OCOP-03` | Xử lý QR sai, hồ sơ đình chỉ và liên kết Batch không nhất quán bằng trạng thái an toàn, không hiển thị dữ liệu cũ như hợp lệ. | `US-OCOP-01` |
| `FR-OCOP-04` | Hiển thị dữ liệu xác nhận gần nhất kèm nguồn/thời điểm/trạng thái khi Provider tạm lỗi, nếu chính sách cho phép. | `US-OCOP-01`, `US-OCOP-03` |
| `FR-OCOP-05` | Hiển thị vùng nguyên liệu, nhà sản xuất và quy trình theo đúng phạm vi Product/SKU/Batch và trạng thái công khai. | `US-OCOP-02` |
| `FR-OCOP-06` | Hỗ trợ nhiều nguyên liệu/nguồn và giữ snapshot lịch sử theo Batch; không thay hồi tố khi nguồn hiện tại đổi. | `US-OCOP-02`, `US-OCOP-05` |
| `FR-OCOP-07` | Chỉ công khai trường/media đã duyệt; che hợp đồng, giá, liên hệ và dữ liệu nội bộ. | `US-OCOP-02`, `BR-AUTH-04` |
| `FR-OCOP-08` | Quản lý chứng nhận với loại, mã, đơn vị cấp, mức/xếp hạng, thời gian, phạm vi, nguồn và trạng thái xác minh. | `US-OCOP-03` |
| `FR-OCOP-09` | Không hiển thị chứng nhận hết hạn, sai phạm vi hoặc chưa xác minh như đang hợp lệ. | `US-OCOP-03` |
| `FR-OCOP-10` | Lưu kết quả xác thực EXT-10 riêng với nguồn/thời điểm; kết quả ngoài không tự công khai hồ sơ. | `US-OCOP-03`, `US-OCOP-06` |
| `FR-OCOP-11` | Cho Manager có quyền tạo hồ sơ/revision cấp Product/SKU/Batch ở trạng thái Draft. | `US-OCOP-04`, `BR-AUTH-04` |
| `FR-OCOP-12` | Kiểm tra quan hệ Product–SKU–Batch và dùng mã lô/NSX/HSD từ EPIC 09 thay vì bản sao có thể mâu thuẫn. | `US-OCOP-04`, `BR-PROD-01`, `BR-PROD-03`, `BR-BATCH-02` |
| `FR-OCOP-13` | Ngăn nhiều hồ sơ hiện hành chồng lấn cho cùng phạm vi và tạo revision mới khi sửa nội dung đã công khai. | `US-OCOP-04` |
| `FR-OCOP-14` | Kiểm soát thay đổi liên kết QR đã phát hành, yêu cầu lý do và không tái sử dụng âm thầm sang Batch khác. | `US-OCOP-04`, `BR-AUDIT-01` |
| `FR-OCOP-15` | Liên kết nguyên liệu với Supplier/nguồn, phạm vi và khoảng hiệu lực; phát hiện khoảng hiệu lực mâu thuẫn. | `US-OCOP-05` |
| `FR-OCOP-16` | Phân biệt Supplier Profile hiện tại với snapshot nguồn áp dụng cho Batch; giữ lịch sử khi nguồn thay đổi. | `US-OCOP-05`, `EPIC 27` |
| `FR-OCOP-17` | Đưa hồ sơ liên quan nguồn/chứng nhận mất hiệu lực vào quy trình xem xét hoặc đình chỉ theo chính sách. | `US-OCOP-03`, `US-OCOP-05`, `US-OCOP-06` |
| `FR-OCOP-18` | Kiểm tra dữ liệu/nguồn bắt buộc trước khi gửi duyệt và không cho sửa âm thầm revision đang review. | `US-OCOP-06`, `BR-PROD-04` |
| `FR-OCOP-19` | Cho actor có quyền phê duyệt/từ chối với lý do; chỉ revision Approved được công khai theo lịch. | `US-OCOP-06`, `BR-AUTH-04`, `BR-PROD-04` |
| `FR-OCOP-20` | Phát hiện xung đột khi hồ sơ thay đổi trong lúc review để quyết định không áp dụng lên phiên bản cũ. | `US-OCOP-06` |
| `FR-OCOP-21` | Đình chỉ hồ sơ công khai có lý do và giữ QR dẫn tới trạng thái an toàn, không làm mất lịch sử. | `US-OCOP-01`, `US-OCOP-06` |
| `FR-OCOP-22` | Kiểm soát riêng quyền tạo/sửa/gửi duyệt/phê duyệt/công khai/đình chỉ và quyền xem dữ liệu nguồn nội bộ. | `US-OCOP-04~06`, `BR-AUTH-03`, `BR-AUTH-04` |
| `FR-OCOP-23` | Ghi Audit cho sửa dữ liệu quan trọng, quyết định duyệt, công khai/đình chỉ và đổi liên kết QR. | `US-OCOP-04~06`, `BR-AUDIT-01` |
| `FR-OCOP-24` | Duy trì trang công khai khi tích hợp ngoài lỗi ở mức có thể, nhưng luôn nêu rõ độ mới/trạng thái xác minh và không bịa dữ liệu. | `US-OCOP-01~03`, `EXT-10` |

## 6. Vòng đời và quy tắc dữ liệu

### 6.1. Vòng đời Traceability Revision

```text
DRAFT → IN_REVIEW → APPROVED → PUBLISHED → ARCHIVED
            │                       │
            └──► REJECTED           └──► SUSPENDED
```

- `DRAFT`: đang chuẩn bị, chỉ người có quyền nội bộ được xem/sửa.
- `IN_REVIEW`: đã gửi kiểm tra; thay đổi nội dung phải quay lại revision/quy trình phù hợp.
- `APPROVED`: đã được xác nhận nội bộ nhưng có thể chưa tới lịch công khai.
- `PUBLISHED`: revision hiện hành được phép hiển thị qua QR/trang nguồn gốc.
- `REJECTED`: không đạt, có lý do; không được công khai.
- `SUSPENDED`: từng công khai nhưng bị đình chỉ do rủi ro/sai lệch/hết hiệu lực; QR hiển thị trạng thái an toàn.
- `ARCHIVED`: lịch sử không còn hiện hành nhưng được bảo toàn để truy vết.

### 6.2. Mô hình phạm vi dữ liệu

```text
Product
  └── SKU/Variant
        └── Batch/Lot
              ├── Ingredient Source Snapshot
              ├── Producer / Process Record
              └── Certificate / Evidence
```

- Dữ liệu cấp Product được kế thừa xuống SKU/Batch chỉ khi chính sách cho phép và phải thể hiện rõ phạm vi.
- Dữ liệu riêng Batch được ưu tiên cho trang QR theo Batch; không bị ghi đè bởi mô tả Product chung.
- Mỗi dữ kiện công khai cần có nguồn, phạm vi, trạng thái xác nhận, revision và thời điểm hiệu lực/cập nhật phù hợp.
- Không dùng nội dung marketing thay thế dữ kiện traceability; nội dung trình bày có thể thuộc EPIC 20 nhưng phải tham chiếu dữ liệu đã công khai của EPIC 19.

### 6.3. QR và tính toàn vẹn lịch sử

- Mã công khai không nên để lộ ID tuần tự/nội bộ hoặc cho phép suy đoán hồ sơ chưa công khai.
- Một mã QR đã gắn/in cho Batch không được tái sử dụng cho Batch khác; hiệu chỉnh cần quy trình có lý do và Audit.
- QR phải tiếp tục trả về trạng thái có kiểm soát khi hồ sơ bị đình chỉ/thu hồi, thay vì lỗi mơ hồ hoặc chuyển sang sản phẩm khác.
- Revision đã dùng làm căn cứ công khai phải bất biến; thay đổi tạo revision mới và giữ quan hệ với bản trước.
- Nếu dữ liệu nguồn thay đổi, hệ thống phân biệt dữ liệu hiện tại với snapshot đã áp dụng cho Batch lịch sử.

### 6.4. Xác minh, công khai và quyền riêng tư

- Xác thực bởi EXT-10, phê duyệt nội bộ và công khai là ba trạng thái/hành động khác nhau.
- Kết quả Provider phải lưu nguồn, tham chiếu, thời điểm, trạng thái và sai lệch; Provider lỗi không được coi là chứng nhận thất bại hay thành công mới.
- Tài liệu chứng nhận/media có thể có bản nội bộ và bản công khai riêng; khách không mặc nhiên được tải file gốc.
- Dữ liệu Supplier công khai phải tối thiểu cần thiết, không lộ hợp đồng, giá mua, thông tin liên hệ riêng hoặc bí mật kinh doanh.
- Quyết định duyệt/từ chối/đình chỉ cần actor, thời điểm, lý do, revision và dữ liệu trước/sau theo chính sách Audit.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-OCOP-01` | `BR-BATCH-01`, `BR-AUTH-04` | EPIC 04, 09, EXT-10 | QR đúng Product/SKU/Batch, Guest xem được bản công khai; mã sai/đình chỉ/provider lỗi không gây hiển thị sai. |
| `US-OCOP-02` | `BR-PROD-04`, `BR-BATCH-02` | EPIC 04, 09, 20, 27 | Nguồn/nguyên liệu/quy trình đúng phạm vi và snapshot; dữ liệu chưa duyệt/nội bộ không bị công khai. |
| `US-OCOP-03` | `BR-PROD-04`, `BR-AUDIT-01` | EXT-09, EXT-10, EPIC 23 | Chứng nhận đúng phạm vi/hiệu lực, có nguồn xác minh; sai lệch hoặc hết hạn không hiển thị như hợp lệ. |
| `US-OCOP-04` | `BR-PROD-01`, `BR-PROD-03`, `BR-BATCH-01`, `BR-AUTH-04` | EPIC 04, 09, 22, 23 | Hồ sơ đúng Product–SKU–Batch, có revision, không trùng hiện hành và liên kết QR được kiểm soát. |
| `US-OCOP-05` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 09, 27 | Nguồn có phạm vi/hiệu lực; Supplier thay đổi không sửa hồi tố Batch và nguồn mất hiệu lực được rà soát. |
| `US-OCOP-06` | `BR-PROD-04`, `BR-AUTH-03`, `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 22, 23, EXT-10 | Chỉ người có quyền duyệt/công khai; sai lệch bị chặn; xung đột phiên bản và đình chỉ được xử lý an toàn. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Ánh xạ actor “Manager” cho `US-OCOP-04~06`: `ACT-12`, `ACT-15` hay vai trò mới; ai tạo, ai duyệt và có bắt buộc phân tách nhiệm vụ không.
- QR được cấp theo Product, SKU, Batch hay từng đơn vị bao gói; ai tạo/in/kích hoạt và quy trình xử lý mã in sai/thất lạc/giả mạo.
- Bộ trường traceability bắt buộc cho từng cấp; dữ liệu nào kế thừa và dữ liệu nào phải có riêng theo Batch.
- Mô hình nguyên liệu nhiều cấp, vùng địa lý, nhà sản xuất/cơ sở, công đoạn và quan hệ với Supplier/PO của EPIC 27.
- Danh mục chứng nhận, trường bắt buộc, phạm vi, ngày hiệu lực/hết hạn và quy trình gia hạn/thu hồi.
- Nguồn nào có thẩm quyền khi dữ liệu nội bộ khác EXT-10; SLA xử lý sai lệch và điều kiện cho phép dùng kết quả xác minh gần nhất.
- Quy trình duyệt: số cấp, quyền tự duyệt, lý do từ chối, lịch công khai và điều kiện khôi phục hồ sơ `SUSPENDED`.
- Chính sách hiển thị hồ sơ lịch sử/chứng nhận hết hạn và thông báo khi QR của sản phẩm thu hồi/đình chỉ.
- Loại/kích thước media, bản công khai so với bản gốc, kiểm tra an toàn, watermark/tải xuống và thời hạn lưu.
- Dữ liệu Supplier/nghệ nhân/cơ sở nào được công khai; căn cứ đồng ý, bảo vệ PII và bí mật kinh doanh.
- Múi giờ/ngày hiệu lực và cách bảo toàn snapshot khi hồ sơ Product, Supplier hoặc Batch được hiệu chỉnh.
- Yêu cầu đa ngôn ngữ, accessibility, SEO và cache của trang QR; nguồn chịu trách nhiệm nội dung dịch.
- Chỉ số theo dõi lượt quét và ranh giới với Analytics/Marketing; không dùng định danh người quét vượt mục đích cho phép.

## 9. UI/UX Reference

- Trang QR công khai cần ưu tiên Product/SKU, Batch, NSX/HSD, trạng thái xác minh và thời điểm cập nhật; cấp dữ liệu phải được ghi rõ.
- Nguồn nguyên liệu nên hiển thị theo từng nguyên liệu/vùng/nhà sản xuất; tránh bản đồ hoặc mô tả khiến khách hiểu sai độ chính xác.
- Chứng nhận cần hiển thị loại, mã, đơn vị cấp, phạm vi, hiệu lực và trạng thái xác minh thay vì chỉ logo/badge.
- QR sai, hồ sơ đình chỉ, Provider lỗi và dữ liệu chưa sẵn sàng cần có trạng thái riêng, thông điệp an toàn và kênh hỗ trợ.
- Manager Workspace cần thể hiện cây Product–SKU–Batch, nguồn dữ liệu, revision, trạng thái duyệt và khác biệt trước/sau.
- Màn hình review cần đặt dữ liệu nội bộ cạnh kết quả EXT-10, làm nổi bật sai lệch và yêu cầu lý do khi từ chối/đình chỉ.
- Khu vực media/tài liệu cần phân biệt bản nội bộ và công khai, trạng thái xử lý và quyền tải xuống.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

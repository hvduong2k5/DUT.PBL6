# EPIC 20 — Content & SEO

## 1. Mục tiêu Epic

Epic này cho phép Content Manager tạo, chỉnh sửa, quản lý media và tối ưu thông tin SEO cho bài viết/sản phẩm; nội dung phải trải qua quy trình duyệt trước khi công khai. Website cung cấp URL ổn định, metadata, liên kết chuẩn và danh mục URL công khai nhất quán để khách hàng lẫn công cụ tìm kiếm tìm thấy nội dung phù hợp.

EPIC 20 sở hữu Article, Content Revision, Publication, Content Media Reference và SEO Metadata. Epic không sở hữu dữ liệu thương mại Product/SKU, dữ kiện OCOP/Batch, kế hoạch chiến dịch Marketing, Analytics nguồn ngoài hoặc tệp media vật lý; các trách nhiệm đó thuộc EPIC 04, 19, 21, EXT-07 và EXT-09.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP v1.0** | `US-CONTENT-01` đến `US-CONTENT-07` | Đội nội dung tạo/duyệt/xuất bản bài viết, media và SEO cho bài viết/sản phẩm với URL công khai nhất quán. |
| **Giai đoạn 2 — đề xuất** | `US-AI-03` | AI hỗ trợ tạo tiêu đề, mô tả và outline dưới dạng bản nháp có người kiểm duyệt. |

EPIC 20 được xếp **MVP** trong Epic Map dù các story `US-CONTENT-01~07` có Priority **Should Have**. `US-AI-03` là **Could Have** và Product Backlog chưa gán release riêng; Giai đoạn 2 là đề xuất cần Product Owner xác nhận.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và hệ thống ngoài

| Actor / hệ thống | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Xem bài viết và thông tin SEO đã được phép công khai. |
| Registered Customer (`ACT-02`) | Xem nội dung công khai tương tự Guest và tiếp tục hành trình liên quan. |
| Content Manager (`ACT-13`) | Tạo/sửa revision, quản lý media và SEO, gửi nội dung đi duyệt. |
| Manager được cấp quyền | Duyệt, từ chối, công khai hoặc ẩn nội dung theo chính sách. |
| Sales Manager (`ACT-12`) | Phối hợp xác nhận dữ liệu sản phẩm/SEO nếu được giao quyền, không thay thế chủ sở hữu Catalog. |
| Search Engine Crawler | Tiêu thụ trang, metadata và danh mục URL công khai; chưa có ID trong Actor Registry. |
| Analytics Platform (`EXT-07`) | Cung cấp dữ liệu hiệu quả bên ngoài nếu tích hợp; phải phân biệt với dữ liệu nội bộ. |
| AI Provider (`EXT-08`) | Hỗ trợ tạo bản nháp nội dung khi tính năng AI được bật. |
| Media / Object Storage (`EXT-09`) | Lưu ảnh/video và cung cấp tài nguyên theo trạng thái/quyền. |

> Product Backlog dùng actor chung “Manager” cho `US-CONTENT-07` nhưng chưa ánh xạ sang một Actor ID cụ thể. Vai trò duyệt cuối cùng và yêu cầu phân tách người soạn/người duyệt cần được chốt.

### 3.2. Business Rules và NFR áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-03` | Tài khoản nội bộ phải có Role phù hợp trước khi truy cập chức năng quản trị. | Content Manager/Manager phải có Role hợp lệ trước khi thao tác CMS. |
| `BR-AUTH-04` | Nhân viên chỉ thực hiện thao tác thuộc quyền được cấp. | Quyền soạn, duyệt, xuất bản, ẩn, sửa SEO và quản lý media phải tách biệt. |
| `BR-PROD-01` | Một Product có thể có nhiều Variant/SKU. | Product SEO phải xác định metadata ở cấp Product và cách xử lý SKU/Variant, tránh trang trùng lặp. |
| `BR-PROD-03` | Sản phẩm thực phẩm phải có thông tin NSX/HSD phù hợp. | Nội dung không được tự tạo dữ kiện sản phẩm/Batch mâu thuẫn với nguồn nghiệp vụ. |
| `BR-PROD-04` | Thông tin sản phẩm hiển thị cho khách phải được quản lý/phê duyệt. | Product SEO hoặc bài viết không được công khai dữ kiện Product chưa được phép. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Duyệt/từ chối/xuất bản/ẩn và thay đổi URL hoặc SEO quan trọng phải truy vết được. |
| `NFR-01` | Các thao tác xem/tìm kiếm phổ biến cần phản hồi phù hợp. | Trang công khai và dữ liệu SEO không được gây trải nghiệm tải kém; ngưỡng đo thuộc NFR/Test Plan. |
| `NFR-02` | Website tối ưu mobile và tương thích ứng dụng. | Nội dung/media cần hiển thị phù hợp trên các kích thước màn hình được hỗ trợ. |
| `NFR-05` | Người dùng chỉ truy cập dữ liệu/chức năng thuộc quyền. | Preview, draft, media và lịch sử revision không được lộ qua URL trực tiếp. |
| `NFR-08` | Dữ liệu cá nhân chỉ được dùng theo mục đích được phép. | Nội dung/media/AI input không được công khai hoặc xử lý PII trái mục đích. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 03 — Product Discovery`: hiển thị và tìm sản phẩm; sử dụng metadata công khai nhưng không sở hữu quy trình biên tập.
- `EPIC 04 — Product, Variant & SKU`: nguồn Product/SKU, giá, trạng thái bán và dữ liệu thực phẩm; EPIC 20 chỉ bổ sung SEO/presentation.
- `EPIC 19 — OCOP Traceability`: nguồn dữ kiện chứng nhận, vùng nguyên liệu và Batch đã xác nhận; bài viết không thay thế hồ sơ truy xuất.
- `EPIC 21 — Marketing`: sở hữu kế hoạch/campaign và phân phối truyền thông; có thể tham chiếu nội dung đã xuất bản của EPIC 20.
- `EPIC 22, 23 — Administration, Audit & Security`: cung cấp Role/Permission và Audit Log cho CMS.
- `EPIC 28 — Notification`: gửi thông báo nội bộ về duyệt/xuất bản nếu cần; EPIC 20 chỉ phát sinh sự kiện nghiệp vụ.
- `EXT-07 — Analytics Platform`: cung cấp metrics bên ngoài; dữ liệu phải được ghi rõ nguồn và không coi là giao dịch nội bộ.
- `EXT-08`, `EXT-09`: hỗ trợ AI và Media; lỗi tích hợp không được làm mất bản nháp hoặc công khai nội dung chưa duyệt.

## 4. User Stories chi tiết

### US-CONTENT-01 — Tạo bài viết

**Actor:** Content Manager (`ACT-13`)
**Ưu tiên:** Should Have
**Phát hành:** MVP v1.0

> Là Content Manager, tôi muốn tạo bài viết để truyền tải câu chuyện văn hóa Huế và thương hiệu.

**Giá trị nghiệp vụ:** Đội nội dung xây dựng tài sản biên tập có cấu trúc, lưu dưới dạng bản nháp và sẵn sàng cho quy trình kiểm duyệt.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo bài viết nháp hợp lệ
  Given Content Manager có quyền tạo nội dung
  When nhập tiêu đề, nội dung, loại/chủ đề và thông tin bắt buộc hợp lệ
  Then hệ thống tạo Article cùng revision đầu tiên ở trạng thái `DRAFT`
  And gắn đúng tác giả và thời điểm
  And không tự công khai bài viết

Scenario: Thiếu dữ liệu bắt buộc
  Given Content Manager đang tạo bài viết
  When tiêu đề, nội dung hoặc trường bắt buộc bị thiếu/không hợp lệ
  Then hệ thống từ chối gửi duyệt hoặc lưu theo quy tắc tương ứng
  And chỉ rõ trường cần sửa

Scenario: Lưu bản nháp đang làm
  Given bài viết chưa hoàn tất
  When Content Manager lưu bản nháp hợp lệ
  Then hệ thống bảo toàn nội dung hiện tại để tiếp tục sau
  And bản nháp không xuất hiện trên trang công khai hoặc danh mục URL indexable

Scenario: Liên kết dữ liệu Product hoặc OCOP
  Given bài viết tham chiếu Product hoặc hồ sơ OCOP
  When Content Manager chọn đối tượng hợp lệ
  Then hệ thống lưu tham chiếu tới nguồn tương ứng
  And không sao chép quyền sửa dữ liệu nguồn sang bài viết

Scenario: Dữ kiện nguồn chưa được công khai
  Given Product hoặc Traceability Profile chưa được phép công khai
  When Content Manager dùng dữ kiện đó trong bài viết chuẩn bị xuất bản
  Then hệ thống cảnh báo hoặc chặn theo chính sách
  And không coi tham chiếu nội bộ là bằng chứng nội dung đã được duyệt

Scenario: Người dùng không có quyền tạo
  Given nhân viên không có quyền tạo Article
  When nhân viên truy cập hoặc gửi nội dung
  Then hệ thống từ chối
  And không tạo bản nháp
```

### US-CONTENT-02 — Chỉnh sửa, xuất bản hoặc ẩn bài viết

**Actor:** Content Manager (`ACT-13`)
**Ưu tiên:** Should Have
**Phát hành:** MVP v1.0

> Là Content Manager, tôi muốn chỉnh sửa, xuất bản hoặc ẩn bài viết.

**Giá trị nghiệp vụ:** Nội dung được cập nhật có phiên bản, công khai đúng thời điểm và có thể gỡ khỏi hiển thị mà không xóa mất lịch sử.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Chỉnh sửa bài viết chưa công khai
  Given revision đang `DRAFT` và Content Manager có quyền
  When sửa nội dung hợp lệ rồi lưu
  Then hệ thống cập nhật revision theo quy tắc phiên bản
  And lưu người/thời điểm chỉnh sửa

Scenario: Sửa bài viết đã xuất bản
  Given Article đang có revision `PUBLISHED`
  When Content Manager thay nội dung
  Then hệ thống tạo revision `DRAFT` mới
  And bản công khai cũ giữ nguyên đến khi revision mới được duyệt/xuất bản

Scenario: Xuất bản revision đã được duyệt
  Given revision ở trạng thái `APPROVED` và actor có quyền xuất bản
  When actor xác nhận công khai ngay
  Then revision chuyển `PUBLISHED`
  And trang công khai dùng đúng revision hiện hành
  And revision cũ được lưu lịch sử

Scenario: Lên lịch xuất bản
  Given revision đã được duyệt và thời điểm tương lai hợp lệ
  When actor có quyền đặt lịch
  Then revision chuyển `SCHEDULED`
  And chỉ được công khai khi đến đúng thời điểm/lịch đã cấu hình

Scenario: Ẩn bài viết đang công khai
  Given Article đang `PUBLISHED`
  When actor có quyền ẩn bài viết và cung cấp lý do nếu bắt buộc
  Then Article chuyển `HIDDEN`
  And không còn xuất hiện trong danh sách công khai, sitemap hoặc kết quả nội bộ dành cho khách
  And lịch sử vẫn được bảo toàn

Scenario: Không cho xuất bản revision chưa duyệt
  Given revision là `DRAFT`, `IN_REVIEW` hoặc `REJECTED`
  When người dùng cố công khai
  Then hệ thống từ chối
  And giữ nguyên nội dung công khai hiện tại

Scenario: Hai người chỉnh sửa đồng thời
  Given revision đã thay đổi sau khi một Content Manager tải dữ liệu
  When người đó lưu trên phiên bản cũ
  Then hệ thống không ghi đè âm thầm bản mới hơn
  And yêu cầu đối chiếu/tải lại trước khi tiếp tục
```

### US-CONTENT-03 — Quản lý hình ảnh/video trong bài viết

**Actor:** Content Manager (`ACT-13`)
**Ưu tiên:** Should Have
**Phát hành:** MVP v1.0

> Là Content Manager, tôi muốn quản lý hình ảnh/video trong bài viết.

**Giá trị nghiệp vụ:** Bài viết có media đúng ngữ cảnh, an toàn và accessible; tài nguyên được quản lý mà không tạo liên kết hỏng hoặc xóa nhầm nội dung đang dùng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tải media hợp lệ
  Given Content Manager có quyền quản lý media
  And tệp đáp ứng loại, dung lượng, số lượng và chính sách an toàn
  When tải tệp lên
  Then hệ thống lưu qua Media Storage
  And tạo Media Reference có trạng thái xử lý rõ ràng

Scenario: Tệp không hợp lệ
  Given Content Manager chọn tệp sai loại, vượt giới hạn hoặc không đạt kiểm tra an toàn
  When tải lên
  Then hệ thống từ chối tệp
  And hiển thị lý do cùng giới hạn áp dụng

Scenario: Tải media bị gián đoạn
  Given quá trình tải chưa hoàn tất
  When kết nối hoặc Media Storage lỗi
  Then hệ thống không gắn tham chiếu hỏng vào revision
  And cho phép thử lại mà không tạo tài nguyên trùng ngoài ý muốn

Scenario: Chèn media vào đúng vị trí
  Given media ở trạng thái dùng được
  When Content Manager chèn vào revision và cung cấp mô tả thay thế/chú thích theo yêu cầu
  Then hệ thống lưu vị trí, thứ tự và metadata trình bày
  And preview phản ánh đúng nội dung dự kiến

Scenario: Thiếu mô tả thay thế bắt buộc
  Given ảnh truyền tải thông tin và chính sách yêu cầu alternative text
  When Content Manager gửi duyệt
  Then hệ thống chặn hoặc cảnh báo theo chính sách
  And không tự tạo mô tả sai nội dung

Scenario: Xóa media đang được sử dụng
  Given media đang được tham chiếu bởi revision hoặc bài viết công khai
  When Content Manager yêu cầu xóa/ẩn
  Then hệ thống cảnh báo các nơi đang sử dụng
  And không tạo liên kết hỏng trên bản công khai ngoài ý muốn

Scenario: Media nội bộ bị truy cập trực tiếp
  Given tài nguyên chỉ thuộc Draft hoặc không được phép công khai
  When người không có quyền mở liên kết
  Then hệ thống từ chối
  And không tiết lộ vị trí lưu trữ trực tiếp
```

### US-CONTENT-04 — Tối ưu SEO cho bài viết

**Actor:** Content Manager (`ACT-13`)
**Ưu tiên:** Should Have
**Phát hành:** MVP v1.0

> Là Content Manager, tôi muốn tối ưu tiêu đề, mô tả, từ khóa và thông tin SEO cho bài viết.

**Giá trị nghiệp vụ:** Mỗi bài viết công khai có thông tin tìm kiếm rõ ràng, nhất quán với nội dung và không tạo preview sai lệch.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Lưu metadata SEO hợp lệ
  Given Content Manager đang chỉnh revision
  When nhập SEO title, description, từ khóa/chủ đề và thông tin chia sẻ hợp lệ
  Then hệ thống lưu metadata vào đúng revision
  And preview hiển thị nội dung dự kiến cho kết quả tìm kiếm/chia sẻ

Scenario: Metadata thiếu hoặc vượt giới hạn khuyến nghị
  Given SEO title/description thiếu hoặc không phù hợp giới hạn cấu hình
  When Content Manager gửi duyệt
  Then hệ thống cảnh báo hoặc chặn theo chính sách
  And chỉ rõ trường cần điều chỉnh

Scenario: Metadata không nhất quán với nội dung
  Given tiêu đề/mô tả SEO đưa ra tuyên bố không có trong bài viết hoặc nguồn đã duyệt
  When người dùng review nội dung
  Then hệ thống cho phép đánh dấu cần sửa/từ chối
  And không tự công khai metadata gây hiểu sai

Scenario: Bài viết chưa công khai
  Given Article đang Draft, In Review, Rejected hoặc Hidden
  When crawler/người dùng công khai truy cập
  Then metadata không được trình bày như một trang indexable đang hoạt động
  And URL không xuất hiện trong sitemap công khai

Scenario: Metadata kế thừa mặc định
  Given một trường SEO được phép dùng giá trị mặc định
  When Content Manager xem preview
  Then hệ thống thể hiện rõ giá trị thực tế sẽ được dùng
  And không để giá trị rỗng hoặc nội dung mẫu bị xuất bản ngoài ý muốn
```

### US-CONTENT-05 — Quản lý SEO cho từng sản phẩm

**Actor:** Content Manager (`ACT-13`)
**Ưu tiên:** Should Have
**Phát hành:** MVP v1.0

> Là Content Manager, tôi muốn quản lý thông tin SEO cho từng sản phẩm để sản phẩm có khả năng được tìm thấy trên công cụ tìm kiếm.

**Giá trị nghiệp vụ:** Trang sản phẩm có metadata tìm kiếm phù hợp nhưng vẫn lấy dữ liệu thương mại và traceability từ nguồn đã phê duyệt.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Cập nhật Product SEO hợp lệ
  Given Product tồn tại và Content Manager có quyền SEO
  When nhập metadata hợp lệ cho đúng Product rồi gửi quy trình duyệt
  Then hệ thống lưu Product SEO tách biệt dữ liệu Catalog
  And không thay đổi tên thương mại, giá, SKU hoặc trạng thái bán của EPIC 04

Scenario: Product có nhiều SKU
  Given Product có nhiều Variant/SKU
  When cấu hình metadata/canonical
  Then hệ thống áp dụng quy tắc cấp Product/SKU đã chốt
  And không tạo nhiều trang trùng lặp cạnh tranh ngoài ý muốn

Scenario: Product chưa được công khai
  Given Product hoặc dữ liệu bắt buộc chưa được phép công khai
  When Content Manager hoàn tất SEO
  Then metadata không làm Product xuất hiện công khai trước quy trình Catalog
  And trang không được đưa vào sitemap indexable

Scenario: Dữ liệu Product thay đổi
  Given tên, trạng thái hoặc dữ liệu Product nguồn đã thay đổi
  When trang sản phẩm được dựng lại hoặc Content Manager review SEO
  Then hệ thống dùng dữ liệu nguồn hiện hành được phép
  And cảnh báo metadata thủ công có thể trở nên mâu thuẫn

Scenario: Dữ kiện OCOP trong Product SEO
  Given metadata/nội dung SEO đề cập chứng nhận hoặc nguồn gốc OCOP
  When gửi duyệt
  Then dữ kiện phải tham chiếu thông tin đã công khai của EPIC 19
  And không tự suy diễn mức chứng nhận hoặc Batch

Scenario: Product ngừng bán
  Given Product chuyển tạm ngừng/ngừng kinh doanh
  When người dùng hoặc crawler truy cập URL cũ
  Then hệ thống áp dụng chính sách giữ trang, noindex, archive hoặc redirect đã được chốt
  And không tiếp tục hiển thị khả năng mua sai trạng thái
```

### US-CONTENT-06 — Quản lý URL và thông tin SEO liên quan

**Actor:** Content Manager (`ACT-13`)
**Ưu tiên:** Should Have
**Phát hành:** MVP v1.0

> Là Content Manager, tôi muốn quản lý URL thân thiện, Meta Title, Meta Description và nội dung liên quan SEO.

**Giá trị nghiệp vụ:** URL dễ hiểu, không trùng, có lịch sử chuyển hướng và xuất hiện chính xác trong canonical/sitemap để tránh mất truy cập hoặc nội dung trùng lặp.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo slug thân thiện duy nhất
  Given Content Manager nhập slug hợp lệ cho Article/Product SEO
  When lưu revision
  Then hệ thống kiểm tra tính duy nhất trong phạm vi URL
  And gắn slug với đúng đối tượng/revision theo chính sách

Scenario: Slug trùng hoặc bị dành riêng
  Given slug đã được dùng hoặc thuộc danh sách đường dẫn hệ thống
  When Content Manager lưu
  Then hệ thống từ chối
  And gợi ý điều chỉnh mà không tự ghi đè URL hiện có

Scenario: Đổi URL của nội dung đã công khai
  Given URL cũ đang có người dùng hoặc crawler truy cập
  When URL mới được duyệt/xuất bản
  Then hệ thống tạo quan hệ chuyển hướng từ URL cũ theo chính sách
  And canonical/sitemap dùng URL hiện hành
  And không tạo chuỗi/vòng chuyển hướng ngoài ý muốn

Scenario: URL cũ được yêu cầu tái sử dụng
  Given slug từng là URL công khai hoặc đang làm nguồn redirect
  When người dùng gán cho nội dung khác
  Then hệ thống cảnh báo hoặc từ chối theo chính sách lịch sử URL
  And không chuyển traffic sang đối tượng sai

Scenario: Cập nhật sitemap khi công khai hoặc ẩn
  Given trạng thái công khai hoặc URL đã thay đổi
  When danh mục URL được cập nhật
  Then sitemap chỉ chứa URL canonical đủ điều kiện index
  And thời điểm cập nhật phản ánh thay đổi nguồn

Scenario: Metadata công khai nhất quán
  Given Article/Product SEO đã công khai
  When người dùng hoặc crawler mở URL canonical
  Then trang cung cấp title, description và thông tin liên quan từ đúng revision hiện hành
  And không trộn metadata của đối tượng hoặc phiên bản khác
```

### US-CONTENT-07 — Duyệt nội dung trước khi xuất bản

**Actor:** Manager được cấp quyền (cần Product Owner ánh xạ Actor ID)
**Ưu tiên:** Should Have
**Phát hành:** MVP v1.0

> Là Manager, tôi muốn duyệt nội dung trước khi xuất bản để đảm bảo thông tin chính xác.

**Giá trị nghiệp vụ:** Nội dung, media, SEO và tuyên bố sản phẩm được kiểm tra bởi người có thẩm quyền trước khi đến khách hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Gửi revision đủ điều kiện đi duyệt
  Given revision có đủ nội dung, media/SEO bắt buộc và tham chiếu hợp lệ
  When Content Manager gửi duyệt
  Then revision chuyển `IN_REVIEW`
  And không được sửa âm thầm trong khi quyết định đang chờ

Scenario: Phê duyệt revision hợp lệ
  Given revision đang `IN_REVIEW` và Manager có quyền
  When Manager xác nhận nội dung chính xác
  Then revision chuyển `APPROVED`
  And lưu người duyệt, thời điểm và ghi chú nếu có
  And chưa tự công khai nếu còn lịch xuất bản riêng

Scenario: Từ chối revision
  Given revision còn lỗi nội dung, SEO, media hoặc căn cứ
  When Manager từ chối với lý do
  Then revision chuyển `REJECTED`
  And Content Manager thấy lý do để tạo/chỉnh revision tiếp theo
  And bản đang công khai không bị thay đổi

Scenario: Người soạn tự duyệt khi chính sách cấm
  Given chính sách yêu cầu phân tách người soạn và người duyệt
  When tác giả cố duyệt revision của chính mình
  Then hệ thống từ chối
  And giữ revision ở trạng thái chờ duyệt

Scenario: Revision thay đổi sau khi người duyệt tải
  Given dữ liệu đã thay đổi sau thời điểm Manager mở review
  When Manager gửi quyết định trên phiên bản cũ
  Then hệ thống không áp dụng quyết định lên nội dung mới hơn
  And yêu cầu tải lại/đối chiếu

Scenario: Manager không có quyền đối với loại nội dung
  Given Manager thiếu quyền duyệt Article/Product SEO tương ứng
  When Manager phê duyệt hoặc từ chối
  Then hệ thống từ chối thao tác
  And không tiết lộ dữ liệu ngoài phạm vi
```

### US-AI-03 — Content Assistant

**Actor:** Content Manager (`ACT-13`)
**Ưu tiên:** Could Have
**Phát hành:** Giai đoạn 2 — đề xuất

> Là Content Manager, tôi muốn được AI hỗ trợ tạo tiêu đề, mô tả và outline bài viết Marketing để tăng hiệu suất sản xuất nội dung.

**Giá trị nghiệp vụ:** Content Manager có điểm bắt đầu nhanh hơn nhưng vẫn kiểm soát tính chính xác, giọng thương hiệu và quyết định lưu/xuất bản.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo gợi ý nội dung
  Given Content Manager có quyền dùng AI và cung cấp brief hợp lệ
  When yêu cầu gợi ý tiêu đề, mô tả hoặc outline
  Then AI trả về nội dung được đánh dấu là bản gợi ý
  And không tự lưu đè revision hoặc xuất bản

Scenario: Chấp nhận gợi ý vào bản nháp
  Given AI đã trả về kết quả
  When Content Manager chọn phần muốn sử dụng và xác nhận
  Then hệ thống đưa nội dung vào revision `DRAFT`
  And nội dung tiếp tục qua validation và quy trình duyệt như nội dung do người viết tạo

Scenario: Gợi ý có tuyên bố sản phẩm hoặc OCOP
  Given kết quả AI chứa giá, thành phần, chứng nhận, nguồn gốc hoặc tuyên bố thực phẩm
  When Content Manager review
  Then hệ thống yêu cầu đối chiếu nguồn nghiệp vụ phù hợp
  And không coi nội dung AI là căn cứ xác nhận

Scenario: Dữ liệu vượt ngoài quyền Content Manager
  Given nguồn ngữ cảnh có dữ liệu nội bộ/nhạy cảm ngoài quyền
  When chuẩn bị yêu cầu AI
  Then hệ thống không cung cấp dữ liệu đó cho AI hoặc trả lại qua kết quả
  And AI không trở thành đường vòng vượt Permission

Scenario: AI Provider không khả dụng
  Given dịch vụ AI lỗi, hết thời gian hoặc bị tắt
  When Content Manager yêu cầu gợi ý
  Then hệ thống thông báo tạm không khả dụng
  And bản nháp hiện có không bị mất hoặc thay đổi

Scenario: Nội dung AI không phù hợp
  Given kết quả sai, vi phạm chính sách hoặc không đúng giọng thương hiệu
  When Content Manager từ chối kết quả
  Then hệ thống không đưa nội dung vào revision
  And không ảnh hưởng nội dung hiện có
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-CONTENT-01` | Cho Content Manager có quyền tạo Article và revision Draft với dữ liệu/tác giả/thời điểm bắt buộc. | `US-CONTENT-01`, `BR-AUTH-04` |
| `FR-CONTENT-02` | Cho lưu bản nháp nhưng không để Draft xuất hiện trên trang/sitemap công khai hoặc qua URL trực tiếp không quyền. | `US-CONTENT-01`, `NFR-05` |
| `FR-CONTENT-03` | Cho liên kết Product/OCOP hợp lệ mà không sao chép quyền sửa dữ liệu nguồn; cảnh báo dữ kiện chưa công khai. | `US-CONTENT-01`, `BR-PROD-04` |
| `FR-CONTENT-04` | Khi sửa nội dung đã Published, tạo revision Draft mới và giữ bản công khai cũ đến khi bản mới được duyệt/xuất bản. | `US-CONTENT-02` |
| `FR-CONTENT-05` | Cho xuất bản ngay/lên lịch/ẩn theo quyền; nội dung chưa Approved không được công khai. | `US-CONTENT-02`, `US-CONTENT-07`, `BR-AUTH-04` |
| `FR-CONTENT-06` | Phát hiện chỉnh sửa đồng thời để không ghi đè revision mới hơn. | `US-CONTENT-02`, `US-CONTENT-07` |
| `FR-CONTENT-07` | Nhận media theo loại, dung lượng, số lượng và kiểm tra an toàn; chỉ liên kết tệp tải hoàn tất. | `US-CONTENT-03` |
| `FR-CONTENT-08` | Quản lý vị trí/thứ tự, caption, alternative text và trạng thái công khai của Media Reference trong revision. | `US-CONTENT-03`, `NFR-02` |
| `FR-CONTENT-09` | Phát hiện media đang được sử dụng trước khi xóa/ẩn và ngăn tạo liên kết hỏng ngoài ý muốn. | `US-CONTENT-03` |
| `FR-CONTENT-10` | Không cho người thiếu quyền truy cập media Draft/nội bộ hoặc vị trí lưu trữ trực tiếp. | `US-CONTENT-03`, `NFR-05` |
| `FR-CONTENT-11` | Quản lý SEO title, description, keywords/chủ đề và metadata chia sẻ theo đúng revision Article. | `US-CONTENT-04` |
| `FR-CONTENT-12` | Kiểm tra trường/giới hạn SEO theo cấu hình, hiển thị preview và không xuất bản nội dung mẫu/rỗng ngoài ý muốn. | `US-CONTENT-04` |
| `FR-CONTENT-13` | Chỉ cung cấp metadata indexable/sitemap cho nội dung Published đủ điều kiện. | `US-CONTENT-04`, `US-CONTENT-06` |
| `FR-CONTENT-14` | Quản lý Product SEO tách khỏi Catalog và không cho metadata làm công khai Product chưa được duyệt. | `US-CONTENT-05`, `BR-PROD-04` |
| `FR-CONTENT-15` | Xử lý Product nhiều SKU bằng canonical/indexability theo quy tắc được chốt, tránh trang trùng lặp ngoài ý muốn. | `US-CONTENT-05`, `BR-PROD-01` |
| `FR-CONTENT-16` | Phát hiện metadata Product có thể mâu thuẫn khi nguồn thay đổi và yêu cầu dữ kiện OCOP dựa trên EPIC 19 đã công khai. | `US-CONTENT-05`, `BR-PROD-03`, `BR-PROD-04` |
| `FR-CONTENT-17` | Quản lý slug thân thiện, hợp lệ, duy nhất và danh sách đường dẫn dành riêng trong toàn phạm vi URL. | `US-CONTENT-06` |
| `FR-CONTENT-18` | Khi đổi URL công khai, duy trì redirect/canonical/sitemap nhất quán; ngăn vòng/chuỗi chuyển hướng và tái sử dụng sai slug lịch sử. | `US-CONTENT-06` |
| `FR-CONTENT-19` | Cập nhật sitemap theo Publication/URL và chỉ chứa URL canonical đủ điều kiện cùng thời điểm cập nhật phù hợp. | `US-CONTENT-06` |
| `FR-CONTENT-20` | Cho gửi revision đủ điều kiện đi duyệt; khóa việc sửa âm thầm trong lúc review. | `US-CONTENT-07` |
| `FR-CONTENT-21` | Cho Manager có quyền duyệt/từ chối với lý do và thực thi phân tách nhiệm vụ nếu chính sách yêu cầu. | `US-CONTENT-07`, `BR-AUTH-04` |
| `FR-CONTENT-22` | AI tạo title/description/outline dưới dạng gợi ý; chỉ đưa vào Draft sau xác nhận của Content Manager. | `US-AI-03` |
| `FR-CONTENT-23` | AI không tự xuất bản, không vượt Permission và không được coi là nguồn xác nhận dữ kiện Product/OCOP. | `US-AI-03`, `BR-PROD-04`, `NFR-05` |
| `FR-CONTENT-24` | Lỗi/tắt AI không làm mất hoặc chặn quy trình biên tập thủ công. | `US-AI-03` |
| `FR-CONTENT-25` | Ghi Audit cho duyệt/từ chối/xuất bản/ẩn và thay đổi URL/SEO quan trọng theo chính sách. | `US-CONTENT-02`, `US-CONTENT-05~07`, `BR-AUDIT-01` |
| `FR-CONTENT-26` | Kiểm soát riêng quyền tạo/sửa, media, SEO, preview, duyệt, xuất bản, ẩn và AI. | `US-CONTENT-01~07`, `US-AI-03`, `BR-AUTH-03`, `BR-AUTH-04` |
| `FR-CONTENT-27` | Phát sinh sự kiện Publication cho Consumer liên quan mà không phụ thuộc lưu trạng thái nội dung vào Analytics/Notification bên ngoài. | `US-CONTENT-02`, `US-CONTENT-06`, `EPIC 21`, `EPIC 28` |

## 6. Vòng đời và quy tắc dữ liệu

### 6.1. Vòng đời Content Revision

```text
DRAFT → IN_REVIEW → APPROVED → SCHEDULED → PUBLISHED
            │                      │             │
            └──► REJECTED          └─────────────┼──► HIDDEN
                                                 └──► ARCHIVED
```

- `DRAFT`: revision đang soạn, không được công khai/index.
- `IN_REVIEW`: đã gửi duyệt; thay đổi cần quay lại Draft/revision phù hợp.
- `APPROVED`: đã duyệt nhưng chưa nhất thiết công khai.
- `SCHEDULED`: chờ thời điểm xuất bản đã cấu hình.
- `PUBLISHED`: revision hiện hành được khách/crawler truy cập.
- `REJECTED`: có lý do, không công khai.
- `HIDDEN`: chủ động gỡ khỏi công khai nhưng giữ lịch sử và chính sách URL.
- `ARCHIVED`: revision lịch sử, không còn hiện hành.

### 6.2. Article, Revision và Publication

- Article là định danh nội dung xuyên suốt; Revision là snapshot biên tập; Publication xác định revision, URL và thời điểm đang công khai.
- Chỉ một revision hiện hành được công khai cho một Article/phạm vi ngôn ngữ tại một thời điểm.
- Revision đã Published không sửa tại chỗ; thay đổi tạo revision mới để review và rollback/đối chiếu.
- Preview phải có quyền/token phù hợp, không biến Draft thành URL công khai có thể index và không dùng dữ liệu của revision khác.
- Lịch xuất bản phải dùng timezone được cấu hình và lưu mốc thời gian tuyệt đối để tránh công khai sai thời điểm.

### 6.3. URL, canonical và sitemap

- Slug phải duy nhất trong namespace, không trùng đường dẫn hệ thống hoặc slug lịch sử cần giữ redirect.
- URL đổi phải có quan hệ từ cũ đến mới; không tạo vòng lặp, chuỗi dài hoặc chuyển sang đối tượng không liên quan.
- Canonical phải trỏ tới URL hiện hành được phép index và không mâu thuẫn sitemap.
- Draft, Rejected, Hidden, trang preview và Product chưa công khai không được đưa vào sitemap indexable.
- Khi nội dung bị ẩn/xóa nghiệp vụ, hành vi URL (giữ trang, noindex, archive, not-found hoặc redirect) phải theo chính sách đã chốt.

### 6.4. SEO, media và dữ liệu nguồn

- Metadata SEO là dữ liệu trình bày; không được thay đổi hoặc thay thế Product/SKU, giá, tồn, chứng nhận hay Batch nguồn.
- Dữ kiện Product/OCOP trong bài viết/metadata phải tham chiếu nguồn đã duyệt và được rà soát khi nguồn thay đổi.
- Media Reference giữ quan hệ với revision; tệp vật lý và trạng thái xử lý do EXT-09 cung cấp.
- Alternative text phải mô tả mục đích/nội dung ảnh, không mặc định nhồi từ khóa hoặc sao chép filename.
- Analytics bên ngoài, nếu hiển thị cho Content Manager, phải ghi rõ nguồn, kỳ dữ liệu và thời điểm cập nhật; không trộn với giao dịch nội bộ.

### 6.5. An toàn AI

- AI output là bản gợi ý, không phải dữ kiện được xác nhận hay revision đã duyệt.
- Chỉ Content Manager có quyền mới được đưa phần đã chọn vào Draft; vẫn áp dụng validation, review và Audit như nội dung thủ công.
- Dữ liệu gửi AI phải tối thiểu cần thiết và nằm trong quyền; không gửi PII, dữ liệu nội bộ hoặc tài liệu chưa được phép.
- Tuyên bố về giá, thành phần, sức khỏe, chứng nhận hoặc nguồn gốc phải được đối chiếu với nguồn nghiệp vụ trước khi công khai.
- AI Provider lỗi không được làm mất bản nháp hoặc ngăn quy trình tạo nội dung thủ công.

## 7. Traceability

| User Story | Business Rule / NFR | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-CONTENT-01` | `BR-AUTH-04`, `BR-PROD-04`, `NFR-05` | EPIC 04, 19, 22 | Tạo Draft đúng quyền, liên kết nguồn hợp lệ và không lộ/công khai dữ kiện nháp. |
| `US-CONTENT-02` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 22, 23, 28 | Revision/version đúng; chỉ Approved được xuất bản; lịch/ẩn/xung đột được xử lý và truy vết. |
| `US-CONTENT-03` | `NFR-02`, `NFR-05`, `NFR-08` | EXT-09 | Media hợp lệ gắn đúng revision, có alt text; lỗi tải/xóa/truy cập không tạo link hỏng hoặc lộ tài nguyên. |
| `US-CONTENT-04` | `BR-PROD-04` | EPIC 20 Publication | SEO metadata đúng revision, có preview; nội dung không công khai không index/sitemap. |
| `US-CONTENT-05` | `BR-PROD-01`, `BR-PROD-03`, `BR-PROD-04` | EPIC 04, 19 | Product SEO không sửa Catalog, xử lý SKU/canonical và không công khai Product/dữ kiện OCOP chưa duyệt. |
| `US-CONTENT-06` | `BR-AUDIT-01`, `NFR-01` | EPIC 03, 23 | Slug duy nhất; đổi URL có redirect/canonical/sitemap đúng, không vòng lặp hoặc tái sử dụng sai. |
| `US-CONTENT-07` | `BR-AUTH-03`, `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 22, 23 | Chỉ đúng Manager duyệt; revision cũ/xung đột/tự duyệt trái chính sách bị chặn và quyết định có lịch sử. |
| `US-AI-03` | `BR-PROD-04`, `NFR-05`, `NFR-08` | EPIC 04, 19, EXT-08 | AI chỉ tạo gợi ý, không vượt quyền/tự xuất bản; tuyên bố cần nguồn và lỗi AI không làm mất Draft. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Ánh xạ actor “Manager” của `US-CONTENT-07`, ma trận duyệt từng loại nội dung và có bắt buộc tách người soạn/người duyệt không.
- Loại Article, trường bắt buộc, taxonomy/category/tag, đa ngôn ngữ và quan hệ với trang thương hiệu/sản phẩm.
- Bộ trạng thái/chuyển trạng thái chính thức, quyền schedule/hide/restore/archive, SLA duyệt và chính sách rollback.
- Quy tắc slug/namespace, độ dài/ký tự, reserved paths, tái sử dụng URL, loại redirect và giới hạn redirect chain.
- Chính sách canonical/indexability cho Product nhiều SKU, Product hết hàng/tạm ngừng/ngừng bán và nội dung Hidden/Archived.
- Trường SEO bắt buộc, giới hạn title/description, metadata chia sẻ, structured data, sitemap và robots ở cấp nghiệp vụ.
- Ngưỡng hiệu năng/SEO/accessibility cụ thể phải nằm trong NFR/Test Plan thay vì hard-code công nghệ hoặc Lighthouse score trong Epic.
- Định dạng/dung lượng/số lượng media, kiểm tra an toàn, crop/thumbnail, caption/alt text, bản quyền và thời hạn lưu.
- Quy trình đồng bộ khi Product/OCOP nguồn thay đổi; cảnh báo hay tự cập nhật metadata nào và ai chịu trách nhiệm duyệt lại.
- Preview Draft được chia sẻ cho ai, thời hạn/quyền của liên kết preview và chống crawler index.
- Các metrics Content/SEO cần theo dõi, nguồn nội bộ/EXT-07, kỳ dữ liệu và ranh giới với EPIC 21.
- Release chính thức của `US-AI-03`, dữ liệu được gửi EXT-08, retention/opt-out, giọng thương hiệu và tiêu chí đánh giá chất lượng.
- Phạm vi Audit và thời hạn lưu Article, revision, media reference, approval, URL history và AI prompt/output.
- Search Engine Crawler có cần bổ sung vào Actor/External System Registry hay chỉ là consumer ẩn danh.

## 9. UI/UX Reference

- Editor cần tách nội dung, media, liên kết nguồn và SEO; thể hiện trạng thái lưu, revision và lỗi tại đúng trường.
- Preview phải hỗ trợ mobile/desktop, URL/metadata dự kiến và nhãn Draft rõ ràng, không dùng nhầm nội dung Published.
- Media Library cần trạng thái xử lý, nơi đang sử dụng, alt text/caption và cảnh báo trước khi ẩn/xóa.
- SEO panel cần preview title/description/URL/canonical, cảnh báo trùng/thiếu và phân biệt giá trị mặc định với override.
- Review screen cần so sánh revision trước/sau, nguồn Product/OCOP, media và SEO; làm nổi bật xung đột dữ liệu.
- Publication calendar cần timezone, lịch dự kiến, trạng thái duyệt và cảnh báo nội dung chưa đủ điều kiện.
- Trang công khai cần mobile-first, media có alternative text và trạng thái phù hợp cho URL cũ/nội dung ẩn.
- AI Assistant cần nhãn gợi ý, cho chọn từng phần để đưa vào Draft và không có hành động xuất bản trực tiếp.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

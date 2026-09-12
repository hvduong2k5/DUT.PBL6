# EPIC 03 — Product Discovery

## 1. Mục tiêu Epic

Epic này giúp khách tìm và hiểu sản phẩm trước khi mua: duyệt danh mục, tìm bằng từ khóa, lọc theo thuộc tính phù hợp, xem sản phẩm nổi bật và khám phá câu chuyện thương hiệu Huế. Ở giai đoạn sau, Epic mở rộng bằng trợ lý AI và gợi ý cá nhân hóa.

Epic chỉ chịu trách nhiệm **khám phá và trình bày sản phẩm công khai**. Việc tạo/quản lý sản phẩm, SKU, giá và trạng thái bán thuộc EPIC 04; giỏ hàng/checkout thuộc EPIC 05–06; soạn và phê duyệt bài viết thuộc EPIC 20; phân tích hành vi chuyên sâu thuộc EPIC 25.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP v1.0** | `US-DISC-01`, `US-DISC-02`, `US-DISC-03` | Khách tìm được sản phẩm phù hợp qua danh mục, từ khóa và bộ lọc. |
| **Giai đoạn 2** | `US-DISC-04`, `US-DISC-05` | Khách được định hướng bởi sản phẩm bán chạy/gợi ý cơ bản và hiểu thêm câu chuyện thương hiệu. |
| **Giai đoạn 3** | `US-AI-01`, `US-AI-04` | Khách được AI hỗ trợ tìm và đề xuất sản phẩm theo nhu cầu hoặc hành vi. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Duyệt, tìm kiếm, lọc và xem thông tin sản phẩm công khai trước khi quyết định mua. |
| Registered Customer (`ACT-02`) | Có toàn bộ khả năng của Guest và có thể nhận gợi ý dựa trên dữ liệu được phép sử dụng. |
| Sales Manager (`ACT-12`) | Cung cấp dữ liệu sản phẩm/SKU, giá và trạng thái bán qua EPIC 04. |
| Content Manager (`ACT-13`) | Soạn và xuất bản nội dung câu chuyện thương hiệu qua EPIC 20. |
| AI Provider (`EXT-08`) | Cung cấp năng lực AI khi các user story AI được triển khai. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-PROD-01` | Mỗi sản phẩm có thể có nhiều Variant/SKU. | Kết quả khám phá và trang chi tiết phải phản ánh đúng lựa chọn SKU/Variant khi khách mua. |
| `BR-PROD-02` | Giá, khối lượng, quy cách đóng gói và tồn kho được quản lý theo SKU. | Bộ lọc và thông tin hiển thị phải lấy dữ liệu theo SKU; trạng thái còn hàng không được suy diễn theo sản phẩm chung. |
| `BR-PROD-03` | Sản phẩm thực phẩm phải có ngày sản xuất và hạn sử dụng phù hợp. | Thông tin thực phẩm hiển thị cho khách cần nhất quán với dữ liệu sản phẩm/lô hàng được phép công khai. |
| `BR-PROD-04` | Thông tin sản phẩm hiển thị cho khách phải được quản lý và phê duyệt trước khi công khai. | Chỉ sản phẩm/nội dung đã được phép công khai mới xuất hiện trong danh mục, tìm kiếm, bộ lọc và gợi ý. |
| `BR-BATCH-05` | Sản phẩm hết hạn không được tiếp tục bán. | Tình trạng còn hàng phục vụ tìm kiếm/lọc không được coi hàng hết hạn là có thể mua. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 04 — Product, Variant & SKU`: là nguồn dữ liệu sản phẩm, SKU, giá, khối lượng, trạng thái bán và thông tin thực phẩm.
- `EPIC 09 — Inventory & Batch`: cung cấp tồn kho khả dụng, Batch/Lot và trạng thái hạn sử dụng để xác định khả năng bán.
- `EPIC 15 — Review & Rating`: cung cấp dữ liệu đánh giá khi bộ lọc/hiển thị theo đánh giá được bật.
- `EPIC 20 — Content & SEO`: quản lý nội dung câu chuyện thương hiệu, ảnh/video và quy trình xuất bản.
- `EPIC 24 — Finance & Business Analytics`: có thể cung cấp dữ liệu bán chạy theo chính sách đã duyệt.
- `EPIC 25 — Customer Analytics & DSS`: cung cấp dữ liệu hành vi/hiệu quả cần thiết cho gợi ý cá nhân hóa.
- `EPIC 05 — Shopping Cart`: nhận sản phẩm/SKU mà khách đã khám phá và chọn mua.

## 4. User Stories chi tiết

### US-DISC-01 — Xem danh mục sản phẩm

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn xem danh mục sản phẩm để tìm sản phẩm phù hợp.

**Giá trị nghiệp vụ:** Khách có điểm bắt đầu rõ ràng để khám phá các dòng mè xửng và đặc sản, giảm thời gian tìm sản phẩm phù hợp.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem danh sách sản phẩm công khai
  Given có sản phẩm đã được phê duyệt để công khai
  When khách mở trang danh mục sản phẩm
  Then hệ thống hiển thị các sản phẩm đang được phép bán/công khai
  And mỗi sản phẩm hiển thị thông tin đủ để khách nhận diện và chọn xem chi tiết

Scenario: Xem sản phẩm theo danh mục
  Given sản phẩm đã được gán vào các danh mục phù hợp
  When khách chọn một danh mục
  Then hệ thống chỉ hiển thị các sản phẩm công khai thuộc danh mục đó
  And khách có thể chuyển sang xem chi tiết một sản phẩm

Scenario: Danh mục không có sản phẩm phù hợp
  Given danh mục được chọn không có sản phẩm công khai phù hợp
  When khách xem danh mục đó
  Then hệ thống thông báo trạng thái không có kết quả
  And cung cấp cách quay lại danh mục hoặc tiếp tục tìm kiếm
```

### US-DISC-02 — Tìm kiếm theo từ khóa

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn tìm kiếm sản phẩm theo từ khóa để nhanh chóng tìm được sản phẩm mong muốn.

**Giá trị nghiệp vụ:** Khách đến với một nhu cầu cụ thể có thể đi thẳng đến sản phẩm thay vì duyệt toàn bộ catalog.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tìm thấy sản phẩm phù hợp với từ khóa
  Given có sản phẩm công khai phù hợp với từ khóa khách nhập
  When khách gửi từ khóa tìm kiếm
  Then hệ thống hiển thị danh sách sản phẩm phù hợp
  And khách có thể chọn một sản phẩm để xem chi tiết

Scenario: Không tìm thấy sản phẩm
  Given không có sản phẩm công khai phù hợp với từ khóa
  When khách thực hiện tìm kiếm
  Then hệ thống thông báo không tìm thấy kết quả
  And gợi ý khách điều chỉnh từ khóa hoặc quay lại danh mục

Scenario: Từ khóa không hợp lệ
  Given khách để trống hoặc nhập dữ liệu không thể dùng để tìm kiếm
  When khách gửi yêu cầu tìm kiếm
  Then hệ thống không thực hiện tìm kiếm không hợp lệ
  And hiển thị hướng dẫn để khách nhập lại từ khóa
```

### US-DISC-03 — Lọc sản phẩm

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Must Have — 1st  
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn lọc theo giá, khối lượng, loại sản phẩm, đánh giá và tình trạng còn hàng để thu hẹp lựa chọn.

**Giá trị nghiệp vụ:** Khách nhanh chóng thu hẹp catalog theo ngân sách, nhu cầu sử dụng và khả năng giao hàng thực tế.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Lọc theo một tiêu chí
  Given khách đang xem danh mục hoặc kết quả tìm kiếm
  When khách chọn một giá trị lọc hợp lệ về giá, khối lượng, loại sản phẩm, đánh giá hoặc tình trạng còn hàng
  Then hệ thống chỉ hiển thị các sản phẩm/SKU phù hợp với tiêu chí đã chọn
  And vẫn chỉ hiển thị dữ liệu sản phẩm được phép công khai

Scenario: Kết hợp nhiều tiêu chí lọc
  Given khách đang xem danh sách sản phẩm
  When khách áp dụng đồng thời nhiều tiêu chí lọc hợp lệ
  Then hệ thống hiển thị kết quả thỏa mãn tất cả tiêu chí đang áp dụng
  And hiển thị các tiêu chí đang được chọn để khách có thể bỏ từng tiêu chí

Scenario: Không có kết quả sau khi lọc
  Given không có sản phẩm/SKU phù hợp với tổ hợp tiêu chí lọc
  When hệ thống áp dụng bộ lọc
  Then hệ thống thông báo không có kết quả phù hợp
  And cho phép khách xóa một phần hoặc toàn bộ bộ lọc
```

### US-DISC-04 — Sản phẩm bán chạy hoặc được đề xuất

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Should Have  
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn xem sản phẩm bán chạy hoặc được đề xuất để dễ lựa chọn.

**Giá trị nghiệp vụ:** Hỗ trợ khách mới ra quyết định nhanh hơn và tạo cơ hội khám phá/cross-sell các sản phẩm phù hợp.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị sản phẩm bán chạy
  Given doanh nghiệp đã có dữ liệu và chính sách xác định sản phẩm bán chạy
  When khách mở khu vực sản phẩm nổi bật
  Then hệ thống hiển thị danh sách sản phẩm bán chạy đang được phép công khai
  And không hiển thị sản phẩm đã ngừng bán hoặc không còn có thể mua theo chính sách tồn kho

Scenario: Hiển thị gợi ý cơ bản khi chưa cá nhân hóa
  Given khách chưa đăng nhập hoặc chưa có đủ dữ liệu cho gợi ý cá nhân hóa
  When khách xem khu vực gợi ý
  Then hệ thống có thể hiển thị các gợi ý chung đã được doanh nghiệp phê duyệt
  And khách vẫn có thể khám phá hoặc tìm kiếm sản phẩm khác

Scenario: Không có dữ liệu nổi bật phù hợp
  Given chưa có sản phẩm bán chạy hoặc gợi ý hợp lệ để công khai
  When khách mở khu vực sản phẩm nổi bật
  Then hệ thống không hiển thị dữ liệu không hợp lệ hoặc thông tin lỗi
  And trải nghiệm duyệt danh mục chính vẫn hoạt động bình thường
```

### US-DISC-05 — Câu chuyện thương hiệu và văn hóa Huế

**Actor:** Guest Customer, Registered Customer  
**Ưu tiên:** Should Have  
**Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn xem câu chuyện thương hiệu và văn hóa Huế để hiểu giá trị của sản phẩm.

**Giá trị nghiệp vụ:** Tạo khác biệt thương hiệu OCOP, tăng niềm tin và giúp khách hiểu giá trị văn hóa đằng sau sản phẩm thay vì chỉ so sánh giá.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem nội dung thương hiệu đã xuất bản
  Given Content Manager đã xuất bản nội dung thương hiệu/văn hóa Huế
  When khách mở khu vực câu chuyện thương hiệu
  Then hệ thống hiển thị nội dung, hình ảnh hoặc video đã được phép công khai
  And khách có thể tiếp tục khám phá sản phẩm liên quan nếu nội dung có liên kết phù hợp

Scenario: Không hiển thị nội dung chưa duyệt
  Given nội dung đang ở trạng thái nháp, bị ẩn hoặc chưa được phê duyệt
  When khách truy cập khu vực câu chuyện thương hiệu
  Then hệ thống không công khai nội dung đó cho khách

Scenario: Nội dung không còn khả dụng
  Given một nội dung đã bị ẩn hoặc gỡ khỏi xuất bản
  When khách mở liên kết cũ đến nội dung đó
  Then hệ thống thông báo nội dung không khả dụng theo cách thân thiện
  And không tiết lộ nội dung chưa được phép công khai
```

### US-AI-01 — Product Search Assistant (AI)

**Actor:** Guest Customer, Registered Customer, AI Provider (`EXT-08`)  
**Ưu tiên:** Could Have  
**Phát hành:** Giai đoạn 3

> Là khách hàng, tôi muốn được AI hỗ trợ tìm sản phẩm phù hợp dựa trên nhu cầu (chat/hỏi đáp) để dễ lựa chọn.

**Giá trị nghiệp vụ:** Khách mô tả nhu cầu tự nhiên như mục đích tặng quà, khẩu vị hoặc ngân sách thay vì phải biết chính xác tên sản phẩm.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Nhận gợi ý từ nhu cầu tự nhiên
  Given AI Search Assistant đang khả dụng
  When khách mô tả nhu cầu tìm sản phẩm
  Then hệ thống trả lời bằng các sản phẩm công khai phù hợp hoặc câu hỏi làm rõ cần thiết
  And gợi ý không bao gồm sản phẩm không được phép công khai hoặc không còn có thể mua

Scenario: Không xác định được nhu cầu phù hợp
  Given yêu cầu của khách quá mơ hồ hoặc không có sản phẩm phù hợp
  When AI Search Assistant xử lý yêu cầu
  Then hệ thống giải thích ngắn gọn giới hạn của kết quả
  And hướng dẫn khách dùng danh mục, tìm kiếm từ khóa hoặc bộ lọc thông thường

Scenario: AI không tự thay đổi dữ liệu nghiệp vụ
  Given khách đang sử dụng AI Search Assistant
  When AI đưa ra câu trả lời hoặc gợi ý
  Then AI không tự tạo, sửa hoặc ẩn sản phẩm, giá, tồn kho hay đơn hàng
  And khách vẫn là người quyết định sản phẩm/SKU được chọn mua
```

### US-AI-04 — Product Recommendation (AI)

**Actor:** Registered Customer, AI Provider (`EXT-08`)  
**Ưu tiên:** Could Have  
**Phát hành:** Giai đoạn 3

> Là khách hàng, tôi muốn được đề xuất sản phẩm phù hợp với lịch sử hoặc hành vi mua hàng.

**Giá trị nghiệp vụ:** Tăng khả năng khách tìm được sản phẩm liên quan và tạo nền tảng cho bán chéo/bán thêm có trách nhiệm.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị đề xuất cá nhân hóa hợp lệ
  Given Customer đã đăng nhập và có dữ liệu lịch sử/hành vi được phép sử dụng
  When Customer xem khu vực đề xuất sản phẩm
  Then hệ thống hiển thị các sản phẩm công khai phù hợp theo dữ liệu đó
  And Customer có thể xem chi tiết hoặc bỏ qua từng gợi ý

Scenario: Fallback khi chưa đủ dữ liệu
  Given Customer chưa có đủ dữ liệu cho đề xuất cá nhân hóa
  When Customer xem khu vực đề xuất
  Then hệ thống dùng gợi ý chung hợp lệ hoặc không hiển thị gợi ý cá nhân hóa
  And không tạo thông tin hành vi giả định về Customer

Scenario: Bảo vệ dữ liệu và quyền quyết định của khách
  Given hệ thống tạo đề xuất cho Customer
  When đề xuất được hiển thị
  Then hệ thống không công khai lịch sử/hành vi riêng tư của Customer cho khách khác
  And đề xuất không tự thêm sản phẩm vào giỏ hàng hoặc tạo đơn thay khách
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-DISC-01` | Hiển thị danh mục và danh sách sản phẩm đã được phê duyệt để công khai. | `US-DISC-01`, `BR-PROD-04` |
| `FR-DISC-02` | Cho phép khách chọn danh mục và mở trang chi tiết sản phẩm từ danh sách. | `US-DISC-01` |
| `FR-DISC-03` | Cho phép khách tìm sản phẩm công khai theo từ khóa và hiển thị trạng thái không có kết quả khi phù hợp. | `US-DISC-02` |
| `FR-DISC-04` | Cho phép lọc kết quả theo giá, khối lượng, loại sản phẩm, đánh giá và tình trạng còn hàng; hỗ trợ kết hợp và xóa bộ lọc. | `US-DISC-03` |
| `FR-DISC-05` | Xác định giá, khối lượng, quy cách và khả năng bán từ dữ liệu SKU/tồn kho hợp lệ thay vì suy diễn ở cấp sản phẩm chung. | `US-DISC-03`, `BR-PROD-01`, `BR-PROD-02` |
| `FR-DISC-06` | Chỉ đưa sản phẩm/SKU còn được phép công khai và không vi phạm quy tắc hết hạn vào kết quả có trạng thái còn hàng hoặc gợi ý mua. | `US-DISC-01~04`, `BR-PROD-04`, `BR-BATCH-05` |
| `FR-DISC-07` | Hiển thị danh sách sản phẩm bán chạy hoặc gợi ý chung theo chính sách dữ liệu đã được phê duyệt. | `US-DISC-04` |
| `FR-DISC-08` | Hiển thị nội dung thương hiệu/văn hóa Huế đã được xuất bản; không công khai nội dung nháp, ẩn hoặc chưa duyệt. | `US-DISC-05`, `EPIC 20` |
| `FR-DISC-09` | Cho phép khách mô tả nhu cầu tự nhiên cho AI Search Assistant và nhận gợi ý sản phẩm trong phạm vi catalog công khai. | `US-AI-01` |
| `FR-DISC-10` | Tạo đề xuất cá nhân hóa từ dữ liệu được phép sử dụng; có fallback gợi ý chung khi thiếu dữ liệu. | `US-AI-04` |
| `FR-DISC-11` | AI/gợi ý không được tự tạo, cập nhật hoặc ẩn dữ liệu sản phẩm; không tự thêm hàng vào giỏ hay tạo đơn cho khách. | `US-AI-01`, `US-AI-04` |

## 6. Quy tắc dữ liệu, trải nghiệm và AI

- Chỉ sử dụng sản phẩm, SKU, giá, tồn kho và nội dung đã được phê duyệt/công khai từ các Epic nguồn.
- Trạng thái “còn hàng” phải dựa trên dữ liệu tồn kho khả dụng; hàng hết hạn không được hiển thị như một lựa chọn có thể mua.
- Bộ lọc theo đánh giá phụ thuộc dữ liệu của EPIC 15. Trước khi EPIC 15 được phát hành, doanh nghiệp cần chốt có ẩn bộ lọc này hay hiển thị trạng thái chưa có đánh giá.
- Dữ liệu hành vi và lịch sử mua dùng cho gợi ý cá nhân hóa phải tuân thủ chính sách quyền riêng tư và chỉ phục vụ Customer tương ứng.
- AI chỉ đưa ra hỗ trợ khám phá; không tự quyết định thay khách hoặc thay đổi dữ liệu kinh doanh quan trọng.
- Chi tiết về thuật toán tìm kiếm, mô hình AI, lưu trữ sự kiện hành vi, API endpoint, cache hay cơ sở dữ liệu thuộc Architecture/Technical Design, không thuộc User Story Specification này.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-DISC-01` | `BR-PROD-04` | EPIC 04 | Danh mục chỉ hiển thị sản phẩm công khai và truy cập được trang chi tiết. |
| `US-DISC-02` | `BR-PROD-04` | EPIC 04 | Tìm thấy sản phẩm theo từ khóa; không có kết quả được thông báo rõ ràng. |
| `US-DISC-03` | `BR-PROD-01`, `BR-PROD-02`, `BR-BATCH-05` | EPIC 04, 09, 15 | Bộ lọc kết hợp đúng dữ liệu SKU/tồn kho; hàng hết hạn không được coi là còn hàng. |
| `US-DISC-04` | `BR-PROD-04`, `BR-BATCH-05` | EPIC 04, 09, 24 | Sản phẩm nổi bật/gợi ý đều hợp lệ để công khai và mua theo chính sách. |
| `US-DISC-05` | `BR-PROD-04` | EPIC 20 | Chỉ nội dung thương hiệu đã xuất bản được hiển thị. |
| `US-AI-01` | `BR-PROD-04` | EPIC 04, 09, 25, EXT-08 | AI chỉ gợi ý catalog công khai và có fallback khi không hiểu nhu cầu. |
| `US-AI-04` | `BR-PROD-04` | EPIC 04, 09, 25, EXT-08 | Đề xuất bảo vệ dữ liệu Customer, có fallback và không tự thêm hàng vào giỏ. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Cấu trúc danh mục sản phẩm, quy tắc gán sản phẩm vào nhiều danh mục và thứ tự hiển thị.
- Quy tắc so khớp từ khóa: tên sản phẩm, SKU, thành phần, từ đồng nghĩa và cách xử lý tiếng Việt có/không dấu.
- Cách diễn giải từng bộ lọc ở cấp Product hay SKU, đặc biệt là giá, khối lượng, tồn kho và rating.
- Khi EPIC 15 chưa có dữ liệu Review/Rating, bộ lọc đánh giá ở MVP sẽ được ẩn, vô hiệu hóa hay hiển thị trạng thái chưa có đánh giá.
- Công thức, khoảng thời gian dữ liệu và quyền phê duyệt để xác định “sản phẩm bán chạy” hoặc gợi ý chung.
- Nội dung thương hiệu nào hiển thị tại trang chủ, trang sản phẩm hay trang nội dung riêng; ai phê duyệt trước khi công khai.
- Chính sách đồng ý sử dụng dữ liệu hành vi/lịch sử mua, giải thích đề xuất và cách khách từ chối cá nhân hóa AI.

## 9. UI/UX Reference

- Trang danh mục, kết quả tìm kiếm và trạng thái không có kết quả.
- Bộ lọc có thể thấy rõ tiêu chí đã áp dụng và hành động xóa/bỏ từng tiêu chí.
- Khu vực sản phẩm bán chạy/gợi ý chung trên trang chủ hoặc trang sản phẩm.
- Khu vực câu chuyện thương hiệu, văn hóa Huế và liên kết sản phẩm liên quan.
- Giao diện hỏi đáp AI Search Assistant cùng đường dẫn quay lại tìm kiếm/bộ lọc thông thường.
- Khu vực đề xuất cá nhân hóa có tùy chọn xem chi tiết hoặc bỏ qua sản phẩm.

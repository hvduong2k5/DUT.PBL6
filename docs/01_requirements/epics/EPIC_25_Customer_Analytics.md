# EPIC 25 — Customer Analytics & Decision Support System

## 1. Mục tiêu Epic

Epic này biến dữ liệu giao dịch và vận hành đã được kiểm soát thành thông tin hỗ trợ quyết định. Executive xem Dashboard tổng quan; Marketing phân tích RFM, nguy cơ không mua lại và mùa vụ; Sales đánh giá liên kết sản phẩm; Supply theo dõi tốc độ bán, dự báo nhu cầu và rủi ro hết hạn; Executive nhận khuyến nghị chiến lược có căn cứ.

EPIC 25 sở hữu định nghĩa chỉ số phân tích, tập dữ liệu phân tích, phiên chạy/phiên bản mô hình, kết quả phân tích và khuyến nghị. Epic không sở hữu giao dịch nguồn, sổ tài chính, Campaign, Combo, giá, tồn kho hay kế hoạch cung ứng. Kết quả DSS chỉ hỗ trợ con người ra quyết định; không tự thay đổi giá, tồn, chương trình Marketing hoặc tạo nghiệp vụ nguồn.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 2 — Should Have, cần PO xác nhận** | `US-DSS-01` | Dashboard tổng quan cho lãnh đạo. |
| **Giai đoạn 3 — Could Have** | `US-DSS-02` đến `US-DSS-10` | Phân tích khách hàng, sản phẩm, cung ứng, mùa vụ và khuyến nghị chiến lược. |

Sprint Planning xếp `US-DSS-01` vào Giai đoạn 2, trong khi Epic Map xếp toàn EPIC 25 vào Giai đoạn 3. Tài liệu ưu tiên kế hoạch chi tiết ở cấp story nhưng giữ đây là quyết định mở cần Product Owner xác nhận.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors

| Actor | Vai trò |
| --- | --- |
| Marketing Staff (`ACT-14`) | Xem phân khúc RFM, churn risk và mùa vụ trong phạm vi được cấp. |
| Inventory/Supply Manager (`ACT-15`) | Xem velocity, demand forecast và expiry risk để lập kế hoạch. |
| Executive / Business Manager (`ACT-16`) | Xem Dashboard, phân tích hiệu quả và đánh giá khuyến nghị chiến lược. |
| Sales Manager (`ACT-12`) | Xem product association/performance và chuyển insight thành đề xuất Combo qua đúng quy trình. |
| Các Epic nguồn | Cung cấp dữ liệu có định danh, thời điểm, trạng thái và chất lượng phù hợp. |

### 3.2. Business Rules và NFR áp dụng

Product Backlog chưa định nghĩa nhóm `BR-DSS`; Epic không tự tạo mã Business Rule mới.

| Mã | Ảnh hưởng đến Epic |
| --- | --- |
| `BR-AUTH-03`, `BR-AUTH-04` | Chỉ nhân viên có Role/Permission và phạm vi phù hợp được xem, export hoặc cấu hình phân tích. |
| `BR-PROD-01`, `BR-PROD-02` | Phân tích sản phẩm phải phân biệt Product và SKU, giữ đúng giá/quy cách ở nguồn. |
| `BR-BATCH-01`, `BR-BATCH-02` | Expiry Risk phải dựa trên Batch/Lot và HSD có thể truy xuất. |
| `BR-BATCH-05` | Cảnh báo gần hết hạn phải dựa trên ngưỡng chính sách, không tự suy diễn. |
| `BR-ORDER-01`, `BR-ORDER-05` | Chỉ Order hợp lệ theo trạng thái được duyệt mới tham gia chỉ số; thay đổi quan trọng phải truy vết. |
| `BR-AUDIT-01` | Thay đổi định nghĩa chỉ số, mô hình, ngưỡng và việc chấp nhận/từ chối khuyến nghị quan trọng cần Audit. |
| `NFR-04`, `NFR-05`, `NFR-08` | Bảo vệ dữ liệu khách hàng, hạn chế quyền và dùng dữ liệu đúng mục đích. |
| `NFR-10` | Truy vấn/tính toán DSS quy mô lớn phải mở rộng được và không ảnh hưởng luồng giao dịch. |
| `NFR-11` | Theo dõi độ mới, lỗi pipeline, chất lượng và trạng thái phiên phân tích. |

### 3.3. Phụ thuộc và ranh giới

- `EPIC 02`, `EPIC 08`: cung cấp Customer hợp lệ và Order; Analytics không sửa hồ sơ hay Order.
- `EPIC 04`, `EPIC 09`: cung cấp Product/SKU, tồn và Batch/HSD; Supply Analytics chỉ đọc và phân tích.
- `EPIC 12`, `EPIC 13`, `EPIC 18`: cung cấp dữ liệu kênh Marketplace, Offline và B2B.
- `EPIC 14`: cung cấp Return/Refund để tránh thổi phồng doanh thu, tần suất và demand.
- `EPIC 17`: sở hữu Combo/Promotion/Loyalty; association chỉ đề xuất, không tự tạo Combo.
- `EPIC 21`: sở hữu Campaign và KPI; DSS không tự kích hoạt chiến dịch.
- `EPIC 22`, `EPIC 23`: cấp quyền và Audit các thao tác nhạy cảm.
- `EPIC 24`: là nguồn chỉ số tài chính đã duyệt cho doanh thu/lợi nhuận trên Dashboard; EPIC 25 không tính một sổ tài chính song song.
- `EPIC 27`: sử dụng demand forecast cho kế hoạch cung ứng nhưng quyết định PO/kế hoạch vẫn thuộc Procurement.
- `EPIC 28`: gửi cảnh báo/notification; DSS chỉ phát sinh insight hoặc điều kiện cảnh báo.

## 4. User Stories chi tiết

### US-DSS-01 — Dashboard tổng quan

**Actor:** Executive / Business Manager (`ACT-16`)

**Ưu tiên:** Should Have — **Phát hành:** Giai đoạn 2, cần PO xác nhận

> Là Executive, tôi muốn xem Dashboard tổng quan về doanh thu, đơn hàng, khách hàng, tồn kho và lợi nhuận để nắm tình hình doanh nghiệp.

```gherkin
Scenario: Xem Dashboard theo kỳ
  Given Executive có quyền và dữ liệu nguồn khả dụng
  When chọn một khoảng thời gian hợp lệ
  Then Dashboard hiển thị doanh thu, Order, Customer, tồn kho và lợi nhuận
  And mỗi chỉ số nêu định nghĩa, đơn vị, độ mới và nguồn

Scenario: So sánh kỳ
  Given người dùng đang xem một kỳ
  When chọn kỳ so sánh
  Then hệ thống dùng biên thời gian nhất quán
  And hiển thị chênh lệch tuyệt đối/tỷ lệ khi hợp lệ

Scenario: Drill-down
  Given người dùng có quyền với chỉ số
  When chọn một card hoặc điểm dữ liệu
  Then hệ thống mở phân tích chi tiết với cùng bộ lọc
  And không vượt phạm vi quyền

Scenario: Nguồn chậm hoặc thiếu
  Given một nguồn chưa đồng bộ đúng SLA
  When mở Dashboard
  Then chỉ số bị ảnh hưởng được đánh dấu stale/partial/unavailable
  And không hiển thị như số đã chốt

Scenario: Định nghĩa Finance và DSS không khớp
  Given chỉ số doanh thu/lợi nhuận chưa có phiên bản được EPIC 24 phê duyệt
  When tổng hợp Dashboard
  Then hệ thống không công bố số tài chính như chính thức
  And chỉ rõ vấn đề định nghĩa dữ liệu
```

### US-DSS-02 — Phân nhóm khách hàng theo RFM

**Actor:** Marketing Staff (`ACT-14`), Executive (`ACT-16`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Marketing/Executive, tôi muốn phân nhóm khách hàng theo hành vi mua hàng để xây dựng chính sách chăm sóc phù hợp.

```gherkin
Scenario: Tính RFM theo phiên bản cấu hình
  Given có lịch sử Order đủ điều kiện và Customer hợp lệ
  When chạy phân tích tại thời điểm chốt dữ liệu
  Then hệ thống tính Recency, Frequency, Monetary theo định nghĩa được duyệt
  And lưu phiên chạy, cửa sổ dữ liệu và phiên bản ngưỡng

Scenario: Gán phân khúc
  Given điểm RFM đã tính
  When áp dụng bộ quy tắc phân khúc
  Then mỗi Customer đủ điều kiện được gán đúng một kết quả trong phiên
  And người dùng xem được lý do/điểm tạo ra phân khúc

Scenario: Guest hoặc Customer chưa hợp nhất
  Given giao dịch không liên kết được một Customer ổn định
  When tính RFM
  Then dữ liệu được loại hoặc đưa nhóm chưa xác định theo chính sách
  And không tự hợp nhất danh tính chỉ vì trùng email/số điện thoại

Scenario: Return, Refund hoặc hủy
  Given Customer có giao dịch bị hủy/hoàn
  When tính RFM
  Then Frequency/Monetary áp dụng quy tắc giao dịch đủ điều kiện
  And không tính doanh thu đã hoàn như mua hợp lệ nếu chính sách loại trừ

Scenario: Export phân khúc
  Given Marketing có quyền và mục đích sử dụng hợp lệ
  When export danh sách phân khúc
  Then hệ thống chỉ xuất trường cần thiết, masking/phạm vi phù hợp
  And ghi nhận actor, mục đích, tham số và thời điểm
```

### US-DSS-03 — Phát hiện nguy cơ không mua lại

**Actor:** Marketing Staff (`ACT-14`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Marketing Staff, tôi muốn phát hiện khách hàng có nguy cơ không mua lại để triển khai chiến dịch giữ chân.

```gherkin
Scenario: Chấm điểm churn risk
  Given có dữ liệu và mô hình/luật đã được phê duyệt
  When phiên phân tích hoàn tất
  Then hệ thống gán risk score/band với phiên bản và thời điểm
  And nêu các yếu tố giải thích được phép

Scenario: Không đủ dữ liệu
  Given Customer không đạt ngưỡng lịch sử tối thiểu
  When chấm điểm
  Then kết quả là insufficient data thay vì tự gán rủi ro thấp

Scenario: Dữ liệu hoặc mô hình đã cũ
  Given kết quả vượt ngưỡng freshness hoặc model version ngừng hiệu lực
  When Marketing xem danh sách
  Then hệ thống cảnh báo stale và không trình bày như dự đoán hiện tại

Scenario: Chuyển insight sang Marketing
  Given Marketing chọn nhóm rủi ro và có quyền
  When tạo đề xuất chiến dịch
  Then hệ thống chuyển tiêu chí/tham chiếu sang EPIC 21
  And không tự gửi thông điệp hoặc cấp ưu đãi

Scenario: Customer không được phép sử dụng dữ liệu
  Given hồ sơ bị hạn chế theo privacy/consent policy
  When tạo danh sách hành động
  Then Customer bị loại hoặc xử lý theo chính sách
  And lý do không tiết lộ dữ liệu nhạy cảm không cần thiết
```

### US-DSS-04 — Hiệu quả sản phẩm

**Actor:** Executive (`ACT-16`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Executive, tôi muốn biết sản phẩm nào bán chạy, bán chậm hoặc đang giảm doanh số.

```gherkin
Scenario: Xếp hạng Product/SKU
  Given dữ liệu bán đủ điều kiện
  When chọn kỳ, kênh và cấp Product/SKU
  Then hệ thống xếp hạng theo chỉ số được chọn
  And hiển thị công thức, nguồn và phạm vi

Scenario: Nhận diện xu hướng giảm
  Given đủ số kỳ so sánh
  When doanh số giảm theo ngưỡng được duyệt
  Then hệ thống đánh dấu xu hướng cùng mức thay đổi và kỳ tham chiếu

Scenario: Product mới không đủ lịch sử
  Given Product chưa có đủ kỳ quan sát
  When phân loại hiệu quả
  Then hệ thống đánh dấu new/insufficient data
  And không mặc định là bán chậm

Scenario: Hết hàng làm sai lệch nhu cầu
  Given SKU không khả dụng trong một phần kỳ
  When phân tích tốc độ bán
  Then hệ thống hiển thị số ngày hết hàng/thiếu dữ liệu
  And không kết luận nhu cầu thấp mà không có cảnh báo

Scenario: Catalog thay đổi
  Given Product đổi tên, nhóm hoặc ngừng bán
  When xem lịch sử
  Then hệ thống giữ định danh ổn định và version/snapshot phù hợp
```

### US-DSS-05 — Tốc độ tiêu thụ SKU

**Actor:** Inventory/Supply Manager (`ACT-15`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Supply Manager, tôi muốn biết tốc độ tiêu thụ từng SKU để lập kế hoạch sản xuất.

```gherkin
Scenario: Tính sales velocity
  Given SKU có lịch sử bán đủ điều kiện
  When chọn cửa sổ và phạm vi kênh
  Then hệ thống tính tốc độ theo công thức/đơn vị đã duyệt
  And lưu cửa sổ, data cutoff và phiên bản định nghĩa

Scenario: So sánh nhiều cửa sổ
  Given dữ liệu đủ cho ngắn hạn và dài hạn
  When người dùng so sánh
  Then hệ thống hiển thị các velocity riêng, không trộn mẫu số

Scenario: Không bán vì hết hàng
  Given SKU hết hàng trong cửa sổ
  When tính velocity
  Then hệ thống thể hiện stockout duration và mức tin cậy

Scenario: Return và đơn bị hủy
  Given có giao dịch không đủ điều kiện
  When tổng hợp lượng tiêu thụ
  Then hệ thống áp dụng chính sách loại trừ/điều chỉnh nhất quán

Scenario: Chuyển sang kế hoạch cung ứng
  Given Supply Manager xem velocity hợp lệ
  When tạo đề xuất kế hoạch
  Then hệ thống chuyển dữ liệu đầu vào sang EPIC 27
  And không tự tạo PO hoặc thay đổi tồn
```

### US-DSS-06 — Dự báo nhu cầu

**Actor:** Executive (`ACT-16`), Inventory/Supply Manager (`ACT-15`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Executive/Supply Manager, tôi muốn dự báo nhu cầu sản phẩm trong tương lai để chuẩn bị nguyên liệu và sản xuất.

```gherkin
Scenario: Tạo forecast theo horizon
  Given dữ liệu đạt điều kiện chất lượng
  When chọn SKU/Product, khu vực/kênh và horizon
  Then hệ thống trả forecast, khoảng bất định, data cutoff và model version

Scenario: Không đủ lịch sử
  Given SKU mới hoặc dữ liệu quá thưa
  When yêu cầu dự báo
  Then hệ thống trả insufficient data hoặc phương án baseline được phê duyệt
  And không tạo độ chính xác giả

Scenario: Có sự kiện bất thường
  Given kỳ lịch sử có promotion, stockout hoặc sự kiện đặc biệt
  When tạo forecast
  Then hệ thống ghi nhận feature/ngoại lệ được sử dụng
  And cho người dùng hiểu giới hạn kết quả

Scenario: Đánh giá forecast sau kỳ thực tế
  Given horizon đã kết thúc và actual khả dụng
  When đánh giá mô hình
  Then hệ thống lưu sai số theo segment/model version
  And cảnh báo khi vượt ngưỡng

Scenario: Sử dụng trong Procurement
  Given forecast còn hiệu lực
  When Supply Manager chuyển thành đầu vào kế hoạch
  Then EPIC 27 nhận snapshot và tham chiếu
  And người có thẩm quyền vẫn phải phê duyệt kế hoạch/PO
```

### US-DSS-07 — Rủi ro hết hạn

**Actor:** Inventory/Supply Manager (`ACT-15`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Supply Manager, tôi muốn biết lượng hàng có nguy cơ hết hạn trong tương lai để chủ động xử lý.

```gherkin
Scenario: Tính rủi ro theo Batch
  Given Batch có tồn khả dụng và HSD hợp lệ
  When kết hợp forecast/velocity với ngưỡng HSD
  Then hệ thống ước tính lượng có nguy cơ theo Batch/SKU và horizon
  And hiển thị căn cứ, giả định và độ mới

Scenario: Batch thiếu hoặc sai HSD
  Given dữ liệu HSD không hợp lệ
  When phân tích
  Then hệ thống đánh dấu data quality issue
  And không tự suy diễn ngày hết hạn

Scenario: Tồn đã giữ hoặc không bán được
  Given tồn gồm available, reserved, damaged hoặc blocked
  When tính rủi ro
  Then hệ thống dùng đúng loại lượng theo định nghĩa
  And trình bày rõ thành phần bị loại

Scenario: Forecast thay đổi
  Given forecast mới làm mức rủi ro đổi
  When phiên mới hoàn tất
  Then hệ thống lưu phiên kết quả mới và không ghi đè mất lịch sử

Scenario: Đề xuất xử lý
  Given Batch vượt ngưỡng rủi ro
  When Supply Manager xem chi tiết
  Then hệ thống cho phép chuyển insight tới Sales/Marketing/Procurement
  And không tự giảm giá, hủy hàng hoặc sửa tồn
```

### US-DSS-08 — Liên kết sản phẩm

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Sales Manager, tôi muốn biết các sản phẩm thường được mua cùng nhau để xây dựng Combo.

```gherkin
Scenario: Tìm cặp/nhóm thường mua cùng
  Given Order đủ điều kiện có nhiều dòng sản phẩm
  When chạy phân tích association
  Then hệ thống trả itemset/rule với support, confidence và lift hoặc chỉ số đã duyệt
  And nêu cửa sổ, kênh và model version

Scenario: Loại giao dịch không hợp lệ
  Given Order hủy, test, gian lận hoặc hoàn toàn phần theo chính sách
  When tạo tập dữ liệu
  Then hệ thống loại/điều chỉnh nhất quán và báo số lượng ảnh hưởng

Scenario: Mẫu quá nhỏ
  Given rule không đạt ngưỡng support hoặc độ tin cậy
  When hiển thị kết quả
  Then hệ thống không gắn nhãn khuyến nghị đáng tin cậy

Scenario: Product không còn bán
  Given itemset chứa SKU ngừng bán hoặc không khả dụng
  When Sales xem insight
  Then hệ thống cảnh báo trạng thái hiện tại
  And không đề xuất kích hoạt trực tiếp

Scenario: Chuyển thành đề xuất Combo
  Given Sales chọn association hợp lệ
  When gửi sang EPIC 17
  Then hệ thống chuyển snapshot insight và căn cứ
  And Combo vẫn qua kiểm tra giá, tồn và phê duyệt riêng
```

### US-DSS-09 — Phân tích mùa vụ

**Actor:** Marketing Staff (`ACT-14`), Executive (`ACT-16`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Marketing/Executive, tôi muốn phân tích nhu cầu theo mùa, lễ hội và dịp Tết để lập kế hoạch kinh doanh.

```gherkin
Scenario: So sánh sự kiện theo nhiều năm
  Given có lịch sự kiện và dữ liệu đủ dài
  When chọn mùa/lễ hội/Tết
  Then hệ thống căn chỉnh cửa sổ sự kiện theo lịch đã duyệt
  And hiển thị nhu cầu, doanh thu hoặc sản lượng cùng kỳ so sánh

Scenario: Ngày sự kiện thay đổi theo năm
  Given sự kiện không cố định theo dương lịch
  When phân tích
  Then hệ thống dùng calendar version đúng năm
  And không so sánh sai chỉ bằng ngày/tháng cố định

Scenario: Promotion gây nhiễu
  Given kỳ sự kiện có campaign/giảm giá lớn
  When xem xu hướng
  Then hệ thống hiển thị promotion context
  And không khẳng định toàn bộ tăng trưởng do mùa vụ

Scenario: Thiếu lịch sử
  Given không đủ kỳ tương đương
  When phân tích
  Then hệ thống báo mức dữ liệu và giới hạn kết luận

Scenario: Chuyển insight sang kế hoạch
  Given Marketing/Executive chọn insight
  When tạo đề xuất
  Then hệ thống chuyển snapshot sang EPIC 21 hoặc 27 phù hợp
  And không tự duyệt ngân sách, Campaign hay PO
```

### US-DSS-10 — Khuyến nghị chiến lược

**Actor:** Executive / Business Manager (`ACT-16`)

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Executive, tôi muốn nhận các khuyến nghị dựa trên dữ liệu để hỗ trợ quyết định về sản phẩm, giá, tồn kho và Marketing.

```gherkin
Scenario: Hiển thị khuyến nghị có căn cứ
  Given insight đạt ngưỡng chất lượng và còn hiệu lực
  When Executive mở danh sách
  Then mỗi khuyến nghị nêu mục tiêu, căn cứ, phạm vi, tác động dự kiến và độ tin cậy
  And liên kết tới phiên dữ liệu/mô hình nguồn

Scenario: Xung đột giữa các mục tiêu
  Given một đề xuất tăng doanh thu nhưng tăng rủi ro tồn/chi phí
  When hiển thị
  Then hệ thống nêu trade-off và giả định
  And không mô tả một phương án là chắc chắn tối ưu

Scenario: Dữ liệu stale hoặc chất lượng thấp
  Given insight vượt freshness hoặc nguồn có lỗi nghiêm trọng
  When sinh khuyến nghị
  Then hệ thống chặn hoặc gắn nhãn không đủ tin cậy theo chính sách

Scenario: Chấp nhận hoặc từ chối
  Given Executive có quyền
  When ghi nhận quyết định cùng lý do
  Then hệ thống lưu actor, thời điểm, trạng thái và Audit
  And chỉ tạo đề xuất nghiệp vụ ở Epic đích, không tự thực thi

Scenario: Theo dõi kết quả khuyến nghị
  Given khuyến nghị được triển khai qua Epic nghiệp vụ
  When có KPI thực tế
  Then hệ thống liên kết outcome với khuyến nghị gốc
  And phân biệt tương quan với quan hệ nhân quả chưa được chứng minh

Scenario: Kết quả không giải thích được
  Given hệ thống không cung cấp đủ căn cứ cho người ra quyết định
  When đánh giá khuyến nghị
  Then khuyến nghị không được coi là đủ điều kiện hành động
```

## 5. Functional Requirements

| Mã | Yêu cầu | Truy vết |
| --- | --- | --- |
| `FR-DSS-01` | Dashboard tổng hợp Finance, Order, Customer, Inventory với định nghĩa, nguồn, độ mới. | `US-DSS-01` |
| `FR-DSS-02` | So sánh kỳ và drill-down nhất quán, đúng quyền. | `US-DSS-01` |
| `FR-DSS-03` | Hiển thị partial/stale/unavailable theo từng chỉ số. | `US-DSS-01` |
| `FR-DSS-04` | Tính RFM theo cửa sổ, giao dịch đủ điều kiện và phiên bản ngưỡng. | `US-DSS-02` |
| `FR-DSS-05` | Gán phân khúc kèm điểm/lý do và xử lý danh tính chưa hợp nhất. | `US-DSS-02` |
| `FR-DSS-06` | Export phân khúc tối thiểu hóa PII, kiểm tra mục đích/quyền và Audit. | `US-DSS-02` |
| `FR-DSS-07` | Chấm churn risk kèm score/band, model version, freshness và giải thích. | `US-DSS-03` |
| `FR-DSS-08` | Không gán risk khi không đủ dữ liệu; tuân thủ privacy/consent. | `US-DSS-03` |
| `FR-DSS-09` | Chuyển audience criteria sang Marketing, không tự chạy Campaign. | `US-DSS-03` |
| `FR-DSS-10` | Phân tích Product/SKU theo kỳ, kênh, xu hướng và chỉ số có phiên bản. | `US-DSS-04` |
| `FR-DSS-11` | Phân biệt Product mới, stockout và thiếu lịch sử khỏi bán chậm. | `US-DSS-04` |
| `FR-DSS-12` | Giữ định danh/snapshot khi catalog thay đổi. | `US-DSS-04` |
| `FR-DSS-13` | Tính sales velocity theo cửa sổ, kênh và đơn vị đã duyệt. | `US-DSS-05` |
| `FR-DSS-14` | Điều chỉnh/ghi chú stockout, hủy, Return và Refund trong velocity. | `US-DSS-05` |
| `FR-DSS-15` | Chuyển snapshot velocity sang Procurement, không tự tạo PO. | `US-DSS-05` |
| `FR-DSS-16` | Forecast theo Product/SKU, phạm vi, horizon, interval và model version. | `US-DSS-06` |
| `FR-DSS-17` | Quản lý insufficient data, sự kiện bất thường và giả định forecast. | `US-DSS-06` |
| `FR-DSS-18` | Đánh giá forecast bằng actual và metric/ngưỡng đã duyệt. | `US-DSS-06` |
| `FR-DSS-19` | Tính expiry risk theo Batch, HSD, loại tồn, velocity/forecast và horizon. | `US-DSS-07` |
| `FR-DSS-20` | Không suy diễn khi HSD/tồn lỗi; lưu phiên kết quả và quality issue. | `US-DSS-07` |
| `FR-DSS-21` | Chuyển expiry insight sang Epic đích, không tự giảm giá/hủy/sửa tồn. | `US-DSS-07` |
| `FR-DSS-22` | Tính association bằng metric, ngưỡng, cửa sổ và dataset version rõ ràng. | `US-DSS-08` |
| `FR-DSS-23` | Loại/điều chỉnh giao dịch không hợp lệ và cảnh báo SKU không khả dụng. | `US-DSS-08` |
| `FR-DSS-24` | Chuyển association sang đề xuất Combo qua EPIC 17. | `US-DSS-08` |
| `FR-DSS-25` | Phân tích mùa vụ bằng event calendar có phiên bản và kỳ so sánh phù hợp. | `US-DSS-09` |
| `FR-DSS-26` | Hiển thị context Promotion, dữ liệu thiếu và giới hạn kết luận. | `US-DSS-09` |
| `FR-DSS-27` | Chuyển seasonal insight sang Marketing/Procurement, không tự phê duyệt. | `US-DSS-09` |
| `FR-DSS-28` | Khuyến nghị nêu căn cứ, mục tiêu, trade-off, tác động, confidence và freshness. | `US-DSS-10` |
| `FR-DSS-29` | Chặn/gắn nhãn insight không đủ chất lượng hoặc không giải thích được. | `US-DSS-10` |
| `FR-DSS-30` | Ghi nhận chấp nhận/từ chối, lý do, Audit và outcome liên kết. | `US-DSS-10` |
| `FR-DSS-31` | Áp dụng quyền theo chỉ số, chiều dữ liệu, export, cấu hình và hành động. | `US-DSS-01` đến `10` |
| `FR-DSS-32` | Version hóa metric, dataset, feature, model/rule, threshold và kết quả. | `US-DSS-01` đến `10` |
| `FR-DSS-33` | Theo dõi pipeline, freshness, quality, runtime và lỗi phiên phân tích. | `US-DSS-01` đến `10` |
| `FR-DSS-34` | Xử lý truy vấn/tính toán lớn không ảnh hưởng giao dịch nguồn. | `US-DSS-01` đến `10` |
| `FR-DSS-35` | Không tự động thực thi thay đổi giá, tồn, Campaign, Combo, PO hoặc thông báo. | `US-DSS-03`, `05`–`10` |

## 6. Dữ liệu, mô hình và vòng đời kết quả

```text
Analysis Run
  analysis_type + scope + data_cutoff + dataset_version
  metric/model/rule_version + status + quality_summary + timestamps

Insight / Recommendation
  subject + value/score + confidence + explanation + assumptions
  valid_from + expires_at + source_run + decision + outcome_reference

Run Status: QUEUED -> RUNNING -> COMPLETED | PARTIAL | FAILED | CANCELLED
Insight Status: ACTIVE -> ACCEPTED | REJECTED | EXPIRED | SUPERSEDED
```

- Dataset phải chống tính trùng, nêu inclusion/exclusion và không làm biến đổi nguồn.
- Metric, threshold và model/rule có owner, phiên bản, ngày hiệu lực và lịch sử thay đổi.
- Không dùng dữ liệu tương lai so với `data_cutoff`; kết quả lịch sử phải tái lập được trong phạm vi retention.
- `PARTIAL` không được trình bày như `COMPLETED`; insight hết hạn không dùng như hiện tại.
- Model/luật phải có tiêu chí đánh giá, baseline, mức chấp nhận và theo dõi drift/sai số phù hợp.
- Công nghệ lưu trữ, thuật toán, lịch chạy và công cụ trực quan thuộc Technical Design.

## 7. Traceability

| Story | FR chính | Epic nguồn/đích |
| --- | --- | --- |
| `US-DSS-01` | `FR-DSS-01`–`03`, `31`–`34` | EPIC 08/09/24 |
| `US-DSS-02` | `FR-DSS-04`–`06`, `31`–`34` | EPIC 02/08/14/21 |
| `US-DSS-03` | `FR-DSS-07`–`09`, `28`–`35` | EPIC 02/08/21 |
| `US-DSS-04` | `FR-DSS-10`–`12`, `31`–`34` | EPIC 04/08/14 |
| `US-DSS-05` | `FR-DSS-13`–`15`, `31`–`35` | EPIC 08/09/27 |
| `US-DSS-06` | `FR-DSS-16`–`18`, `31`–`35` | EPIC 08/09/27 |
| `US-DSS-07` | `FR-DSS-19`–`21`, `31`–`35` | EPIC 09/17/27 |
| `US-DSS-08` | `FR-DSS-22`–`24`, `31`–`35` | EPIC 08/14/17 |
| `US-DSS-09` | `FR-DSS-25`–`27`, `31`–`35` | EPIC 08/21/27 |
| `US-DSS-10` | `FR-DSS-28`–`35` | EPIC 17/21/24/27 |

`FR-25 — Analytics` và `FR-26 — DSS` trong Functional Requirements cấp cao được chi tiết hóa bởi `FR-DSS-01` đến `FR-DSS-35`.

## 8. Quyết định còn mở

- Xác nhận `US-DSS-01` thuộc Giai đoạn 2 hay toàn EPIC thuộc Giai đoạn 3.
- Định nghĩa từng KPI Dashboard và nguồn chuẩn từ EPIC 24/Order/Inventory/Customer.
- Cửa sổ, giao dịch đủ điều kiện, ngưỡng và taxonomy phân khúc RFM.
- Định nghĩa churn, horizon dự báo, nhãn thực tế và tiêu chí đánh giá.
- Công thức velocity; cách xử lý stockout, Return/Refund và kênh bán.
- Horizon/interval/metric chấp nhận cho Demand Forecast và ngưỡng drift.
- Công thức Expiry Risk, loại tồn sử dụng và ngưỡng hành động.
- Metric/ngưỡng association và điều kiện đủ để đề xuất Combo.
- Event calendar, cách căn chỉnh Tết/lễ hội và xử lý Promotion gây nhiễu.
- Tiêu chí đủ điều kiện, giải thích, confidence và hết hạn của khuyến nghị.
- Privacy/consent, mức tổng hợp, ngưỡng chống tái nhận diện và retention dataset/model.
- Owner phê duyệt metric/model, SLA freshness, tần suất chạy và quy trình rollback.

## 9. UI/UX Reference

- Executive Dashboard: KPI, xu hướng, freshness/quality và drill-down có context.
- Customer Analytics: RFM/churn với giải thích, bộ lọc và export có kiểm soát.
- Product & Supply Analytics: performance, velocity, forecast, expiry theo Product/SKU/Batch.
- Association & Seasonality: metric, cỡ mẫu, kỳ so sánh và cảnh báo giới hạn.
- Strategic Insights: căn cứ, trade-off, confidence, quyết định và outcome.
- Mọi màn hình cần loading, empty, insufficient data, stale, partial, failed và permission denied.

Liên kết Figma sẽ được bổ sung khi thiết kế UI/UX được phê duyệt.

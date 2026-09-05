# EPIC 24 — Finance & Business Analytics

## 1. Mục tiêu Epic

Epic này cung cấp góc nhìn tài chính nhất quán cho Executive/Business Manager và Accountant/Finance Staff: phân tích doanh thu theo thời gian, kênh, khu vực; theo dõi chi phí và lợi nhuận sản phẩm; đối soát, theo dõi thuế và xuất báo cáo/hóa đơn.

EPIC 24 sở hữu định nghĩa chỉ số tài chính đã duyệt, lớp dữ liệu báo cáo, kỳ báo cáo, báo cáo xuất và trạng thái tích hợp thuế/hóa đơn. Epic không sở hữu Order, Payment, Refund, settlement Marketplace, giao dịch Offline, Product hay phân tích/khuyến nghị nâng cao. Số liệu phải truy ngược được về nguồn, độ mới và trạng thái đối soát; dữ liệu thiếu hoặc ước tính không được trình bày như số đã chốt.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 3 — Could Have** | `US-FIN-01` đến `US-FIN-07` | Lãnh đạo và Kế toán xem báo cáo tài chính đa chiều, đối soát, theo dõi thuế và xuất báo cáo/hóa đơn theo quyền. |

EPIC 24 không có User Story thuộc MVP hoặc Giai đoạn 2 theo Product Backlog hiện tại.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và hệ thống ngoài

| Actor / hệ thống | Vai trò |
| --- | --- |
| Executive / Business Manager (`ACT-16`) | Xem, so sánh chỉ số và xuất báo cáo quản trị trong phạm vi được cấp. |
| Accountant / Finance Staff (`ACT-19`) | Kiểm tra số liệu, xử lý đối soát, theo dõi thuế và quản lý báo cáo/hóa đơn. |
| Tax / Invoice Provider (`EXT-11`) | Trao đổi dữ liệu thuế/hóa đơn khi tích hợp được bật. |
| Các Epic nghiệp vụ nguồn | Cung cấp Order, Payment, Refund, phí, chi phí, kênh, khu vực và dữ liệu tham chiếu. |

### 3.2. Business Rules và NFR áp dụng

Product Backlog chưa định nghĩa nhóm `BR-FIN`; Epic không tự tạo mã mới. Công thức/chính sách còn thiếu được ghi tại mục 8 để Product Owner và Finance chốt.

| Mã | Ảnh hưởng đến Epic |
| --- | --- |
| `BR-AUTH-03`, `BR-AUTH-04` | Chỉ tài khoản nội bộ có Role/Permission phù hợp được xem, xử lý, export hoặc phát hành hóa đơn. |
| `BR-ORDER-03`, `BR-ORDER-05` | Chỉ dùng Payment đã xác nhận theo cơ sở ghi nhận được duyệt; thay đổi trạng thái phải truy vết được. |
| `BR-MKTPLACE-03`, `BR-MKTPLACE-07` | Dữ liệu Marketplace phải có trạng thái đối soát và tách giá hàng, trợ giá, phí, số thực nhận. |
| `BR-OFFLINE-05`, `BR-OFFLINE-07` | Doanh thu Offline đến từ Order nội bộ hợp lệ và thể hiện chênh lệch tiền thực thu. |
| `BR-REFUND-01` đến `BR-REFUND-04` | Refund phải liên kết Order, đúng giới hạn/phương thức và được phản ánh không mất dấu vết. |
| `BR-AUDIT-01` | Phân bổ, điều chỉnh, xử lý sai lệch, khóa/mở kỳ, export và hóa đơn là thao tác cần Audit. |
| `NFR-04`, `NFR-05`, `NFR-08` | Bảo vệ dữ liệu tài chính/PII và giới hạn theo quyền, phạm vi, mục đích. |
| `NFR-06` | Tổng hợp không được che giấu sai lệch giữa Order, Payment và nguồn liên quan. |
| `NFR-10` | Truy vấn lớn và export phải phân trang/giới hạn hoặc xử lý nền. |
| `NFR-11` | Theo dõi độ mới dữ liệu, lỗi tổng hợp và tình trạng tích hợp `EXT-11`. |

### 3.3. Phụ thuộc và ranh giới

- `EPIC 04`, `EPIC 08`: cung cấp Product/SKU và Order/snapshot; Finance không sửa dữ liệu nguồn.
- `EPIC 07`: sở hữu Payment và đối soát Payment–Order; Finance dùng kết quả đã xác nhận.
- `EPIC 12`, `EPIC 13`: sở hữu settlement/đối soát Marketplace và giao dịch/đối soát Offline.
- `EPIC 14`: sở hữu Return/Refund; Finance phản ánh tác động theo chính sách kỳ.
- `EPIC 18`: cung cấp snapshot thông tin doanh nghiệp/hóa đơn B2B.
- `EPIC 21`: cung cấp chi phí Marketing đã phê duyệt khi dùng trong báo cáo.
- `EPIC 22`, `EPIC 23`: cấp quyền và lưu Audit cho thao tác tài chính nhạy cảm.
- `EPIC 25`: sở hữu Customer Analytics, dự báo, DSS/khuyến nghị; EPIC 24 chỉ cung cấp chỉ số tài chính.
- `EXT-11` là tích hợp tùy chọn; lỗi bên ngoài không được làm mất Order hoặc dữ liệu tài chính nội bộ.

## 4. User Stories chi tiết

### US-FIN-01 — Doanh thu theo thời gian

**Actor:** `ACT-16`, `ACT-19`

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Giám đốc hoặc Kế toán, tôi muốn xem doanh thu theo ngày/tháng/năm để đánh giá tình hình kinh doanh.

```gherkin
Scenario: Xem doanh thu theo kỳ
  Given người dùng có quyền và dữ liệu đã được tổng hợp
  When chọn khoảng thời gian cùng độ phân giải ngày, tháng hoặc năm
  Then hệ thống hiển thị từng kỳ và tổng khoảng thời gian
  And nêu timezone, tiền tệ, độ mới và định nghĩa chỉ số

Scenario: So sánh với kỳ trước
  Given một khoảng thời gian hợp lệ
  When bật so sánh kỳ trước hoặc cùng kỳ
  Then hệ thống dùng biên thời gian nhất quán
  And hiển thị giá trị cùng tỷ lệ thay đổi khi mẫu số hợp lệ

Scenario: Có hủy, Return hoặc Refund
  Given nghiệp vụ giảm trừ đã có hiệu lực
  When báo cáo được tính
  Then hệ thống phản ánh theo chính sách ghi nhận đã duyệt
  And phân biệt doanh thu gốc, giảm trừ và doanh thu thuần

Scenario: Dữ liệu đến muộn hoặc điều chỉnh
  Given dữ liệu kỳ trước đến sau hoặc được điều chỉnh
  When tổng hợp lại
  Then hệ thống áp dụng chính sách kỳ mở/đóng
  And hiển thị phiên bản hoặc thời điểm làm mới

Scenario: Nguồn chưa hoàn chỉnh
  Given một nguồn chưa đồng bộ hoặc đối soát
  When xem báo cáo
  Then hệ thống đánh dấu phạm vi chưa hoàn chỉnh
  And không diễn giải dữ liệu thiếu thành doanh thu bằng không

Scenario: Không có quyền
  Given người dùng thiếu quyền xem Finance
  When truy cập báo cáo hoặc drill-down
  Then hệ thống từ chối và không tiết lộ số liệu
```

### US-FIN-02 — Doanh thu theo kênh

**Actor:** `ACT-16`, `ACT-19`

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Giám đốc hoặc Kế toán, tôi muốn xem doanh thu theo Website, App, Marketplace, B2B và Offline.

```gherkin
Scenario: Phân bổ theo kênh nguồn
  Given Order có snapshot kênh hợp lệ
  When xem báo cáo theo kênh
  Then hệ thống phân loại theo taxonomy đã chốt
  And tổng các kênh cùng nhóm chưa xác định khớp tổng chung

Scenario: Drill-down một kênh
  Given người dùng có quyền chi tiết
  When chọn kênh và kỳ
  Then hệ thống giữ bộ lọc, định nghĩa và trạng thái đối soát
  And chỉ trả dữ liệu nguồn được phép

Scenario: Kênh thiếu hoặc chưa ánh xạ
  Given giá trị kênh không hợp lệ
  When tổng hợp
  Then hệ thống đưa vào nhóm chưa xác định
  And không tự gán sang Website

Scenario: Marketplace có nhiều thành phần tiền
  Given có giá hàng, giảm giá, trợ giá, phí và số thực nhận
  When xem kênh Marketplace
  Then hệ thống trình bày riêng theo định nghĩa
  And không gọi số thực nhận là doanh thu gộp

Scenario: Ánh xạ kênh thay đổi
  Given cấu hình đổi sau khi Order đã chốt
  When xem lịch sử
  Then hệ thống áp dụng versioning đã duyệt
  And không sửa hồi tố mất dấu vết

Scenario: Quyền bị giới hạn theo kênh
  Given người dùng chỉ được xem một số kênh
  When xem hoặc export đa kênh
  Then hệ thống loại dữ liệu ngoài phạm vi kể cả tổng và drill-down
```

### US-FIN-03 — Chi phí bán hàng và vận hành

**Actor:** `ACT-16`, `ACT-19`

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Giám đốc hoặc Kế toán, tôi muốn xem chi phí bán hàng và vận hành để đánh giá hiệu quả kinh doanh.

```gherkin
Scenario: Xem chi phí theo kỳ và nhóm
  Given dữ liệu đến từ nguồn được phê duyệt
  When chọn kỳ và nhóm chi phí
  Then hệ thống hiển thị tổng, nhóm con, nguồn và trạng thái xác nhận

Scenario: Phân bổ chi phí dùng chung
  Given chi phí áp dụng cho nhiều kênh hoặc sản phẩm
  When áp dụng quy tắc hợp lệ
  Then hệ thống lưu phiên bản, căn cứ và kết quả phân bổ
  And tổng phân bổ đối chiếu được với giá trị gốc

Scenario: Chưa có căn cứ phân bổ
  Given thiếu quy tắc hoặc dữ liệu
  When tổng hợp
  Then chi phí nằm trong nhóm chưa phân bổ
  And không tự chia đều

Scenario: Điều chỉnh chi phí
  Given Accountant có quyền và chứng từ hợp lệ
  When ghi nhận điều chỉnh cùng lý do
  Then hệ thống tạo bản ghi liên kết thay vì ghi đè dữ liệu gốc

Scenario: Nguồn chi phí chưa đầy đủ
  Given một nguồn chưa đồng bộ hoặc xác nhận
  When xem tổng chi phí
  Then báo cáo được đánh dấu chưa hoàn chỉnh và nêu phạm vi ảnh hưởng

Scenario: Không có quyền xem chi phí
  Given người dùng chỉ có quyền doanh thu
  When xem chỉ số dẫn xuất hoặc export
  Then hệ thống không tiết lộ hay cho suy ra chi phí bị hạn chế
```

### US-FIN-04 — Lợi nhuận theo sản phẩm

**Actor:** `ACT-16`, `ACT-19`

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Giám đốc hoặc Kế toán, tôi muốn xem lợi nhuận theo sản phẩm để biết sản phẩm nào mang lại giá trị cao.

```gherkin
Scenario: Tính lợi nhuận theo công thức đã duyệt
  Given doanh thu, giảm trừ, giá vốn và chi phí cần thiết khả dụng
  When xem báo cáo
  Then hệ thống hiển thị từng thành phần, lợi nhuận và phiên bản công thức

Scenario: Xem theo Product hoặc SKU
  Given Product có nhiều SKU
  When đổi cấp tổng hợp
  Then tổng SKU đối chiếu được với Product cùng bộ lọc

Scenario: Thiếu giá vốn hoặc chi phí
  Given một số dòng thiếu dữ liệu bắt buộc
  When tính lợi nhuận
  Then hệ thống đánh dấu chưa xác định/chưa hoàn chỉnh
  And không mặc định giá vốn bằng không

Scenario: Return hoặc Refund
  Given sản phẩm có nghiệp vụ hợp lệ
  When tính theo kỳ
  Then doanh thu, giá vốn hoàn nhập và chi phí được xử lý theo chính sách

Scenario: Product đổi tên hoặc ngừng bán
  Given catalog thay đổi sau giao dịch
  When xem lịch sử
  Then hệ thống giữ định danh ổn định và nhãn/snapshot phù hợp

Scenario: Sắp xếp theo lợi nhuận
  Given dữ liệu đủ điều kiện
  When xếp hạng theo lợi nhuận hoặc biên lợi nhuận
  Then hệ thống dùng cùng công thức và xử lý rõ số âm/mẫu số không hợp lệ
```

### US-FIN-05 — Doanh thu theo khu vực

**Actor:** `ACT-16`, `ACT-19`

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Giám đốc hoặc Kế toán, tôi muốn xem doanh thu theo khu vực để đánh giá thị trường.

```gherkin
Scenario: Tổng hợp theo khu vực
  Given Order có dữ liệu địa lý đủ điều kiện
  When chọn cấp khu vực được hỗ trợ
  Then hệ thống tổng hợp theo kỳ
  And nêu nguồn xác định từ giao hàng, điểm bán hay quy tắc khác

Scenario: Drill-down có bảo vệ riêng tư
  Given người dùng chọn một khu vực
  When xem chi tiết
  Then chỉ hiển thị mức tổng hợp hoặc giao dịch được cấp quyền
  And không lộ địa chỉ cá nhân không cần thiết

Scenario: Không ánh xạ được địa lý
  Given Order thiếu hoặc sai dữ liệu địa lý
  When tổng hợp
  Then hệ thống giữ nhóm chưa xác định thay vì loại bỏ âm thầm

Scenario: Khách đổi địa chỉ sau mua
  Given hồ sơ hiện tại khác snapshot Order
  When xem lịch sử
  Then hệ thống dùng snapshot tại giao dịch theo chính sách

Scenario: Hủy hoặc Refund
  Given Order khu vực có giảm trừ hợp lệ
  When tính báo cáo
  Then tác động thống nhất với US-FIN-01 và tổng khu vực khớp tổng chung

Scenario: Nhóm quá nhỏ hoặc nhạy cảm
  Given bộ lọc có nguy cơ nhận diện cá nhân
  When xem hoặc export
  Then hệ thống ẩn/gộp theo ngưỡng riêng tư đã duyệt
```

### US-FIN-06 — Đối soát và nghĩa vụ thuế

**Actor:** `ACT-19`

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Kế toán, tôi muốn đối soát giao dịch và theo dõi nghĩa vụ thuế qua `EXT-11` khi tích hợp để quản lý tài chính.

```gherkin
Scenario: Tổng hợp kết quả đối soát
  Given Payment, Marketplace và Offline cung cấp kết quả
  When mở kỳ đối soát tài chính
  Then hệ thống hiển thị số khớp, chưa khớp, đang xử lý và chênh lệch
  And cho truy vết nguồn mà không tự sửa dữ liệu nguồn

Scenario: Phát hiện chênh lệch
  Given tổng báo cáo không khớp nguồn ngoài ngưỡng
  When kiểm tra kỳ
  Then hệ thống tạo ngoại lệ với loại, chênh lệch và tham chiếu
  And không cho chốt kỳ khi còn điều kiện chặn

Scenario: Xử lý ngoại lệ có thẩm quyền
  Given Accountant có quyền và bằng chứng
  When ghi nhận phương án cùng lý do
  Then trạng thái được cập nhật, dữ liệu gốc được giữ và Audit được tạo

Scenario: Theo dõi thuế theo kỳ
  Given chính sách và dữ liệu tính thuế khả dụng
  When chọn kỳ thuế
  Then hệ thống hiển thị căn cứ, số dự kiến/xác nhận/đã nộp và hạn
  And phân biệt số nội bộ với xác nhận EXT-11

Scenario: EXT-11 xử lý thành công
  Given tích hợp được bật và yêu cầu hợp lệ
  When gửi hoặc nhận dữ liệu
  Then hệ thống lưu tham chiếu, trạng thái, thời điểm đúng kỳ
  And yêu cầu lặp không tạo hồ sơ trùng

Scenario: EXT-11 lỗi hoặc timeout
  Given nhà cung cấp không xác nhận
  When đồng bộ
  Then hệ thống giữ trạng thái chờ/lỗi và cho retry an toàn
  And không đánh dấu đã xác nhận hoặc đã nộp

Scenario: Không có quyền xử lý
  Given người dùng chỉ được xem
  When xử lý ngoại lệ, khóa kỳ hoặc đồng bộ thuế
  Then hệ thống từ chối và không đổi trạng thái
```

### US-FIN-07 — Xuất báo cáo và hóa đơn

**Actor:** `ACT-16`, `ACT-19`

**Ưu tiên:** Could Have — **Phát hành:** Giai đoạn 3

> Là Giám đốc hoặc Kế toán, tôi muốn xuất báo cáo tài chính/kinh doanh và kết nối `EXT-11` khi cần để phục vụ họp và ra quyết định.

```gherkin
Scenario: Tạo file theo bộ lọc
  Given người dùng có quyền và chọn kỳ, phạm vi, chỉ số, định dạng
  When yêu cầu export
  Then hệ thống tạo Export Job với snapshot tham số và quyền
  And file phản ánh đúng phiên bản dữ liệu

Scenario: Báo cáo lớn
  Given phạm vi vượt ngưỡng trực tiếp
  When gửi yêu cầu
  Then hệ thống xử lý nền và hiển thị trạng thái
  And không gây nghẽn luồng giao dịch

Scenario: File có dữ liệu nhạy cảm
  Given báo cáo chứa chi phí, thuế hoặc PII
  When tạo file
  Then hệ thống áp dụng masking, phạm vi quyền và thời hạn tải

Scenario: Tái lập báo cáo
  Given file đã hoàn tất
  When xem metadata
  Then hệ thống cho biết tham số, thời điểm, độ mới và phiên bản chính sách

Scenario: Phát hành hóa đơn
  Given Order đủ điều kiện và snapshot hóa đơn hợp lệ
  When Accountant yêu cầu phát hành
  Then hệ thống gửi đúng dữ liệu, lưu tham chiếu/trạng thái
  And yêu cầu lặp không tạo hóa đơn trùng

Scenario: Nhà cung cấp hóa đơn lỗi
  Given EXT-11 từ chối, timeout hoặc chưa rõ kết quả
  When xử lý phản hồi
  Then không ghi đã phát hành nếu thiếu bằng chứng
  And cho retry/đối soát an toàn

Scenario: Điều chỉnh, thay thế hoặc hủy
  Given hóa đơn đã phát hành cần thay đổi hợp lệ
  When Accountant thao tác với lý do và tham chiếu
  Then hệ thống giữ quan hệ với chứng từ gốc và không xóa lịch sử

Scenario: Không có quyền export/phát hành
  Given người dùng chỉ được xem dashboard
  When export hoặc phát hành hóa đơn
  Then hệ thống từ chối và ghi nhận theo chính sách Audit
```

## 5. Functional Requirements

| Mã | Yêu cầu | Truy vết |
| --- | --- | --- |
| `FR-FIN-01` | Tổng hợp doanh thu theo ngày/tháng/năm, timezone, tiền tệ, độ mới và định nghĩa. | `US-FIN-01` |
| `FR-FIN-02` | So sánh kỳ trước/cùng kỳ bằng biên thời gian nhất quán. | `US-FIN-01` |
| `FR-FIN-03` | Tách doanh thu gốc, giảm trừ và doanh thu thuần theo chính sách. | `US-FIN-01`, `04` |
| `FR-FIN-04` | Quản lý dữ liệu đến muộn, điều chỉnh và trạng thái kỳ không mất dấu vết. | `US-FIN-01`, `06` |
| `FR-FIN-05` | Phân loại kênh Website/App/Marketplace/B2B/Offline/chưa xác định. | `US-FIN-02` |
| `FR-FIN-06` | Đối chiếu tổng theo chiều phân tích với tổng chung. | `US-FIN-02`, `04`, `05` |
| `FR-FIN-07` | Drill-down tới nguồn trong phạm vi quyền. | `US-FIN-01`, `02`, `04`, `05` |
| `FR-FIN-08` | Tách các thành phần tiền Marketplace theo định nghĩa. | `US-FIN-02` |
| `FR-FIN-09` | Tổng hợp chi phí theo kỳ, nhóm, nguồn và trạng thái. | `US-FIN-03` |
| `FR-FIN-10` | Phiên bản hóa quy tắc/kết quả phân bổ, bảo toàn giá trị gốc. | `US-FIN-03` |
| `FR-FIN-11` | Hiển thị chi phí chưa phân bổ hoặc dữ liệu thiếu. | `US-FIN-03`, `04` |
| `FR-FIN-12` | Ghi điều chỉnh có tham chiếu, actor, thời điểm, lý do. | `US-FIN-03` |
| `FR-FIN-13` | Tính lợi nhuận Product/SKU theo công thức có phiên bản. | `US-FIN-04` |
| `FR-FIN-14` | Không công bố lợi nhuận hoàn chỉnh khi thiếu giá vốn/chi phí. | `US-FIN-04` |
| `FR-FIN-15` | Giữ định danh/snapshot Product/SKU cho báo cáo lịch sử. | `US-FIN-04` |
| `FR-FIN-16` | Tổng hợp theo cấp khu vực và nguồn địa lý được duyệt. | `US-FIN-05` |
| `FR-FIN-17` | Giữ nhóm chưa xác định và áp dụng ngưỡng riêng tư. | `US-FIN-05` |
| `FR-FIN-18` | Tổng hợp kết quả đối soát nguồn mà không sửa dữ liệu nguồn. | `US-FIN-06` |
| `FR-FIN-19` | Phát hiện/phân loại/theo dõi ngoại lệ vượt ngưỡng. | `US-FIN-06` |
| `FR-FIN-20` | Kiểm soát quyền xử lý ngoại lệ và khóa/mở kỳ; bắt buộc Audit. | `US-FIN-06` |
| `FR-FIN-21` | Theo dõi căn cứ, thuế dự kiến/xác nhận/đã nộp và hạn. | `US-FIN-06` |
| `FR-FIN-22` | Phân biệt tính nội bộ với xác nhận `EXT-11`. | `US-FIN-06` |
| `FR-FIN-23` | Tích hợp `EXT-11` chống trùng, retry/đối soát an toàn. | `US-FIN-06`, `07` |
| `FR-FIN-24` | Tạo Export Job từ snapshot bộ lọc, quyền và phiên bản dữ liệu. | `US-FIN-07` |
| `FR-FIN-25` | Xử lý export lớn ở nền và cung cấp trạng thái. | `US-FIN-07` |
| `FR-FIN-26` | Bảo vệ file bằng quyền, masking và thời hạn truy cập. | `US-FIN-07` |
| `FR-FIN-27` | Phát hành hóa đơn từ Order/snapshot hợp lệ, lưu tham chiếu ngoài. | `US-FIN-07`, `FR-24` |
| `FR-FIN-28` | Chống hóa đơn trùng và liên kết chứng từ điều chỉnh/thay thế/hủy. | `US-FIN-07` |
| `FR-FIN-29` | Tách quyền xem doanh thu/chi phí/lợi nhuận/thuế, xử lý, export, hóa đơn. | `US-FIN-01` đến `07` |
| `FR-FIN-30` | Hiển thị dữ liệu đầy đủ/chưa hoàn chỉnh/chưa đối soát rõ ràng. | `US-FIN-01` đến `06` |
| `FR-FIN-31` | Audit phân bổ, điều chỉnh, ngoại lệ, kỳ, export và vòng đời hóa đơn. | `US-FIN-03`, `06`, `07` |
| `FR-FIN-32` | Quan sát lỗi/độ trễ tổng hợp, Export Job và `EXT-11`. | `US-FIN-01` đến `07` |

## 6. Dữ liệu và quy tắc nhất quán

```text
Financial Fact
  source_type + source_id + source_version + event_type
  occurred_at + recognized_period + channel + region + product_id + sku_id
  gross + discount + refund + fee + tax + net + currency
  reconciliation_status + data_quality_status

Reporting Period: OPEN -> REVIEWING -> CLOSED
Reconciliation: NOT_CHECKED -> MATCHED | MISMATCH -> IN_REVIEW -> RESOLVED
Export Job: QUEUED -> PROCESSING -> COMPLETED | FAILED | EXPIRED
External Tax/Invoice: NOT_SENT -> PENDING -> ACCEPTED | REJECTED | UNKNOWN
```

- Khóa nguồn + phiên bản/sự kiện phải chống tính trùng khi nhận lại dữ liệu.
- `occurred_at`, thời điểm ghi nhận tài chính và thời điểm vào báo cáo là ba khái niệm riêng.
- Điều chỉnh liên kết bản ghi gốc; báo cáo mang `data_cutoff_at` và `policy_version`.
- `UNKNOWN` không phải `ACCEPTED`; mở kỳ đã đóng là thao tác đặc quyền có lý do/Audit.
- Mỗi chỉ số phải có định nghĩa, công thức, nguồn, owner, phiên bản và ngày hiệu lực.
- Công nghệ tổng hợp, biểu đồ và giao thức tích hợp thuộc Technical Design.

## 7. Traceability

| User Story | FR chính | Nguồn/đích |
| --- | --- | --- |
| `US-FIN-01` | `FR-FIN-01`–`04`, `07`, `30`, `32` | Order, Payment, Refund |
| `US-FIN-02` | `FR-FIN-05`–`08`, `29`, `30` | Website/App, EPIC 12/13/18 |
| `US-FIN-03` | `FR-FIN-09`–`12`, `29`–`31` | Phí/chi phí nguồn |
| `US-FIN-04` | `FR-FIN-03`, `06`, `11`, `13`–`15` | EPIC 04/08/09/14 |
| `US-FIN-05` | `FR-FIN-06`, `07`, `16`, `17`, `29` | Order/Shipping/Offline |
| `US-FIN-06` | `FR-FIN-18`–`23`, `29`–`32` | EPIC 07/12/13/23, EXT-11 |
| `US-FIN-07` | `FR-FIN-23`–`29`, `31`, `32` | EPIC 18/22/23, EXT-11 |

`FR-24 — Finance` trong Functional Requirements cấp cao được chi tiết hóa bởi `FR-FIN-01` đến `FR-FIN-32`.

## 8. Quyết định còn mở

- Cơ sở ghi nhận doanh thu và xử lý COD, hủy, Return, Refund.
- Công thức doanh thu gộp/thuần, giá vốn, lợi nhuận, biên lợi nhuận và làm tròn.
- Nguồn/taxonomy chi phí và cách phân bổ chi phí chung.
- Kỳ kế toán, timezone, khóa/mở kỳ và dữ liệu đến muộn.
- Chính sách tiền tệ/tỷ giá nếu hỗ trợ đa tiền tệ.
- Taxonomy kênh; nguồn/cấp khu vực và ngưỡng bảo vệ riêng tư.
- Quy tắc thuế/hóa đơn theo quy định pháp luật hiện hành.
- Nhà cung cấp/phạm vi `EXT-11`, ký/phát hành/điều chỉnh/hủy và xử lý gián đoạn.
- Định dạng, giới hạn, bảo vệ, retention của file export.
- SLA độ mới, ngưỡng sai lệch, điều kiện khóa kỳ và owner ngoại lệ.

## 9. UI/UX Reference

- Finance Dashboard: bộ lọc kỳ, kênh, khu vực, Product/SKU; chỉ số kèm định nghĩa và độ mới.
- Revenue/Cost/Profit: biểu đồ và bảng chung bộ lọc, drill-down giữ context/trạng thái đối soát.
- Reconciliation Workspace: ngoại lệ theo nguồn/trạng thái, expected–actual và xử lý có lý do.
- Tax & Invoice Workspace: kỳ thuế, nghĩa vụ, đồng bộ, hóa đơn và lịch sử điều chỉnh.
- Export Center: job, tham số, requester, trạng thái, thời hạn file và lỗi có thể hành động.
- Các màn hình phải có loading, empty, partial/stale, permission denied, integration unavailable và export failed.

Liên kết Figma sẽ được bổ sung khi thiết kế UI/UX được phê duyệt.

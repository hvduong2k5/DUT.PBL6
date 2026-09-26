# EPIC 13 — Offline Sales

## 1. Mục tiêu Epic

Epic này quản lý hoạt động bán hàng tại cửa hàng, chợ, tạp hóa, đại lý hoặc điểm bán lưu động. Offline Sales Staff nhận hàng từ kho, ghi nhận từng giao dịch bán, báo cáo tồn/hỏng/trả theo kỳ; Sales Manager theo dõi doanh thu/tồn và đối soát hàng hóa cùng tiền thực thu.

Mỗi giao dịch bán Offline phải trở thành một Order nội bộ để dữ liệu khách hàng, SKU, doanh thu và tồn kho được thống nhất với các kênh khác. Hàng được cấp cho nhân viên/điểm bán vẫn phải truy xuất theo SKU/Batch và mọi chênh lệch phải được giải thích bằng giao dịch bán, tồn cuối kỳ, hàng hỏng/thất thoát hoặc hàng hoàn trả.

Epic không quản lý Product/SKU và chính sách giá gốc, tự sửa số dư Inventory, xử lý Payment dùng chung, duyệt Return/Refund, quản trị quyền hay lập báo cáo tài chính tổng hợp. Các trách nhiệm này lần lượt thuộc EPIC 04, EPIC 09, EPIC 07, EPIC 14, EPIC 22 và EPIC 24.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 2** | `US-OFF-01` đến `US-OFF-06` | Điểm bán quản lý được hàng nhận, giao dịch bán, tồn/hỏng/trả, báo cáo định kỳ và đối soát hàng hóa/doanh thu. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Offline Customer (`ACT-05`) | Mua trực tiếp tại điểm bán; có thể là khách vãng lai hoặc được liên kết Customer khi có đủ thông tin/đồng ý. |
| Warehouse Staff (`ACT-06`) | Chuẩn bị, bàn giao và nhận lại hàng từ nhân viên/điểm bán qua nghiệp vụ Inventory. |
| Offline Sales Staff (`ACT-10`) | Nhận hàng, ghi nhận bán, kiểm đếm tồn/hỏng/trả và nộp báo cáo theo kỳ. |
| Sales Manager (`ACT-12`) | Theo dõi doanh thu/tồn, kiểm soát giá/chiết khấu và đối soát hoạt động điểm bán. |
| Accountant / Finance Staff (`ACT-19`) | Kiểm tra tiền thực thu và sử dụng dữ liệu đã đối soát cho nghiệp vụ tài chính theo quyền. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-04` | Nhân viên chỉ được thực hiện thao tác thuộc quyền được cấp. | Nhận hàng, bán, điều chỉnh, nộp báo cáo và đối soát đều phải kiểm tra actor/quyền. |
| `BR-PROD-02` | Giá, quy cách và tồn kho được quản lý theo SKU. | Cấp hàng và giao dịch bán phải ghi đúng SKU, số lượng và giá áp dụng. |
| `BR-BATCH-01` | Tồn thực phẩm phải truy xuất theo Batch/Lot. | Hàng cấp/bán/trả/hỏng cần duy trì Batch khi chính sách yêu cầu truy xuất. |
| `BR-BATCH-05` | Hàng hết hạn không được phép tiếp tục bán. | Hàng hết hạn phải bị chặn bán và tách khỏi lượng khả dụng tại điểm bán. |
| `BR-BATCH-06` | Nên ưu tiên Batch có HSD gần hơn khi phù hợp chính sách. | Kho/điểm bán áp dụng phân bổ FEFO có điều kiện từ EPIC 09. |
| `BR-ORDER-01` | Order phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Mỗi giao dịch Offline tạo Order với nguồn/trạng thái phù hợp và không được ghi nhận bán ngoài Order. |
| `BR-ORDER-05` | Thay đổi trạng thái quan trọng phải được ghi nhận. | Hủy/điều chỉnh giao dịch Offline phải có lịch sử và nguồn thao tác. |
| `BR-OFFLINE-01` | Offline Sales Staff phải được xác định danh tính. | Không cho ghi nhận nhận hàng, bán hoặc báo cáo dưới actor không xác định. |
| `BR-OFFLINE-02` | Hàng lấy từ kho cho Offline Staff phải được ghi nhận. | Mọi lần cấp hàng phải có chứng từ, SKU/Batch, số lượng, người giao/nhận và điểm bán. |
| `BR-OFFLINE-03` | Nhân viên báo cáo bán, tồn, hỏng và trả theo kỳ. | Báo cáo phải phân loại đủ các lượng để đối soát hàng đã cấp. |
| `BR-OFFLINE-04` | Tồn Offline phải được đối soát với tồn hệ thống. | Chênh lệch không được tự điều chỉnh mất dấu vết; phải qua quy trình xử lý có quyền. |
| `BR-OFFLINE-05` | Mỗi giao dịch bán Offline phải được ghi thành Order nội bộ. | Ghi bán và tạo Order là một kết quả nghiệp vụ thống nhất, chống tạo trùng. |
| `BR-OFFLINE-06` | Staff chỉ bán đúng giá hoặc chiết khấu trong thẩm quyền. | Hệ thống phải kiểm tra giá/ưu đãi tại thời điểm xác nhận giao dịch. |
| `BR-OFFLINE-07` | Tiền thực thu cuối ca phải khớp tổng giá trị hàng đã bán. | Đối soát phải phân tách tiền mặt/chuyển khoản và ghi rõ mọi chênh lệch. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Điều chỉnh giá, hủy giao dịch, chênh lệch và xử lý đối soát phải truy vết được. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 01/02 — Authentication & Customer Profile`: xác định nhân viên và liên kết Customer khi khách cung cấp thông tin hợp lệ; không bắt buộc khách tại điểm bán tạo tài khoản.
- `EPIC 04 — Product, Variant & SKU`: cung cấp SKU, giá Offline, quy cách và trạng thái bán.
- `EPIC 07 — Payment`: ghi nhận/đối soát khoản thu tiền mặt hoặc chuyển khoản theo mô hình Payment được chốt.
- `EPIC 08 — Order Management`: tạo/lưu Order nguồn Offline và quản lý vòng đời sau ghi nhận.
- `EPIC 09 — Inventory & Batch`: cấp/chuyển tồn đến điểm bán, ghi nhận bán/hỏng/trả và áp dụng điều chỉnh được duyệt.
- `EPIC 14 — Return / Refund / Complaint`: xử lý khách trả hàng/hoàn tiền; không nhầm với hàng tồn chưa bán trả về kho.
- `EPIC 17 — Promotion & Loyalty`: cung cấp khuyến mãi/chiết khấu hợp lệ và tích điểm khi phạm vi được triển khai.
- `EPIC 22 — User / Role / Permission Administration`: quyết định thẩm quyền giá, chiết khấu, điều chỉnh và đối soát.
- `EPIC 23 — Audit & Security`: lưu dấu vết thao tác nhạy cảm.
- `EPIC 24 — Finance & Business Analytics`: sử dụng dữ liệu Offline đã đối soát cho báo cáo tài chính đa kênh.

## 4. User Stories chi tiết

### US-OFF-01 — Nhận hàng từ kho

**Actor:** Offline Sales Staff (`ACT-10`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Offline Sales Staff, tôi muốn nhận hàng từ kho để mang đi bán tại điểm bán.

**Giá trị nghiệp vụ:** Hàng được bàn giao có người chịu trách nhiệm, đúng SKU/Batch và số lượng trước khi bắt đầu bán.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xác nhận nhận hàng hợp lệ
  Given Warehouse Staff đã tạo chứng từ cấp hàng cho một nhân viên/điểm bán
  When Offline Sales Staff kiểm tra và xác nhận đúng SKU, Batch cùng số lượng thực nhận
  Then hệ thống ghi nhận người giao, người nhận, điểm bán và thời điểm
  And EPIC 09 chuyển số lượng sang phạm vi tồn Offline tương ứng
  And chứng từ sẵn sàng làm số đầu kỳ để đối soát

Scenario: Số lượng thực nhận không khớp
  Given chứng từ cấp hàng có số lượng dự kiến
  When Offline Sales Staff ghi nhận số thực nhận khác dự kiến
  Then hệ thống không tự coi chứng từ đã khớp hoàn toàn
  And lưu chênh lệch theo SKU/Batch để Warehouse Staff hoặc Manager xử lý

Scenario: Batch không đủ điều kiện bán
  Given chứng từ có Batch hết hạn, bị cách ly hoặc không còn đủ điều kiện
  When Staff kiểm tra nhận hàng
  Then hệ thống không đưa số lượng đó vào tồn có thể bán tại điểm
  And yêu cầu xử lý qua EPIC 09

Scenario: Người không đúng phân công nhận hàng
  Given chứng từ được giao cho Staff/điểm bán cụ thể
  When người khác không có quyền cố xác nhận nhận
  Then hệ thống từ chối thao tác
  And trách nhiệm hàng hóa không bị chuyển sai

Scenario: Xác nhận nhận bị gửi lặp
  Given chứng từ đã được xác nhận nhận thành công
  When cùng yêu cầu được gửi lại
  Then hệ thống không chuyển tồn thêm lần nữa
  And trả về kết quả nhất quán của chứng từ hiện có
```

### US-OFF-02 — Ghi nhận số lượng hàng đã bán

**Actor:** Offline Sales Staff (`ACT-10`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Offline Sales Staff, tôi muốn ghi nhận số lượng hàng đã bán để cập nhật tình hình kinh doanh.

**Giá trị nghiệp vụ:** Mỗi lần bán làm giảm đúng tồn điểm bán, tạo Order và ghi nhận đúng khoản tiền phải thu/thực thu.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Ghi nhận giao dịch bán hợp lệ
  Given Staff đã được cấp hàng và SKU còn đủ tồn Offline có thể bán
  When Staff xác nhận SKU, số lượng, giá hợp lệ và phương thức thanh toán
  Then hệ thống tạo một Order nguồn Offline
  And ghi giảm đúng tồn Offline theo SKU/Batch qua EPIC 09
  And ghi nhận nghĩa vụ/kết quả thu tiền phù hợp qua EPIC 07

Scenario: Khách vãng lai mua tại điểm bán
  Given Offline Customer không có tài khoản
  When Staff ghi nhận giao dịch hợp lệ
  Then hệ thống vẫn tạo Order mà không bắt buộc tạo Customer Account
  And chỉ thu thập dữ liệu khách cần thiết theo chính sách giao dịch

Scenario: Không đủ tồn Offline
  Given số lượng khách mua lớn hơn tồn Offline có thể bán của Staff/điểm bán
  When Staff xác nhận giao dịch
  Then hệ thống từ chối số lượng vượt khả dụng
  And không tạo Order hoàn tất hoặc khoản thu không có hàng tương ứng

Scenario: Giá hoặc chiết khấu vượt thẩm quyền
  Given Staff nhập giá khác giá Offline hoặc chiết khấu vượt mức được cấp
  When Staff xác nhận giao dịch
  Then hệ thống từ chối áp dụng giá ngoài thẩm quyền
  And yêu cầu giá hợp lệ hoặc phê duyệt theo chính sách

Scenario: Hàng hết hạn
  Given SKU còn số lượng vật lý tại điểm bán nhưng Batch đã hết hạn
  When Staff cố ghi nhận bán
  Then hệ thống từ chối giao dịch đối với lượng hết hạn
  And yêu cầu tách hàng khỏi tồn có thể bán

Scenario: Yêu cầu bán bị gửi lặp
  Given một giao dịch Offline đã tạo Order thành công
  When cùng yêu cầu được gửi lại
  Then hệ thống không tạo Order, trừ tồn hoặc ghi nhận tiền lần thứ hai
```

### US-OFF-03 — Ghi nhận tồn, hỏng hoặc hàng trả lại

**Actor:** Offline Sales Staff (`ACT-10`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Offline Sales Staff, tôi muốn ghi nhận hàng còn tồn, hỏng hoặc trả lại.

**Giá trị nghiệp vụ:** Số lượng tại điểm bán phản ánh thực tế và phân biệt được hàng còn bán, hàng hỏng với hàng được trả về kho.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Ghi nhận kiểm đếm tồn cuối kỳ
  Given Staff đang kiểm đếm hàng thuộc phạm vi của mình
  When Staff nhập số lượng thực tế theo SKU/Batch
  Then hệ thống lưu kết quả kiểm đếm cùng thời điểm
  And tính chênh lệch với số dư hệ thống mà chưa tự ý xóa lịch sử

Scenario: Ghi nhận hàng hỏng
  Given Staff phát hiện hàng hỏng tại điểm bán
  When Staff ghi nhận SKU/Batch, số lượng, lý do và bằng chứng nếu bắt buộc
  Then lượng hàng được tách khỏi tồn có thể bán
  And yêu cầu biến động Inventory được tạo theo thẩm quyền/phê duyệt

Scenario: Trả hàng chưa bán về kho
  Given điểm bán còn hàng chưa bán đủ điều kiện trả kho
  When Staff và Warehouse Staff xác nhận bàn giao/nhận lại
  Then EPIC 09 chuyển đúng SKU/Batch và số lượng khỏi tồn Offline về phạm vi kho phù hợp
  And không ghi nhận đây là Customer Return hoặc doanh thu âm

Scenario: Khách trả sản phẩm đã mua
  Given Offline Customer trả sản phẩm của một Order đã bán
  When Staff tiếp nhận yêu cầu
  Then giao dịch được chuyển sang quy trình EPIC 14
  And hàng không tự nhập lại tồn bán được trước khi kiểm tra chất lượng

Scenario: Số lượng báo cáo không hợp lệ
  Given tổng lượng tồn/hỏng/trả khai báo vượt lượng Staff chịu trách nhiệm mà không có căn cứ
  When Staff gửi báo cáo
  Then hệ thống cảnh báo chênh lệch
  And không áp dụng điều chỉnh làm số dư sai hoặc âm
```

### US-OFF-04 — Gửi báo cáo bán hàng theo kỳ

**Actor:** Offline Sales Staff (`ACT-10`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Offline Sales Staff, tôi muốn gửi báo cáo bán hàng theo ngày/tháng.

**Giá trị nghiệp vụ:** Doanh nghiệp có bản tổng hợp theo kỳ để kiểm tra doanh số, tồn và tiền của từng nhân viên/điểm bán.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Lập báo cáo từ dữ liệu giao dịch
  Given Staff có giao dịch bán, hàng nhận, tồn, hỏng hoặc trả trong kỳ
  When Staff mở báo cáo kỳ
  Then hệ thống tổng hợp số đầu kỳ, nhận thêm, bán, tồn, hỏng, trả và doanh thu theo dữ liệu đã ghi nhận
  And phân tách tiền mặt, chuyển khoản cùng phương thức khác nếu có

Scenario: Nộp báo cáo đầy đủ
  Given các trường bắt buộc và kiểm đếm cuối kỳ đã hoàn tất
  When Staff xác nhận nộp báo cáo
  Then hệ thống lưu phiên bản báo cáo cùng người nộp và thời điểm
  And chuyển báo cáo sang trạng thái chờ đối soát hoặc trạng thái phù hợp

Scenario: Báo cáo còn thiếu dữ liệu
  Given báo cáo thiếu kiểm đếm, số tiền thực thu hoặc dữ liệu bắt buộc
  When Staff cố nộp
  Then hệ thống từ chối nộp hoàn chỉnh
  And chỉ rõ dữ liệu cần bổ sung

Scenario: Sửa báo cáo đã nộp
  Given báo cáo đã được nộp
  When Staff phát hiện sai và yêu cầu hiệu chỉnh
  Then hệ thống không ghi đè làm mất phiên bản đã nộp
  And áp dụng quy trình mở lại/điều chỉnh theo quyền với lý do và Audit

Scenario: Không có giao dịch trong kỳ
  Given Staff không phát sinh bán nhưng vẫn chịu trách nhiệm tồn tại điểm
  When đến kỳ báo cáo
  Then hệ thống cho phép nộp báo cáo không bán với kiểm đếm tồn và tiền phù hợp
  And không tự suy diễn không có giao dịch là không cần đối soát
```

### US-OFF-05 — Xem doanh thu và tồn theo nhân viên/điểm bán

**Actor:** Sales Manager (`ACT-12`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Sales Manager, tôi muốn xem doanh thu và tồn kho theo từng nhân viên/điểm bán.

**Giá trị nghiệp vụ:** Manager so sánh được hiệu quả và lượng hàng chịu trách nhiệm tại từng điểm để điều phối bán hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem tổng quan theo kỳ
  Given có dữ liệu Offline Sales trong khoảng thời gian được chọn
  When Sales Manager mở báo cáo
  Then hệ thống hiển thị doanh thu, số Order, số lượng bán và tồn theo nhân viên/điểm bán
  And nêu rõ kỳ, nguồn dữ liệu và trạng thái đối soát

Scenario: Phân biệt đã đối soát và chưa đối soát
  Given một số báo cáo kỳ chưa được đối soát
  When Manager xem doanh thu/tồn
  Then hệ thống phân biệt dữ liệu tạm ghi với dữ liệu đã xác nhận
  And không trình bày toàn bộ như số liệu cuối cùng

Scenario: Drill-down đến dữ liệu nguồn
  Given Manager xem một chỉ số tổng hợp
  When chọn xem chi tiết
  Then hệ thống liên kết được tới các Order, chứng từ cấp hàng và báo cáo kỳ tương ứng
  And tổng chi tiết nhất quán với số tổng hợp trong cùng phạm vi

Scenario: Không có dữ liệu
  Given kỳ hoặc điểm bán không có dữ liệu đủ điều kiện
  When Manager xem báo cáo
  Then hệ thống hiển thị trạng thái chưa có dữ liệu
  And không tạo doanh thu hoặc tỷ lệ gây hiểu nhầm

Scenario: Giới hạn quyền theo điểm bán
  Given Manager chỉ được quyền xem một nhóm điểm bán
  When Manager truy cập báo cáo
  Then hệ thống chỉ tổng hợp và hiển thị dữ liệu trong phạm vi đó
```

### US-OFF-06 — Đối soát hàng cấp, hàng bán và tồn

**Actor:** Sales Manager (`ACT-12`), Accountant / Finance Staff (`ACT-19`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Manager, tôi muốn đối soát lượng hàng đã cấp cho điểm bán với lượng hàng đã bán và còn tồn.

**Giá trị nghiệp vụ:** Doanh nghiệp phát hiện thất thoát hàng hóa và chênh lệch tiền theo từng kỳ, nhân viên và điểm bán.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đối soát hàng hóa khớp
  Given có số đầu kỳ, hàng nhận thêm, bán, tồn cuối kỳ, hỏng và trả kho của một phạm vi
  When Manager thực hiện đối soát
  Then hệ thống so sánh các thành phần theo công thức đã phê duyệt
  And đánh dấu khớp khi toàn bộ SKU/Batch cân bằng

Scenario: Đối soát tiền thực thu khớp
  Given có tổng giá trị Order Offline và số tiền thực thu theo từng phương thức
  When người có quyền đối soát cuối ca/kỳ
  Then hệ thống so sánh số phải thu với tiền mặt, chuyển khoản và điều chỉnh hợp lệ
  And đánh dấu khớp khi chênh lệch bằng mức cho phép

Scenario: Phát hiện chênh lệch hàng hoặc tiền
  Given số lượng hoặc tiền thực tế không khớp dữ liệu hệ thống
  When thực hiện đối soát
  Then hệ thống phân loại chênh lệch theo SKU/Batch hoặc phương thức thanh toán
  And yêu cầu lý do/xử lý theo quy trình thay vì tự cân bằng số liệu

Scenario: Xử lý sai lệch theo thẩm quyền
  Given một sai lệch đang mở
  And người dùng có quyền thực hiện hành động xử lý
  When người dùng ghi nhận kết quả cùng lý do và bằng chứng bắt buộc
  Then trạng thái đối soát được cập nhật
  And biến động Inventory/Payment chỉ được áp dụng qua Epic chịu trách nhiệm
  And toàn bộ thay đổi được ghi Audit

Scenario: Người không có quyền đóng đối soát
  Given người dùng chỉ có quyền xem
  When người dùng cố đóng hoặc điều chỉnh sai lệch
  Then hệ thống từ chối thao tác
  And dữ liệu hàng, tiền và báo cáo không bị thay đổi

Scenario: Mở lại kỳ đã đối soát
  Given kỳ đã được đối soát hoàn tất
  When phát hiện dữ liệu đến muộn hoặc sai sót cần điều chỉnh
  Then chỉ người có thẩm quyền được mở lại với lý do
  And phiên bản/kết quả trước vẫn được bảo toàn để truy vết
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-OFF-01` | Tạo chứng từ cấp hàng gắn với điểm bán, Staff, SKU/Batch, số lượng, người giao/nhận và trạng thái bàn giao. | `US-OFF-01`, `BR-OFFLINE-01~02` |
| `FR-OFF-02` | Chuyển tồn qua EPIC 09 khi hai phía xác nhận số lượng thực giao/nhận và không xử lý lặp cùng chứng từ. | `US-OFF-01`, `EPIC 09` |
| `FR-OFF-03` | Ghi nhận chênh lệch giao/nhận theo SKU/Batch mà không tự coi chứng từ đã khớp. | `US-OFF-01` |
| `FR-OFF-04` | Tạo đúng một Order nguồn Offline cho mỗi giao dịch bán hợp lệ và liên kết Staff/điểm bán. | `US-OFF-02`, `BR-OFFLINE-05`, `EPIC 08` |
| `FR-OFF-05` | Kiểm tra SKU, Batch/HSD, tồn Offline khả dụng, giá và thẩm quyền chiết khấu trước khi xác nhận bán. | `US-OFF-02`, `BR-BATCH-05`, `BR-OFFLINE-06` |
| `FR-OFF-06` | Ghi giảm tồn Offline qua EPIC 09 và ghi nghĩa vụ/kết quả thu qua EPIC 07 như một kết quả nhất quán của giao dịch. | `US-OFF-02`, `EPIC 07`, `EPIC 09` |
| `FR-OFF-07` | Hỗ trợ Offline Customer không có tài khoản và chỉ thu thập dữ liệu khách cần thiết theo chính sách. | `US-OFF-02` |
| `FR-OFF-08` | Chống tạo trùng Order, trừ tồn hoặc ghi nhận tiền khi cùng giao dịch được gửi lại. | `US-OFF-02` |
| `FR-OFF-09` | Ghi nhận kiểm đếm tồn thực tế theo Staff/điểm bán/SKU/Batch và tính chênh lệch với số dư hệ thống. | `US-OFF-03`, `BR-OFFLINE-03~04` |
| `FR-OFF-10` | Ghi nhận hàng hỏng/thất thoát bằng yêu cầu biến động có lý do, bằng chứng và phê duyệt khi bắt buộc. | `US-OFF-03`, `EPIC 09` |
| `FR-OFF-11` | Chuyển hàng chưa bán trả kho bằng chứng từ hai phía, không nhầm với Customer Return. | `US-OFF-03`, `EPIC 09`, `EPIC 14` |
| `FR-OFF-12` | Chuyển yêu cầu khách trả hàng đã bán sang EPIC 14 và không tự nhập lại tồn bán được. | `US-OFF-03`, `EPIC 14` |
| `FR-OFF-13` | Tổng hợp báo cáo kỳ gồm đầu kỳ, nhận, bán, tồn, hỏng, trả và doanh thu theo phương thức thanh toán. | `US-OFF-04`, `BR-OFFLINE-03` |
| `FR-OFF-14` | Kiểm tra dữ liệu bắt buộc trước khi nộp, lưu phiên bản/người nộp/thời điểm và kiểm soát việc mở lại báo cáo. | `US-OFF-04`, `BR-AUDIT-01` |
| `FR-OFF-15` | Cho Sales Manager xem doanh thu, Order, số lượng bán và tồn theo kỳ/Staff/điểm bán với trạng thái đối soát. | `US-OFF-05` |
| `FR-OFF-16` | Cho phép drill-down từ chỉ số tổng hợp tới Order, chứng từ cấp hàng và báo cáo nguồn trong phạm vi quyền. | `US-OFF-05`, `BR-AUTH-04` |
| `FR-OFF-17` | Đối soát phương trình hàng hóa theo SKU/Batch giữa đầu kỳ, nhận thêm, bán, hỏng/thất thoát, trả kho và tồn cuối kỳ. | `US-OFF-06`, `BR-OFFLINE-04` |
| `FR-OFF-18` | Đối soát số phải thu với tiền thực thu theo tiền mặt, chuyển khoản và các điều chỉnh hợp lệ. | `US-OFF-06`, `BR-OFFLINE-07`, `EPIC 07` |
| `FR-OFF-19` | Phân loại chênh lệch hàng/tiền và xử lý theo thẩm quyền mà không tự ghi đè số liệu nguồn. | `US-OFF-06`, `BR-AUTH-04` |
| `FR-OFF-20` | Chỉ áp dụng điều chỉnh Inventory/Payment qua Epic sở hữu và bảo toàn liên kết với kết quả đối soát. | `US-OFF-06`, `EPIC 07`, `EPIC 09` |
| `FR-OFF-21` | Cung cấp dữ liệu Offline đã đối soát cho EPIC 24 nhưng không thay thế báo cáo tài chính tổng hợp. | `US-OFF-05`, `US-OFF-06`, `EPIC 24` |
| `FR-OFF-22` | Ghi Audit cho vượt giá/chiết khấu, hủy/điều chỉnh giao dịch, mở lại báo cáo và xử lý sai lệch. | `US-OFF-02~06`, `BR-AUDIT-01` |

## 6. Mô hình đối soát và quy tắc dữ liệu

### 6.1. Phương trình hàng hóa cơ sở

```text
Tồn đầu kỳ
+ Hàng nhận thêm
= Hàng đã bán
 + Hàng hỏng/thất thoát đã xác nhận
 + Hàng trả về kho
 + Tồn cuối kỳ
 + Chênh lệch chưa xử lý
```

### 6.2. Phương trình tiền cơ sở

```text
Giá trị các Order Offline hợp lệ
- Giảm giá/hoàn/điều chỉnh được phê duyệt
= Tiền mặt thực thu
 + Chuyển khoản đã xác nhận
 + Phương thức thu khác được hỗ trợ
 + Chênh lệch chưa xử lý
```

Công thức chính thức phải quy định dấu của chênh lệch, thời điểm ghi nhận doanh thu và cách xử lý giao dịch hủy/trả. Không được tự dùng “chênh lệch” để làm cân phương trình mà không mở trường hợp cần xử lý.

### 6.3. Quy tắc dữ liệu và an toàn nghiệp vụ

- Tồn Offline là một phạm vi/vị trí chịu trách nhiệm trong Inventory, không phải số lượng tách rời không thể đối chiếu với kho trung tâm.
- Mọi giao dịch phải xác định Staff, điểm bán, ca/kỳ, SKU, số lượng, giá, phương thức thu và Order nội bộ duy nhất.
- Giá/chiết khấu phải được kiểm tra ở thời điểm giao dịch; client hoặc Staff không được tự quyết giá ngoài thẩm quyền.
- Hàng hết hạn/hỏng/cách ly không thuộc tồn có thể bán dù vẫn đang hiện diện vật lý tại điểm.
- Khách mua trực tiếp không bắt buộc có tài khoản. Chỉ liên kết giao dịch với Customer khi có dữ liệu và căn cứ hợp lệ; không tự tạo/hợp nhất hồ sơ từ thông tin mơ hồ.
- Báo cáo đã nộp và kỳ đã đối soát không được ghi đè. Hiệu chỉnh tạo phiên bản hoặc giao dịch điều chỉnh có người/lý do.
- Tiền mặt, chuyển khoản và phương thức khác phải được đối soát riêng; không dùng tổng tiền khai báo để che sai lệch từng phương thức.
- Hàng tồn chưa bán trả kho khác với Customer Return. Customer Return phải qua EPIC 14 và kiểm tra chất lượng trước khi tái nhập tồn bán được.
- Cùng một thao tác gửi lặp không được tạo nhiều Order, trừ/chuyển tồn, ghi nhận tiền hay đóng báo cáo nhiều lần.
- “Offline Sales” chỉ tên kênh bán trực tiếp; không mặc nhiên yêu cầu hệ thống hoạt động khi mất mạng. Offline-first cần User Story và quy tắc xung đột riêng nếu được phê duyệt.
- Chi tiết endpoint, cơ sở dữ liệu cục bộ, đồng bộ khi có mạng, thiết bị POS, máy in hóa đơn, máy quét và giao thức thanh toán thuộc Architecture/Technical Design.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-OFF-01` | `BR-OFFLINE-01`, `BR-OFFLINE-02`, `BR-BATCH-01`, `BR-BATCH-05` | EPIC 09, 22 | Nhận đúng người/SKU/Batch; chênh lệch được ghi; yêu cầu lặp không chuyển tồn hai lần. |
| `US-OFF-02` | `BR-OFFLINE-01`, `BR-OFFLINE-05~06`, `BR-ORDER-01` | EPIC 04, 07, 08, 09 | Mỗi lần bán tạo một Order, đúng giá/tồn/tiền; Guest mua được; xử lý lặp không nhân đôi. |
| `US-OFF-03` | `BR-OFFLINE-03~04`, `BR-BATCH-05` | EPIC 09, 14 | Tồn/hỏng/trả được phân loại đúng; hàng trả kho khác Customer Return; không điều chỉnh âm hoặc mất dấu vết. |
| `US-OFF-04` | `BR-OFFLINE-03`, `BR-AUDIT-01` | EPIC 07, 09, 23 | Báo cáo đủ các thành phần, lưu phiên bản và chỉ mở lại có quyền/lý do. |
| `US-OFF-05` | `BR-AUTH-04`, `BR-OFFLINE-04`, `BR-OFFLINE-07` | EPIC 08, 09, 24 | Chỉ số đúng Staff/điểm/kỳ, phân biệt đã đối soát và drill-down được tới nguồn trong phạm vi quyền. |
| `US-OFF-06` | `BR-OFFLINE-02~07`, `BR-AUDIT-01` | EPIC 07, 09, 23, 24 | Hàng và tiền được đối soát riêng; sai lệch không tự cân; xử lý đúng quyền và bảo toàn lịch sử. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Mô hình điểm bán: cửa hàng cố định, quầy/chợ lưu động, đại lý ký gửi hay tất cả; phạm vi trách nhiệm tồn theo Staff hay theo location.
- Định nghĩa ca bán/kỳ báo cáo, thời điểm mở/đóng, múi giờ, giao ca và xử lý giao dịch đến muộn.
- Có yêu cầu hoạt động khi mất mạng hay không. Nếu có, phải bổ sung User Story cho offline-first, định danh tạm, đồng bộ lại và giải quyết xung đột giá/tồn/Order.
- Mốc chuyển tồn từ kho trung tâm sang tồn Offline và quy trình bàn giao một phía/hai phía.
- Có quản lý Batch tại điểm bán đến từng giao dịch hay chỉ khi cấp/trả; cách áp dụng FEFO trong bán trực tiếp.
- Chính sách giá Offline, mức chiết khấu theo role/Staff/điểm bán và quy trình phê duyệt vượt thẩm quyền.
- Phương thức thanh toán hỗ trợ, cách xác nhận chuyển khoản tại điểm và trách nhiệm nộp/đối soát tiền mặt.
- Trạng thái Order Offline và thời điểm coi giao dịch `PAID`, `DELIVERED`, `COMPLETED`; không áp dụng máy móc state machine D2C.
- Giao dịch hủy/sửa trước và sau đóng ca; dùng Order đảo/điều chỉnh hay chỉnh trực tiếp.
- Công thức đối soát chính thức, quy tắc làm tròn, dung sai tiền và dấu của chênh lệch thừa/thiếu.
- Ngưỡng chênh lệch hàng/tiền cần phê duyệt, actor chịu trách nhiệm và bằng chứng bắt buộc.
- Quy trình hàng chưa bán trả kho, hàng hỏng, thất thoát và Customer Return; điều kiện nhập lại tồn bán được.
- Dữ liệu tối thiểu của Offline Customer, yêu cầu hóa đơn và quy tắc liên kết với Customer Account/Loyalty.
- Thiết bị POS, in phiếu/hóa đơn, quét SKU/Batch và quản lý tiền mặt có nằm trong phạm vi dự án hay không.
- Thời hạn lưu báo cáo ca, chứng từ tiền mặt, biên bản chênh lệch và dữ liệu Audit.

## 9. UI/UX Reference

- Màn hình nhận hàng hiển thị chứng từ, SKU/Batch, dự kiến, thực nhận và chênh lệch; xác nhận rõ người/điểm nhận.
- Giao diện bán ưu tiên thao tác nhanh nhưng luôn hiển thị SKU, số lượng, giá, chiết khấu, tồn khả dụng và phương thức thu trước khi xác nhận.
- Cảnh báo giá vượt quyền, tồn thiếu hoặc Batch hết hạn phải chặn giao dịch và nêu hành động xử lý.
- Kiểm đếm cuối kỳ phân tách tồn bán được, hỏng/cách ly và trả kho; Customer Return có lối vào riêng.
- Báo cáo kỳ hiển thị phương trình hàng và tiền, đánh dấu từng chênh lệch chưa xử lý trước khi nộp.
- Dashboard Manager phân biệt dữ liệu tạm ghi với đã đối soát và cho drill-down tới Order/chứng từ nguồn.
- Màn hình đối soát hiển thị hàng hóa và tiền theo hai phần riêng; hành động điều chỉnh phải nêu phạm vi tác động và yêu cầu lý do.
- Chỉ tiêu hiệu năng, offline sync, endpoint, thiết bị POS và công nghệ in/quét được đặc tả trong NFR hoặc Technical Design.

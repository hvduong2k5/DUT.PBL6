# EPIC 10 — Packing & Fulfillment

## 1. Mục tiêu Epic

Epic này hỗ trợ Packing Staff nhận đúng Order cần xử lý, lấy đúng SKU/Batch và số lượng, thực hiện các bước kiểm tra, lưu bằng chứng khi được yêu cầu và xác nhận kiện hàng sẵn sàng chuyển sang giao vận. Danh sách công việc phải phản ánh điều kiện Order và thứ tự ưu tiên theo SLA giao hàng.

Packing quản lý Packing Task, người được phân công, checklist, kết quả đóng gói và bằng chứng media. Việc hoàn tất Packing chỉ được ghi nhận khi đúng người có quyền thực hiện, các bước bắt buộc đã đạt và bằng chứng bắt buộc đã được liên kết thành công.

Epic không quyết định Payment, quản lý state machine tổng thể của Order, tự chọn Batch trái chính sách Inventory, ghi biến động xuất kho, tạo Shipment, gửi thông báo hoặc xử lý khiếu nại/hoàn tiền. Các trách nhiệm này lần lượt thuộc EPIC 07, EPIC 08, EPIC 09, EPIC 11, EPIC 28, EPIC 16 và EPIC 14.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 2nd** | `US-PACK-01` đến `US-PACK-03`, `US-PACK-07`, `US-PACK-08` | Packing Staff xem được hàng đợi, lấy/kiểm tra đúng hàng, ưu tiên đúng SLA và bàn giao Order đã đóng gói sang giao vận. |
| **Giai đoạn 2** | `US-PACK-04` đến `US-PACK-06` | Video đóng gói được lưu, liên kết và tra cứu có kiểm soát để làm bằng chứng. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actor

| Actor | Vai trò trong Epic |
| --- | --- |
| Packing Staff (`ACT-07`) | Xem hàng đợi, nhận/phân công Packing Task, kiểm tra hàng, lưu bằng chứng và xác nhận hoàn tất. |
| Customer Service (`ACT-11`) | Tra cứu bằng chứng đóng gói khi hỗ trợ khách hoặc xử lý khiếu nại nếu được cấp quyền. |
| Sales Manager (`ACT-12`) | Giám sát tiến độ, SLA và tra cứu bằng chứng theo quyền. |
| System Administrator (`ACT-17`) | Quản trị truy cập hệ thống; chỉ xem Packing Video khi được cấp quyền nghiệp vụ phù hợp. |
| Media / Object Storage (`EXT-09`) | Lưu trữ Packing Video thông qua hạ tầng media được thiết kế. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-04` | Nhân viên chỉ được thực hiện thao tác thuộc quyền được cấp. | Xem Task, hoàn tất Packing và truy cập Video đều phải kiểm tra quyền. |
| `BR-PROD-02` | Khối lượng và quy cách đóng gói được quản lý theo SKU. | Packing Task phải hiển thị đúng SKU/quy cách từ snapshot Order. |
| `BR-BATCH-01` | Tồn kho thực phẩm phải truy xuất được theo Batch/Lot. | Hàng thực lấy phải liên kết được với Batch do Inventory phân bổ hoặc xác nhận. |
| `BR-BATCH-05` | Hàng hết hạn không được phép tiếp tục bán. | Packing không được xác nhận sử dụng Batch hết hạn cho Order. |
| `BR-BATCH-06` | Nên ưu tiên Batch có HSD gần hơn khi phù hợp chính sách. | Danh sách lấy hàng sử dụng phân bổ FEFO có điều kiện từ EPIC 09. |
| `BR-ORDER-01` | Order phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Chỉ Order đủ điều kiện mới vào hàng đợi; kết quả hoàn tất được gửi sang EPIC 08 để chuyển `PACKED`. |
| `BR-ORDER-05` | Thay đổi trạng thái quan trọng của Order phải được ghi nhận. | Kết quả Packing và actor thực hiện phải có lịch sử liên kết Order. |
| `BR-PACK-01` | Chỉ Packing Staff được phân công mới được xác nhận hoàn tất. | Hệ thống phải xác minh phân công tại thời điểm hoàn tất. |
| `BR-PACK-02` | Nhân viên phải kiểm tra SKU và số lượng trước khi đóng gói. | Checklist SKU/số lượng là điều kiện bắt buộc để hoàn tất. |
| `BR-PACK-03` | Order có thể yêu cầu bằng chứng ảnh/video đóng gói. | Packing Task phải thể hiện rõ khi media là bắt buộc. |
| `BR-PACK-04` | Packing Video phải liên kết với Order/Shipment tương ứng. | Media không có liên kết hợp lệ không được coi là bằng chứng hoàn chỉnh. |
| `BR-PACK-05` | Người không có quyền không được tùy ý xem hoặc tải Packing Video. | Mọi đường truy cập media phải kiểm tra actor, quyền và phạm vi Order. |
| `BR-AUDIT-01` | Thao tác quan trọng phải được ghi Audit Log. | Hoàn tất, mở lại, thay bằng chứng và truy cập nhạy cảm cần có khả năng truy vết. |

### 3.3. Phụ thuộc giữa Epic

- `EPIC 04 — Product, Variant & SKU`: cung cấp dữ liệu catalog; Packing dùng snapshot SKU/quy cách của Order để tránh thay đổi hồi tố.
- `EPIC 08 — Order Management`: cung cấp Order đủ điều kiện, SLA và nhận kết quả Packing để áp dụng trạng thái `PACKED`.
- `EPIC 09 — Inventory & Batch`: phân bổ/xác nhận Batch, kiểm soát HSD và ghi nhận xuất kho; Packing không tự sửa tồn.
- `EPIC 11 — Shipping & Delivery`: tiếp nhận Order/kiện hàng đã đóng gói để tạo Shipment và bàn giao vận chuyển.
- `EPIC 12 — Marketplace` và `EPIC 18 — B2B`: cung cấp yêu cầu đóng gói riêng theo nguồn Order ở giai đoạn tương ứng.
- `EPIC 14 — Return / Refund / Complaint`: sử dụng Packing Video làm bằng chứng trong quy trình đổi trả/khiếu nại.
- `EPIC 16 — Customer Service`: cho CSKH tra cứu Order và yêu cầu sử dụng bằng chứng khi hỗ trợ khách.
- `EPIC 22 — User / Role / Permission Administration`: quyết định quyền thao tác và quyền truy cập Packing Video.
- `EPIC 23 — Audit & Security`: lưu dấu vết thao tác nhạy cảm và thay đổi Packing Task/media.
- `EXT-09 — Media / Object Storage`: lưu dữ liệu media; lựa chọn công nghệ thuộc Technical Design.

## 4. User Stories chi tiết

### US-PACK-01 — Xem danh sách đơn cần đóng gói

**Actor:** Packing Staff (`ACT-07`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Packing Staff, tôi muốn xem danh sách đơn cần đóng gói để biết đơn nào phải xử lý.

**Giá trị nghiệp vụ:** Nhân viên có một hàng đợi thống nhất, chỉ gồm các Order đủ điều kiện cho công đoạn đóng gói.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem hàng đợi đóng gói
  Given Packing Staff có quyền xem Packing Task
  When nhân viên mở danh sách cần đóng gói
  Then hệ thống hiển thị các Order đủ điều kiện cho công đoạn Packing
  And mỗi dòng có mã Order, thời điểm/hạn xử lý, số dòng hàng, nguồn và trạng thái phân công cần thiết

Scenario: Không đưa Order chưa đủ điều kiện vào hàng đợi sẵn sàng
  Given Order chưa đáp ứng điều kiện Payment, reservation hoặc trạng thái trước đó
  When Packing Staff xem hàng đợi
  Then Order không được trình bày như công việc có thể bắt đầu
  And người có quyền có thể nhận biết điều kiện đang chặn khi cần

Scenario: Lọc Packing Task theo trạng thái
  Given có Task chưa phân công, đang xử lý, bị chặn và đã hoàn tất
  When Packing Staff chọn bộ lọc được hỗ trợ
  Then hệ thống hiển thị đúng các Task phù hợp
  And không làm thay đổi trạng thái Task chỉ vì xem hoặc lọc

Scenario: Nhân viên không có quyền
  Given nhân viên không có quyền xem hàng đợi Packing
  When nhân viên truy cập danh sách
  Then hệ thống từ chối truy cập
  And không tiết lộ thông tin Order trong hàng đợi
```

### US-PACK-02 — Xem chi tiết sản phẩm cần đóng gói

**Actor:** Packing Staff (`ACT-07`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Packing Staff, tôi muốn xem chi tiết sản phẩm và số lượng cần đóng gói để tránh lấy sai hàng.

**Giá trị nghiệp vụ:** Nhân viên lấy đúng SKU, quy cách, Batch và số lượng đã đặt, giảm giao thiếu hoặc giao nhầm.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem danh sách lấy hàng của Order
  Given Packing Task gắn với một Order hợp lệ
  When Packing Staff mở chi tiết Task
  Then hệ thống hiển thị đúng SKU, tên/quy cách snapshot, số lượng và thông tin nhận diện cần thiết
  And hiển thị Batch/vị trí được EPIC 09 phân bổ nếu có

Scenario: Dữ liệu Product thay đổi sau khi đặt hàng
  Given tên, ảnh hoặc quy cách catalog đã thay đổi sau khi Order được tạo
  When Packing Staff xem Task
  Then thông tin phải đóng gói vẫn dựa trên snapshot Order
  And không tự thay SKU theo dữ liệu catalog mới

Scenario: Batch phân bổ không còn hợp lệ
  Given Batch được đề xuất đã hết hạn, bị cách ly hoặc không còn đủ số lượng
  When Packing Staff mở hoặc xác nhận lấy hàng
  Then hệ thống không cho sử dụng Batch đó
  And yêu cầu EPIC 09 phân bổ/xác nhận Batch hợp lệ khác

Scenario: Order có nhiều SKU hoặc nhiều Batch
  Given một Order có nhiều dòng hoặc một dòng được phân bổ từ nhiều Batch
  When Packing Staff xem chi tiết
  Then hệ thống phân biệt rõ từng SKU, số lượng và Batch tương ứng
  And tổng lượng cần lấy của từng SKU khớp snapshot Order
```

### US-PACK-03 — Xác nhận các bước kiểm tra đóng gói

**Actor:** Packing Staff (`ACT-07`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Packing Staff, tôi muốn xác nhận từng bước kiểm tra đơn trước khi hoàn tất đóng gói.

**Giá trị nghiệp vụ:** Checklist giúp giảm lỗi lấy sai SKU, sai số lượng hoặc bỏ sót yêu cầu đóng gói trước khi niêm phong kiện hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hoàn thành checklist bắt buộc
  Given Packing Staff được phân công một Task đang xử lý
  When nhân viên xác nhận đã kiểm tra từng SKU, số lượng và các bước bắt buộc khác
  Then hệ thống lưu kết quả từng bước cùng người và thời điểm thực hiện
  And Task đủ điều kiện kiểm tra để chuyển sang hoàn tất nếu các điều kiện khác cũng đạt

Scenario: Checklist còn thiếu
  Given Packing Task còn ít nhất một bước bắt buộc chưa đạt
  When Packing Staff cố xác nhận hoàn tất
  Then hệ thống từ chối hoàn tất
  And chỉ rõ bước còn thiếu hoặc không đạt

Scenario: Phát hiện sai SKU hoặc số lượng
  Given hàng thực lấy không khớp danh sách của Order
  When Packing Staff ghi nhận kết quả kiểm tra không đạt
  Then Task được giữ ở trạng thái chưa hoàn tất hoặc bị chặn phù hợp
  And hệ thống không chuyển Order sang `PACKED`
  And sai lệch được lưu để xử lý

Scenario: Người không được phân công sửa checklist
  Given Packing Task đã được phân công cho nhân viên A
  When nhân viên B không có quyền thay thế cố xác nhận bước kiểm tra
  Then hệ thống từ chối thao tác
  And kết quả checklist không bị thay đổi
```

### US-PACK-04 — Lưu video quá trình đóng gói

**Actor:** Packing Staff (`ACT-07`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Packing Staff, tôi muốn lưu video quá trình đóng gói để làm bằng chứng khi cần kiểm tra.

**Giá trị nghiệp vụ:** Doanh nghiệp có bằng chứng về SKU, số lượng và tình trạng đóng gói để hỗ trợ điều tra khi phát sinh khiếu nại.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Lưu Packing Video hợp lệ
  Given Packing Staff đang xử lý một Task cho phép hoặc yêu cầu video
  When nhân viên hoàn tất việc ghi/tải video đáp ứng chính sách media
  Then hệ thống lưu bản ghi media với trạng thái thành công
  And lưu người tạo, thời điểm và Packing Task liên quan

Scenario: Video là bằng chứng bắt buộc
  Given Packing Task được đánh dấu bắt buộc có video
  When video chưa được lưu thành công hoặc không đáp ứng điều kiện
  Then hệ thống không cho hoàn tất Packing
  And hiển thị yêu cầu bổ sung hoặc thử lại

Scenario: Lưu video thất bại
  Given Packing Staff đang gửi một Packing Video
  When việc lưu media không hoàn tất
  Then hệ thống không trình bày video như bằng chứng đã sẵn sàng
  And cho phép thử lại mà không tạo nhiều bản media hoàn chỉnh trùng nhau

Scenario: Media không hợp lệ
  Given tệp/video không đáp ứng loại, kích thước, thời lượng hoặc chính sách an toàn được cấu hình
  When Packing Staff gửi media
  Then hệ thống từ chối sử dụng làm bằng chứng
  And thông báo điều kiện cần điều chỉnh
```

### US-PACK-05 — Liên kết video với Order

**Actor:** Packing Staff (`ACT-07`)

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Packing Staff, tôi muốn liên kết video đóng gói với Order để việc tra cứu sau này dễ dàng.

**Giá trị nghiệp vụ:** Bằng chứng luôn truy được về đúng giao dịch và kiện hàng, tránh nhầm video giữa các Order.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Liên kết video với đúng Order
  Given Packing Video đã được lưu thành công cho một Packing Task
  When hệ thống hoàn tất liên kết bằng chứng
  Then video gắn với đúng Order của Task
  And gắn với Shipment/kiện hàng tương ứng khi định danh đó đã tồn tại

Scenario: Ngăn liên kết chéo Order
  Given Packing Staff đang xử lý Order A
  When nhân viên cố liên kết video của Task với Order B không tương ứng
  Then hệ thống từ chối liên kết
  And bằng chứng của cả hai Order không bị thay đổi

Scenario: Một Order có nhiều kiện hàng hoặc video
  Given chính sách cho phép Order có nhiều kiện hoặc nhiều đoạn bằng chứng
  When các video được liên kết
  Then mỗi video xác định được Packing Task và kiện hàng liên quan
  And hệ thống không làm mất thứ tự hoặc quan hệ của bằng chứng

Scenario: Thay thế bằng chứng sau khi hoàn tất
  Given Packing Task đã hoàn tất và có video liên kết
  When người có quyền yêu cầu thay thế hoặc bổ sung bằng chứng
  Then hệ thống không ghi đè làm mất bản gốc
  And lưu phiên bản/liên kết mới cùng lý do và Audit
```

### US-PACK-06 — Tra cứu Packing Video

**Actor:** Customer Service (`ACT-11`), Sales Manager (`ACT-12`), System Administrator (`ACT-17`) được cấp quyền

**Ưu tiên:** Should Have

**Phát hành:** Giai đoạn 2

> Là Admin/Manager/CSKH được cấp quyền, tôi muốn tra cứu video đóng gói của một đơn hàng để kiểm tra khi xảy ra sai sót hoặc khiếu nại.

**Giá trị nghiệp vụ:** Nhân viên có thẩm quyền có thể xác minh quá trình đóng gói mà không mở quyền truy cập media cho toàn bộ nhân sự.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: CSKH có quyền xem video của Order khiếu nại
  Given CSKH có quyền và đang xử lý ticket liên quan đến một Order
  When CSKH tra cứu bằng chứng đóng gói
  Then hệ thống hiển thị danh sách video liên kết với đúng Order/kiện hàng
  And cho phép xem theo phạm vi quyền được cấp

Scenario: Người không có quyền truy cập
  Given nhân viên không có quyền xem Packing Video của Order
  When nhân viên mở liên kết trực tiếp hoặc yêu cầu tải/xem video
  Then hệ thống từ chối truy cập
  And không tiết lộ nội dung hoặc vị trí lưu trữ media

Scenario: Video chưa sẵn sàng hoặc không tồn tại
  Given Order không có video hoặc media đang lỗi/chưa hoàn tất
  When người có quyền tra cứu
  Then hệ thống hiển thị đúng trạng thái bằng chứng
  And không hiển thị liên kết hỏng như video có thể sử dụng

Scenario: Ghi nhận truy cập nhạy cảm
  Given người có quyền xem hoặc tải Packing Video
  When thao tác truy cập được thực hiện
  Then hệ thống ghi nhận actor, Order/media, thời điểm và loại thao tác theo chính sách Audit

Scenario: CSKH không tự thay đổi bằng chứng
  Given CSKH chỉ có quyền tra cứu Packing Video
  When CSKH cố xóa, thay thế hoặc liên kết lại media
  Then hệ thống từ chối thao tác
  And bằng chứng gốc được bảo toàn
```

### US-PACK-07 — Ưu tiên đơn có thời hạn giao gần nhất

**Actor:** Packing Staff (`ACT-07`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Packing Staff, tôi muốn biết đơn nào có thời hạn giao gần nhất để ưu tiên đóng gói.

**Giá trị nghiệp vụ:** Nhân viên xử lý trước các Order có nguy cơ trễ SLA giao hàng, giảm chậm bàn giao cho đơn vị vận chuyển.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Sắp xếp hàng đợi theo hạn xử lý
  Given có nhiều Packing Task đủ điều kiện
  When Packing Staff xem hàng đợi ưu tiên
  Then hệ thống sắp xếp hoặc đánh dấu theo thời hạn giao/SLA đang áp dụng
  And hiển thị hạn cùng lý do ưu tiên của từng Task

Scenario: Cảnh báo Task sắp hoặc đã quá hạn
  Given một Packing Task sắp đến hoặc đã vượt hạn đóng gói
  When hệ thống đánh giá SLA
  Then Task được đánh dấu mức cảnh báo tương ứng
  And Packing Staff và Sales Manager có quyền nhận biết tình trạng chậm

Scenario: Task bị chặn không được trình bày như sẵn sàng
  Given Task có hạn gần nhưng đang bị chặn bởi dữ liệu hoặc điều kiện nghiệp vụ
  When Packing Staff xem danh sách ưu tiên
  Then Task được thể hiện là bị chặn thay vì có thể xử lý ngay
  And lý do chặn được hiển thị cho actor có quyền

Scenario: Hạn thay đổi sau cập nhật Shipping hoặc Order
  Given thời hạn giao của Order được cập nhật hợp lệ
  When Packing nhận dữ liệu SLA mới
  Then vị trí/mức ưu tiên của Task được tính lại
  And không làm mất lịch sử hạn trước nếu thay đổi cần Audit
```

### US-PACK-08 — Xác nhận hoàn tất đóng gói

**Actor:** Packing Staff (`ACT-07`)

**Ưu tiên:** Must Have — 2nd

**Phát hành:** MVP

> Là Packing Staff, tôi muốn xác nhận đơn đã đóng gói xong để chuyển sang công đoạn giao hàng.

**Giá trị nghiệp vụ:** Chỉ kiện hàng đã được kiểm tra đầy đủ mới được bàn giao cho Shipping, giảm sai sót và duy trì trạng thái Order chính xác.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hoàn tất Packing thành công
  Given Packing Staff được phân công cho Task
  And checklist bắt buộc đã đạt
  And SKU, số lượng và Batch thực tế đều hợp lệ
  And bằng chứng bắt buộc đã liên kết thành công nếu Order yêu cầu
  When Packing Staff xác nhận hoàn tất
  Then Packing Task được ghi nhận hoàn tất đúng một lần
  And kết quả được gửi sang EPIC 08 để chuyển Order sang `PACKED`
  And dữ liệu kiện hàng cần thiết sẵn sàng cho EPIC 11

Scenario: Người không được phân công xác nhận hoàn tất
  Given Task được phân công cho Packing Staff A
  When Packing Staff B không có quyền thay thế cố hoàn tất Task
  Then hệ thống từ chối thao tác
  And Task cùng Order không bị thay đổi

Scenario: Điều kiện hoàn tất chưa đủ
  Given checklist, phân bổ Batch, thông tin kiện hàng hoặc bằng chứng bắt buộc còn thiếu
  When Packing Staff xác nhận hoàn tất
  Then hệ thống từ chối hoàn tất
  And chỉ rõ điều kiện chưa đáp ứng
  And không chuyển Order sang `PACKED`

Scenario: Gửi lặp yêu cầu hoàn tất
  Given Packing Task đã hoàn tất và Order đã nhận kết quả
  When cùng yêu cầu hoàn tất được gửi lại
  Then hệ thống trả về kết quả nhất quán
  And không ghi xuất kho, chuyển trạng thái hoặc tạo bàn giao Shipping nhiều lần

Scenario: Mở lại Packing Task
  Given Task đã hoàn tất nhưng phát hiện sai sót trước khi bàn giao
  When người có thẩm quyền yêu cầu mở lại và cung cấp lý do
  Then hệ thống chỉ mở lại nếu trạng thái Order/Shipment còn cho phép
  And lưu trạng thái trước/sau, actor và lý do để Audit
```

## 5. Functional Requirements

| Mã FR | Yêu cầu chức năng | User Story nguồn |
| --- | --- | --- |
| `FR-PACK-01` | Tạo/cập nhật Packing Task cho Order đủ điều kiện và không tạo Task trùng cho cùng phạm vi đóng gói. | `US-PACK-01`, `EPIC 08` |
| `FR-PACK-02` | Cho Packing Staff có quyền xem, tìm và lọc hàng đợi theo trạng thái, phân công, nguồn và SLA. | `US-PACK-01`, `US-PACK-07` |
| `FR-PACK-03` | Không trình bày Order chưa đáp ứng Payment, reservation hoặc trạng thái trước đó như Task sẵn sàng xử lý. | `US-PACK-01`, `EPIC 07`, `EPIC 08`, `EPIC 09` |
| `FR-PACK-04` | Hiển thị snapshot SKU, quy cách, số lượng cùng Batch/vị trí phân bổ cần thiết cho việc lấy hàng. | `US-PACK-02`, `EPIC 08`, `EPIC 09` |
| `FR-PACK-05` | Kiểm tra Batch thực tế còn đủ điều kiện, không hết hạn/cách ly và khớp SKU trước khi xác nhận lấy hàng. | `US-PACK-02`, `BR-BATCH-05` |
| `FR-PACK-06` | Quản lý checklist bắt buộc theo Packing Task và lưu kết quả từng bước cùng actor, thời điểm. | `US-PACK-03`, `BR-PACK-02` |
| `FR-PACK-07` | Không cho hoàn tất Task khi checklist không đạt hoặc phát hiện sai SKU/số lượng chưa được xử lý. | `US-PACK-03`, `US-PACK-08` |
| `FR-PACK-08` | Cho Packing Staff lưu media đáp ứng chính sách và phản ánh rõ trạng thái đang xử lý, thành công hoặc thất bại. | `US-PACK-04` |
| `FR-PACK-09` | Bắt buộc có bằng chứng hợp lệ trước khi hoàn tất nếu Packing Task được cấu hình yêu cầu media. | `US-PACK-04`, `BR-PACK-03` |
| `FR-PACK-10` | Liên kết Packing Video với đúng Task, Order và Shipment/kiện hàng khi có; ngăn liên kết chéo. | `US-PACK-05`, `BR-PACK-04` |
| `FR-PACK-11` | Bảo toàn media gốc; bổ sung/thay thế sau hoàn tất phải theo quyền, có phiên bản hoặc lịch sử và Audit. | `US-PACK-05`, `BR-AUDIT-01` |
| `FR-PACK-12` | Cho actor được cấp quyền tra cứu/xem/tải Packing Video trong đúng phạm vi Order và mục đích nghiệp vụ. | `US-PACK-06`, `BR-PACK-05` |
| `FR-PACK-13` | Từ chối mọi truy cập trực tiếp hoặc gián tiếp vào Packing Video khi actor không có quyền. | `US-PACK-06`, `BR-AUTH-04`, `BR-PACK-05` |
| `FR-PACK-14` | Ghi nhận hoạt động truy cập Packing Video nhạy cảm theo chính sách Audit. | `US-PACK-06`, `BR-AUDIT-01` |
| `FR-PACK-15` | Tính/nhận hạn xử lý, sắp xếp hoặc đánh dấu Packing Task theo SLA và cảnh báo Task sắp/đã quá hạn. | `US-PACK-07`, `EPIC 08` |
| `FR-PACK-16` | Tính lại ưu tiên khi hạn giao hoặc điều kiện Order thay đổi mà không làm mất lịch sử cần truy vết. | `US-PACK-07`, `BR-ORDER-05` |
| `FR-PACK-17` | Chỉ cho Packing Staff được phân công xác nhận hoàn tất khi checklist, SKU/số lượng, Batch, kiện hàng và bằng chứng bắt buộc đều hợp lệ. | `US-PACK-08`, `BR-PACK-01~04` |
| `FR-PACK-18` | Gửi kết quả hoàn tất đúng một lần cho EPIC 08 và dữ liệu bàn giao cần thiết cho EPIC 11. | `US-PACK-08`, `EPIC 08`, `EPIC 11` |
| `FR-PACK-19` | Phối hợp ghi nhận xuất kho qua EPIC 09 tại mốc nghiệp vụ đã chốt, không tự sửa số dư Inventory. | `US-PACK-02`, `US-PACK-08`, `EPIC 09` |
| `FR-PACK-20` | Chỉ mở lại Task đã hoàn tất khi actor có quyền và Order/Shipment còn cho phép; bắt buộc lưu lý do và Audit. | `US-PACK-08`, `EPIC 08`, `EPIC 11`, `EPIC 23` |

## 6. Vòng đời Packing Task và quy tắc dữ liệu

### 6.1. Vòng đời nghiệp vụ đề xuất

```text
READY
  ├── ASSIGNED
  │      ├── IN_PROGRESS
  │      │      ├── BLOCKED
  │      │      │      └── IN_PROGRESS
  │      │      └── COMPLETED
  │      └── READY (khi hủy phân công hợp lệ)
  └── CANCELLED (khi Order không còn cần đóng gói)
```

Tên trạng thái chính thức và ma trận chuyển trạng thái cần được chốt trước Technical Design. `COMPLETED` của Packing Task là kết quả để EPIC 08 xem xét chuyển Order sang `PACKED`; hai trạng thái không phải cùng một bản ghi.

### 6.2. Quy tắc dữ liệu và an toàn nghiệp vụ

- Mỗi Packing Task phải liên kết với đúng Order và phạm vi kiện hàng. Một Order có thể có nhiều Task/kiện nếu chính sách cho phép, nhưng không được tạo trùng cùng phạm vi.
- Dữ liệu lấy hàng dùng snapshot Order kết hợp phân bổ Batch hiện hành từ Inventory; thay đổi catalog không được sửa danh sách đã đặt.
- Phân công phải có actor, thời điểm và trạng thái hiệu lực. Chỉ người đang được phân công hoặc người thay thế có thẩm quyền mới xác nhận checklist/hoàn tất.
- Checklist phải lưu từng bước và kết quả, không chỉ một cờ hoàn tất chung. Bước không đạt phải chặn hoàn tất cho đến khi được xử lý hợp lệ.
- Media phải có định danh, trạng thái lưu, người tạo, thời điểm, loại bằng chứng và liên kết Task/Order/kiện hàng. Bản lưu thất bại không được coi là bằng chứng.
- Quyền biết Order không mặc nhiên cấp quyền xem video. Quyền media phải được kiểm tra tại mỗi lần truy cập, kể cả khi có liên kết trực tiếp.
- Bằng chứng sau hoàn tất không được xóa/ghi đè tùy ý. Mọi bổ sung, thay thế hoặc vô hiệu hóa phải bảo toàn bản gốc và lịch sử.
- Kết quả hoàn tất gửi lặp không được làm Order chuyển trạng thái, trừ tồn hoặc tạo Shipment nhiều lần.
- Packing không tự chuyển Order sang `PACKED`; EPIC 08 kiểm tra và áp dụng chuyển trạng thái. Packing không tự tạo mã vận đơn; EPIC 11 phụ trách Shipping.
- Chi tiết thiết bị quay, định dạng/codec, kích thước, thời lượng, endpoint, object storage, URL truy cập và cơ chế phát sự kiện thuộc Technical Design/NFR.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-PACK-01` | `BR-AUTH-04`, `BR-ORDER-01` | EPIC 07, 08, 09 | Chỉ Order đủ điều kiện xuất hiện; actor không quyền không xem được hàng đợi. |
| `US-PACK-02` | `BR-PROD-02`, `BR-BATCH-01`, `BR-BATCH-05~06` | EPIC 04, 08, 09 | Hiển thị đúng snapshot SKU/số lượng/Batch; Batch không hợp lệ bị chặn. |
| `US-PACK-03` | `BR-PACK-02` | EPIC 08, 09 | Checklist lưu từng bước; thiếu bước hoặc sai SKU/số lượng không thể hoàn tất. |
| `US-PACK-04` | `BR-PACK-03` | EXT-09 | Media hợp lệ được lưu đúng trạng thái; video bắt buộc chưa thành công chặn hoàn tất. |
| `US-PACK-05` | `BR-PACK-04`, `BR-AUDIT-01` | EPIC 08, 11, 23 | Video liên kết đúng Order/kiện; không liên kết chéo; thay thế bảo toàn bản gốc. |
| `US-PACK-06` | `BR-AUTH-04`, `BR-PACK-05` | EPIC 14, 16, 22, 23 | Chỉ actor có quyền truy cập đúng video; truy cập nhạy cảm có Audit; CSKH không sửa bằng chứng. |
| `US-PACK-07` | `BR-ORDER-01`, `BR-ORDER-05` | EPIC 08, 11 | Task ưu tiên đúng hạn/SLA, cảnh báo quá hạn và thể hiện rõ Task bị chặn. |
| `US-PACK-08` | `BR-PACK-01~04`, `BR-ORDER-05` | EPIC 08, 09, 11, 23 | Chỉ người được phân công hoàn tất khi đủ điều kiện; yêu cầu lặp không gây tác động trùng; mở lại có Audit. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Một Order tương ứng một Packing Task/kiện hay có thể tách nhiều kiện, đóng gói một phần và hoàn tất từng kiện.
- Điều kiện chính xác để Order vào hàng đợi Packing, đặc biệt với COD và các Order nguồn Marketplace/B2B.
- Cơ chế phân công: tự nhận việc, quản lý chỉ định, tự động phân công hay kết hợp; quy tắc chuyển giao giữa nhân viên.
- Bộ checklist bắt buộc theo loại SKU, kênh, loại bao bì/quà tặng và yêu cầu đặc biệt của Order.
- Cách xác nhận SKU/Batch thực lấy: nhập tay, quét mã hay phương án khác; chi tiết thiết bị thuộc Technical Design.
- Mốc ghi xuất kho chính thức: khi lấy hàng, hoàn tất đóng gói hay bàn giao Shipping; cách bù trừ khi mở lại/hủy Task.
- Công thức SLA Packing và nguồn hạn giao; cách xử lý ngày lễ, giờ làm việc, đơn ưu tiên và Task bị chặn.
- Order nào bắt buộc quay video; tiêu chí bắt đầu/kết thúc để video đủ giá trị bằng chứng.
- Giới hạn định dạng, kích thước, thời lượng, chất lượng video và cách xử lý kết nối gián đoạn.
- Chính sách lưu giữ/xóa Packing Video, chi phí lưu trữ, trường hợp legal hold và quyền tải xuống.
- Video có thể chứa nhãn giao hàng/thông tin cá nhân; cần quy tắc giảm thiểu dữ liệu và che thông tin khi chia sẻ phục vụ điều tra.
- Actor nào được xem, tải, bổ sung, thay thế hoặc vô hiệu hóa video; System Administrator không nên mặc nhiên có quyền nghiệp vụ chỉ vì quản trị hệ thống.
- Có cho phép hoàn tất Packing khi media tùy chọn đang tải dở hay không; media bắt buộc luôn phải sẵn sàng trước khi hoàn tất.
- Quy tắc mở lại Task sau `COMPLETED`, đặc biệt khi đã xuất kho, tạo Shipment hoặc bàn giao vận chuyển.
- Quan hệ giữa Packing Video với Order, Packing Task, Package và Shipment khi Shipment chỉ được tạo sau đóng gói.

## 9. UI/UX Reference

- Hàng đợi Packing hiển thị mã Order, số dòng, nguồn, phân công, hạn/SLA, mức ưu tiên và trạng thái bị chặn.
- Chi tiết Task đặt thông tin SKU/quy cách/số lượng/Batch cạnh checklist để giảm chuyển màn hình và nhầm hàng.
- Checklist thể hiện rõ bước chưa làm, đạt hoặc không đạt; không cho nút hoàn tất ở trạng thái có thể gây hiểu nhầm.
- Khu vực media hiển thị tiến trình lưu, kết quả thành công/thất bại và trạng thái bắt buộc; không coi thumbnail cục bộ là đã lưu bằng chứng.
- Nút hoàn tất hiển thị điều kiện còn thiếu, chống thao tác lặp và chỉ xuất hiện cho người được phân công có quyền.
- Trang điều tra hiển thị Order, kiện hàng, Packing Task và media liên quan nhưng chỉ cung cấp hành động đúng quyền của CSKH/Manager/Admin.
- Liên kết video không được để lộ vị trí lưu trữ hoặc cho phép truy cập lâu dài ngoài cơ chế kiểm soát quyền.
- Chỉ tiêu hiệu năng, upload resumable, codec, endpoint và công nghệ lưu trữ được đặc tả trong NFR hoặc Technical Design.

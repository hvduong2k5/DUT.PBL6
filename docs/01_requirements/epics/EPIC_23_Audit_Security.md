# EPIC 23 — Audit & Security

## 1. Mục tiêu Epic

Epic này tạo và bảo vệ dấu vết kiểm toán cho các thao tác nghiệp vụ quan trọng. System Administrator/Auditor có thể tra cứu ai đã làm gì, khi nào, ở đâu, trên đối tượng nào; Auditor xem giá trị trước–sau và lý do; Security Operator phát hiện bản ghi bị thay đổi hoặc chuỗi Audit mất tính liên tục.

EPIC 23 sở hữu Audit Event, chính sách tra cứu/xuất, retention và kết quả kiểm chứng tính toàn vẹn. Epic không sở hữu Role/Permission, dữ liệu nghiệp vụ nguồn, log hiệu năng kỹ thuật hay quy trình xử lý sự cố hoàn chỉnh; các trách nhiệm đó thuộc EPIC 22, từng Epic nghiệp vụ, NFR Observability và quy trình Security/Incident Response cần được thiết kế riêng.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 3rd** | `US-AUDIT-01`, `US-AUDIT-02` | Người có đặc quyền tra cứu lịch sử thao tác và đối chiếu giá trị trước–sau của thay đổi quan trọng. |
| **Giai đoạn 2** | `US-AUDIT-03` đến `US-AUDIT-05` | Audit bổ sung nguồn truy cập, lý do nghiệp vụ và cơ chế phát hiện thay đổi/mất tính liên tục. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và nguồn sự kiện

| Actor / nguồn | Vai trò trong Epic |
| --- | --- |
| System Administrator (`ACT-17`) | Tra cứu/trích xuất Audit khi được cấp đặc quyền tương ứng; không mặc nhiên có mọi quyền Audit. |
| Auditor (`ACT-18`) | Điều tra lịch sử, before/after, nguồn và lý do trong phạm vi được cấp. |
| Security Operator (`ACT-18`) | Chạy/xem kết quả kiểm chứng tính toàn vẹn và tiếp nhận cảnh báo sai lệch. |
| Epic/module nghiệp vụ | Phát sinh Audit Event cho thao tác quan trọng với ngữ cảnh cần thiết. |
| Notification Provider (`EXT-06`) | Chuyển cảnh báo integrity qua EPIC 28 nếu chính sách yêu cầu. |

### 3.2. Business Rules và NFR áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-03` | Tài khoản nội bộ phải có Role phù hợp trước khi truy cập quản trị. | Người dùng Audit Console phải có System Access và Role hợp lệ. |
| `BR-AUTH-04` | Nhân viên chỉ thực hiện thao tác thuộc quyền được cấp. | Quyền tra cứu, xem dữ liệu nhạy cảm, trích xuất và kiểm chứng integrity phải tách biệt. |
| `BR-REFUND-04` | Mọi thao tác duyệt/từ chối/thực hiện Refund phải lưu Audit Log. | Refund là luồng bắt buộc kiểm tra đầy đủ quyết định, số tiền, phương thức và kết quả. |
| `BR-AUDIT-01` | Thao tác quan trọng phải được ghi Audit Log. | Các Epic nguồn phải xác định catalog sự kiện bắt buộc và gửi đủ dữ liệu. |
| `BR-AUDIT-02` | Audit xác định Who, What, When, Where, Why, Object, Before và After. | Audit Event cần schema/ngữ nghĩa chung, cho phép trường không áp dụng nhưng không được bịa dữ liệu. |
| `BR-AUDIT-03` | Audit Log phải được bảo vệ chống sửa/xóa trái phép. | Bản ghi đã chấp nhận không được sửa/xóa qua nghiệp vụ thông thường và phải có thể kiểm chứng. |
| `BR-AUDIT-04` | Chỉ Auditor/System Administrator đặc quyền được tra cứu/trích xuất. | Có Role phù hợp chưa đủ; phải có Permission đặc quyền và phạm vi dữ liệu cụ thể. |
| `BR-AUDIT-05` | Audit Log lưu tối thiểu theo thời gian quy định và không có chức năng xóa thủ công trên UI. | Retention phải thực thi tự động/theo chính sách; không cung cấp nút xóa từng bản ghi. |
| `NFR-04` | Dữ liệu Customer, Employee, Order, Payment, Supplier và Audit phải được bảo vệ. | Audit thường chứa dữ liệu nhạy cảm nên cần masking, scope và bảo vệ export. |
| `NFR-05` | Người dùng chỉ truy cập dữ liệu/chức năng thuộc quyền. | Tìm kiếm hoặc biết ID không cho phép vượt phạm vi Audit. |
| `NFR-07` | Audit Log phải phát hiện sửa/xóa trái phép. | Integrity verification phải nhận biết bản ghi đổi, thiếu hoặc chuỗi không liên tục. |
| `NFR-08` | Dữ liệu cá nhân chỉ dùng theo mục đích được phép. | Audit chỉ lưu/hiển thị PII cần thiết và áp dụng redaction theo quyền. |
| `NFR-10` | Hệ thống cần xử lý tải truy vấn dữ liệu lớn. | Tra cứu/xuất Audit phải phân trang/giới hạn và không làm ảnh hưởng luồng nghiệp vụ. |
| `NFR-11` | Hệ thống phải theo dõi lỗi, hiệu năng và trạng thái xử lý/tích hợp. | Sức khỏe pipeline Audit và kiểm chứng integrity phải quan sát được, nhưng telemetry không thay Audit Event. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 22 — Administration`: cấp Permission tra cứu/trích xuất/verify; phát sinh Audit cho thay đổi Account/Role/Permission.
- `EPIC 07, 08, 09, 14`: cung cấp sự kiện Payment, Order, Inventory và Refund nhạy cảm; dữ liệu nguồn vẫn thuộc Epic tương ứng.
- `EPIC 10, 15, 16, 18–21, 24–28`: phát sinh sự kiện khi thao tác quan trọng theo catalog Audit được chốt.
- `EPIC 28 — Notification`: gửi cảnh báo integrity hoặc pipeline; EPIC 23 sở hữu điều kiện/căn cứ cảnh báo.
- `NFR-11 — Observability`: giám sát lỗi/độ trễ/sức khỏe. Operational log/trace không được coi là Audit chỉ vì có timestamp.
- Technical Design quyết định cơ chế tamper-evident, lưu trữ và tính toán integrity; Epic chỉ quy định kết quả phải phát hiện được thay đổi/mất liên tục.

## 4. User Stories chi tiết

### US-AUDIT-01 — Tra cứu lịch sử thao tác

**Actor:** System Administrator (`ACT-17`), Auditor (`ACT-18`)
**Ưu tiên:** Must Have — 3rd
**Phát hành:** MVP

> Là System Administrator/Auditor, tôi muốn tra cứu lịch sử thao tác để biết ai đã thực hiện hành động gì, khi nào và trên đối tượng nào.

**Giá trị nghiệp vụ:** Người có trách nhiệm xác định được chuỗi hành động liên quan tới một actor/đối tượng và có căn cứ điều tra sai lệch.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tra cứu Audit theo đối tượng
  Given Auditor có quyền với phạm vi dữ liệu
  When tìm theo loại và mã Object/Resource
  Then hệ thống hiển thị các Audit Event phù hợp theo thứ tự thời gian xác định
  And mỗi event thể hiện actor, action, thời điểm, object và outcome được phép

Scenario: Lọc theo actor, action và thời gian
  Given có nhiều Audit Event trong phạm vi
  When người dùng áp dụng actor, action, module, outcome và khoảng thời gian hợp lệ
  Then hệ thống trả đúng tập kết quả có phân trang
  And hiển thị timezone/thời điểm truy vấn rõ ràng

Scenario: Không có kết quả
  Given bộ lọc hợp lệ nhưng không có event phù hợp
  When thực hiện tra cứu
  Then hệ thống hiển thị trạng thái không có dữ liệu
  And không diễn giải như Audit Event đã bị xóa

Scenario: Người dùng không có đặc quyền Audit
  Given tài khoản đăng nhập nhưng thiếu Permission tra cứu Audit
  When mở Audit Console hoặc truy cập bản ghi trực tiếp
  Then hệ thống từ chối
  And không tiết lộ sự tồn tại/nội dung event ngoài quyền

Scenario: Có quyền Audit nhưng ngoài phạm vi dữ liệu
  Given Auditor chỉ được xem một phạm vi module/đơn vị
  When tìm event ngoài phạm vi
  Then hệ thống loại hoặc che kết quả theo chính sách
  And không cho vượt scope bằng bộ lọc hoặc export

Scenario: Audit Event đến muộn hoặc khác thứ tự tiếp nhận
  Given event xảy ra trước nhưng được ghi nhận sau event khác
  When hiển thị timeline
  Then hệ thống phân biệt thời điểm xảy ra và thời điểm tiếp nhận/ghi
  And không sắp xếp gây hiểu sai mà không nêu căn cứ

Scenario: Tra cứu Audit cũng là thao tác nhạy cảm
  Given người dùng tra cứu hoặc xem chi tiết Audit
  When thao tác đáp ứng chính sách cần Audit
  Then hệ thống ghi nhận hoạt động truy cập tương ứng
  And không tạo vòng ghi vô hạn hoặc làm mất event gốc

Scenario: Trích xuất kết quả có kiểm soát
  Given người dùng có Permission trích xuất riêng và bộ lọc hợp lệ
  When yêu cầu export
  Then hệ thống tạo kết quả gắn người yêu cầu, thời điểm, phạm vi/bộ lọc và trạng thái
  And chỉ chứa trường/bản ghi người dùng được phép xem
```

### US-AUDIT-02 — Xem giá trị trước và sau

**Actor:** Auditor (`ACT-18`)
**Ưu tiên:** Must Have — 3rd
**Phát hành:** MVP

> Là Auditor, tôi muốn xem giá trị trước và sau khi dữ liệu quan trọng thay đổi để xác định nguyên nhân sai lệch.

**Giá trị nghiệp vụ:** Auditor biết trường nào thay đổi và giá trị chuyển đổi thực tế, thay vì chỉ thấy một nhãn “đã cập nhật”.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem diff của thao tác cập nhật
  Given Audit Event ghi nhận một thay đổi dữ liệu quan trọng
  When Auditor mở chi tiết
  Then hệ thống hiển thị before và after cho các trường được phép
  And làm nổi bật trường thay đổi, actor, action, object và thời điểm

Scenario: Audit thao tác tạo mới
  Given event đại diện cho việc tạo đối tượng
  When xem before/after
  Then hệ thống thể hiện before là không tồn tại/không áp dụng
  And after là snapshot được phép của đối tượng vừa tạo

Scenario: Audit thao tác xóa hoặc vô hiệu hóa
  Given event đại diện cho xóa logic/vô hiệu hóa
  When xem chi tiết
  Then hệ thống giữ snapshot before cần thiết và trạng thái after
  And không làm mất căn cứ chỉ vì đối tượng nguồn không còn hoạt động

Scenario: Trường bí mật hoặc nhạy cảm thay đổi
  Given before/after có credential, token, dữ liệu thanh toán hoặc PII bị hạn chế
  When Auditor xem event
  Then hệ thống mask/loại trường theo chính sách và Permission
  And không lưu/hiển thị bí mật đọc được chỉ để chứng minh có thay đổi

Scenario: Giá trị lớn hoặc tài nguyên ngoài
  Given thay đổi liên quan media/tài liệu hoặc payload vượt giới hạn inline
  When event được ghi và hiển thị
  Then Audit giữ metadata/hash/tham chiếu an toàn đủ đối chiếu theo chính sách
  And không sao chép toàn bộ tài nguyên nhạy cảm không cần thiết

Scenario: Đối tượng nguồn tiếp tục thay đổi
  Given dữ liệu hiện tại khác after của event lịch sử
  When Auditor xem event
  Then hệ thống giữ nguyên before/after tại thời điểm thao tác
  And không thay event cũ bằng trạng thái hiện tại

Scenario: Event thiếu before/after khi nghiệp vụ yêu cầu
  Given catalog Audit quy định action phải có before/after
  When event thiếu trường bắt buộc
  Then hệ thống đánh dấu event/pipeline không đầy đủ theo chính sách
  And không tự bịa snapshot từ trạng thái hiện tại

Scenario: Người dùng thiếu quyền xem giá trị chi tiết
  Given người dùng có quyền danh sách nhưng không có quyền xem dữ liệu nhạy cảm
  When mở before/after
  Then hệ thống che hoặc từ chối phần ngoài quyền
  And vẫn bảo vệ toàn bộ event gốc
```

### US-AUDIT-03 — Xem nguồn truy cập hoặc vị trí kỹ thuật

**Actor:** Auditor (`ACT-18`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Auditor, tôi muốn biết nguồn truy cập hoặc vị trí kỹ thuật của thao tác khi thông tin này khả dụng để phục vụ điều tra.

**Giá trị nghiệp vụ:** Auditor có thêm bối cảnh kênh/nguồn/phiên/correlation để nối các thao tác liên quan mà không suy diễn vị trí khi dữ liệu không tồn tại.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị nguồn truy cập khả dụng
  Given Audit Event có dữ liệu nguồn được phép như kênh, client, network source, session hoặc service
  When Auditor mở chi tiết
  Then hệ thống hiển thị giá trị cùng loại nguồn
  And không trình bày network source như vị trí địa lý chính xác nếu chưa có căn cứ

Scenario: Nguồn không khả dụng
  Given thao tác không cung cấp được một trường nguồn
  When xem event
  Then hệ thống hiển thị `không khả dụng` hoặc `không áp dụng`
  And không điền giá trị suy đoán

Scenario: Thao tác qua hệ thống ngoài
  Given event bắt nguồn từ Provider/webhook/integration
  When xem nguồn
  Then hệ thống phân biệt external system, actor nghiệp vụ nếu biết và correlation/reference
  And không gán thao tác cho một nhân viên không có căn cứ

Scenario: Thao tác do hệ thống tự động
  Given scheduler/system process tạo thay đổi
  When event được ghi
  Then actor type thể hiện là system/process với định danh tác vụ phù hợp
  And giữ nguyên initiating actor/correlation nếu chuỗi bắt đầu từ người dùng

Scenario: Nhiều hop hoặc proxy
  Given yêu cầu đi qua nhiều lớp/kênh
  When nguồn được ghi nhận
  Then hệ thống sử dụng nguồn tin cậy theo Technical Design
  And không tin giá trị client có thể tự khai báo như bằng chứng tuyệt đối

Scenario: Tìm theo correlation
  Given nhiều event thuộc cùng một luồng nghiệp vụ
  When Auditor tra cứu correlation/reference hợp lệ
  Then hệ thống trả các event liên quan trong phạm vi quyền
  And vẫn phân biệt từng object/action/outcome

Scenario: Dữ liệu nguồn bị hạn chế vì riêng tư
  Given source data thuộc loại nhạy cảm
  When Auditor không có Permission chi tiết
  Then hệ thống mask/giới hạn theo chính sách
  And không cho export vượt phạm vi xem
```

### US-AUDIT-04 — Xem lý do nghiệp vụ

**Actor:** Auditor (`ACT-18`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Auditor, tôi muốn biết lý do nghiệp vụ của các thay đổi nhạy cảm khi thao tác đó yêu cầu người dùng cung cấp lý do.

**Giá trị nghiệp vụ:** Quyết định như Refund, điều chỉnh tồn, khóa tài khoản hoặc thay quyền có căn cứ do actor cung cấp tại thời điểm thao tác.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Hiển thị lý do của thao tác bắt buộc
  Given action nguồn yêu cầu lý do và event đã ghi reason hợp lệ
  When Auditor mở chi tiết
  Then hệ thống hiển thị reason cùng action, actor, object và thời điểm
  And phân biệt lý do do người dùng nhập với mô tả do hệ thống sinh

Scenario: Chặn thao tác nguồn khi thiếu lý do
  Given chính sách nghiệp vụ yêu cầu reason trước khi thực hiện
  When actor bỏ trống hoặc nhập lý do không hợp lệ
  Then module nguồn phải từ chối thay đổi nghiệp vụ
  And Audit không được bổ sung lý do giả sau đó để hợp thức hóa

Scenario: Event thiếu reason ngoài ý muốn
  Given action yêu cầu reason nhưng event đã tiếp nhận không có reason
  When hệ thống kiểm tra completeness
  Then event được đánh dấu thiếu dữ liệu/cảnh báo theo chính sách
  And bản ghi gốc không bị sửa để tự thêm lý do

Scenario: Action không yêu cầu reason
  Given thao tác không có yêu cầu lý do
  When Auditor xem event
  Then trường reason thể hiện không áp dụng/không được cung cấp
  And không coi đó là lỗi nếu catalog không yêu cầu

Scenario: Bổ sung giải trình sau sự kiện
  Given cần thêm nhận xét điều tra sau khi Audit Event đã được ghi
  When người có quyền thêm annotation
  Then hệ thống tạo bản ghi/annotation riêng với tác giả và thời điểm
  And không sửa reason nguyên gốc

Scenario: Reason chứa dữ liệu nhạy cảm
  Given người dùng nhập PII/bí mật không cần thiết trong reason
  When event được xử lý/hiển thị
  Then hệ thống áp dụng validation/redaction và cảnh báo theo chính sách
  And không mở rộng quyền xem chỉ vì dữ liệu nằm trong reason

Scenario: Lý do thay đổi quyết định
  Given một quyết định cũ được đảo/thay qua quy trình hợp lệ
  When event mới được ghi
  Then event mới tham chiếu quyết định/event trước và chứa lý do mới
  And không sửa/xóa lý do của quyết định cũ
```

### US-AUDIT-05 — Phát hiện Audit Log bị thay đổi hoặc mất tính liên tục

**Actor:** Security Operator (`ACT-18`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Security Operator, tôi muốn phát hiện Audit Log bị thay đổi hoặc mất tính liên tục để bảo vệ tính toàn vẹn của nhật ký.

**Giá trị nghiệp vụ:** Doanh nghiệp nhận biết được Audit Event bị sửa, mất, chèn sai hoặc không thể kiểm chứng và có căn cứ khoanh vùng điều tra.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Kiểm chứng phạm vi toàn vẹn thành công
  Given các Audit Event trong phạm vi kiểm tra không bị thay đổi/mất liên tục
  When Security Operator chạy hoặc xem lần verification
  Then kết quả là `VALID`
  And lưu phạm vi, thời điểm, phiên bản phương pháp và số event đã kiểm tra

Scenario: Phát hiện bản ghi bị thay đổi
  Given nội dung của một Audit Event đã khác dữ liệu được bảo vệ
  When verification chạy
  Then kết quả là `INVALID`
  And chỉ rõ phạm vi/điểm sai lệch đủ để điều tra mà không sửa event
  And phát sinh cảnh báo theo chính sách

Scenario: Phát hiện khoảng trống hoặc mất tính liên tục
  Given một event/segment bị thiếu hoặc quan hệ continuity không còn hợp lệ
  When verification chạy
  Then kết quả là `INCOMPLETE` hoặc `INVALID` theo quy tắc đã chốt
  And thể hiện khoảng/phạm vi bị ảnh hưởng

Scenario: Event đến muộn hoặc retry hợp lệ
  Given pipeline có event đến muộn, gửi lặp hoặc xử lý lại hợp lệ
  When verification thực hiện
  Then hệ thống áp dụng quy tắc ordering/dedup đã thiết kế
  And không cảnh báo giả chỉ vì thứ tự tiếp nhận khác thời điểm xảy ra

Scenario: Chỉ kiểm tra một phần phạm vi
  Given verification chỉ bao phủ một khoảng thời gian/partition/module
  When kết quả được hiển thị
  Then hệ thống nêu rõ coverage
  And không trình bày toàn bộ Audit Log là hợp lệ ngoài phạm vi đã kiểm tra

Scenario: Verification lỗi hoặc không hoàn tất
  Given tác vụ kiểm chứng bị lỗi/timeout hoặc thiếu dữ liệu cần thiết
  When kết thúc lần chạy
  Then kết quả là `ERROR` hoặc `INCOMPLETE`, không phải `VALID`
  And lưu lỗi, thời điểm và phạm vi đã xử lý
  And cho phép retry an toàn

Scenario: Không tạo cảnh báo trùng liên tục
  Given cùng một sai lệch đã có cảnh báo đang hoạt động
  When verification lặp lại mà không có thay đổi trạng thái
  Then hệ thống không tạo nhiều cảnh báo giống nhau ngoài ý muốn
  And cập nhật lần quan sát gần nhất theo chính sách

Scenario: Người dùng không có quyền verification
  Given người dùng thiếu Permission kiểm chứng integrity
  When chạy verification hoặc xem chi tiết sai lệch
  Then hệ thống từ chối
  And ghi nhận nỗ lực truy cập theo chính sách Security Audit
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-AUDIT-01` | Tiếp nhận Audit Event cho action quan trọng theo catalog với định danh duy nhất và chống ghi trùng khi producer retry. | `US-AUDIT-01`, `BR-AUDIT-01` |
| `FR-AUDIT-02` | Lưu actor/type, action, occurred/recorded time, source, reason nếu áp dụng, object/resource, before/after, outcome và correlation theo schema chung. | `US-AUDIT-01~04`, `BR-AUDIT-02` |
| `FR-AUDIT-03` | Cho người có đặc quyền lọc theo actor, action, module, object, outcome, correlation và khoảng thời gian với phân trang. | `US-AUDIT-01`, `BR-AUDIT-04`, `NFR-10` |
| `FR-AUDIT-04` | Thực thi scope/field-level access cho tra cứu, chi tiết và export; không lộ event ngoài quyền qua ID/bộ lọc. | `US-AUDIT-01~04`, `BR-AUTH-04`, `NFR-05` |
| `FR-AUDIT-05` | Phân biệt occurred time với recorded time, thể hiện timezone và xử lý event đến muộn không gây timeline sai lệch. | `US-AUDIT-01`, `US-AUDIT-03` |
| `FR-AUDIT-06` | Ghi Audit cho việc tra cứu/xem/export nhạy cảm theo chính sách mà không tạo vòng ghi vô hạn. | `US-AUDIT-01`, `BR-AUDIT-01` |
| `FR-AUDIT-07` | Export phải gắn người yêu cầu, thời điểm, filter/scope, trạng thái và chỉ chứa dữ liệu được phép. | `US-AUDIT-01`, `BR-AUDIT-04` |
| `FR-AUDIT-08` | Hiển thị diff before/after cho trường thay đổi và biểu diễn rõ create/delete/disable khi một phía không áp dụng. | `US-AUDIT-02`, `BR-AUDIT-02` |
| `FR-AUDIT-09` | Giữ snapshot lịch sử bất biến với trạng thái tại thời điểm action, không thay theo dữ liệu nguồn hiện tại. | `US-AUDIT-02`, `BR-AUDIT-03` |
| `FR-AUDIT-10` | Không lưu/hiển thị credential, token hoặc bí mật đọc được; mask/redact PII và trường nhạy cảm theo chính sách/quyền. | `US-AUDIT-02`, `US-AUDIT-04`, `NFR-04`, `NFR-08` |
| `FR-AUDIT-11` | Với payload lớn/media, lưu metadata/hash/tham chiếu an toàn đủ đối chiếu thay vì sao chép dữ liệu không cần thiết. | `US-AUDIT-02` |
| `FR-AUDIT-12` | Phát hiện event thiếu trường bắt buộc theo catalog và không tự dựng before/after/reason từ trạng thái hiện tại. | `US-AUDIT-02`, `US-AUDIT-04` |
| `FR-AUDIT-13` | Lưu nguồn khả dụng với loại/độ tin cậy phù hợp; phân biệt client, service, system process và external system. | `US-AUDIT-03`, `BR-AUDIT-02` |
| `FR-AUDIT-14` | Hiển thị `không khả dụng/không áp dụng` khi thiếu source và không suy diễn vị trí/network/actor. | `US-AUDIT-03` |
| `FR-AUDIT-15` | Cho tra cứu chuỗi event theo correlation/reference trong phạm vi quyền và vẫn phân biệt từng action/object/outcome. | `US-AUDIT-03` |
| `FR-AUDIT-16` | Lưu reason nguyên gốc cho action yêu cầu lý do và phân biệt với system description/annotation điều tra. | `US-AUDIT-04`, `BR-AUDIT-02` |
| `FR-AUDIT-17` | Annotation bổ sung phải là bản ghi riêng có tác giả/thời điểm, không sửa reason hoặc Audit Event gốc. | `US-AUDIT-04`, `BR-AUDIT-03` |
| `FR-AUDIT-18` | Hỗ trợ cơ chế tamper-evident có thể phát hiện event bị sửa, thiếu/chèn sai hoặc mất continuity theo Technical Design. | `US-AUDIT-05`, `BR-AUDIT-03`, `NFR-07` |
| `FR-AUDIT-19` | Tạo Integrity Verification Result có trạng thái, phạm vi/coverage, thời điểm, phương pháp/version, số event và điểm sai lệch/lỗi. | `US-AUDIT-05` |
| `FR-AUDIT-20` | Không coi partial/error là valid; hỗ trợ retry và xử lý late/duplicate event theo quy tắc nhất quán. | `US-AUDIT-05`, `NFR-11` |
| `FR-AUDIT-21` | Tạo/deduplicate cảnh báo integrity và chuyển qua EPIC 28 mà không sửa Audit Event bị nghi ngờ. | `US-AUDIT-05`, `EPIC 28` |
| `FR-AUDIT-22` | Áp dụng retention tối thiểu theo chính sách; không cung cấp xóa thủ công từng Audit Event trên UI. | `US-AUDIT-01~05`, `BR-AUDIT-05` |
| `FR-AUDIT-23` | Kiểm soát riêng quyền search, view detail, sensitive fields, export, annotate và verify integrity. | `US-AUDIT-01~05`, `BR-AUTH-03`, `BR-AUTH-04`, `BR-AUDIT-04` |
| `FR-AUDIT-24` | Theo dõi sức khỏe/độ trễ/lỗi của pipeline Audit và verification nhưng không trộn telemetry thành Audit Event nghiệp vụ. | `US-AUDIT-01`, `US-AUDIT-05`, `NFR-11` |
| `FR-AUDIT-25` | Bảo toàn Audit khi dữ liệu nghiệp vụ/Account nguồn bị khóa, archive hoặc xóa logic và vẫn quy được actor/object lịch sử. | `US-AUDIT-01~05`, `BR-AUDIT-03`, `BR-AUDIT-05` |

## 6. Mô hình dữ liệu, integrity và retention

### 6.1. Audit Event tối thiểu

```text
Audit Event
  ├── event_id / schema_version
  ├── actor_type / actor_id / actor_snapshot
  ├── action / outcome
  ├── object_type / object_id
  ├── occurred_at / recorded_at
  ├── source_context / correlation
  ├── reason (nếu áp dụng)
  ├── before / after hoặc diff được phép
  └── integrity metadata
```

- `actor_snapshot` chỉ giữ thông tin tối thiểu cần nhận diện lịch sử; không sao chép toàn bộ Employee/Customer Profile.
- `outcome` phân biệt thao tác thành công, bị từ chối hoặc thất bại nếu catalog yêu cầu ghi; event không tự chứng minh giao dịch nghiệp vụ thành công nếu outcome chưa xác định.
- Before/after phải phản ánh đúng thời điểm action và schema version; không lấy dữ liệu hiện tại để lấp khoảng trống lịch sử.
- Producer/module nguồn chịu trách nhiệm cung cấp ngữ nghĩa action/object/reason; EPIC 23 kiểm tra schema/completeness và bảo vệ bản ghi.

### 6.2. Vòng đời tiếp nhận Audit Event

```text
RECEIVED → VALIDATED → PROTECTED → RETAINED → ARCHIVED/DISPOSED_BY_POLICY
              │
              └──► INVALID/INCOMPLETE
```

- Đây là trạng thái xử lý, không cho phép người dùng sửa nội dung event sau khi ghi.
- Event thiếu/bất hợp lệ phải được nhận biết và xử lý theo chính sách; không được im lặng bỏ qua nếu action bắt buộc Audit.
- `DISPOSED_BY_POLICY`, nếu được phép sau retention/legal hold, là quy trình quản trị tự động/được kiểm soát; không phải nút xóa thủ công trên UI.
- Audit của đối tượng đã xóa logic/khóa vẫn được giữ theo retention và quy được định danh lịch sử.

### 6.3. Integrity Verification Result

```text
NOT_CHECKED → RUNNING → VALID
                    ├──► INVALID
                    ├──► INCOMPLETE
                    └──► ERROR
```

- `VALID` chỉ áp dụng cho coverage đã kiểm tra, không phải mọi Audit Event của toàn hệ thống.
- `INVALID` chỉ ra bằng chứng không khớp/thay đổi; `INCOMPLETE` chỉ ra khoảng trống hoặc dữ liệu chưa đủ; `ERROR` là lần verify không hoàn tất.
- Verification không được sửa event để “khắc phục” sai lệch; xử lý sự cố tạo case/annotation/cảnh báo riêng.
- Cơ chế bảo vệ cụ thể thuộc Technical Design, miễn đáp ứng khả năng tamper-evident và kiểm thử được.

### 6.4. Bảo mật, privacy và export

- Audit Log có độ nhạy cao vì tập trung lịch sử actor, dữ liệu trước/sau và nguồn truy cập; quyền truy cập phải theo least privilege và có scope.
- Không ghi password, secret, token, mã xác thực hoặc dữ liệu thẻ/credential đọc được. Với trường nhạy cảm, chỉ ghi trạng thái thay đổi hoặc giá trị đã bảo vệ khi đủ căn cứ.
- Search, detail và export có thể có quyền khác nhau; export không được mở rộng field/scope hơn giao diện người dùng được phép.
- Tệp export cần thời hạn truy cập/lưu trữ, định danh người yêu cầu và trạng thái tạo; chính sách tải/chia sẻ phải được chốt.
- Retention, archive, legal hold và disposal phải áp dụng theo loại sự kiện/quy định; không dùng “vĩnh viễn” mặc định khi chưa có quyết định.

### 6.5. Quan hệ với Observability và nghiệp vụ nguồn

- Audit Event trả lời trách nhiệm và thay đổi nghiệp vụ; operational log/metric/trace trả lời sức khỏe/hiệu năng. Một bản ghi kỹ thuật không tự trở thành Audit Event.
- Correlation có thể nối Audit với workflow/trace khi được phép nhưng không làm lộ telemetry nhạy cảm.
- Nếu pipeline Audit không khả dụng, chính sách fail-open/fail-closed phải được quyết định theo mức độ action; Epic không mặc định cho mọi thao tác tiếp tục hoặc dừng.
- Catalog Audit cần owner và version để biết module/action nào bắt buộc, trường nào required/sensitive và retention nào áp dụng.
- Thay đổi catalog/schema phải tương thích với event lịch sử và không làm verification cũ mất ý nghĩa.

## 7. Traceability

| User Story | Business Rule / NFR | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-AUDIT-01` | `BR-AUDIT-01`, `BR-AUDIT-04`, `NFR-05`, `NFR-10` | EPIC 22, các Epic nguồn | Tra cứu/filter/export đúng scope, phân trang, timezone; late event và audit-of-access không gây hiểu sai/vòng lặp. |
| `US-AUDIT-02` | `BR-AUDIT-02`, `BR-AUDIT-03`, `NFR-04`, `NFR-08` | Các Epic nguồn | Before/after đúng thời điểm, create/delete rõ; bí mật/PII được bảo vệ và thiếu snapshot không bị tự dựng. |
| `US-AUDIT-03` | `BR-AUDIT-02`, `BR-AUTH-04`, `NFR-08` | EPIC 22, hệ thống ngoài | Source/correlation đúng loại, không bịa khi thiếu, system/external actor rõ và field nhạy cảm được che. |
| `US-AUDIT-04` | `BR-AUDIT-02`, `BR-AUDIT-03` | EPIC 09, 14, 22 | Reason nguyên gốc đúng action; thiếu reason bị nhận biết; giải trình sau là annotation riêng. |
| `US-AUDIT-05` | `BR-AUDIT-03`, `BR-AUDIT-05`, `NFR-07`, `NFR-11` | EPIC 22, 28 | Verify phát hiện sửa/mất continuity, coverage rõ, partial/error không thành valid và cảnh báo không trùng. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Catalog action bắt buộc Audit theo từng Epic; có ghi denied/failed/read access không và mức chi tiết cho mỗi loại.
- Schema chuẩn, versioning, định danh actor/object khi nguồn bị xóa logic, correlation và quy tắc occurred/recorded time.
- Trường before/after bắt buộc, cách diff collection/quan hệ, giới hạn payload và xử lý media/tài liệu lớn.
- Danh sách secret/PII/financial fields cấm ghi hoặc phải mask; quyền xem field nhạy cảm và quy trình redaction khi producer gửi nhầm.
- Source fields được coi là đáng tin cậy, cách xử lý proxy/multiple hop, device/location và thời hạn lưu dữ liệu network.
- Action nào bắt buộc reason, validation/độ dài/danh mục lý do và quy tắc annotation/case điều tra.
- Permission matrix cho search/detail/sensitive fields/export/annotation/verify và phạm vi module/đơn vị/kho/kênh.
- Retention chính thức theo loại event thay cho ví dụ “1 năm hoặc vĩnh viễn”; archive, legal hold, disposal và bằng chứng đã hủy theo chính sách.
- Cơ chế tamper-evident cụ thể, đơn vị/segment của continuity, key management, rotation, backup/restore và verification schedule.
- Quy tắc với late/duplicate/out-of-order event để không tạo integrity alert giả và vẫn nhận biết event thật sự bị thiếu.
- Chính sách fail-open/fail-closed khi không ghi được Audit cho từng nhóm action nhạy cảm; retry/dead-letter/reconciliation.
- SLA cảnh báo, dedup/escalation, owner Incident Response và quy trình điều tra/khắc phục sau khi integrity invalid.
- Export format, giới hạn thời gian/kích thước, phê duyệt, watermark, expiry, storage và audit việc tải/chia sẻ.
- Mục tiêu hiệu năng truy vấn/ghi/verify, giới hạn filter/range và cách tránh Audit làm chậm giao dịch nghiệp vụ.

## 9. UI/UX Reference

- Audit Search cần bộ lọc actor/action/module/object/outcome/time/correlation, timezone rõ, phân trang và scope hiện tại.
- Result table cần hiển thị occurred time, actor, action, object, outcome và integrity status mà không lộ field nhạy cảm.
- Detail view cần tách metadata, source, reason và before/after diff; chỉ làm nổi bật trường thực sự thay đổi.
- Timeline/correlation view cần phân biệt event time với recorded time và từng actor/system/external source.
- Export dialog phải hiển thị filter, phạm vi, trường, ước lượng dữ liệu, lý do/phê duyệt nếu có và trạng thái file.
- Integrity dashboard cần coverage, lần verify gần nhất, valid/invalid/incomplete/error, điểm sai lệch và trạng thái cảnh báo.
- Annotation/incident link phải tách khỏi event gốc, thể hiện tác giả/thời điểm và không cung cấp hành động “sửa Audit”.
- Không có nút xóa thủ công từng Audit Event; chức năng retention/legal hold nếu có phải nằm ở khu vực quản trị riêng theo quyền.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

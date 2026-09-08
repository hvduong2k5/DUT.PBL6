# EPIC 16 — Customer Service

## 1. Mục tiêu Epic

Epic này quản lý luồng giao tiếp và tương tác hỗ trợ khách hàng: tiếp nhận yêu cầu, tạo Ticket, phân loại/ưu tiên, phân công, duy trì lịch sử hội thoại đa kênh, theo dõi SLA và cảnh báo các vấn đề chưa được xử lý. CSKH có đủ ngữ cảnh được cấp quyền để trả lời chính xác và chuyển yêu cầu sang quy trình nghiệp vụ phù hợp.

Epic không sở hữu quyết định đổi/trả/hoàn tiền, trạng thái Order/Payment/Shipment, dữ liệu Batch hay Review. Khi một Ticket liên quan các nghiệp vụ đó, EPIC 16 giữ hội thoại và SLA, còn trạng thái/quyết định nghiệp vụ vẫn thuộc EPIC 14, 08, 07, 11, 09 hoặc 15 tương ứng.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP v1.0** | `US-CS-01` | Khách gửi được câu hỏi/yêu cầu hỗ trợ và nhận mã Ticket để theo dõi. |
| **Giai đoạn 2** | `US-CS-02` đến `US-CS-06` | CSKH vận hành hàng đợi ưu tiên, lịch sử đa kênh, ngữ cảnh giao dịch, cảnh báo tồn đọng và phối hợp xử lý hàng sắp hết hạn. |
| **Sau Giai đoạn 2 — đề xuất** | `US-AI-02`, `US-AI-06` | AI hỗ trợ tổng hợp, soạn nháp và tìm kiếm tri thức nhưng không tự gửi trả lời hoặc thay đổi dữ liệu nghiệp vụ. |

Hai story AI có mức **Could Have** và Product Backlog chưa ấn định bản phát hành cụ thể; mốc “Sau Giai đoạn 2” là đề xuất cần được Product Owner xác nhận.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và hệ thống ngoài

| Actor / hệ thống | Vai trò trong Epic |
| --- | --- |
| Guest Customer (`ACT-01`) | Gửi yêu cầu hỗ trợ bằng thông tin liên hệ và dữ liệu xác minh theo chính sách. |
| Registered Customer (`ACT-02`) | Gửi, xem và phản hồi Ticket thuộc tài khoản của mình. |
| Customer Service (`ACT-11`) | Tiếp nhận, phân loại, xử lý, giao tiếp, ghi chú và chuyển tiếp Ticket theo quyền. |
| Sales Manager (`ACT-12`) | Phối hợp xử lý vấn đề bán hàng/hàng sắp hết hạn và xem phạm vi Ticket được cấp quyền. |
| Inventory/Supply Manager (`ACT-15`) | Cung cấp và phối hợp xử lý dữ liệu Batch/hạn sử dụng thuộc EPIC 09. |
| Nhân viên nội bộ được cấp quyền | Tìm kiếm kho tri thức trong phạm vi công việc ở `US-AI-06`. |
| Notification Provider (`EXT-06`) | Chuyển thông báo Email/SMS/Push/Zalo qua hạ tầng EPIC 28. |
| AI Provider (`EXT-08`) | Hỗ trợ tổng hợp, tạo Draft Reply và tìm kiếm tri thức khi tính năng AI được bật. |

### 3.2. Business Rules áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-04` | Nhân viên chỉ được thực hiện thao tác thuộc quyền được cấp. | Quyền xem dữ liệu khách/giao dịch, nhận Ticket, ghi chú, trả lời, chuyển cấp và dùng AI phải được kiểm soát riêng. |
| `BR-ORDER-01` | Order phải có trạng thái rõ ràng trong toàn bộ vòng đời. | Workspace CSKH chỉ hiển thị trạng thái từ nguồn Order, không tự suy diễn hoặc thay đổi trạng thái. |
| `BR-ORDER-03` | Order chỉ chuyển Paid khi giao dịch được xác nhận hợp lệ. | CSKH không được coi lời khai hoặc ảnh chụp của khách là xác nhận Payment thành công. |
| `BR-ORDER-05` | Thay đổi trạng thái quan trọng của Order phải được ghi nhận để truy vết. | Lịch sử hỗ trợ tham chiếu timeline nguồn, không ghi đè lịch sử Order. |
| `BR-PACK-04` | Packing Video phải liên kết với đúng Order/Shipment. | Ticket chỉ tham chiếu video đúng giao dịch/Case liên quan. |
| `BR-PACK-05` | Người không có quyền không được tùy ý xem/tải Packing Video. | Quyền xem Ticket hoặc Order không mặc nhiên cấp quyền xem video. |
| `BR-BATCH-03` | Hệ thống phải cảnh báo trước khi Batch/Lot đến hạn sử dụng. | Danh sách hàng sắp hết hạn của CSKH/Sales phải lấy từ cảnh báo nguồn EPIC 09. |
| `BR-BATCH-04` | Hàng gần hết hạn phải được ưu tiên xử lý theo chính sách. | Hoạt động phối hợp bán phải bám quyết định/chính sách hợp lệ. |
| `BR-BATCH-05` | Hàng hết hạn không được tiếp tục bán. | Danh sách phối hợp không được biến Batch đã hết hạn thành hàng có thể bán. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Phân công, chuyển cấp, thay đổi ưu tiên/SLA, đóng/mở lại và truy cập nhạy cảm phải có khả năng truy vết. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 07, 08, 11 — Payment, Order, Shipping`: cung cấp dữ liệu nguồn hiện tại; EPIC 16 chỉ hiển thị ngữ cảnh và liên kết xử lý.
- `EPIC 09 — Inventory & Batch`: sở hữu Batch, hạn sử dụng và cảnh báo; CSKH/Sales chỉ nhận danh sách và ghi nhận phối hợp.
- `EPIC 10 — Packing`: sở hữu Packing Video và quyền tài nguyên; Ticket chỉ tham chiếu bằng chứng đúng Order khi được phép.
- `EPIC 14 — Return / Refund / Complaint`: sở hữu Case và state machine đổi/trả/hoàn tiền. Ticket tiếp nhận, giao tiếp, kích hoạt Case và phản ánh kết quả cho khách.
- `EPIC 15 — Review & Rating`: phát hiện Review tiêu cực và chuyển ngữ cảnh sang Ticket; EPIC 16 không kiểm duyệt Review.
- `EPIC 20 — Content & SEO`: cung cấp nội dung sản phẩm công khai; không phải kho tri thức nội bộ mặc định.
- `EPIC 22, 23 — Administration, Audit & Security`: cung cấp Role/Permission, Audit Log và kiểm soát truy cập dữ liệu nhạy cảm.
- `EPIC 28 — Omnichannel Notification`: chịu trách nhiệm gửi thông báo và Internal Alert; EPIC 16 tạo sự kiện/nội dung nghiệp vụ và theo dõi kết quả liên quan.

## 4. User Stories chi tiết

### US-CS-01 — Gửi câu hỏi hoặc yêu cầu hỗ trợ

**Actor:** Guest Customer, Registered Customer
**Ưu tiên:** Must Have — 3rd
**Phát hành:** MVP v1.0

> Là khách hàng, tôi muốn gửi câu hỏi hoặc yêu cầu hỗ trợ.

**Giá trị nghiệp vụ:** Khách có một đầu mối chính thức để mô tả vấn đề, nhận mã theo dõi và bổ sung thông tin mà không phải gửi lại yêu cầu từ đầu.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Khách đã đăng nhập tạo Ticket
  Given khách hàng đã đăng nhập
  When khách chọn chủ đề, nhập nội dung hợp lệ và gửi yêu cầu
  Then hệ thống tạo một Ticket có mã duy nhất và trạng thái `NEW`
  And liên kết Ticket với đúng tài khoản khách hàng
  And hiển thị xác nhận cùng mã Ticket

Scenario: Guest gửi yêu cầu hỗ trợ
  Given khách chưa đăng nhập và kênh hỗ trợ cho phép Guest
  When khách cung cấp thông tin liên hệ cùng nội dung bắt buộc hợp lệ
  Then hệ thống tạo Ticket gắn với định danh liên hệ chưa xác minh theo chính sách
  And không tự liên kết Ticket với tài khoản hoặc Order chỉ từ dữ liệu dễ đoán

Scenario: Dữ liệu bắt buộc không hợp lệ
  Given khách đang tạo yêu cầu hỗ trợ
  When chủ đề, nội dung hoặc thông tin liên hệ bắt buộc bị thiếu hay không hợp lệ
  Then hệ thống từ chối tạo Ticket
  And chỉ rõ dữ liệu cần điều chỉnh

Scenario: Gắn Order do chính khách sở hữu
  Given khách đã đăng nhập và có một Order thuộc tài khoản
  When khách chọn Order đó làm ngữ cảnh yêu cầu
  Then hệ thống liên kết Ticket với đúng Order
  And lưu snapshot tham chiếu cần thiết mà không sao chép quyền thay đổi Order sang Ticket

Scenario: Khách cố gắn Order không thuộc quyền
  Given Order không thuộc khách hoặc Guest chưa hoàn tất xác minh cần thiết
  When khách cố liên kết Order với Ticket
  Then hệ thống từ chối liên kết
  And không tiết lộ trạng thái hoặc dữ liệu của Order

Scenario: Gửi lại cùng yêu cầu do lỗi mạng
  Given yêu cầu trước đã được tiếp nhận nhưng khách chưa nhận phản hồi giao diện
  When cùng thao tác được gửi lại trong phạm vi chống trùng
  Then hệ thống không tạo nhiều Ticket ngoài ý muốn
  And trả về kết quả Ticket đã được tiếp nhận nếu có thể xác định an toàn

Scenario: Xem Ticket của chính mình
  Given khách đã xác thực và sở hữu Ticket
  When khách mở chi tiết Ticket
  Then hệ thống hiển thị trạng thái cùng các trao đổi được phép công khai cho khách
  And không hiển thị ghi chú nội bộ hoặc dữ liệu ngoài quyền
```

### US-CS-02 — Xử lý danh sách yêu cầu theo ưu tiên

**Actor:** Customer Service (`ACT-11`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là CSKH, tôi muốn xem danh sách yêu cầu hỗ trợ để xử lý theo mức độ ưu tiên.

**Giá trị nghiệp vụ:** Đội CSKH xử lý đúng Ticket khẩn cấp hoặc gần vi phạm SLA, phân công rõ người chịu trách nhiệm và giảm bỏ sót.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem hàng đợi Ticket theo ưu tiên
  Given CSKH có quyền xem hàng đợi
  When CSKH mở danh sách yêu cầu
  Then hệ thống sắp xếp theo quy tắc ưu tiên và mốc SLA đang áp dụng
  And hiển thị rõ trạng thái, chủ đề, kênh, người phụ trách và thời gian còn lại

Scenario: Lọc và tìm Ticket
  Given hàng đợi có nhiều Ticket
  When CSKH lọc theo trạng thái, ưu tiên, chủ đề, kênh, người phụ trách hoặc khoảng thời gian
  Then hệ thống chỉ hiển thị Ticket phù hợp trong phạm vi quyền
  And kết quả không làm thay đổi dữ liệu Ticket

Scenario: Nhận xử lý Ticket chưa được phân công
  Given Ticket đang chờ và CSKH có quyền nhận xử lý
  When CSKH nhận Ticket
  Then hệ thống gán người phụ trách và cập nhật trạng thái phù hợp
  And lưu người thực hiện cùng thời điểm

Scenario: Hai nhân viên nhận cùng Ticket
  Given hai CSKH đang xem cùng một Ticket chưa phân công
  When cả hai gửi yêu cầu nhận xử lý gần đồng thời
  Then hệ thống chỉ chấp nhận một kết quả hợp lệ
  And thông báo dữ liệu mới cho người còn lại

Scenario: Chuyển Ticket cho hàng đợi hoặc nhân viên khác
  Given CSKH có quyền chuyển Ticket và cung cấp lý do hợp lệ
  When CSKH thực hiện chuyển
  Then hệ thống cập nhật đơn vị/người phụ trách
  And bảo toàn toàn bộ lịch sử cùng SLA theo quy tắc chuyển

Scenario: Nhân viên không có quyền xem hoặc nhận Ticket
  Given nhân viên không có quyền đối với hàng đợi hoặc phạm vi Ticket
  When nhân viên truy cập hay nhận xử lý
  Then hệ thống từ chối thao tác
  And không tiết lộ dữ liệu khách hàng
```

### US-CS-03 — Xem lịch sử trao đổi với khách hàng

**Actor:** Customer Service (`ACT-11`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là CSKH, tôi muốn xem lịch sử trao đổi với khách hàng để hỗ trợ chính xác.

**Giá trị nghiệp vụ:** Nhân viên hiểu đầy đủ bối cảnh qua các kênh, tránh yêu cầu khách lặp lại thông tin và phân biệt trao đổi công khai với ghi chú nội bộ.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem timeline hội thoại đa kênh
  Given CSKH có quyền xem Ticket
  When CSKH mở lịch sử trao đổi
  Then hệ thống hiển thị các tin theo thứ tự thời gian với người gửi, kênh, thời điểm và trạng thái gửi
  And các tin thuộc cùng Ticket được hợp nhất mà không làm mất nguồn gốc kênh

Scenario: Gửi phản hồi cho khách
  Given CSKH có quyền trả lời và Ticket chưa ở trạng thái khóa giao tiếp
  When CSKH gửi nội dung hợp lệ qua kênh được hỗ trợ
  Then hệ thống thêm phản hồi vào timeline
  And ghi nhận trạng thái gửi độc lập với việc lưu nội dung Ticket

Scenario: Gửi thông báo ra ngoài thất bại
  Given phản hồi đã được lưu nhưng nhà cung cấp kênh gửi thất bại hoặc chưa xác nhận
  When hệ thống nhận kết quả gửi
  Then timeline thể hiện trạng thái thất bại hoặc đang chờ
  And không đánh dấu khách đã nhận tin khi chưa có căn cứ
  And cho phép thử lại theo chính sách mà không nhân đôi nội dung

Scenario: Thêm ghi chú nội bộ
  Given CSKH có quyền ghi chú
  When CSKH thêm ghi chú nội bộ hợp lệ
  Then hệ thống gắn ghi chú vào Ticket với tác giả và thời điểm
  And không hiển thị ghi chú đó cho khách hoặc gửi qua kênh bên ngoài

Scenario: Phân biệt khách và nhân viên trong hội thoại
  Given timeline có tin từ khách, nhân viên và hệ thống
  When người có quyền xem lịch sử
  Then giao diện phân biệt rõ vai trò từng nguồn
  And không biến nội dung do khách nhập thành ghi chú/quyết định của nhân viên

Scenario: Hội thoại đến không xác định được Ticket
  Given hệ thống nhận một tin từ kênh ngoài nhưng chưa liên kết an toàn với Ticket
  When tiến hành tiếp nhận
  Then hệ thống đưa tin vào hàng đợi cần đối chiếu hoặc tạo luồng mới theo chính sách
  And không tự gắn tin vào khách/Ticket chỉ dựa trên dữ liệu không đủ tin cậy
```

### US-CS-04 — Xem ngữ cảnh giao dịch của khách

**Actor:** Customer Service (`ACT-11`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là CSKH, tôi muốn xem đơn hàng, thanh toán, vận chuyển và lịch sử mua hàng của khách để xử lý vấn đề.

**Giá trị nghiệp vụ:** CSKH kiểm tra đúng dữ liệu nguồn khi trả lời, giảm chuyển tuyến không cần thiết và tránh đưa ra cam kết dựa trên thông tin lỗi thời.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem tổng quan giao dịch liên quan
  Given CSKH có quyền và Ticket liên kết hợp lệ với khách/Order
  When CSKH mở workspace xử lý
  Then hệ thống hiển thị Order, trạng thái Payment, Shipment và lịch sử mua cần thiết từ nguồn tương ứng
  And nêu rõ thời điểm dữ liệu được cập nhật

Scenario: Xem timeline Order nhưng không sửa dữ liệu nguồn
  Given CSKH đang xem một Order liên quan
  When CSKH mở lịch sử trạng thái
  Then hệ thống hiển thị các mốc được phép theo nguồn EPIC 08
  And quyền xem trong Ticket không tự cho phép sửa Order, Payment hoặc Shipment

Scenario: Dữ liệu nguồn thay đổi khi Ticket đang mở
  Given trạng thái giao dịch đã thay đổi sau khi workspace được tải
  When CSKH tải lại hoặc chuẩn bị thực hiện hành động phụ thuộc dữ liệu
  Then hệ thống hiển thị trạng thái mới nhất cùng dấu thời gian
  And không âm thầm dùng snapshot cũ làm trạng thái hiện tại

Scenario: Che dữ liệu nhạy cảm
  Given workspace chứa thông tin thanh toán, địa chỉ hoặc liên hệ
  When CSKH xem Ticket
  Then hệ thống chỉ hiển thị trường cần thiết theo quyền và mục đích hỗ trợ
  And che hoặc loại bỏ dữ liệu nhạy cảm ngoài phạm vi

Scenario: Tra cứu Packing Video
  Given Ticket liên quan khiếu nại của một Order và CSKH có quyền xem video
  When CSKH mở bằng chứng đóng gói
  Then hệ thống chỉ hiển thị video liên kết đúng Order/Shipment từ EPIC 10
  And không cho sửa hoặc thay thế bằng chứng gốc

Scenario: Có quyền xem Ticket nhưng không có quyền xem video
  Given CSKH được phép xử lý Ticket nhưng thiếu quyền Packing Video
  When CSKH mở liên kết video
  Then hệ thống từ chối truy cập
  And không tiết lộ vị trí lưu trữ trực tiếp
```

### US-CS-05 — Cảnh báo vấn đề hoặc khiếu nại chưa xử lý

**Actor:** Customer Service (`ACT-11`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là CSKH, tôi muốn nhận cảnh báo các khách hàng có vấn đề hoặc khiếu nại chưa được xử lý.

**Giá trị nghiệp vụ:** Các Ticket gần quá hạn, bị bỏ quên hoặc đang chờ nội bộ được đưa trở lại đúng hàng đợi trước khi ảnh hưởng nghiêm trọng tới trải nghiệm khách hàng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Cảnh báo Ticket gần vi phạm SLA
  Given Ticket đang mở và thời hạn SLA sắp đến ngưỡng cảnh báo
  When hệ thống đánh giá SLA
  Then tạo Internal Alert cho người phụ trách hoặc hàng đợi phù hợp qua EPIC 28
  And cảnh báo tham chiếu đúng Ticket và mốc SLA

Scenario: Escalate Ticket quá hạn
  Given Ticket chưa hoàn tất mốc SLA và đã quá hạn
  When hệ thống đánh giá quy tắc chuyển cấp
  Then Ticket được đánh dấu quá hạn
  And gửi cảnh báo đến vai trò/cấp xử lý được cấu hình
  And lưu thời điểm cùng mốc bị vi phạm

Scenario: Không tạo cảnh báo trùng liên tục
  Given cảnh báo cho cùng Ticket, loại và ngưỡng SLA đã tồn tại
  When tác vụ đánh giá chạy lại mà không có mốc trạng thái mới
  Then hệ thống không tạo nhiều cảnh báo giống nhau ngoài ý muốn

Scenario: Ticket được giải quyết trước ngưỡng
  Given Ticket đã chuyển sang trạng thái không còn cần cảnh báo cho mốc hiện tại
  When hệ thống kiểm tra SLA
  Then không tạo cảnh báo quá hạn mới cho mốc đó
  And giữ lịch sử cảnh báo cũ để truy vết

Scenario: Ticket đang chờ khách
  Given Ticket ở trạng thái `WAITING_CUSTOMER`
  When hệ thống tính SLA
  Then áp dụng quy tắc dừng hoặc tiếp tục đồng hồ đã được cấu hình
  And hiển thị rõ căn cứ thay vì tự mặc định SLA đã dừng

Scenario: Nhân viên không có quyền xem cảnh báo
  Given nhân viên không có quyền đối với Ticket hoặc hàng đợi
  When nhân viên truy cập Internal Alert
  Then hệ thống không cung cấp nội dung nhạy cảm
  And không cho phép mở Ticket qua liên kết cảnh báo
```

### US-CS-06 — Phối hợp xử lý sản phẩm sắp hết hạn

**Actor:** Customer Service (`ACT-11`), Sales Manager (`ACT-12`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là CSKH/Sales, tôi muốn nhận danh sách sản phẩm sắp hết hạn để phối hợp triển khai chương trình bán phù hợp.

**Giá trị nghiệp vụ:** CSKH và Sales sử dụng cảnh báo tồn kho đã xác minh để phối hợp hoạt động phù hợp, giảm lãng phí nhưng không bán hàng hết hạn hoặc tự sửa dữ liệu kho.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem danh sách sản phẩm sắp hết hạn
  Given EPIC 09 đã xác định các Batch đạt ngưỡng cảnh báo
  When CSKH hoặc Sales Manager có quyền mở danh sách phối hợp
  Then hệ thống hiển thị Product/SKU, Batch, hạn sử dụng, số lượng liên quan và mức cảnh báo được phép
  And nêu rõ thời điểm dữ liệu nguồn được cập nhật

Scenario: Ghi nhận hoạt động phối hợp
  Given một sản phẩm sắp hết hạn cần kế hoạch xử lý
  When người có quyền thêm người phụ trách, ghi chú hoặc trạng thái phối hợp
  Then hệ thống lưu thông tin phối hợp tách biệt với dữ liệu tồn kho gốc
  And không tự tạo hoặc phê duyệt chương trình bán ngoài Epic chịu trách nhiệm

Scenario: Batch đã hết hạn
  Given Batch đã chuyển sang hết hạn theo dữ liệu EPIC 09
  When danh sách được cập nhật
  Then hệ thống đánh dấu không được bán và loại khỏi đề xuất chương trình bán
  And không cho hoạt động phối hợp ghi đè quy tắc chặn bán

Scenario: Số lượng hoặc hạn sử dụng thay đổi
  Given EPIC 09 cập nhật số lượng hay hạn sử dụng của Batch
  When người dùng tải lại danh sách hoặc chuẩn bị hành động
  Then hệ thống dùng dữ liệu nguồn mới nhất
  And cảnh báo nếu thông tin phối hợp trước đó dựa trên dữ liệu cũ

Scenario: Người dùng không có quyền
  Given nhân viên không được cấp quyền xem dữ liệu Batch/hạn sử dụng
  When nhân viên truy cập danh sách
  Then hệ thống từ chối truy cập hoặc che phạm vi không được phép
  And không tiết lộ dữ liệu tồn kho nội bộ
```

### US-AI-02 — Customer Support Assistant

**Actor:** Customer Service (`ACT-11`)
**Ưu tiên:** Could Have
**Phát hành:** Sau Giai đoạn 2 — đề xuất

> Là CSKH, tôi muốn được hệ thống AI hỗ trợ tổng hợp thông tin liên quan đến câu hỏi của khách (Draft reply, tra cứu đơn) để trả lời nhanh hơn.

**Giá trị nghiệp vụ:** Giảm thời gian đọc nhiều nguồn và soạn câu trả lời, trong khi quyết định gửi và trách nhiệm nghiệp vụ vẫn thuộc nhân viên.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tổng hợp ngữ cảnh Ticket có căn cứ
  Given CSKH có quyền xem Ticket và các nguồn dữ liệu liên quan
  When CSKH yêu cầu AI tổng hợp
  Then hệ thống tạo bản tóm tắt dựa trên dữ liệu được phép
  And chỉ ra nguồn/mốc dữ liệu để CSKH kiểm tra trước khi sử dụng

Scenario: Tạo Draft Reply
  Given Ticket có đủ ngữ cảnh và CSKH yêu cầu gợi ý phản hồi
  When AI tạo Draft Reply
  Then bản nháp được đánh dấu rõ là nội dung AI chưa gửi
  And CSKH phải xem xét, chỉnh sửa nếu cần và chủ động xác nhận trước khi gửi

Scenario: Thiếu hoặc mâu thuẫn dữ liệu
  Given nguồn Order, Payment hoặc Shipment thiếu hay có thông tin mâu thuẫn
  When AI tổng hợp hoặc soạn nháp
  Then kết quả nêu rõ phần chưa xác định
  And không tự khẳng định trạng thái hay cam kết kết quả nghiệp vụ không có căn cứ

Scenario: AI không được tự thực hiện hành động nghiệp vụ
  Given AI đề xuất hủy đơn, hoàn tiền, đổi trả hoặc thay đổi Ticket
  When đề xuất được hiển thị
  Then hệ thống không tự thực hiện hành động
  And yêu cầu actor có quyền dùng quy trình nghiệp vụ tương ứng

Scenario: Dữ liệu vượt ngoài quyền CSKH
  Given nguồn dữ liệu chứa trường mà CSKH không được phép xem
  When hệ thống chuẩn bị ngữ cảnh cho AI
  Then trường đó không được cung cấp hoặc hiển thị qua kết quả AI
  And AI không được dùng để vượt kiểm soát quyền trực tiếp

Scenario: AI Provider không khả dụng
  Given dịch vụ AI lỗi, hết thời gian hoặc bị tắt
  When CSKH yêu cầu hỗ trợ AI
  Then hệ thống thông báo tính năng tạm không khả dụng
  And quy trình xử lý Ticket thủ công vẫn tiếp tục được
```

### US-AI-06 — Internal Knowledge Assistant

**Actor:** Nhân viên nội bộ/Customer Service được cấp quyền
**Ưu tiên:** Could Have
**Phát hành:** Sau Giai đoạn 2 — đề xuất

> Là nhân viên nội bộ/CSKH, tôi muốn tìm kiếm thông tin nghiệp vụ/sản phẩm bằng AI trong kho tri thức của doanh nghiệp để giải quyết công việc.

**Giá trị nghiệp vụ:** Nhân viên tìm nhanh chính sách và thông tin sản phẩm có căn cứ, giảm trả lời không nhất quán hoặc dùng tài liệu đã hết hiệu lực.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tìm câu trả lời từ kho tri thức được phép
  Given nhân viên có quyền truy cập các tài liệu liên quan
  When nhân viên đặt câu hỏi nghiệp vụ hoặc sản phẩm
  Then AI trả lời dựa trên nội dung trong phạm vi quyền
  And cung cấp nguồn, phiên bản hoặc ngày hiệu lực để người dùng kiểm tra

Scenario: Tài liệu có nhiều phiên bản
  Given kho tri thức có chính sách cũ và mới
  When AI trả lời câu hỏi
  Then hệ thống ưu tiên phiên bản đang hiệu lực theo metadata được quản lý
  And cảnh báo nếu không xác định được phiên bản áp dụng

Scenario: Không tìm thấy căn cứ đáng tin cậy
  Given kho tri thức không có nội dung đủ hỗ trợ câu trả lời
  When nhân viên đặt câu hỏi
  Then AI nêu rõ không tìm thấy căn cứ phù hợp
  And không bịa chính sách, giá, cam kết hoặc quy trình

Scenario: Tài liệu ngoài quyền người dùng
  Given một nguồn tri thức không thuộc quyền của nhân viên
  When nhân viên tìm kiếm trực tiếp hoặc qua AI
  Then nguồn đó không được dùng để tạo kết quả cho người dùng
  And không tiết lộ tiêu đề, trích đoạn hoặc sự tồn tại nếu chính sách cấm

Scenario: Câu trả lời AI không thay đổi dữ liệu
  Given AI đưa ra hướng dẫn hoặc đề xuất
  When nhân viên xem kết quả
  Then kết quả chỉ mang tính hỗ trợ
  And không tự cập nhật Ticket, Product, Order, chính sách hay dữ liệu kinh doanh quan trọng
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-CS-01` | Cho Registered Customer và Guest được phép gửi yêu cầu với chủ đề, nội dung, thông tin liên hệ và ngữ cảnh hợp lệ. | `US-CS-01` |
| `FR-CS-02` | Tạo mã Ticket duy nhất, trạng thái ban đầu và xác nhận tiếp nhận; chống tạo trùng khi cùng yêu cầu được gửi lại. | `US-CS-01` |
| `FR-CS-03` | Chỉ liên kết tài khoản/Order sau khi xác minh đủ quyền sở hữu; không tiết lộ Order qua dữ liệu dễ đoán. | `US-CS-01`, `BR-AUTH-04` |
| `FR-CS-04` | Cho khách đã xác thực xem trạng thái và trao đổi công khai của Ticket thuộc quyền nhưng không thấy ghi chú nội bộ. | `US-CS-01`, `US-CS-03` |
| `FR-CS-05` | Cung cấp hàng đợi với ưu tiên, SLA, trạng thái, chủ đề, kênh, người phụ trách và bộ lọc trong phạm vi quyền. | `US-CS-02` |
| `FR-CS-06` | Cho nhận, phân công và chuyển Ticket; xử lý xung đột nhận đồng thời và bảo toàn lịch sử/SLA. | `US-CS-02` |
| `FR-CS-07` | Duy trì timeline đa kênh với nguồn gửi, actor, kênh, thời điểm và trạng thái gửi của từng trao đổi. | `US-CS-03` |
| `FR-CS-08` | Lưu phản hồi trước khi gửi ra kênh ngoài, theo dõi kết quả và cho thử lại có kiểm soát mà không nhân đôi. | `US-CS-03`, `EPIC 28` |
| `FR-CS-09` | Quản lý ghi chú nội bộ tách biệt hoàn toàn với nội dung được phép hiển thị/gửi cho khách. | `US-CS-03` |
| `FR-CS-10` | Đưa trao đổi chưa xác định an toàn được Ticket vào hàng đợi đối chiếu thay vì tự liên kết sai. | `US-CS-03` |
| `FR-CS-11` | Hiển thị ngữ cảnh Order, Payment, Shipment và lịch sử mua từ nguồn tương ứng kèm thời điểm cập nhật. | `US-CS-04`, `BR-ORDER-01`, `BR-ORDER-03` |
| `FR-CS-12` | Giới hạn workspace CSKH ở quyền xem/khởi tạo luồng phù hợp; không cho sửa dữ liệu nguồn chỉ nhờ quyền Ticket. | `US-CS-04`, `BR-AUTH-04` |
| `FR-CS-13` | Che dữ liệu thanh toán, liên hệ, địa chỉ và thông tin nhạy cảm không cần thiết cho mục đích hỗ trợ. | `US-CS-04` |
| `FR-CS-14` | Chỉ cho xem Packing Video đúng Order/Shipment khi có quyền và không cho sửa bằng chứng gốc. | `US-CS-04`, `BR-PACK-04`, `BR-PACK-05` |
| `FR-CS-15` | Quản lý các mốc SLA theo loại Ticket/ưu tiên/kênh và nêu rõ quy tắc với trạng thái chờ khách/chờ nội bộ. | `US-CS-02`, `US-CS-05` |
| `FR-CS-16` | Tạo cảnh báo gần quá hạn/quá hạn và chuyển cấp qua EPIC 28; chống tạo cảnh báo trùng cho cùng mốc. | `US-CS-05`, `EPIC 28` |
| `FR-CS-17` | Dừng tạo cảnh báo mới khi mốc đã hoàn tất nhưng vẫn bảo toàn lịch sử cảnh báo và vi phạm SLA. | `US-CS-05` |
| `FR-CS-18` | Nhận danh sách Batch sắp hết hạn từ EPIC 09, hiển thị dữ liệu nguồn và thời điểm cập nhật cho CSKH/Sales có quyền. | `US-CS-06`, `BR-BATCH-03` |
| `FR-CS-19` | Cho ghi nhận người phụ trách/ghi chú/trạng thái phối hợp tách biệt dữ liệu kho; không tự duyệt chương trình bán. | `US-CS-06`, `BR-BATCH-04` |
| `FR-CS-20` | Loại Batch đã hết hạn khỏi đề xuất bán và không cho thao tác phối hợp bỏ qua chặn bán. | `US-CS-06`, `BR-BATCH-05` |
| `FR-CS-21` | AI tổng hợp Ticket và tạo Draft Reply có dẫn nguồn trong phạm vi quyền; bắt buộc nhân viên xác nhận trước khi gửi. | `US-AI-02` |
| `FR-CS-22` | AI phải biểu thị dữ liệu thiếu/mâu thuẫn, không tự cam kết hay thực hiện thay đổi nghiệp vụ và có đường lui xử lý thủ công khi lỗi. | `US-AI-02` |
| `FR-CS-23` | Knowledge Assistant chỉ tìm trong nguồn thuộc quyền, ưu tiên nội dung hiệu lực, dẫn nguồn và thừa nhận khi không đủ căn cứ. | `US-AI-06` |
| `FR-CS-24` | Không cho AI trở thành đường vòng vượt Permission hoặc tự thay đổi dữ liệu kinh doanh quan trọng. | `US-AI-02`, `US-AI-06`, `BR-AUTH-04` |
| `FR-CS-25` | Ghi Audit cho phân công/chuyển cấp, thay đổi ưu tiên/SLA, đóng/mở lại, truy cập bằng chứng và hành động quan trọng. | `US-CS-02~05`, `BR-AUDIT-01` |
| `FR-CS-26` | Kiểm soát riêng quyền xem hàng đợi, dữ liệu khách/giao dịch, ghi chú, trả lời, chuyển Ticket, xem video, cảnh báo và dùng AI. | `US-CS-01~06`, `US-AI-02`, `US-AI-06`, `BR-AUTH-04` |

## 6. Vòng đời Ticket, SLA và quy tắc dữ liệu

### 6.1. Vòng đời Ticket đề xuất

```text
NEW → TRIAGED → ASSIGNED → IN_PROGRESS
                         ↙             ↘
          WAITING_CUSTOMER          WAITING_INTERNAL
                         ↘             ↙
                           RESOLVED → CLOSED
                               ▲          │
                               └── REOPEN ┘
```

- `NEW`: yêu cầu đã được tiếp nhận nhưng chưa hoàn tất phân loại.
- `TRIAGED`: đã có chủ đề, ưu tiên và SLA phù hợp nhưng chưa chắc đã có người xử lý.
- `ASSIGNED`: đã xác định cá nhân/hàng đợi chịu trách nhiệm.
- `IN_PROGRESS`: đang được xử lý hoặc trao đổi chủ động.
- `WAITING_CUSTOMER`: cần khách bổ sung/xác nhận; quy tắc tạm dừng SLA phải được cấu hình rõ.
- `WAITING_INTERNAL`: chờ đơn vị/quy trình nội bộ; không mặc nhiên được dừng SLA.
- `RESOLVED`: đã có kết quả xử lý nhưng còn trong thời gian có thể mở lại.
- `CLOSED`: kết thúc theo chính sách; mở lại phải giữ lịch sử và lý do.

### 6.2. Quy tắc dữ liệu Ticket và hội thoại

- Ticket phải có mã duy nhất, chủ thể khách/định danh Guest, chủ đề, mức ưu tiên, kênh khởi tạo, trạng thái, SLA, người/hàng đợi phụ trách và tham chiếu nghiệp vụ nếu có.
- Mỗi Message phải lưu Ticket, nguồn gửi, hướng gửi/nhận, kênh, thời điểm, trạng thái chuyển và nội dung/đính kèm được phép; thứ tự hiển thị không chỉ dựa vào lúc đồng bộ về hệ thống.
- Internal Note là loại dữ liệu riêng và không bao giờ được gửi ra kênh khách hàng chỉ vì thay đổi bộ lọc/giao diện.
- Ticket liên kết Case/Order/Review bằng tham chiếu; không sao chép quyền sở hữu state machine hoặc sửa dữ liệu nguồn.
- Snapshot phục vụ điều tra phải được phân biệt với dữ liệu hiện tại và ghi rõ thời điểm để tránh đưa ra cam kết dựa trên thông tin cũ.
- Đóng, mở lại, hợp nhất hoặc xác định Ticket trùng phải bảo toàn lịch sử, liên kết và lý do; không xóa cứng trao đổi đã dùng làm căn cứ.

### 6.3. SLA và ưu tiên

- SLA cần tách tối thiểu thời gian phản hồi đầu tiên và thời gian xử lý/giải quyết; không dùng một mốc “quá hạn” mơ hồ cho mọi loại Ticket.
- Ưu tiên phải được tính hoặc chọn theo quy tắc có thể giải thích dựa trên loại vấn đề, mức ảnh hưởng, kênh và thời gian chờ; thay đổi thủ công cần lý do nếu chính sách yêu cầu.
- Việc chuyển hàng đợi/nhân viên không tự đặt lại SLA trừ khi chính sách được phê duyệt quy định rõ.
- Cảnh báo và Ticket là hai thực thể khác nhau: đóng thông báo không đồng nghĩa Ticket đã giải quyết.
- Đồng hồ SLA phải dùng múi giờ và lịch làm việc được cấu hình, đồng thời giữ mốc thời gian tuyệt đối để truy vết.

### 6.4. An toàn AI và dữ liệu nhạy cảm

- AI chỉ nhận dữ liệu nằm trong quyền của người yêu cầu và tối thiểu cần thiết cho tác vụ; kết quả không được làm lộ dữ liệu nguồn bị che.
- Summary/Draft phải có khả năng đối chiếu nguồn. Nội dung AI không được coi là trạng thái giao dịch hoặc quyết định nghiệp vụ chính thức.
- Draft Reply không được tự gửi; mọi cam kết với khách phải do actor có trách nhiệm xem xét và xác nhận.
- Knowledge source cần có owner, trạng thái công khai nội bộ, phiên bản/ngày hiệu lực và phạm vi quyền trước khi được AI sử dụng.
- Lỗi hoặc việc tắt AI không được chặn nhân viên tiếp tục xử lý Ticket bằng quy trình thủ công.

## 7. Traceability

| User Story | Business Rule | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-CS-01` | `BR-AUTH-04` | EPIC 08, 28 | Khách gửi được yêu cầu hợp lệ, nhận Ticket duy nhất; không liên kết/xem Order ngoài quyền. |
| `US-CS-02` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 22, 23 | Hàng đợi đúng ưu tiên/SLA; phân công và chuyển Ticket không xung đột, không mất lịch sử. |
| `US-CS-03` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 28 | Timeline giữ đúng nguồn/kênh/trạng thái gửi; Internal Note không bị gửi cho khách. |
| `US-CS-04` | `BR-ORDER-01`, `BR-ORDER-03`, `BR-PACK-04`, `BR-PACK-05` | EPIC 07, 08, 10, 11, 14 | CSKH thấy ngữ cảnh hiện tại trong quyền nhưng không sửa nguồn hoặc xem video trái phép. |
| `US-CS-05` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 23, 28 | Cảnh báo đúng mốc, không trùng, chuyển cấp đúng người và không làm lộ Ticket. |
| `US-CS-06` | `BR-BATCH-03`, `BR-BATCH-04`, `BR-BATCH-05` | EPIC 09, 17, 21 | Danh sách lấy đúng nguồn; phối hợp không sửa tồn và không đề xuất bán Batch hết hạn. |
| `US-AI-02` | `BR-AUTH-04` | EPIC 07, 08, 11, EXT-08 | Summary/Draft có căn cứ, không vượt quyền, không tự gửi hoặc thay đổi nghiệp vụ; lỗi AI có fallback. |
| `US-AI-06` | `BR-AUTH-04` | EPIC 04, 20, 22, EXT-08 | Tìm đúng nguồn hiệu lực trong quyền, có dẫn nguồn và không bịa khi thiếu căn cứ. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Guest có được tạo Ticket không; phương thức xác minh Email/SĐT/Order và cách Guest theo dõi Ticket an toàn.
- Danh mục chủ đề, mức ưu tiên, quy tắc tự phân loại, hàng đợi và ma trận phân công/escalation.
- Bộ trạng thái Ticket chính thức, điều kiện chuyển trạng thái, thời gian mở lại và quyền đóng/mở lại.
- SLA phản hồi đầu tiên/giải quyết theo chủ đề, ưu tiên, kênh; lịch làm việc, ngày nghỉ và quy tắc dừng đồng hồ.
- Các kênh Phase 2 thực sự hỗ trợ hai chiều; quy tắc định danh khách, hợp nhất hội thoại và xử lý tin đến không nhận diện được.
- Điều kiện chống trùng/hợp nhất Ticket và ai có quyền thực hiện; cách giữ mã tham chiếu cũ.
- Dữ liệu Order/Payment/Shipment/lịch sử mua nào CSKH được xem; chính sách che PII và ghi Audit khi xem dữ liệu nhạy cảm.
- Loại tệp đính kèm, giới hạn lưu trữ, kiểm tra an toàn và thời hạn lưu hội thoại/Ticket theo chính sách pháp lý.
- Ngưỡng cảnh báo sắp hết hạn và trách nhiệm giữa Supply, Sales, Marketing, CSKH; EPIC nào sở hữu chương trình bán được tạo ra.
- Hai mốc phát hành AI; phạm vi nguồn tri thức, owner/quy trình duyệt tài liệu và tiêu chí đánh giá chất lượng câu trả lời.
- Dữ liệu được phép gửi tới External AI Provider, yêu cầu ẩn danh/lưu giữ, lựa chọn opt-out và quy trình ứng phó sự cố.
- Hành động nào bắt buộc Audit; thời hạn lưu Ticket/Message/Internal Note/AI prompt-output và quyền xuất dữ liệu.

## 9. UI/UX Reference

- Form khách gửi yêu cầu cần chỉ rõ trường bắt buộc, Order có thể liên kết, tiến trình gửi và mã Ticket sau khi tiếp nhận.
- Customer Portal cần tách trạng thái Ticket, timeline công khai và hành động bổ sung thông tin; không hiển thị Internal Note.
- Agent Workspace nên bố trí hàng đợi/ưu tiên/SLA, hội thoại, thông tin khách và ngữ cảnh giao dịch trên cùng luồng nhưng phân vùng theo quyền.
- Timeline cần phân biệt khách, nhân viên, hệ thống, Internal Note, kênh và trạng thái gửi; lỗi gửi phải nhìn thấy và có thao tác thử lại an toàn.
- Cảnh báo SLA cần thể hiện mốc nào sắp/quá hạn, người phụ trách và hành động tiếp theo; đóng cảnh báo không được gây hiểu nhầm là đóng Ticket.
- Danh sách hàng sắp hết hạn phải thể hiện nguồn EPIC 09, Batch/HSD, thời điểm cập nhật và trạng thái không được bán.
- AI Summary/Draft phải có nhãn AI, nguồn tham chiếu, cảnh báo dữ liệu thiếu và nút xác nhận riêng; không dùng giao diện khiến người dùng tưởng nội dung đã gửi.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

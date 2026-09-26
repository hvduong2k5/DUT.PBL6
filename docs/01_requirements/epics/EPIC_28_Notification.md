# EPIC 28 — Omnichannel Notification System

## 1. Mục tiêu Epic

Epic này cung cấp hạ tầng thông báo cắt ngang hệ thống qua Email, SMS/Zalo ZNS, Push trên Mobile/Web và Internal Alert. Mỗi thông báo phải xuất phát từ một sự kiện nghiệp vụ hợp lệ, đến đúng đối tượng, dùng đúng kênh/nội dung và có trạng thái gửi truy vết được.

EPIC 28 sở hữu Notification Request, template/version, lựa chọn kênh, preference/consent, delivery attempt và trạng thái nhận từ Provider. Epic nguồn sở hữu điều kiện phát sinh, dữ liệu nghiệp vụ và ý nghĩa của thông báo. Việc gửi thất bại không được hoàn tác Order, Payment, Shipment, Ticket hoặc nghiệp vụ nguồn đã hoàn tất.

Trạng thái “Provider accepted/delivered” không đồng nghĩa người nhận đã đọc hoặc hành động. Retry và callback lặp không được tạo thông báo nghiệp vụ trùng ngoài ý muốn.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **MVP — ưu tiên 2nd** | `US-NOTI-01` | Gửi Email OTP, xác nhận Order, hóa đơn và thông báo quan trọng. |
| **Giai đoạn 2 — Should Have** | `US-NOTI-02` đến `US-NOTI-04` | Gửi SMS/Zalo, Push và Internal Alert theo sự kiện/quyền phù hợp. |

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và hệ thống ngoài

| Actor / hệ thống | Vai trò |
| --- | --- |
| Guest Customer (`ACT-01`) | Nhận thông báo giao dịch theo thông tin liên hệ đã cung cấp/xác minh. |
| Registered Customer (`ACT-02`) | Nhận thông báo giao dịch, Push/Marketing theo preference và consent. |
| Nhân viên nội bộ | Nhận Internal Alert trong Role, đơn vị và phạm vi dữ liệu được cấp. |
| Notification Provider (`EXT-06`) | Tiếp nhận yêu cầu gửi Email/SMS/Push/Zalo và trả trạng thái khi tích hợp. |
| Epic/module nguồn | Quyết định sự kiện nào cần thông báo, audience nghiệp vụ và dữ liệu tham chiếu. |

### 3.2. Business Rules và NFR áp dụng

Product Backlog chưa định nghĩa nhóm `BR-NOTI`; Epic không tự tạo mã Business Rule mới.

| Mã | Ảnh hưởng đến Epic |
| --- | --- |
| `BR-AUTH-01`, `BR-AUTH-02` | Guest vẫn nhận xác nhận Order; OTP/khôi phục phải gắn đúng định danh và không làm lộ tài khoản tồn tại. |
| `BR-AUTH-03`, `BR-AUTH-04` | Internal Alert chỉ đến nhân viên có System Access, Role/Permission và phạm vi phù hợp. |
| `BR-ORDER-05` | Thông báo thay đổi trạng thái phải tham chiếu đúng transition có thể truy vết. |
| `BR-BATCH-03`, `BR-BATCH-05` | Cảnh báo gần hết hạn đến đúng người; không dùng thông báo để cho phép bán hàng đã hết hạn. |
| `BR-AUDIT-01` | Thay đổi template/routing, gửi lại thủ công và truy cập payload nhạy cảm phải Audit theo catalog. |
| `NFR-02` | Push/Web notification và deep link phải tương thích phạm vi nền tảng Mobile/Web. |
| `NFR-03` | Hạ tầng thông báo cần chịu được giai đoạn cao điểm; suy giảm kênh phải được nhận biết. |
| `NFR-04`, `NFR-05`, `NFR-08` | Bảo vệ OTP, token thiết bị, PII, nội dung và giới hạn truy cập đúng mục đích. |
| `NFR-10` | Hỗ trợ tải lớn, giới hạn tốc độ và xử lý nền mà không chặn giao dịch nguồn. |
| `NFR-11` | Theo dõi queue/backlog, latency, lỗi, retry và tình trạng Provider. |

### 3.3. Phụ thuộc và ranh giới

- `EPIC 01`: phát sinh OTP/xác minh/khôi phục; EPIC 28 chỉ chuyển thông điệp, không xác thực OTP.
- `EPIC 07`, `EPIC 08`, `EPIC 11`: phát sinh trạng thái Payment, Order, Shipment; lỗi gửi không làm lùi trạng thái.
- `EPIC 09`, `EPIC 16`, `EPIC 23`: sở hữu điều kiện cảnh báo Batch, Ticket SLA và Audit integrity.
- `EPIC 17`, `EPIC 21`: sở hữu Promotion/Campaign/audience; EPIC 28 thực thi gửi theo consent và policy.
- `EPIC 24`: cung cấp artifact/trạng thái hóa đơn; Notification không tự phát hành hóa đơn.
- `EPIC 26`: cung cấp audience buyer/recipient và hide-price policy cho nội dung quà tặng.
- `EPIC 22`: cấp quyền quản trị template/routing và phạm vi nhận Internal Alert.
- `EXT-06`: nhà cung cấp kênh; trạng thái ngoài phải được xác minh và tách khỏi trạng thái nội bộ.

## 4. User Stories chi tiết

### US-NOTI-01 — Email Notification

**Actor:** Guest Customer (`ACT-01`), Registered Customer (`ACT-02`)

**Ưu tiên:** Must Have — 2nd — **Phát hành:** MVP

> Là khách hàng, tôi muốn nhận Email xác nhận đơn hàng, mã OTP, hóa đơn và các thông báo quan trọng để nắm bắt thông tin kịp thời.

```gherkin
Scenario: Gửi Email xác nhận Order
  Given Order đã được tạo hợp lệ và có địa chỉ Email được phép sử dụng
  When EPIC 08 phát sinh sự kiện xác nhận
  Then hệ thống tạo đúng một Notification cho sự kiện/recipient
  And Email chứa mã Order và thông tin cần thiết theo template hiện hành

Scenario: Gửi OTP an toàn
  Given EPIC 01 phát sinh OTP còn hiệu lực cho mục đích xác định
  When tạo Email OTP
  Then nội dung chỉ dùng OTP trong đúng ngữ cảnh và không ghi bí mật vào log/metadata dễ đọc
  And thời hạn/hướng dẫn an toàn được hiển thị theo policy

Scenario: Gửi hóa đơn
  Given EPIC 24 có artifact hóa đơn hợp lệ và người nhận được phép
  When yêu cầu gửi Email hóa đơn
  Then hệ thống gửi link/attachment được bảo vệ theo policy
  And không gửi artifact của Order/pháp nhân khác

Scenario: Email không hợp lệ hoặc chưa được phép
  Given địa chỉ thiếu, sai định dạng hoặc không được dùng cho loại thông báo
  When xử lý yêu cầu
  Then hệ thống không gửi
  And ghi trạng thái/lý do để Epic nguồn hoặc người có quyền xử lý

Scenario: Provider lỗi hoặc timeout
  Given EXT-06 không xác nhận kết quả
  When gửi Email
  Then attempt chuyển trạng thái retryable/unknown/failed phù hợp
  And nghiệp vụ nguồn không bị rollback

Scenario: Callback hoặc request lặp
  Given cùng idempotency key/sự kiện đã được xử lý
  When nhận lại request hoặc callback
  Then hệ thống không tạo Email nghiệp vụ trùng
  And cập nhật trạng thái attempt nhất quán

Scenario: Email bị bounce hoặc complaint
  Given Provider báo địa chỉ không nhận được hoặc khiếu nại
  When xử lý callback hợp lệ
  Then hệ thống cập nhật delivery status và suppression/preference theo policy
  And không tự vô hiệu thông báo bảo mật bắt buộc ngoài quy tắc đã chốt

Scenario: Nội dung dành cho buyer và recipient
  Given Gift Order có audience và hide-price policy
  When tạo Email tương ứng
  Then hệ thống dùng đúng template/dữ liệu cho từng audience
  And không gửi nhầm giá hoặc thông tin buyer cho recipient
```

### US-NOTI-02 — SMS / Zalo ZNS

**Actor:** Guest Customer (`ACT-01`), Registered Customer (`ACT-02`)

**Ưu tiên:** Should Have — **Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn nhận SMS hoặc Zalo cập nhật trạng thái giao hàng để theo dõi hành trình đơn hàng.

```gherkin
Scenario: Gửi cập nhật Shipment quan trọng
  Given EPIC 11 phát sinh mốc được cấu hình và số điện thoại hợp lệ
  When routing chọn SMS hoặc Zalo
  Then hệ thống gửi nội dung ngắn gọn với Order/Shipment reference phù hợp
  And không chứa dữ liệu Payment/PII không cần thiết

Scenario: Chọn kênh theo preference và khả năng
  Given recipient có preference hợp lệ và các kênh khả dụng
  When xử lý Notification
  Then hệ thống chọn kênh theo routing policy/version
  And lưu căn cứ lựa chọn

Scenario: Kênh ưu tiên không khả dụng
  Given Zalo/SMS ưu tiên bị từ chối hoặc không hỗ trợ recipient
  When policy cho phép fallback
  Then hệ thống thử kênh thay thế theo thứ tự và giới hạn
  And không gửi đồng thời trùng nội dung ngoài chủ đích

Scenario: Số điện thoại sai hoặc đổi chủ
  Given contact không còn hợp lệ hoặc không xác minh được
  When gửi thông báo
  Then hệ thống không tiết lộ dữ liệu Order chi tiết
  And cập nhật lỗi để xử lý contact

Scenario: Nhiều cập nhật giống nhau
  Given cùng Shipment transition được xử lý lại
  When tạo request
  Then hệ thống deduplicate theo event/recipient/template purpose

Scenario: Trạng thái thay đổi nhanh
  Given nhiều mốc Shipment đến gần nhau hoặc out-of-order
  When routing thông báo
  Then hệ thống áp dụng ordering/coalescing policy đã duyệt
  And không gửi nội dung cũ gây hiểu nhầm sau trạng thái mới hơn

Scenario: Opt-out Marketing không chặn giao dịch bắt buộc
  Given Customer đã từ chối Marketing
  When có cập nhật giao hàng cần thiết
  Then hệ thống xử lý theo classification giao dịch đã duyệt
  And không kèm nội dung quảng cáo nếu thiếu consent

Scenario: Provider báo delivered
  Given EXT-06 trả trạng thái đã giao
  When cập nhật attempt
  Then hệ thống ghi delivered_at và reference
  And không diễn giải là khách đã đọc
```

### US-NOTI-03 — In-app Push Notification

**Actor:** Registered Customer (`ACT-02`)

**Ưu tiên:** Should Have — **Phát hành:** Giai đoạn 2

> Là khách hàng, tôi muốn nhận Push trên Mobile App/Web về trạng thái đơn hàng và khuyến mãi.

```gherkin
Scenario: Push trạng thái Order
  Given Customer có thiết bị/browser đăng ký hợp lệ và sở hữu Order
  When có mốc Order được cấu hình
  Then hệ thống gửi Push tới endpoint được phép
  And deep link trỏ tới resource phù hợp

Scenario: Deep link vẫn kiểm tra quyền
  Given người dùng bấm Push trên thiết bị
  When mở Order hoặc nội dung đích
  Then ứng dụng yêu cầu xác thực/quyền hiện tại
  And không coi possession của notification là quyền truy cập

Scenario: Nhiều thiết bị
  Given Customer đăng ký nhiều endpoint còn hiệu lực
  When gửi Push
  Then routing áp dụng policy đa thiết bị và dedup hiển thị phù hợp

Scenario: Token hết hạn hoặc bị thu hồi
  Given Provider báo endpoint không hợp lệ
  When xử lý kết quả
  Then hệ thống vô hiệu endpoint tương ứng
  And không vô hiệu nhầm các thiết bị khác

Scenario: Push khuyến mãi theo consent
  Given Customer có consent và preference phù hợp
  When Campaign yêu cầu gửi Push
  Then hệ thống gửi trong phạm vi audience/frequency policy
  And lưu campaign/purpose reference

Scenario: Không có consent Marketing
  Given Customer từ chối hoặc chưa cấp consent
  When nhận request Push khuyến mãi
  Then hệ thống suppress request
  And không fallback sang kênh Marketing khác

Scenario: Quiet hours hoặc frequency cap
  Given notification không khẩn nằm trong giờ yên lặng hoặc vượt tần suất
  When lập lịch gửi
  Then hệ thống trì hoãn/suppress theo policy và timezone

Scenario: Push không đến nhưng dữ liệu vẫn xem được
  Given Push thất bại
  When Customer mở Notification Center hoặc Order
  Then dữ liệu nghiệp vụ hiện tại vẫn khả dụng theo quyền
  And lỗi Push không làm mất sự kiện nguồn
```

### US-NOTI-04 — Internal Alert

**Actor:** Nhân viên nội bộ được cấp quyền

**Ưu tiên:** Should Have — **Phát hành:** Giai đoạn 2

> Là nhân viên nội bộ, tôi muốn nhận cảnh báo in-app khi có đơn khẩn, Ticket quá hạn SLA hoặc hàng hóa sắp hết hạn.

```gherkin
Scenario: Cảnh báo đơn khẩn
  Given EPIC 08 xác định Order đạt điều kiện ưu tiên
  When tạo Internal Alert
  Then hệ thống route đến đúng Role/hàng đợi/phạm vi
  And alert tham chiếu đúng Order và căn cứ ưu tiên

Scenario: Cảnh báo Ticket quá hạn
  Given EPIC 16 phát sinh mốc SLA hợp lệ
  When tạo alert
  Then hệ thống gửi tới owner/hàng đợi/escalation target theo policy
  And không tự thay đổi Ticket status

Scenario: Cảnh báo hàng sắp hết hạn
  Given EPIC 09 xác định Batch đạt ngưỡng
  When tạo alert
  Then hệ thống route tới Supply/Sales/CSKH được cấu hình
  And không biến Batch hết hạn thành hàng được phép bán

Scenario: Không tạo alert trùng liên tục
  Given alert cùng source, type, threshold và audience đang active
  When nhận lại sự kiện không có thay đổi đáng kể
  Then hệ thống cập nhật last observed hoặc bỏ qua theo policy
  And không tạo nhiều alert giống nhau

Scenario: Đọc/xác nhận không đồng nghĩa giải quyết
  Given nhân viên mở hoặc acknowledge alert
  When cập nhật trạng thái cá nhân
  Then hệ thống ghi read/acknowledged riêng
  And không đánh dấu nghiệp vụ nguồn resolved

Scenario: Nguồn được giải quyết
  Given Order/Ticket/Batch không còn điều kiện cảnh báo
  When nhận sự kiện giải quyết
  Then alert chuyển resolved/closed theo policy
  And giữ lịch sử

Scenario: Người dùng mất quyền
  Given nhân viên không còn quyền với resource
  When xem danh sách hoặc mở deep link
  Then hệ thống không hiển thị nội dung nhạy cảm
  And quyền cũ không được giữ chỉ vì đã nhận alert

Scenario: Escalation quá hạn
  Given alert chưa acknowledged/resolved sau ngưỡng
  When đánh giá escalation
  Then hệ thống route tới cấp tiếp theo theo policy
  And deduplicate từng escalation stage
```

## 5. Functional Requirements

| Mã | Yêu cầu | Truy vết |
| --- | --- | --- |
| `FR-NOTI-01` | Tiếp nhận Notification Request với source event, purpose, audience, priority và idempotency key. | `US-NOTI-01`–`04` |
| `FR-NOTI-02` | Chống tạo/gửi trùng theo event, recipient, purpose và policy. | `US-NOTI-01`–`04` |
| `FR-NOTI-03` | Quản lý template/version, locale, audience và biến dữ liệu bắt buộc. | `US-NOTI-01`–`04` |
| `FR-NOTI-04` | Validate/render nội dung an toàn, không lộ secret/PII ngoài mục đích. | `US-NOTI-01`–`04` |
| `FR-NOTI-05` | Phân loại security/transactional/operational/marketing và áp dụng policy riêng. | `US-NOTI-01`–`04` |
| `FR-NOTI-06` | Gửi Email xác nhận Order từ sự kiện hợp lệ. | `US-NOTI-01` |
| `FR-NOTI-07` | Gửi OTP đúng purpose/expiry và không ghi bí mật dễ đọc. | `US-NOTI-01` |
| `FR-NOTI-08` | Gửi link/attachment hóa đơn đúng Order/recipient với bảo vệ phù hợp. | `US-NOTI-01` |
| `FR-NOTI-09` | Theo dõi bounce/complaint/suppression mà không tự sai lệch policy bảo mật. | `US-NOTI-01` |
| `FR-NOTI-10` | Tách nội dung buyer/recipient và tuân thủ hide-price policy. | `US-NOTI-01`, `EPIC 26` |
| `FR-NOTI-11` | Gửi SMS/Zalo cho mốc Shipment với nội dung tối thiểu. | `US-NOTI-02` |
| `FR-NOTI-12` | Chọn/fallback kênh theo preference, capability và routing version. | `US-NOTI-02` |
| `FR-NOTI-13` | Xử lý contact sai, ordering/coalescing và cập nhật Shipment lặp. | `US-NOTI-02` |
| `FR-NOTI-14` | Phân biệt delivered với read và lưu Provider reference/timestamp. | `US-NOTI-02`, `03` |
| `FR-NOTI-15` | Quản lý endpoint Push theo Customer, thiết bị/browser, trạng thái và consent. | `US-NOTI-03` |
| `FR-NOTI-16` | Gửi Push Order/Promotion đúng scope; deep link luôn kiểm tra quyền hiện tại. | `US-NOTI-03` |
| `FR-NOTI-17` | Hỗ trợ đa thiết bị và vô hiệu riêng endpoint hỏng/thu hồi. | `US-NOTI-03` |
| `FR-NOTI-18` | Áp dụng consent, quiet hours, timezone và frequency cap cho Marketing. | `US-NOTI-03` |
| `FR-NOTI-19` | Duy trì Notification Center độc lập với kết quả Push ngoài. | `US-NOTI-03` |
| `FR-NOTI-20` | Route Internal Alert theo Role, hàng đợi, đơn vị và resource scope hiện tại. | `US-NOTI-04` |
| `FR-NOTI-21` | Hỗ trợ Order urgency, Ticket SLA, Batch expiry và loại alert mở rộng có source reference. | `US-NOTI-04` |
| `FR-NOTI-22` | Deduplicate alert theo source/type/threshold/audience và escalation stage. | `US-NOTI-04` |
| `FR-NOTI-23` | Tách delivered/read/acknowledged/resolved và không sửa trạng thái nguồn. | `US-NOTI-04` |
| `FR-NOTI-24` | Re-evaluate quyền khi liệt kê/mở alert hoặc deep link. | `US-NOTI-03`, `04` |
| `FR-NOTI-25` | Quản lý delivery attempt, retry, backoff, timeout, unknown và terminal failure. | `US-NOTI-01`–`04` |
| `FR-NOTI-26` | Xác minh, deduplicate và xử lý callback Provider out-of-order. | `US-NOTI-01`–`03` |
| `FR-NOTI-27` | Không rollback nghiệp vụ nguồn khi gửi thất bại; cung cấp trạng thái để xử lý. | `US-NOTI-01`–`04` |
| `FR-NOTI-28` | Audit thay đổi template/routing, resend thủ công và truy cập payload nhạy cảm. | `US-NOTI-01`–`04`, `EPIC 23` |
| `FR-NOTI-29` | Bảo vệ dữ liệu contact, OTP, endpoint, payload, file/link và retention theo policy. | `US-NOTI-01`–`04` |
| `FR-NOTI-30` | Quan sát backlog, latency, delivery rate, retry/failure và Provider health theo kênh. | `US-NOTI-01`–`04`, `NFR-11` |
| `FR-NOTI-31` | Áp dụng rate limit, quota và backpressure không làm mất dấu request. | `US-NOTI-01`–`04`, `NFR-10` |
| `FR-NOTI-32` | Hỗ trợ tìm kiếm trạng thái theo source/recipient/kênh trong quyền; masking dữ liệu nhạy cảm. | `US-NOTI-01`–`04` |

## 6. Mô hình dữ liệu và vòng đời

```text
Notification Request
  request_id + idempotency_key + source_type + source_id + source_event
  purpose + classification + audience + priority + locale + template_version

Notification -> Channel Delivery -> Delivery Attempt
  recipient_reference + channel + destination_masked + provider_reference
  status + failure_category + retry_count + timestamps

Request: ACCEPTED -> ROUTED -> PROCESSING -> COMPLETED | PARTIAL | FAILED | SUPPRESSED
Delivery: QUEUED -> SENDING -> ACCEPTED -> DELIVERED | FAILED | UNKNOWN | SUPPRESSED
Internal Alert: ACTIVE -> READ | ACKNOWLEDGED -> RESOLVED | CLOSED
```

- Một request có thể sinh nhiều channel delivery theo policy nhưng không được gửi trùng ngoài chủ đích.
- `ACCEPTED` chỉ là Provider nhận yêu cầu; `DELIVERED` là trạng thái kênh; `READ` chỉ có khi có bằng chứng phù hợp.
- Callback đến muộn/out-of-order không được làm trạng thái terminal quay lùi sai.
- Payload lưu tối thiểu, destination được masking; OTP/token/link bí mật không xuất hiện trong log thông thường.
- Template đã dùng phải truy vết được version; thay template không sửa nội dung lịch sử.
- Chi tiết broker/queue, nhà cung cấp, giao thức realtime và SDK thuộc Technical Design.

## 7. Traceability

| User Story | FR chính | Epic nguồn/đích |
| --- | --- | --- |
| `US-NOTI-01` | `FR-NOTI-01`–`10`, `25`–`32` | EPIC 01/08/24/26, EXT-06 |
| `US-NOTI-02` | `FR-NOTI-01`–`05`, `11`–`14`, `25`–`32` | EPIC 11, EXT-06 |
| `US-NOTI-03` | `FR-NOTI-01`–`05`, `14`–`19`, `24`–`32` | EPIC 08/17/21, EXT-06 |
| `US-NOTI-04` | `FR-NOTI-01`–`05`, `20`–`32` | EPIC 08/09/16/23 |

`FR-30 — Omnichannel Notification` trong Functional Requirements cấp cao được chi tiết hóa bởi `FR-NOTI-01` đến `FR-NOTI-32`.

## 8. Quyết định còn mở

- Event catalog: sự kiện nào phải gửi, kênh, audience, priority, SLA và owner nội dung.
- Phân loại transactional/security/operational/marketing; trường hợp bắt buộc và quyền opt-out.
- Preference/consent source, thời điểm hiệu lực, quiet hours, timezone và frequency cap.
- Thứ tự routing/fallback, số retry, backoff, timeout, quota và điều kiện dừng.
- Provider cho từng kênh, chiến lược failover và nguồn có thẩm quyền khi callback mâu thuẫn.
- Quy tắc ordering/coalescing cho nhiều trạng thái đến nhanh hoặc out-of-order.
- Template approval, locale, branding, accessibility và chính sách nội dung/link/attachment.
- OTP expiry/resend/rate limit thuộc EPIC 01 và Notification phối hợp ở mức nào.
- Internal Alert routing, acknowledgement/escalation SLA và điều kiện resolve/close.
- Retention payload/delivery history, masking, quyền resend/search/export và Audit catalog.
- SLO về acceptance/delivery latency và cách đo “delivered” khác nhau giữa các Provider.

## 9. UI/UX Reference

- Customer Notification Preferences: kênh theo classification, consent, quiet hours và trạng thái contact/endpoint.
- Notification Center: unread/read, loại, thời gian, deep link; luôn kiểm tra quyền khi mở.
- Internal Alert Center: priority, source, owner, SLA, acknowledge, resolve và escalation.
- Template/Route Administration: version, locale, preview, test, approval và Audit.
- Delivery Operations: search theo source/recipient/kênh, timeline attempts, lỗi, retry/resend có quyền.
- Mọi màn hình cần loading, empty, partial, suppressed, delayed, provider unavailable, permission denied và expired link.

Liên kết Figma sẽ được bổ sung khi thiết kế UI/UX được phê duyệt.

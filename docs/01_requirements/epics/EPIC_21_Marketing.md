# EPIC 21 — Marketing

## 1. Mục tiêu Epic

Epic này quản lý vòng đời chiến dịch Marketing từ lập kế hoạch, chuẩn bị nội dung/chương trình truyền thông, gửi duyệt và triển khai đến theo dõi KPI, đồng bộ dữ liệu hiệu quả theo kênh và so sánh chi phí với doanh thu được quy gán.

EPIC 21 sở hữu Campaign, Campaign Revision, kế hoạch kênh/nội dung, mục tiêu KPI, liên kết tài sản và dữ liệu performance theo chiến dịch. Epic không sở hữu quy trình xuất bản Article, Promotion/Coupon, hạ tầng gửi thông báo, dữ liệu giao dịch tài chính hay mô hình phân khúc khách hàng; các trách nhiệm đó thuộc EPIC 20, 17, 28, 24 và 25.

## 2. Phạm vi phát hành

| Bản phát hành | User Story | Kết quả nghiệp vụ |
| --- | --- | --- |
| **Giai đoạn 2** | `US-MKTG-01` đến `US-MKTG-07` | Marketing lập/duyệt/triển khai chiến dịch có mục tiêu và ngân sách; theo dõi KPI đa kênh; Executive đánh giá hiệu quả chi phí–doanh thu trên dữ liệu có nguồn. |

Toàn bộ User Story trong Epic có mức ưu tiên **Should Have** theo Product Backlog.

## 3. Actor, quy tắc và phụ thuộc

### 3.1. Actors và hệ thống ngoài

| Actor / hệ thống | Vai trò trong Epic |
| --- | --- |
| Marketing Staff (`ACT-14`) | Lập/sửa kế hoạch, chuẩn bị nội dung/kênh, gửi duyệt, triển khai và theo dõi KPI theo quyền. |
| Executive / Business Manager (`ACT-16`) | Duyệt/từ chối kế hoạch và xem hiệu quả chi phí–doanh thu ở phạm vi được cấp. |
| Sales Manager (`ACT-12`) | Phối hợp sản phẩm/Promotion/doanh số hoặc duyệt nếu được ủy quyền; không mặc nhiên có quyền Executive. |
| Content Manager (`ACT-13`) | Tạo/duyệt/xuất bản Article và Content Media qua EPIC 20 khi chiến dịch yêu cầu. |
| Accountant / Finance Staff (`ACT-19`) | Cung cấp/đối soát chi phí và doanh thu nội bộ qua EPIC 24. |
| Analytics Platform (`EXT-07`) | Cung cấp traffic, conversion và dữ liệu campaign bên ngoài. |
| Media / Object Storage (`EXT-09`) | Lưu Marketing Video/media theo quyền; Campaign chỉ giữ tham chiếu. |
| Social / Marketing Platform (`EXT-12`) | Nhận nội dung/cấu hình và cung cấp performance của kênh khi tích hợp. |

### 3.2. Business Rules và NFR áp dụng

| Mã | Quy tắc | Ảnh hưởng đến Epic |
| --- | --- | --- |
| `BR-AUTH-03` | Tài khoản nội bộ phải có Role phù hợp trước khi truy cập chức năng quản trị. | Marketing Staff, Executive và actor phối hợp phải có Role hợp lệ. |
| `BR-AUTH-04` | Nhân viên chỉ thực hiện thao tác thuộc quyền được cấp. | Quyền lập, gửi duyệt, duyệt, triển khai, sửa ngân sách và xem chi phí/doanh thu phải tách biệt. |
| `BR-PROD-04` | Thông tin Product công khai phải được quản lý/phê duyệt. | Campaign không được phát hành tuyên bố Product/OCOP chưa được duyệt qua Epic nguồn. |
| `BR-ORDER-03` | Order chỉ chuyển Paid khi giao dịch được xác nhận hợp lệ. | Doanh thu quy gán không được lấy từ sự kiện Purchase bên ngoài như bằng chứng Payment nội bộ. |
| `BR-AUDIT-01` | Các thao tác quan trọng phải được ghi Audit Log. | Duyệt/từ chối, thay ngân sách/KPI, triển khai/dừng và điều chỉnh attribution phải truy vết được. |
| `NFR-04` | Dữ liệu Customer/Order/Payment phải được bảo vệ. | Báo cáo chiến dịch không được làm lộ dữ liệu giao dịch hoặc khách hàng ngoài quyền. |
| `NFR-05` | Người dùng chỉ truy cập dữ liệu/chức năng thuộc quyền. | Campaign, audience, chi phí và ROI phải giới hạn theo vai trò/phạm vi. |
| `NFR-08` | Dữ liệu cá nhân chỉ được dùng theo mục đích được phép. | Nhóm đối tượng và hoạt động truyền thông phải tuân thủ mục đích/consent áp dụng. |
| `NFR-10` | Hệ thống phải xử lý tốt tải truy vấn Analytics lớn. | Dashboard cần giới hạn/phân trang/tổng hợp phù hợp; ngưỡng cụ thể thuộc NFR/Test Plan. |
| `NFR-11` | Hệ thống phải theo dõi lỗi/hiệu năng/trạng thái tích hợp ngoài. | Lần đồng bộ kênh phải có trạng thái, thời điểm, lỗi và khả năng thử lại an toàn. |

### 3.3. Phụ thuộc và ranh giới giữa Epic

- `EPIC 04 — Product, Variant & SKU`: nguồn dữ liệu Product/SKU; Campaign chỉ tham chiếu đối tượng được phép.
- `EPIC 17 — Promotion & Loyalty`: sở hữu Coupon/Combo/quyền lợi; Campaign gắn Promotion hiện có hoặc yêu cầu tạo qua đúng quy trình.
- `EPIC 20 — Content & SEO`: sở hữu Article, Content Revision, Publication và SEO; Campaign quản lý brief/lịch/liên kết nội dung.
- `EPIC 22, 23 — Administration, Audit & Security`: cung cấp Role/Permission và Audit Log.
- `EPIC 24 — Finance & Business Analytics`: nguồn chi phí/doanh thu tài chính được đối soát; EPIC 21 tính góc nhìn hiệu quả Campaign.
- `EPIC 25 — Customer Analytics & DSS`: cung cấp segment/insight nếu được phép; EPIC 21 không tự xây dựng mô hình RFM/Churn.
- `EPIC 28 — Omnichannel Notification`: thực hiện gửi thông báo Marketing và quản lý delivery; Campaign chỉ xác định audience/content/schedule được duyệt.
- `EXT-07`, `EXT-12`: cung cấp dữ liệu kênh ngoài; dữ liệu này phải phân biệt với Order/Payment nội bộ.

## 4. User Stories chi tiết

### US-MKTG-01 — Lập kế hoạch Marketing theo chiến dịch

**Actor:** Marketing Staff (`ACT-14`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Marketing Staff, tôi muốn lập kế hoạch Marketing theo chiến dịch để tổ chức hoạt động quảng bá.

**Giá trị nghiệp vụ:** Mỗi chiến dịch có mục tiêu, đối tượng, kênh, ngân sách, thời gian, người chịu trách nhiệm và KPI thống nhất trước khi triển khai.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo Campaign Draft hợp lệ
  Given Marketing Staff có quyền tạo chiến dịch
  When nhập tên, mục tiêu, thời gian, kênh, đối tượng, ngân sách dự kiến, owner và KPI mục tiêu hợp lệ
  Then hệ thống tạo Campaign cùng revision đầu tiên ở trạng thái `DRAFT`
  And gắn người tạo và thời điểm
  And không tự triển khai chiến dịch

Scenario: Thiếu dữ liệu kế hoạch bắt buộc
  Given Marketing Staff đang tạo hoặc gửi duyệt Campaign
  When thiếu mục tiêu, kỳ, owner, kênh hoặc dữ liệu bắt buộc khác
  Then hệ thống từ chối gửi duyệt
  And chỉ rõ trường cần hoàn thiện

Scenario: Khoảng thời gian không hợp lệ
  Given ngày kết thúc không sau ngày bắt đầu hoặc lịch kênh nằm ngoài kỳ Campaign
  When Marketing Staff lưu/gửi duyệt
  Then hệ thống từ chối dữ liệu thời gian không nhất quán
  And không tự sửa lịch

Scenario: Liên kết Product hoặc Promotion
  Given Product/Promotion tồn tại và được phép sử dụng
  When Marketing Staff gắn vào Campaign
  Then hệ thống lưu tham chiếu đúng đối tượng
  And không sao chép quyền sửa Catalog hoặc Promotion sang Campaign

Scenario: Chọn audience từ EPIC 25
  Given segment đã được cung cấp và Marketing Staff có quyền sử dụng
  When gắn segment vào Campaign
  Then hệ thống lưu tham chiếu/phiên bản segment phù hợp
  And không tự mở rộng audience ngoài mục đích được phép

Scenario: Hai người chỉnh sửa cùng revision
  Given Campaign Revision đã thay đổi sau khi một người tải dữ liệu
  When người đó lưu trên phiên bản cũ
  Then hệ thống không ghi đè âm thầm dữ liệu mới hơn
  And yêu cầu đối chiếu/tải lại

Scenario: Người dùng không có quyền tạo Campaign
  Given nhân viên không có quyền Marketing Plan
  When nhân viên tạo hoặc sửa kế hoạch
  Then hệ thống từ chối
  And không tạo thay đổi
```

### US-MKTG-02 — Tạo nội dung và chương trình truyền thông cho Campaign

**Actor:** Marketing Staff (`ACT-14`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Marketing Staff, tôi muốn tạo nội dung video, bài viết và chương trình truyền thông cho từng chiến dịch.

**Giá trị nghiệp vụ:** Mọi nội dung/tài sản/kênh được tổ chức theo Campaign, có brief và trạng thái sẵn sàng rõ ràng trước khi phân phối.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Tạo Content Brief trong Campaign
  Given Campaign đang cho phép chuẩn bị nội dung
  When Marketing Staff nhập thông điệp, audience, kênh, định dạng, deadline và owner hợp lệ
  Then hệ thống tạo Content Brief gắn đúng Campaign Revision
  And không coi Brief là Article/Media đã xuất bản

Scenario: Liên kết Article từ EPIC 20
  Given Article/Revision tồn tại và Marketing Staff có quyền xem
  When gắn Article vào kế hoạch kênh
  Then Campaign lưu đúng tham chiếu và trạng thái Publication
  And không tự xuất bản hoặc sửa Article

Scenario: Liên kết Promotion từ EPIC 17
  Given Promotion/Coupon tồn tại
  When Marketing Staff gắn chương trình vào Campaign
  Then hệ thống hiển thị trạng thái, hiệu lực và điều kiện cần thiết
  And không tự kích hoạt hoặc thay đổi hạn mức Promotion

Scenario: Tải Marketing Media hợp lệ
  Given Marketing Staff có quyền media và tệp đáp ứng chính sách
  When tải video/ảnh lên
  Then hệ thống lưu qua Media Storage và gắn đúng Campaign/Brief
  And thể hiện trạng thái xử lý, quyền sử dụng và phiên bản hiện hành

Scenario: Media không hợp lệ hoặc tải lỗi
  Given tệp sai loại/vượt giới hạn/không an toàn hoặc quá trình tải thất bại
  When hệ thống xử lý
  Then tệp không được đánh dấu sẵn sàng triển khai
  And không tạo tham chiếu hỏng trong kế hoạch kênh

Scenario: Asset thay đổi sau khi kế hoạch đã gửi duyệt
  Given Campaign Revision đang `IN_REVIEW` hoặc đã `APPROVED`
  When nội dung/Promotion/Media liên kết thay đổi đáng kể
  Then hệ thống đánh dấu kế hoạch cần rà soát hoặc tạo revision mới theo chính sách
  And không âm thầm triển khai asset khác bản đã duyệt

Scenario: Nội dung chứa dữ kiện chưa được duyệt
  Given nội dung đề cập Product/OCOP/giá/ưu đãi chưa được phép công khai
  When Campaign được gửi duyệt hoặc triển khai
  Then hệ thống cảnh báo/chặn theo chính sách
  And yêu cầu dùng dữ liệu từ Epic nguồn đã được duyệt
```

### US-MKTG-03 — Gửi kế hoạch đi duyệt

**Actor:** Marketing Staff (`ACT-14`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Marketing Staff, tôi muốn gửi kế hoạch cho người có quyền duyệt trước khi triển khai.

**Giá trị nghiệp vụ:** Kế hoạch được đóng gói thành một revision nhất quán để người duyệt đánh giá mục tiêu, chi phí, audience, kênh, nội dung và rủi ro.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Gửi Campaign Revision đủ điều kiện
  Given revision có đủ kế hoạch, KPI, ngân sách, kênh và asset bắt buộc
  When Marketing Staff gửi duyệt
  Then revision chuyển `IN_REVIEW`
  And ghi người gửi cùng thời điểm
  And không được triển khai khi chưa có quyết định hợp lệ

Scenario: Kế hoạch vượt ngưỡng ngân sách
  Given ngân sách nằm ngoài thẩm quyền hoặc yêu cầu cấp duyệt cao hơn
  When Marketing Staff gửi duyệt
  Then hệ thống định tuyến tới vai trò/cấp duyệt phù hợp
  And không tự hạ ngân sách hoặc bỏ bước duyệt

Scenario: Asset/Promotion chưa sẵn sàng
  Given nội dung, Promotion hoặc quyền sử dụng Media bắt buộc chưa hợp lệ
  When gửi duyệt
  Then hệ thống từ chối hoặc đánh dấu điều kiện còn thiếu theo chính sách
  And không trình bày Campaign như sẵn sàng triển khai

Scenario: Thu hồi kế hoạch đang chờ duyệt
  Given revision đang `IN_REVIEW` và chưa có quyết định
  When Marketing Staff có quyền thu hồi với lý do
  Then revision quay về `DRAFT` hoặc trạng thái được cấu hình
  And yêu cầu duyệt hiện tại hết hiệu lực
  And lịch sử được giữ lại

Scenario: Sửa kế hoạch sau khi gửi duyệt
  Given revision đang chờ duyệt
  When Marketing Staff cần thay đổi nội dung quan trọng
  Then hệ thống yêu cầu thu hồi/tạo revision mới
  And không sửa âm thầm dữ liệu người duyệt đang xem

Scenario: Gửi lặp cùng yêu cầu duyệt
  Given revision đã có yêu cầu duyệt đang hoạt động
  When thao tác gửi bị lặp
  Then hệ thống không tạo nhiều yêu cầu duyệt ngoài ý muốn
  And trả về trạng thái hiện tại
```

### US-MKTG-04 — Duyệt hoặc từ chối kế hoạch Marketing

**Actor:** Executive / Business Manager (`ACT-16`), Manager được ủy quyền
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Manager/Executive, tôi muốn duyệt hoặc từ chối kế hoạch Marketing.

**Giá trị nghiệp vụ:** Người có thẩm quyền kiểm soát ngân sách, audience, thông điệp, kênh và KPI trước khi doanh nghiệp cam kết triển khai.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Duyệt Campaign Revision hợp lệ
  Given revision đang `IN_REVIEW` và Executive có đủ thẩm quyền
  When Executive phê duyệt
  Then revision chuyển `APPROVED`
  And lưu người duyệt, thời điểm và phạm vi ngân sách được duyệt
  And chưa tự kích hoạt kênh nếu còn bước triển khai riêng

Scenario: Từ chối kế hoạch
  Given revision đang chờ duyệt
  When người có quyền từ chối với lý do
  Then revision chuyển `REJECTED`
  And Marketing Staff nhận được lý do để tạo revision tiếp theo
  And không có hoạt động triển khai mới được tạo từ revision đó

Scenario: Yêu cầu chỉnh sửa thay vì quyết định cuối
  Given kế hoạch cần bổ sung nhưng chưa bị từ chối hoàn toàn
  When người duyệt yêu cầu chỉnh sửa với nhận xét
  Then revision chuyển trạng thái cần sửa theo quy trình
  And giữ nhận xét, người yêu cầu và lịch sử

Scenario: Người duyệt không đủ thẩm quyền ngân sách
  Given chi phí Campaign vượt giới hạn của người duyệt
  When người đó cố phê duyệt
  Then hệ thống từ chối hoặc chuyển cấp theo chính sách
  And không đánh dấu kế hoạch Approved

Scenario: Revision thay đổi sau khi được tải
  Given dữ liệu đã thay đổi/supersede sau lúc người duyệt mở màn hình
  When người duyệt gửi quyết định trên bản cũ
  Then hệ thống không áp dụng quyết định cho revision mới hơn
  And yêu cầu tải lại/đối chiếu

Scenario: Tự duyệt khi chính sách phân tách nhiệm vụ
  Given người gửi kế hoạch đồng thời là người duyệt và chính sách cấm tự duyệt
  When người đó phê duyệt
  Then hệ thống từ chối
  And giữ kế hoạch ở trạng thái chờ người phù hợp

Scenario: Hủy hoặc thu hồi phê duyệt
  Given Campaign đã Approved nhưng chưa hoặc đang triển khai
  When người có quyền thu hồi với lý do
  Then hệ thống áp dụng trạng thái `CANCELLED` hoặc `PAUSED` theo tình trạng thực tế
  And không xóa lịch sử phê duyệt/hoạt động đã phát sinh
```

### US-MKTG-05 — Theo dõi KPI của chiến dịch

**Actor:** Marketing Staff (`ACT-14`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Marketing Staff, tôi muốn theo dõi KPI của chiến dịch sau khi triển khai.

**Giá trị nghiệp vụ:** Marketing so sánh kết quả thực tế với mục tiêu trên cùng định nghĩa, kỳ và nguồn dữ liệu để phát hiện sớm chênh lệch.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem mục tiêu và thực tế KPI
  Given Campaign đã triển khai và có dữ liệu hợp lệ
  When Marketing Staff mở dashboard
  Then hệ thống hiển thị target, actual, chênh lệch và tỷ lệ hoàn thành cho từng KPI
  And nêu rõ công thức, nguồn, kỳ và thời điểm cập nhật

Scenario: Phân tích KPI theo kênh
  Given Campaign sử dụng nhiều kênh
  When người dùng nhóm/lọc theo kênh
  Then hệ thống hiển thị KPI đúng nguồn và Campaign mapping
  And không cộng các metric khác định nghĩa/đơn vị như cùng một chỉ số

Scenario: Chưa có dữ liệu thực tế
  Given Campaign vừa bắt đầu hoặc kênh chưa trả dữ liệu
  When xem dashboard
  Then hệ thống hiển thị trạng thái chưa có/chưa đồng bộ
  And không biểu diễn thiếu dữ liệu thành giá trị 0

Scenario: Dữ liệu đến muộn được cập nhật
  Given kênh bổ sung hoặc điều chỉnh dữ liệu cho kỳ trước
  When lần đồng bộ mới hoàn tất
  Then dashboard cập nhật actual và thời điểm làm mới
  And không sửa target đã duyệt hồi tố

Scenario: KPI mục tiêu cần thay đổi sau khi Campaign chạy
  Given Campaign đang Active và Marketing Staff đề nghị sửa target
  When gửi thay đổi
  Then hệ thống yêu cầu quyền/lý do hoặc revision theo chính sách
  And giữ target ban đầu để đối chiếu

Scenario: Không có quyền xem KPI
  Given người dùng không có quyền Campaign hoặc dữ liệu kênh
  When mở dashboard
  Then hệ thống từ chối hoặc giới hạn phạm vi
  And không làm lộ audience/chi phí/dữ liệu nhạy cảm
```

### US-MKTG-06 — Lấy dữ liệu hiệu quả từ các kênh Marketing

**Actor:** Marketing Staff (`ACT-14`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Marketing Staff, tôi muốn lấy dữ liệu hiệu quả từ các kênh Marketing để đánh giá chiến dịch.

**Giá trị nghiệp vụ:** Dữ liệu bên ngoài được liên kết đúng Campaign, chuẩn hóa có kiểm soát và giữ nguồn để báo cáo không trộn lẫn hoặc đếm trùng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Đồng bộ dữ liệu kênh thành công
  Given Campaign đã ánh xạ đúng tài khoản/kênh/campaign ngoài và quyền kết nối còn hiệu lực
  When thực hiện đồng bộ
  Then hệ thống lưu metric theo nguồn, Campaign, kỳ, múi giờ và đơn vị
  And ghi trạng thái/thời điểm của lần đồng bộ

Scenario: Mapping kênh không tồn tại hoặc mơ hồ
  Given dữ liệu ngoài chưa thể ánh xạ duy nhất tới Campaign nội bộ
  When hệ thống tiếp nhận
  Then không tự gán metric vào Campaign ngẫu nhiên
  And đưa vào trạng thái cần đối chiếu

Scenario: Dữ liệu bị gửi lặp
  Given cùng bản ghi/kỳ/nguồn đã được tiếp nhận
  When Provider gửi lại
  Then hệ thống không cộng đôi metric
  And cập nhật revision nguồn theo quy tắc nếu Provider sửa số liệu

Scenario: Kênh dùng timezone hoặc tiền tệ khác
  Given dữ liệu nguồn có timezone/đơn vị tiền khác Campaign
  When chuẩn hóa để báo cáo
  Then hệ thống giữ cả giá trị nguồn và giá trị quy đổi nếu có
  And nêu rõ tỷ giá, timezone và quy tắc quy đổi áp dụng

Scenario: Đồng bộ một phần
  Given Provider trả thiếu một số metric hoặc kỳ
  When xử lý kết quả
  Then lần đồng bộ được đánh dấu partial
  And dashboard thể hiện phạm vi thiếu thay vì coi đã đầy đủ

Scenario: Provider lỗi hoặc mất quyền kết nối
  Given EXT-07/EXT-12 lỗi, timeout hoặc credential không còn hợp lệ
  When đồng bộ
  Then hệ thống ghi nhận lỗi và cảnh báo người phụ trách
  And giữ dữ liệu hợp lệ gần nhất cùng độ mới
  And cho phép thử lại an toàn

Scenario: Phân biệt dữ liệu bên ngoài và nội bộ
  Given kênh báo Purchase/Conversion và hệ thống có Order/Payment nội bộ
  When hiển thị báo cáo
  Then metric ngoài được ghi nhãn theo nguồn
  And không được coi là doanh thu tài chính đã xác nhận nếu chưa đối chiếu nguồn nội bộ
```

### US-MKTG-07 — So sánh chi phí Marketing với doanh thu

**Actor:** Executive / Business Manager (`ACT-16`)
**Ưu tiên:** Should Have
**Phát hành:** Giai đoạn 2

> Là Executive, tôi muốn so sánh chi phí Marketing với doanh thu tạo ra để đánh giá hiệu quả đầu tư.

**Giá trị nghiệp vụ:** Lãnh đạo đánh giá hiệu quả Campaign trên chi phí và doanh thu có nguồn, công thức và mức độ tin cậy rõ ràng.

**Tiêu chí chấp nhận:**

```gherkin
Scenario: Xem chi phí và doanh thu quy gán
  Given Campaign có chi phí và doanh thu nội bộ được quy gán theo chính sách
  When Executive chọn Campaign/kỳ phân tích
  Then hệ thống hiển thị planned, committed, actual cost và attributed revenue theo dữ liệu sẵn có
  And hiển thị chỉ số hiệu quả cùng công thức được cấu hình
  And nêu rõ nguồn, kỳ và thời điểm cập nhật

Scenario: Phân biệt doanh thu quy gán với tổng doanh thu
  Given doanh nghiệp có Order không liên quan Campaign
  When báo cáo tính hiệu quả
  Then chỉ doanh thu đáp ứng quy tắc attribution được đưa vào attributed revenue
  And tổng doanh thu toàn doanh nghiệp không bị trình bày như do Campaign tạo ra

Scenario: Conversion ngoài không khớp Order nội bộ
  Given nền tảng Marketing báo conversion nhưng không có giao dịch nội bộ được xác nhận
  When tính doanh thu/ROI tài chính
  Then conversion vẫn hiển thị như metric kênh nếu phù hợp
  And không tự chuyển thành doanh thu đã xác nhận

Scenario: Order hủy, hoàn hoặc Refund
  Given Order từng được quy gán cho Campaign nhưng sau đó hủy/hoàn tiền
  When dữ liệu tài chính được cập nhật
  Then attributed revenue được điều chỉnh theo chính sách
  And báo cáo giữ khả năng giải thích chênh lệch theo kỳ

Scenario: Thiếu chi phí hoặc attribution chưa đủ
  Given Campaign thiếu actual cost hoặc không đủ dữ liệu nối doanh thu
  When Executive xem báo cáo
  Then hệ thống đánh dấu kết quả chưa đầy đủ/không thể tính
  And không hiển thị giá trị 0 hoặc ROI chắc chắn gây hiểu sai

Scenario: So sánh nhiều Campaign
  Given các Campaign dùng cùng định nghĩa chỉ số, kỳ và đơn vị có thể so sánh
  When Executive chọn nhiều Campaign
  Then hệ thống hiển thị kết quả cạnh nhau
  And cảnh báo khi attribution window/công thức hoặc độ đầy đủ khác nhau

Scenario: Người dùng không có quyền tài chính
  Given người dùng có quyền xem KPI Marketing nhưng không có quyền chi phí/doanh thu
  When mở báo cáo ROI
  Then hệ thống che hoặc từ chối phần tài chính theo quyền
  And không suy ra dữ liệu nhạy cảm qua tổng hợp chi tiết
```

## 5. Functional Requirements

| ID | Yêu cầu | Traceability |
| --- | --- | --- |
| `FR-MKTG-01` | Cho Marketing Staff có quyền tạo Campaign/Revision Draft với mục tiêu, kỳ, owner, audience, kênh, ngân sách và KPI mục tiêu hợp lệ. | `US-MKTG-01`, `BR-AUTH-04` |
| `FR-MKTG-02` | Kiểm tra khoảng thời gian/lịch kênh và phát hiện chỉnh sửa đồng thời để không ghi đè revision mới hơn. | `US-MKTG-01` |
| `FR-MKTG-03` | Liên kết Product, Promotion và audience segment đúng nguồn/quyền mà không sao chép quyền sửa dữ liệu nguồn. | `US-MKTG-01`, `BR-PROD-04`, `NFR-08` |
| `FR-MKTG-04` | Quản lý Content Brief với thông điệp, audience, kênh, định dạng, deadline, owner và quan hệ Campaign Revision. | `US-MKTG-02` |
| `FR-MKTG-05` | Liên kết Article/Publication của EPIC 20 và Promotion của EPIC 17; không tự xuất bản/kích hoạt hoặc thay đổi hạn mức. | `US-MKTG-02` |
| `FR-MKTG-06` | Quản lý Marketing Media Reference, phiên bản, quyền sử dụng và trạng thái xử lý; không liên kết tệp tải lỗi như sẵn sàng. | `US-MKTG-02`, `EXT-09` |
| `FR-MKTG-07` | Khi asset/Promotion thay đổi sau gửi duyệt, đánh dấu cần rà soát hoặc tạo revision thay vì triển khai khác bản đã duyệt. | `US-MKTG-02`, `US-MKTG-03` |
| `FR-MKTG-08` | Kiểm tra Campaign Revision đủ kế hoạch/KPI/ngân sách/kênh/asset trước khi gửi duyệt. | `US-MKTG-03` |
| `FR-MKTG-09` | Định tuyến yêu cầu duyệt theo loại Campaign, ngân sách và ma trận thẩm quyền; chống gửi lặp. | `US-MKTG-03`, `BR-AUTH-04` |
| `FR-MKTG-10` | Cho thu hồi kế hoạch đang chờ theo quyền/lý do và không cho sửa âm thầm revision `IN_REVIEW`. | `US-MKTG-03`, `BR-AUDIT-01` |
| `FR-MKTG-11` | Cho Executive/Manager có quyền duyệt, từ chối hoặc yêu cầu sửa với lý do; áp dụng phân tách nhiệm vụ nếu chính sách yêu cầu. | `US-MKTG-04`, `BR-AUTH-04` |
| `FR-MKTG-12` | Không tự triển khai chỉ vì Approved; thu hồi quyết định phải Paused/Cancelled phù hợp và giữ lịch sử hoạt động. | `US-MKTG-04` |
| `FR-MKTG-13` | Lưu mục tiêu KPI theo Campaign Revision với tên, định nghĩa, công thức, đơn vị, nguồn, kỳ và target. | `US-MKTG-01`, `US-MKTG-05` |
| `FR-MKTG-14` | Hiển thị target, actual, chênh lệch, tỷ lệ hoàn thành theo Campaign/kênh cùng độ mới và nguồn. | `US-MKTG-05` |
| `FR-MKTG-15` | Phân biệt thiếu/chưa đồng bộ với giá trị 0; giữ target đã duyệt và kiểm soát thay đổi sau khi chạy. | `US-MKTG-05` |
| `FR-MKTG-16` | Quản lý mapping duy nhất giữa Campaign nội bộ và Campaign/tài khoản/kênh ngoài; dữ liệu mơ hồ phải chờ đối chiếu. | `US-MKTG-06` |
| `FR-MKTG-17` | Lưu metric ngoài theo nguồn, Campaign, kỳ, timezone, đơn vị và revision; chống cộng trùng, hỗ trợ correction/late data. | `US-MKTG-06` |
| `FR-MKTG-18` | Theo dõi trạng thái đồng bộ success/partial/failed, lỗi và thời điểm; giữ dữ liệu hợp lệ gần nhất và thử lại an toàn. | `US-MKTG-06`, `NFR-11` |
| `FR-MKTG-19` | Giữ giá trị nguồn khi chuẩn hóa timezone/tiền tệ và nêu rõ quy tắc/tỷ giá áp dụng. | `US-MKTG-06` |
| `FR-MKTG-20` | Phân biệt metric Purchase/Conversion bên ngoài với Order/Payment/doanh thu nội bộ đã xác nhận. | `US-MKTG-06`, `US-MKTG-07`, `BR-ORDER-03` |
| `FR-MKTG-21` | Nhận planned/committed/actual cost và attributed revenue từ nguồn phù hợp, kèm kỳ/trạng thái đối soát. | `US-MKTG-07`, `EPIC 24` |
| `FR-MKTG-22` | Tính chỉ số hiệu quả theo công thức/attribution version được chốt; loại hoặc điều chỉnh Order hủy/refund theo chính sách. | `US-MKTG-07` |
| `FR-MKTG-23` | Đánh dấu báo cáo không đầy đủ khi thiếu chi phí/doanh thu/attribution và cảnh báo khi so sánh khác định nghĩa. | `US-MKTG-07` |
| `FR-MKTG-24` | Kiểm soát riêng quyền lập/sửa/gửi duyệt/duyệt/triển khai, media, audience, KPI, dữ liệu kênh và tài chính. | `US-MKTG-01~07`, `BR-AUTH-03`, `BR-AUTH-04`, `NFR-05` |
| `FR-MKTG-25` | Ghi Audit cho duyệt/từ chối/thu hồi, thay ngân sách/KPI, triển khai/dừng, mapping và attribution quan trọng. | `US-MKTG-03~07`, `BR-AUDIT-01` |
| `FR-MKTG-26` | Phát sinh yêu cầu triển khai/gửi thông qua Epic/kênh đích nhưng không phụ thuộc lưu trạng thái Campaign vào kết quả tích hợp ngoài. | `US-MKTG-02~06`, `EPIC 20`, `EPIC 28` |

## 6. Vòng đời và quy tắc dữ liệu

### 6.1. Vòng đời Campaign Revision

```text
DRAFT → IN_REVIEW → APPROVED → SCHEDULED → ACTIVE → COMPLETED
            │          │            │          │
            ├─► REJECTED            ├──────────┴─► PAUSED
            └─► NEEDS_CHANGES        └────────────► CANCELLED
```

- `DRAFT`: đang lập kế hoạch, chưa được triển khai.
- `IN_REVIEW`: đã gửi duyệt; thay đổi quan trọng cần thu hồi/revision mới.
- `NEEDS_CHANGES`, `REJECTED`: cần sửa hoặc bị từ chối, không được triển khai.
- `APPROVED`: đã duyệt phạm vi/ngân sách nhưng chưa mặc nhiên kích hoạt kênh.
- `SCHEDULED`: chờ lịch bắt đầu đã cấu hình.
- `ACTIVE`: đang chạy; thay đổi trọng yếu cần quy trình kiểm soát.
- `PAUSED`: tạm dừng hoạt động mới, giữ dữ liệu đã phát sinh.
- `COMPLETED`, `CANCELLED`: kết thúc theo kết quả/lý do; không xóa performance hoặc quyết định cũ.

### 6.2. Campaign, Revision và Execution

- Campaign là định danh xuyên suốt; Revision là snapshot kế hoạch được duyệt; Execution là lần triển khai theo kênh/asset/lịch cụ thể.
- Chỉ revision Approved/Scheduled hợp lệ mới được dùng tạo Execution; sửa revision không thay ngầm Execution đang chạy.
- Mỗi Execution phải giữ kênh, account, asset/publication/promotion version, audience reference, lịch và trạng thái ngoài nếu có.
- Dừng/thu hồi kế hoạch phải phân biệt ngừng Campaign nội bộ với kết quả thực tế ở từng platform; lỗi dừng một kênh không được trình bày như đã dừng toàn bộ.
- Audience/segment phải có snapshot hoặc version để giải thích phạm vi đã duyệt, nhưng không sao chép PII không cần thiết vào Campaign.

### 6.3. KPI và dữ liệu kênh

- KPI target phải có định nghĩa, công thức, đơn vị, phạm vi, nguồn và kỳ; cùng tên nhưng khác định nghĩa không được tự cộng/so sánh.
- Metric ngoài phải giữ source platform, source account/campaign, thời gian sự kiện/kỳ, timezone, currency/unit, sync revision và độ mới.
- Dữ liệu gửi lặp/correction/late-arriving phải cập nhật có kiểm soát, không làm cộng đôi.
- Trạng thái `no data`, `not synced`, `partial`, `not applicable` khác với số 0.
- Analytics bên ngoài là dữ liệu đo lường kênh; Order/Payment/Finance nội bộ là nguồn giao dịch. Cả hai có thể đối chiếu nhưng không thay thế nhau.

### 6.4. Chi phí, doanh thu và attribution

- Planned Budget, Committed Cost và Actual Cost là các khái niệm khác nhau; báo cáo phải ghi rõ loại nào đang dùng.
- Attributed Revenue chỉ gồm giao dịch đáp ứng attribution model/window/version được chốt, không phải toàn bộ doanh thu trong kỳ.
- Chỉ số hiệu quả phải hiển thị công thức, ví dụ tỷ lệ doanh thu quy gán trên chi phí hoặc lợi nhuận phù hợp; tên và mẫu số/tử số chính thức cần được chốt.
- Order hủy/return/refund và chi phí điều chỉnh sau kỳ phải được phản ánh theo chính sách, đồng thời giữ khả năng giải thích báo cáo cũ.
- Khi thiếu dữ liệu hoặc không thể nối Campaign–Order đáng tin cậy, hệ thống phải hiển thị độ đầy đủ thay vì suy diễn giá trị chắc chắn.

### 6.5. Quyền riêng tư và kiểm soát

- Audience và dữ liệu cá nhân chỉ được dùng theo mục đích/consent/kênh được phép; quyền xem segment không mặc nhiên cho phép xuất PII.
- Dữ liệu tài chính chỉ hiển thị cho vai trò có quyền, kể cả khi người dùng được xem KPI kênh.
- Content/Promotion phải được duyệt ở Epic nguồn; Campaign approval không tự thay thế approval đó.
- Thay đổi ngân sách, target KPI, attribution mapping hoặc kết quả thủ công phải có quyền, lý do và Audit theo chính sách.
- Tích hợp kênh lỗi không được làm mất Campaign Plan, quyết định duyệt hoặc dữ liệu nguồn hợp lệ đã nhận.

## 7. Traceability

| User Story | Business Rule / NFR | Epic phụ thuộc | Kết quả cần kiểm thử |
| --- | --- | --- | --- |
| `US-MKTG-01` | `BR-AUTH-04`, `BR-PROD-04`, `NFR-08` | EPIC 04, 17, 25 | Campaign Draft đủ mục tiêu/kỳ/kênh/audience/ngân sách/KPI; link nguồn đúng và sửa đồng thời không ghi đè. |
| `US-MKTG-02` | `BR-PROD-04`, `NFR-05` | EPIC 17, 20, EXT-09 | Brief/asset đúng Campaign; không tự xuất bản/kích hoạt; asset đổi sau duyệt phải rà soát. |
| `US-MKTG-03` | `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 22, 23, 28 | Chỉ revision đủ điều kiện được gửi; định tuyến đúng thẩm quyền, chống lặp và không sửa âm thầm khi review. |
| `US-MKTG-04` | `BR-AUTH-03`, `BR-AUTH-04`, `BR-AUDIT-01` | EPIC 22, 23 | Đúng actor duyệt/từ chối/yêu cầu sửa; ngân sách vượt quyền, revision cũ và tự duyệt trái chính sách bị chặn. |
| `US-MKTG-05` | `NFR-05`, `NFR-10` | EXT-07, EXT-12 | KPI có target/actual/công thức/nguồn/kỳ; thiếu dữ liệu khác 0 và target cũ không bị sửa hồi tố. |
| `US-MKTG-06` | `BR-ORDER-03`, `NFR-11` | EXT-07, EXT-12 | Mapping đúng, không đếm trùng, xử lý partial/error/late data và phân biệt metric ngoài với doanh thu nội bộ. |
| `US-MKTG-07` | `BR-ORDER-03`, `NFR-04`, `NFR-05` | EPIC 07, 08, 14, 24 | Chi phí/doanh thu đúng nguồn; attribution có công thức/version; hủy/refund/thiếu dữ liệu và quyền tài chính được xử lý. |

## 8. Điểm cần chốt trước khi thiết kế kỹ thuật

- Các trường Campaign bắt buộc, taxonomy loại chiến dịch, owner/team, mục tiêu và quy tắc Campaign trùng/chồng lịch.
- Ma trận duyệt theo ngân sách/kênh/rủi ro, actor “Manager” được ủy quyền và yêu cầu phân tách người lập/người duyệt.
- Trạng thái/chuyển trạng thái chính thức, quyền pause/cancel/restore và cách xử lý từng Execution khi Campaign bị thu hồi.
- Planned/committed/actual budget, tiền tệ/tỷ giá, ngưỡng vượt ngân sách và nguồn chi phí từ EPIC 24/platform.
- Quan hệ giữa Campaign–Content Brief–Article/Media–Promotion; thay đổi nào bắt buộc duyệt lại.
- Kênh tích hợp ở Phase 2, quyền/tài khoản kết nối, tần suất đồng bộ, thời gian giữ raw data và SLA lỗi.
- Từ điển KPI: impression, reach, click, engagement, conversion, cost và các công thức/đơn vị chuẩn theo kênh.
- Quy tắc mapping Campaign ngoài–nội bộ, UTM/tracking identifier và xử lý dữ liệu không ánh xạ/ánh xạ nhiều Campaign.
- Timezone, currency, tỷ giá và kỳ báo cáo; cách xử lý correction/late data sau khi báo cáo đã chốt.
- Attribution model/window ưu tiên, multi-touch hay single-touch, nguồn nối Order với Campaign và cách xử lý cross-device/offline/marketplace.
- Định nghĩa “doanh thu tạo ra”, trạng thái Order/Payment được tính, thuế/phí/discount/refund và công thức ROI/ROAS/lợi nhuận chính thức.
- Audience consent, opt-out/suppression, giới hạn tần suất và ranh giới với EPIC 25/28; quyền export danh sách khách.
- Loại/kích thước Marketing Media, bản quyền, thời hạn sử dụng, approval và retention.
- Phạm vi Audit, quyền điều chỉnh dữ liệu thủ công và cơ chế đánh dấu số liệu estimated/imported/verified.

## 9. UI/UX Reference

- Campaign Builder cần tách mục tiêu, kỳ, audience, kênh, ngân sách, KPI và owner; hiển thị trạng thái revision/lưu rõ ràng.
- Content/Channel Plan cần thể hiện Brief, Article/Publication, Promotion, Media, owner, deadline và readiness của từng asset.
- Approval screen cần so sánh revision, ngân sách/KPI, audience/kênh và cảnh báo asset chưa sẵn sàng hoặc vượt thẩm quyền.
- Campaign Calendar cần timezone, trạng thái Approved/Scheduled/Active/Paused và drill-down từng Execution/kênh.
- KPI Dashboard cần target–actual–variance, nguồn/độ mới, trạng thái missing/partial và bộ lọc theo Campaign/kênh/kỳ.
- Integration Monitor cần mapping, lần đồng bộ, success/partial/failed, lỗi, phạm vi thiếu và thao tác retry an toàn.
- ROI view cần tách planned/committed/actual cost, attributed revenue, công thức/version, độ đầy đủ và cảnh báo khi so sánh khác định nghĩa.
- Dữ liệu audience/tài chính nhạy cảm phải được che theo quyền; không dựa chỉ vào ẩn nút ở giao diện.
- Chưa có liên kết Figma/mockup được xác nhận; bổ sung sau khi các điểm mở ở Mục 8 được chốt.

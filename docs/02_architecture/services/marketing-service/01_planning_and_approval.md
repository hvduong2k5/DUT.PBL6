# MS-14 · Thiết kế triển khai lát 1

Thiết kế cho US-MKTG-01 đến US-MKTG-04. Nghiệm thu Campaign Draft, Content Brief, liên kết Product, review và audit qua Admin API. Màn hình review dùng snapshot đã gửi. APPROVED xác nhận kế hoạch và ngân sách. Điều phối kênh, KPI actual và Finance thuộc các lát tiếp theo.

[Kiến trúc và chính sách pilot](README.md) · [Admin API draft](../../../03_api_specs/admin-draft-planned.openapi.yaml) · [Ma trận trạng thái](../../../03_api_specs/admin-api-state-matrix.md#marketing-planning-and-approval-draft)

## Workflow và quyền

| Lệnh | Trạng thái trước → sau | Quyền và điều kiện |
| --- | --- | --- |
| Create | ∅ → DRAFT | PLAN_EDIT và BUDGET_EDIT. Department là scope bất biến, actor lấy từ phiên |
| Edit | DRAFT → DRAFT | PLAN_EDIT, thêm BUDGET_EDIT khi đổi tiền. PATCH thay toàn bộ trường plan và kiểm expectedVersion |
| Submit | DRAFT → IN_REVIEW | PLAN_SUBMIT. Lịch hợp lệ, có kênh và KPI, mỗi kênh có brief, Product đã duyệt và công khai nếu có |
| Withdraw | IN_REVIEW → DRAFT | PLAN_SUBMIT. Khớp request đang mở, có lý do. Lưu snapshot cũ và đóng request |
| Approve | IN_REVIEW → APPROVED | PLAN_APPROVE. Khớp request và version, actor khác người lập và người gửi. Duyệt toàn bộ ngân sách đã gửi. Áp dụng trần nếu pilot có cấu hình |
| Reject / request changes | IN_REVIEW → REJECTED / NEEDS_CHANGES | PLAN_APPROVE. Cùng quy tắc phân tách nhiệm vụ, bắt buộc lý do |
| New revision | APPROVED / REJECTED / NEEDS_CHANGES → DRAFT mới | PLAN_EDIT. Copy plan từ revision hiện hành đã đóng review. Revision trước giữ snapshot và quyết định |
| Read / list | Mọi trạng thái trong lát này | CAMPAIGN_VIEW. Có thể đọc revision cũ theo ID để đối chiếu |

Các mã quyền có tiền tố MARKETING_. Với từng quyền cần thiết, grant phải khớp department hoặc GLOBAL của chính quyền đó. Budget trong lát này là số tiền kế hoạch và số tiền duyệt hiển thị cho người có CAMPAIGN_VIEW trong scope. Dữ liệu chi phí thực tế và doanh thu thuộc quyền tài chính ở lát sau.

Draft chấp nhận danh sách kênh/KPI rỗng và brief trống để chuẩn bị. Khoảng thời gian phải hợp lệ ở mọi lần lưu, lịch kênh nằm trong kỳ campaign và timezone là tên IANA hợp lệ. Pilot có tối đa một kế hoạch cho mỗi mã kênh trong revision. Mục tiêu KPI cấp campaign dùng kỳ campaign, là số thập phân theo metric code và đơn vị hợp lệ, không trùng metric code. Submit yêu cầu ít nhất một kênh và một KPI, brief có nội dung sau khi trim, ngân sách VND nguyên không âm và Product reference không trùng. Nhu cầu Article, Promotion và Media ghi trong brief. Chuẩn hóa mã và trim chuỗi trước khi kiểm trùng và tính hash.

## Dữ liệu và transaction

| Bảng | Khóa và ràng buộc |
| --- | --- |
| campaigns | UUID PK, department_id, current_revision_id, next_revision_number, updated_at. FK tổ hợp giữ current revision trong cùng campaign. Index theo department_id, updated_at, id |
| campaign_revisions | UUID PK, campaign_id FK, revision_no, base_revision_id, version bắt đầu từ 1, created_by, state, owner_id, name, objective, starts_at, ends_at, timezone, planned_budget_vnd. Unique campaign_id + revision_no. CHECK ends_at > starts_at và tiền không âm |
| channel_plans / kpi_targets / product_references | FK revision_id. Unique theo revision + channel/metric code/product_id. Brief nằm cùng channel plan trong lát này. Product reference giữ UUID và slug |
| approval_requests | UUID PK, campaign_id, revision_id, snapshot JSONB, snapshot_hash, policy_version, submitted_by/at, state. Unique partial theo campaign_id với state OPEN. Snapshot lưu plan và bằng chứng Product gồm ID/version/trạng thái/thời điểm kiểm |
| approval_decisions | request_id PK/FK, action, actor_id, decided_at, reason, approved_budget_vnd. Một kết quả đóng cho mỗi request. Số tiền duyệt chỉ có ở APPROVED, lý do bắt buộc cho các kết quả đóng còn lại |
| audit_entries | UUID PK, campaign_id, revision_id, approval_request_id khi có, action, actor_id, department_id, request_id, from/to state, version, reason và changed field names. Ghi cùng transaction với nghiệp vụ |
| idempotency_records | Unique actor_id + department_id + action + key. Lưu request hash và kết quả HTTP. Hash bao gồm path params và body đã chuẩn hóa, bao gồm expectedVersion |

Mỗi command kiểm IAM trước. Submit đọc Catalog Admin ngoài transaction theo productId, kiểm đúng slug và trạng thái APPROVED/ACTIVE, lưu revision Catalog dưới tên version trong evidence. Trong transaction, khóa campaign trước rồi revision và request, xác nhận expectedVersion và lưu snapshot. Vòng review dùng dữ liệu snapshot; kênh chạy thực tế sẽ xác minh readiness ở lát triển khai. Nếu Catalog trả 404, ID lệch hoặc trạng thái không đủ điều kiện, trả 422; timeout hoặc nguồn lỗi trả 503. Với campaign không gắn Product, submit không cần gọi Catalog.

Mọi cập nhật revision tăng version. Với lệnh tạo revision mới, expectedVersion thuộc revision cơ sở và revision đó phải còn hiện hành. Mọi lần ghi làm mới updated_at của campaign. Quyết định, trạng thái, audit và kết quả idempotency commit cùng transaction. Không giữ lock khi gọi Identity/Catalog. Lỗi tạm thời không ghi kết quả idempotency, để client có thể thử lại.

- **Hai người sửa cùng revision.** Người ghi đầu tiên commit, người thứ hai nhận 409 STALE_REVISION và tải lại dữ liệu.
- **Submit lặp.** Cùng key và canonical payload trả kết quả đã lưu sau khi kiểm lại quyền. Cùng key khác payload trả 409 IDEMPOTENCY_KEY_REUSED. Với key khác, nếu cùng revision và expectedVersion của lần submit đã mở request, trả request đang mở; không thêm audit hay request. Withdraw rồi submit lại cần version và key mới.
- **Approve đồng thời với withdraw.** Khóa theo cùng thứ tự. Chỉ lệnh đầu tiên có request OPEN và version đúng. Lệnh còn lại nhận 409 STALE_REVISION hoặc STATE_CONFLICT.
- **Danh sách và chi tiết.** List chỉ trả department được cấp, lọc theo department/state, sắp updated_at DESC rồi id DESC. Cursor được ràng buộc với filter và principal, mỗi request kiểm lại scope. Total đếm tập kết quả theo filter trước phân trang. ID không tồn tại hoặc ngoài phạm vi hiển thị trả 404. Thiếu quyền cho một action trên tài nguyên nhìn thấy trả 403.

## Tích hợp và nghiệm thu

| Nguồn | Interface đã thấy trên origin/dev tại 239c8db | Kiểm tra trước pilot |
| --- | --- | --- |
| Identity / Kong | GET /api/v1/users/me trả scoped permissionGrants. Analytics có tiền lệ dùng session qua Identity | Xác nhận mã DEPARTMENT, seed quyền Marketing, test staff ACTIVE với grant đúng/sai department, GLOBAL và phiên hết hạn. Kiểm luồng BFF/session và bảo vệ CSRF qua Kong |
| Catalog | GET /api/v1/admin/catalog/products/{productId} trả productId, slug, revision, approvalStatus và listingStatus. Endpoint public chỉ trả bản chiếu công khai | Đối chiếu response thật của Product công khai, bị ẩn và slug sai. Test timeout và Product đổi version giữa hai lần submit |
| Audit | Có schema audit.events.v1, audit nội bộ đủ cho lát đầu | Test audit commit cùng command, rollback khi lỗi. Bật publish tập trung sau khi xác nhận consumer và contract |
| Content / Promotion / Media | Chưa có provider phù hợp cho hồ sơ pilot trên dev đã đọc | Kiểm brief mô tả nhu cầu được lưu và review đầy đủ. Chốt readiness từng nguồn trước lát triển khai |

Test domain cho toàn bộ transition, quyền thiếu, tự duyệt, ngân sách vượt trần, lịch và dữ liệu submit. Test PostgreSQL thật cho hai lần sửa, submit lặp, approve/withdraw đồng thời, rollback và revision lịch sử. Contract test kiểm mọi response đã công bố, error envelope, version, idempotency và list rỗng. Smoke test qua Kong dùng staff thật và Catalog thật là điều kiện mở pilot. Có thể dùng fixture để viết domain và migration trước khi có mã phòng ban và tài khoản thử.

Các lựa chọn của bản thiết kế gồm PATCH thay toàn bộ plan, budget VND nguyên, một brief mỗi kênh và cùng quyền CAMPAIGN_VIEW cho ngân sách kế hoạch/đã duyệt. Các mã quyền và hình dạng API đang ở draft. Chốt với IAM và frontend khi thực hiện handoff.

## Quyết định triển khai persistence

Lát 1 dùng Spring JDBC để giữ các truy vấn khóa và transaction tường minh. Migration dùng FK tổ hợp được kiểm khi commit cho con trỏ current revision, cho phép tạo campaign và revision đầu tiên trong cùng transaction. Yêu cầu duyệt lưu submitted_revision_version để nhận biết retry submit bằng key khác. Snapshot review lưu JSONB độc lập với các bảng plan của draft.


Các command dùng khóa hàng Campaign làm điểm tuần tự hóa cho tất cả cập nhật revision và yêu cầu duyệt của Campaign. Một advisory transaction lock riêng theo actor, department, action và key tuần tự hóa các lần thử cùng khóa idempotency, kể cả khi Campaign chưa tồn tại. Hồ sơ review giữ mức trần áp dụng lúc submit cùng snapshot, để quyết định dựa trên chính sách đã lưu.

Catalog Admin hiện xác thực bằng bearer token. Submit có Product yêu cầu staff bearer có CATALOG_VIEW; submit chỉ có phiên cookie trả 403 CATALOG_AUTHORIZATION_REQUIRED. Submit không có Product dùng được cả cookie và bearer. Cookie mutation yêu cầu Origin khớp MARKETING_ADMIN_ORIGIN. Web Admin dùng prefix /api/v1/admin/marketing-api/campaigns. Identity kiểm employee đang active, dùng bearer của phiên và chuyển tiếp vào prefix /api/v1/admin/marketing/campaigns qua gateway. Trình duyệt giữ cookie phiên. BFF dùng cấu hình gateway hiện có và cho phép đúng method/path của lát 1. Service không nhận token Catalog do client khai báo trong payload.

## Chạy service và kiểm thử

Build image với context riêng của service bằng docker build -t marketing-service:local services/marketing-service. API mặc định lắng nghe cổng 8014. Flyway chạy migration khi API khởi động. Cấu hình MARKETING_DB_URL, MARKETING_DB_USER, MARKETING_DB_PASSWORD, MARKETING_IDENTITY_URL và MARKETING_CATALOG_URL từ môi trường triển khai. Chỉ đặt MARKETING_ADMIN_ORIGIN khi cần bật luồng cookie từ origin Web Admin đã xác định. MARKETING_BUDGET_CEILING_VND là trần tùy chọn.

Chạy sh gradlew test trong thư mục service. Cấp MARKETING_DB_URL, MARKETING_DB_USER và MARKETING_DB_PASSWORD trỏ tới PostgreSQL test cô lập để chạy test transaction và HTTP. Test HTTP dùng fixture Identity; bước mở pilot vẫn cần smoke test với Kong, Identity và Catalog thật.

Kong định tuyến /api/v1/admin/marketing/campaigns tới marketing-service:8014 và tắt retry upstream cho command. Migration IAM V14 khai báo năm permission của lát này. IAM cấp role và assignment theo DEPARTMENT trong môi trường pilot, dựa trên mã phòng ban chính thức. Migration chỉ thêm danh mục quyền.

### Kết quả smoke test stack thật

Trên nhánh Marketing đã đồng bộ origin/dev tại 14757f2, smoke test Docker qua Kong và route Identity BFF dùng bearer do Keycloak local phát cho hai staff fixture. Identity xác nhận grant DEPARTMENT. Catalog Admin trả revision 7 và Marketing lưu đúng evidence lúc submit. Replay cùng key trả cùng approval request, tự duyệt trả 403, reviewer chuyển Campaign sang APPROVED với 125000 VND. Audit PostgreSQL ghi đúng actor cho CREATE, SUBMIT và APPROVE. Lượt này dùng bearer vào BFF. Việc đăng nhập trình duyệt và chuyển cookie OMAMA_SESSION thành bearer cần kiểm riêng.

Smoke test cookie tiếp theo dùng HTTP client có cookie jar để thực hiện form login Keycloak, authorization code callback qua Kong và gọi Identity BFF chỉ bằng OMAMA_SESSION. Staff tạo và gửi duyệt Campaign có Product, snapshot lưu revision 7. Reviewer đăng nhập phiên riêng và duyệt thành công. Keycloak chỉ được đưa ra loopback qua relay tạm phục vụ test. Lượt này kiểm OAuth và cookie ở mức HTTP, chưa kiểm giao diện trình duyệt.

# Thiết kế chi tiết Traceability phiên bản đầu

Trạng thái: thiết kế triển khai theo [quyết định hiện hành](./01_decisions_and_peer_code_review.md)
và [kế hoạch](./02_development_plan.md). Ngày 2026-10-17. Backend theo Batch;
R1 dùng fixture dependency có nhãn, R2 cần provider/identity contract thật.

## 1. Thành phần và hướng phụ thuộc

`HTTP → Application → Domain`; Application dùng các port Repository/Transaction,
CatalogVerifier, BatchVerifier, IdentityVerifier, Clock, Random và EventPublisher.
PostgreSQL/Kafka/JWT/fixture là adapter. Composition ở `cmd/server`; domain không
import HTTP, SQL hoặc Kafka. Mỗi API ghi tương ứng một application command.

Không FK/SQL JOIN xuyên DB. Không thêm runtime dependency vào Order/Shipping.
Public đọc snapshot local đã duyệt; không gọi peer trong request scan QR.

## 2. Định danh, nội dung và mode

Profile có UUID, mode, Product UUID, sku_code, source namespace và stable Batch ID.
Binding này bất biến ngay khi tạo. `batch_code` chỉ là nhãn snapshot, không là khóa
identity vì provider hiện chưa bảo đảm uniqueness. Sandbox fixture có namespace
riêng; profile sandbox không được tái sử dụng trong integration. DB metadata giữ
mode và startup từ chối mode khác để tránh đổi env làm lộ dữ liệu sandbox.

Revision chứa nội dung snapshot: tên sản phẩm, producer/workshop, artisan tùy chọn,
nguyên liệu/vùng nguồn, process, certificate list và evidence metadata. Không nhập
NSX/HSD làm dữ liệu authoritative: lấy từ BatchVerifier khi submit, giữ snapshot.
Thông tin manual verification tách khỏi dependency verification; reviewer phải
ghi rõ bằng chứng và thời điểm kiểm tra, không tự gán xác minh ngoài.

Certificate: id, type (OCOP/ATTP/OTHER), number, issuer, subject scope/reference,
issued_on, valid_until hoặc NO_EXPIRY_STATED + review_due_at, stars cho OCOP,
verification_method MANUAL, verified_at, evidence IDs và public attributes.
Submit cần ít nhất một OCOP hợp lệ đúng phạm vi và đủ origin/producer/process.
ATTP/artisan/video không được bịa nếu thiếu; public dùng nullable/status riêng.

Mỗi evidence có ID, internal reference, kind, description và public permission.
V1 không upload hoặc fetch URL từ người dùng. Public video URL chỉ nhận HTTPS
thuộc allowlist host cấu hình; chứng cứ nội bộ không có trong public DTO.

## 3. Máy trạng thái và quyền

Profile ACTIVE/SUSPENDED/ARCHIVED; revision DRAFT/IN_REVIEW/APPROVED/REJECTED/
PUBLISHED/ARCHIVED. Một working revision (Draft/In review/Approved) mỗi profile.
Một public pointer tùy chọn, chỉ trỏ revision Published cùng profile.

| Command | Quyền | Guard | Kết quả |
| --- | --- | --- | --- |
| Create | create | Binding mới, nội dung đúng kiểu | Profile Active + Draft 1, chưa public |
| Edit | edit | Draft, Active, expected_version đúng | Thay full content snapshot và ghi contributor |
| New revision | edit | Active, version đúng; nếu đang có working revision thì bắt buộc reason thu hồi | Archive working revision cũ, clone bản chỉ định thành Draft, contributor kế thừa |
| Submit | submit | Draft đầy đủ, peer xác minh, version đúng | Freeze content, snapshot dependency, In review |
| Approve | approve | In review, không là contributor, cert còn hợp lệ | Approved và immutable decision |
| Reject | approve | In review, reason bắt buộc, không tự duyệt | Rejected và immutable decision |
| Publish | publish | Approved, Active, peer recheck không đổi, cert hợp lệ | Atomic public pointer/audit/outbox |
| Suspend | suspend | Chưa archive, reason bắt buộc | Profile suspended, public 410 |
| Archive | archive | Reason bắt buộc | Profile archived, public 410 |

Publisher có thể chính là reviewer nhưng không được bỏ qua approval. Không có
ADMIN bypass. Chưa có restore; tạo revision mới không được vượt suspension.
Published content không sửa tại chỗ; thay public pointer chỉ archive metadata bản
cũ và giữ nội dung/decision/evidence. Reject không mở lại content cho sửa; clone
thành Draft mới. Contributor kế thừa khi clone, thêm mọi actor sửa nội dung; tránh
đổi creator để tự duyệt nội dung mình đã biên soạn. Quyết định mới cần actor độc lập.
New revision có thể thu hồi working revision Draft/In review/Approved bằng reason
và audit trong cùng transaction, không đổi bản public hiện hành. Điều này cho
phép sửa nguồn đã thay đổi sau approve mà không kẹt ở Approved không thể publish.
Không có thao tác sửa nội dung Approved; approval cũ vẫn lưu nhưng không áp dụng
lên bản mới. Nếu clone từ bản khác, kế thừa contributor của cả nội dung nguồn và
working revision bị thu hồi để không mở đường lách kiểm soát tự duyệt.

Quyền read_internal không tự có read_evidence. Response admin che evidence
reference nếu principal thiếu quyền; không lộ bằng URL trực tiếp. V1 không có
multi-tenant/warehouse scoped delegation: mọi permission trên instance có phạm vi
instance, không tuyên bố đã hỗ trợ scope đó; nếu R2 cần thì bổ sung contract trước.

Create/edit cần cả quyền thao tác và read_internal vì trả lại hồ sơ/history.
Các command khác trả history cũng cần read_internal. Approve/reject đều cần
reason/căn cứ không rỗng; suspend/archive lặp bằng key mới trả conflict khi đã ở
trạng thái đích. Retry cùng key/body replay kết quả đã commit.
Authorize lại trước lookup/replay và redact theo quyền hiện tại. Replay create
đã commit không gọi lại provider; cache miss phải xác minh nguồn ngoài transaction,
sau đó transaction runner kiểm tra dedup lần nữa dưới khóa để tránh race.

## 4. Transaction, concurrency và idempotency

Mỗi profile có `version` tăng sau command thành công. Mọi command ngoài create
cần expected_version, kể cả approve/publish/suspend. Lock profile row trước,
revision sau theo thứ tự cố định; concurrency stale trả 409. Hai publish không
được tạo hai public pointers. Không giữ DB transaction trong network call.

Command fingerprint là SHA-256 canonical decoded request + method/action + resource;
principal và action namespace idempotency key. Key dài 1..128 ký tự an toàn;
record hoàn tất và response cùng transaction nghiệp vụ. Retry cùng key/body
replay response gốc, key/body khác trả 409. Permission được kiểm tra cả khi replay,
không cho key trở thành cách lấy response ngoài quyền. Key không được reuse do TTL.

Submit/publish: đọc expected profile version → query peer với deadline ngoài TX →
lock/recheck version → áp dụng result nếu freshness <= 30 giây → commit. Nếu
provider timeout/contract thiếu thì 503, không tự fallback fixture trong integration.
Snapshot hash bao gồm identity, fields và source version nếu có, không gồm checked_at.
Publish recheck phải khớp snapshot đã submit; khác → 409 yêu cầu revision/duyệt lại.
Không claim atomicity xuyên service: source vẫn có thể đổi sau query; R2 cần version
và correction contract từ owner. Public biểu thị verified_at để phản ánh giới hạn này.

Publish đổi revision status/pointer + local audit + business outbox nguyên tử.
Mọi command quan trọng có audit local append-only, ghi reason, actor, before/after
đã redacted, request ID; không log JWT/secret/evidence nội bộ trong telemetry.

## 5. Public QR và thời gian

Token CSPRNG 128 bit trở lên, base64url không padding, unique DB; collision retry
có giới hạn, không thay binding đã tạo. Token cấp lúc tạo profile, chỉ public sau
publish. URL trang `{PUBLIC_WEB_BASE_URL}/trace/{token}` và API `/api/v1/trace/{token}`.

| Tình trạng | HTTP | Dữ liệu |
| --- | --- | --- |
| Không có token/chưa từng public | 404 | Error an toàn, không lộ Draft |
| Active + Published + eligibility đạt | 200 | DTO allowlist, revision/verified_at/mode |
| Từng public, suspended/archived | 410 | reason riêng, không trả valid content cũ |
| Chứng nhận expired/revoked/review_due | 410 | CERTIFICATE_*; giữ history nội bộ |
| Batch expired/known restricted | 410 | BATCH_*; không nhầm với certificate expiry |
| Batch depleted nhưng không expired/restricted | 200 nếu đủ điều kiện khác | Không coi hết tồn là mất nguồn gốc |

`Cache-Control: no-store` cho mọi public response. Eligibility tính trên mỗi read
với clock inject, không đợi cron. `valid_until` certificate inclusive theo
Asia/Ho_Chi_Minh; hết từ đầu ngày kế tiếp. `review_due_at` hết tại đúng instant.
Ngày phát hành tương lai không hợp lệ. Batch dùng best_before instant do provider
cung cấp; không tự đổi quy tắc ngày của Inventory. UTC cho stored timestamps.
Không có data correction event thì không biết peer vừa sửa Batch; không claim
real-time verification. Known correction chặn publish, hoặc suspend có audit.

Public `verified_at` là thời điểm kiểm tra OCOP thủ công. `source_verified_at` là
checked_at của snapshot nguồn đã đóng băng; publish recheck hash không sửa snapshot
đã submit. Producer/process có trong DTO; artisan/ATTP/video thiếu giữ null.

## 6. Outbox, worker và Audit contract còn lệch

Business CloudEvents dùng event ID ổn định, profile_id làm Kafka key, aggregate
version tăng và draft schema Traceability. At-least-once; lease/claim bằng DB,
mark published sau broker ack; retry giữ ID/payload. Per-profile ordering cần ưu
tiên oldest chưa sent; consumer vẫn kiểm tra version/id vì crash có thể gửi trùng.
Sandbox relay disabled phải giữ Pending; không dùng log làm proof broker delivered.

Expiry worker chỉ ghi eligibility transition/event idempotent, không tự chuyển
profile sang Suspended (v1 không restore); revision mới chứng nhận hợp lệ vẫn có
thể publish khi profile Active. Read-time guard bảo vệ cả khi worker ngừng.

Phát hiện khi thiết kế: schema Audit chung hiện yêu cầu prev_entity_hash và
current_entity_hash ở producer, trong khi HLD mô tả Audit Service tính chain.
Không bịa hash hoặc tuyên bố consumer đã chấp nhận. R1 giữ audit local bền vững;
audit dispatch phải ghi draft/integration gate X-09 tới khi owner xác nhận schema/
trách nhiệm. Local audit không thay thế hash-chain/ingestion thật của MS-18.

## 7. Runtime và điều kiện sẵn sàng

Go module độc lập; PostgreSQL 16, REST 8003. `/livez` chỉ process; `/readyz` cần
DB, migration version, runtime DB role/mode và verifier/dependency capability theo
mode. Integration thiếu identity thì không mở admin; Batch identity chưa đủ thì
command cần xác minh trả 503. Production chưa được hỗ trợ.

Sandbox host-native bind loopback. Container sandbox chỉ được bật bằng cấu hình
explicit và Compose publish host 127.0.0.1, network riêng; test keys vẫn chỉ sandbox.
Secrets qua env/file sinh lúc chạy, không commit; logs che DSN. Shutdown dừng
HTTP accept, drain transactions/workers trong timeout rồi đóng resources.

## 8. Giới hạn API

Body tối đa 256 KiB, tối đa 20 certificates/50 evidence/20 ingredients, string
name <= 250, text <= 8000, reason 1..1000. Reject unknown fields, trailing JSON,
negative/zero expected_version, malformed UUID/date, unsafe URL và duplicate IDs.
List page_size 1..100, mặc định 20, cursor ổn định theo created_at + id.
OpenAPI mô tả field/nullability/errors. Không đổi contract Web/GraphQL tổng một cách ngầm định.

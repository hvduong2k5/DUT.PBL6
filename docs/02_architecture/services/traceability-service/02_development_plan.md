# Kế hoạch phát triển Traceability Service

Ngày lập: 2026-10-17. Nhánh: `feat/traceability-service`.
Trạng thái cập nhật: **R1 sẵn sàng review trong phạm vi sandbox Batch backend**.
P0..P7 có bằng chứng local; P8/R2 chờ contract. Xem
[đối chiếu cuối R1](../../../04_testing/traceability-service/04_final_r1_audit.md).
Sự tồn tại của file code không đồng nghĩa hạng mục đã được kiểm chứng hoàn tất.

Mục tiêu là tạo backend MS-03 chạy độc lập và có thể review toàn luồng:
**tạo hồ sơ Batch → gửi duyệt → duyệt → publish → quét QR → đình chỉ**.
Sau đó kết nối các provider/consumer thật khi contract đủ điều kiện. Không lấy
việc hoàn thành sandbox làm bằng chứng đã tích hợp toàn hệ thống hoặc xong EPIC 19.

## 1. Cơ sở và phạm vi

Nguồn quyết định hiện hành:
[Quyết định nội bộ và đối chiếu code peer](./01_decisions_and_peer_code_review.md).
Nguồn yêu cầu:
[EPIC 19](../../../01_requirements/epics/EPIC_19_OCOP_Traceability.md).
Các vấn đề gốc:
[Readiness review](./00_readiness_review.md).

### Trong phiên bản đầu

- Hồ sơ cấp Batch liên kết Product/SKU/Batch; nguồn gốc và chứng nhận có bằng chứng.
- Kiểm tra chứng nhận thủ công, có người/thời điểm/phạm vi, không giả xác minh EXT-10.
- Revision có lịch sử; duyệt và publish tách riêng; người biên soạn không tự duyệt.
- QR ngẫu nhiên ổn định theo Batch, không đổi sang lô khác khi sửa revision.
- Admin REST, public REST, PostgreSQL, audit cục bộ và transactional outbox.
- Xử lý xung đột, yêu cầu lặp, chứng nhận hết hiệu lực, Batch hết HSD và đình chỉ.
- Sandbox có fixture rõ nhãn; integration có adapter thật và kiểm tra điều kiện nghiêm ngặt.
- README, migration, OpenAPI, tests và hướng dẫn reviewer chạy lại.

### Hoãn khỏi phiên bản đầu

Hồ sơ Product/SKU độc lập, QR từng hộp/chống sao chép, restore hồ sơ đình chỉ,
hẹn giờ publish, genealogy nguyên liệu nhiều cấp, tự nhận Procurement/
PackingCompleted, tích hợp EXT-10, upload/media pipeline và thống kê scan nâng cao.
Không sửa code peer hoặc thay contract chung trong phạm vi kế hoạch này nếu chưa
có yêu cầu tích hợp cụ thể. Giao diện Web/Mobile không thuộc implementation backend.

## 2. Hai mốc hoàn thành riêng biệt

| Mốc | Cần chứng minh | Chưa được tuyên bố |
| --- | --- | --- |
| **R1 — Backend độc lập có thể review** | REST chạy thật, PostgreSQL thật, identity test có chữ ký, Catalog/Batch fixture, toàn luồng và các failure cases có test; outbox lưu bền vững | Chưa coi fixture là Inventory/Catalog thật, chưa coi local audit là Audit Service đã ingest |
| **R2 — Tích hợp các peer được chỉ định** | Contract được chốt, adapter và roundtrip với provider thật, identity/Gateway thật, UI/consumer liên quan được kiểm chứng | Không suy rộng sang peer chưa test, production hoặc toàn bộ Epic |

R1 không phải chế độ bỏ qua bảo mật hoặc bỏ qua kiểm tra nguồn. Fixture xác minh
được quan hệ trong tập dữ liệu tổng hợp; hồ sơ sandbox không được dùng như dữ liệu thật.

## 3. Thứ tự triển khai

Thực hiện P0 → P7 để đạt R1. P8 dành cho R2 và chỉ làm những integration đã có
contract. Viết tests cùng từng phần; P7 tổng hợp bằng chứng, không để đến đó mới test.
Mỗi phần chia thành PR/commit nhỏ theo đầu ra nghiệp vụ; kế hoạch này không tự commit/push.

### P0 — Đặc tả chi tiết và contract nội bộ

**Công việc**

- Viết `03_service_design.md`: domain model, state transition/permission matrix,
  public eligibility, snapshot, QR lifecycle và các transaction boundary.
- Viết `04_database_design.md`: ERD, bảng/cột/index/constraints, draft/public pointer,
  immutable content, contributor history, audit, outbox và idempotency.
- Viết OpenAPI service tại `docs/03_api_specs/traceability-service.openapi.yaml`.
  Ghi rõ đây là contract Traceability, không tự thay OpenAPI tổng/GraphQL của nhóm.
- Viết `05_test_matrix.md`: ánh xạ FR/US → hành vi → test → bằng chứng.
- Làm rõ bằng thiết kế nội bộ: một working revision chưa kết thúc tại một thời
  điểm; cách đánh dấu revision bị thay thế; contributor từ bản clone; thay đổi
  bằng chứng/verification sau submit; snapshot nào bị invalid khi nguồn đổi.
- Quy định CAS cho cả quyết định và public pointer; recheck dependency không mở
  transaction DB trong suốt network call. Ghi rõ giới hạn không có atomic transaction
  xuyên service và yêu cầu version/correction event từ nguồn nếu cần bảo đảm mạnh hơn.
- Định nghĩa giới hạn payload/pagination, command fingerprint, error taxonomy,
  certificate review_due và date semantics; dùng UTC timestamp + certificate date tại Huế.
- Ghi dependency capability matrix và các câu hỏi X-01..X-09, không biến đề nghị
  thêm field cho peer thành API đã tồn tại.

**Hoàn thành khi**: không còn luật nội bộ bỏ trống cho luồng R1; API thể hiện
được guard/lỗi; schema giữ được lịch sử và ngăn duplicate current record.
Contract ngoài còn mở được ghi thành integration gate, không dùng giả định ngầm.

### P1 — Khung chạy, cấu hình và môi trường kiểm thử

**Công việc**

- Khởi tạo Go module trong `services/traceability-service`, Clean Architecture:
  domain → application/ports → infrastructure → HTTP transport; composition ở cmd/server.
- Pin toolchain/dependencies, tạo config validation, injectable clock/ID/token generator,
  structured logging có request ID và graceful shutdown.
- Tách sandbox/integration; production chưa được mở. Sandbox bind loopback,
  fixture và test issuer không tự được chọn khi integration thiếu cấu hình.
- Tạo `/livez`, `/readyz`, Dockerfile, Compose riêng với PostgreSQL 16 và env example.
- Tạo test harness PostgreSQL cô lập, khóa test sinh lúc chạy; không ảnh hưởng
  DB/volume/container peer và không commit private key/token.

**Kiểm chứng**: build/vet; config thiếu hoặc sai bị từ chối; readiness không báo
healthy khi DB/schema chưa sẵn sàng; startup/shutdown smoke. Chưa có endpoint
ghi dữ liệu được mở nếu identity chưa được cài đặt.

### P2 — Persistence và các ràng buộc cốt lõi

**Công việc**

- Tạo migration versioned theo P0; runtime role riêng, không dùng quyền thay schema
  để chạy API. Không tự reset database trong startup.
- Cài repositories/transaction manager cho profile/revision/certification/evidence,
  decision/contributor, public pointer, QR, verification, local audit/outbox và idempotency.
- Constraints và locking bảo đảm một hồ sơ hiện hành cho Batch binding, một Draft/
  working revision theo policy và một public pointer; không FK tới DB peer.
- DB bảo vệ nội dung revision đã submit/publish khỏi sửa tại chỗ; không dựa chỉ
  vào UI hoặc handler. Chỉnh trạng thái metadata qua lệnh hợp lệ có audit.
- Cài atomic command execution và dedup theo principal/action/key/fingerprint;
  không để idempotency record hoàn tất trước transaction nghiệp vụ.

**Kiểm chứng bằng PostgreSQL thật**: migration trên DB trống, constraint violations,
CAS, rollback, duplicate key/body mismatch, concurrent command, nội dung đóng băng
và audit/outbox cùng transaction. Migration down chỉ thử trong DB test dùng một lần.

### P3 — Lát cắt tạo và chỉnh sửa hồ sơ nháp

**Công việc**

- Cài identity verifier/permission enforcement trước khi mở admin API.
  Sandbox dùng token ký với issuer/audience riêng; không tin role/header client.
- Cài create/read/list profile, tạo/sửa Draft, evidence metadata và chứng nhận;
  lưu lịch sử actor đã biên soạn, expected version và validation theo trạng thái.
- Cài CatalogVerifier/BatchVerifier fixture có stable IDs; hỗ trợ not-found,
  mismatch, timeout, unavailable và source correction bằng dữ liệu tổng hợp.
- Tách `verification_method=MANUAL` của bằng chứng nguồn khỏi xác minh quan hệ
  Product/SKU/Batch bằng dependency; không gộp hai khái niệm thành một cờ verified.

**Kiểm chứng**: hai nhân viên có quyền khác nhau thao tác được đúng phạm vi;
token sai/expired/wrong audience, header giả, ID ngoài scope, edit stale và
quan hệ đã biết sai bị chặn. Draft thiếu dữ liệu lưu được nhưng không public.

### P4 — Gửi duyệt và quyết định duyệt

**Công việc**

- Cài submit, approve, reject theo state matrix và contributor policy.
- Submit kiểm tra required fields, certificate scope/dates, evidence và kết quả
  dependency. Bản đã submit không sửa âm thầm; bị từ chối phải tạo Draft phù hợp.
- Approve/reject lưu actor, time, version, lý do/căn cứ; không tự publish.
- Source/evidence thay đổi sau duyệt phải invalid quyết định hoặc yêu cầu revision
  mới theo P0; không cho giữ approval cũ để công khai nội dung khác.

**Kiểm chứng**: self-approval và ADMIN bypass bị chặn; approve trên version cũ,
approve/reject đồng thời, thiếu dữ liệu/bằng chứng và lệnh gửi lặp được xử lý đúng.
Không thấy dữ liệu qua public API sau khi approve nhưng chưa publish.

### P5 — Publish, QR, public read, suspend và archive

**Công việc**

- Publish chỉ từ Approved, hồ sơ Active và kiểm tra nguồn/chứng nhận đạt.
  Public pointer + audit + outbox được cập nhật trong một local transaction.
- Cấp QR bằng CSPRNG ít nhất 128 bit; stable theo Batch, không tái gán lô;
  URL trang `/trace/{code}`, public endpoint `/api/v1/trace/{code}`.
- Cài public DTO allowlist, verification metadata và nullable/status fields.
  Không bịa artisan/chứng nhận/URL để thỏa DTO Web cũ.
- Public trả 404 cho mã sai hoặc chưa từng public; 410 theo reason cho hồ sơ đã
  public bị đình chỉ/archive, chứng nhận hết hiệu lực hoặc Batch hết HSD/restricted.
  Depleted không mặc nhiên chặn truy xuất. Response không lộ tài liệu nội bộ.
- Kiểm tra eligibility theo clock khi đọc, `Cache-Control: no-store`; không gọi
  provider đồng bộ mỗi lượt scan. Không hứa phát hiện correction chưa nhận từ peer.
- Suspend/archive bắt buộc quyền/reason; giữ history; không cho publish bản mới
  hoặc dùng QR mới để âm thầm vượt suspension của cùng hồ sơ/Batch binding.

**Kiểm chứng**: public đúng revision; edit tạo bản mới không đổi bản khách đang xem;
publish đồng thời chỉ có một public pointer; QR collision/retry không gắn nhầm Batch;
read sau suspend không trả valid content; certificate date boundary, batch expiry,
no-expiry-stated review_due; anonymous không truy cập Draft/evidence.

**Checkpoint R1-A**: reviewer chạy được toàn luồng R1 qua HTTP với PostgreSQL thật,
dependency fixture và hai identity test. Ghi rõ Kafka relay/peer integration chưa đạt.

### P6 — Giao event bền vững và vận hành tối thiểu

**Công việc**

- Hoàn thiện outbox relay, message key/profile version/CloudEvents/trace context,
  claim/lease, retry/backoff và trạng thái lỗi quan sát được; không đánh dấu sent
  trước khi broker xác nhận. Không tuyên bố exactly-once delivery.
- Event Traceability theo draft contract; Audit theo schema chung. Không tự phát
  nhiều alias event để đoán consumer, không tự ghi hash-chain của Audit Service.
- Test relay bằng Kafka thật trong stack test cô lập khi bật relay. Có thể tắt
  relay sandbox nhưng phải giữ PENDING và công bố chưa kiểm chứng delivery.
- Bổ sung worker kiểm tra hạn chứng nhận/review_due và ghi chuyển trạng thái/
  event idempotent; read-time guard vẫn bảo vệ khi worker trễ hoặc ngừng.
- Metrics/log tối thiểu: request/error/latency, outbox pending/failed/oldest age,
  worker failure, dependency availability. Không log credential/PII/evidence gốc.

**Kiểm chứng**: broker lỗi không làm mất outbox hoặc làm public read sập;
crash sau publish trước mark-sent cho phép duplicate có event ID ổn định;
retry không đổi payload; worker chạy lại không tạo status event vô hạn;
graceful shutdown không bỏ transaction hoặc che publish failure.

### P7 — Gói bàn giao để review R1

**Công việc**

- README, env example, migration instructions, synthetic seed chạy lại an toàn,
  smoke script và mô tả các mode/capability.
- Chạy formatting, build, vet, unit/HTTP tests, PostgreSQL integration và race
  tests phù hợp; broker tests nếu relay được bật. Test bị skip phải báo NOT_RUN.
- Hoàn thiện OpenAPI, event schema draft, mẫu request/response và test report;
  đồng bộ quyết định thực tế vào tài liệu, không giữ tuyên bố đã bỏ trong code.
- Báo cáo rollback/backup cấu hình và giới hạn: chưa restore profile v1, chưa
  xác minh EXT-10, chưa peer integration, chưa production.
- Chuẩn bị nội dung PR nhắm `dev`, checklist peer review và danh sách X còn mở.
  Tạo/push PR là bước bàn giao riêng khi được yêu cầu.

**R1 hoàn thành khi** reviewer có một bộ lệnh dựng DB/API và tái hiện được flow,
negative cases quan trọng; mọi fixture được ghi rõ; tests có kết quả thực tế;
không còn lỗi làm lộ Draft/evidence, sai QR binding, self-approve hoặc publish
không nguyên tử. Kafka delivery chỉ được ghi PASS nếu đã test với broker thật.

### P8 — Tích hợp thật theo từng peer để đạt R2

| Thứ tự | Công việc | Điều kiện đầu vào | Bằng chứng cần có |
| --- | --- | --- | --- |
| I1 | Catalog adapter thật | X-03: provider proto và phạm vi lookup được chốt | Codegen đúng package, query Product chứa SKU, not-found/non-public/history behavior |
| I2 | Inventory adapter thật | X-01/X-02: Batch identity, retention/correction và QR ownership rõ | UUID/mismatch/date/status/warehouse semantics, lô depleted và correction được kiểm chứng |
| I3 | Identity/Gateway | X-04: nhánh/contract, permission và route | Token thật, role mapping, chống header giả, public anonymous/admin protected |
| I4 | Web public | X-05: DTO/status semantics được đồng thuận | UI hiển thị manual/nullable/410 đúng, không gắn nhãn hợp lệ cho trạng thái bị chặn |
| I5 | Catalog/Content consumer | X-06: event contract có owner | Publish/suspend/certificate-status làm projection cập nhật đúng, chống duplicate/out-of-order |
| I6 | Audit ingestion | X-09: provider consumer có thật | Outbox tới Audit, dedup và field/privacy mapping; local audit không thay bằng chứng này |

Mỗi integration chạy với ref/image/config được ghi trong báo cáo. Khi source
chưa sẵn sàng, đánh dấu BLOCKED_BY_PEER hoặc NOT_RUN, vẫn giữ kết quả R1 riêng.
Fulfillment X-07, Procurement X-08, Analytics/Mobile BFF/EXT-10 là backlog sau,
không tự thêm vào điều kiện đạt backend R1 hoặc cam kết R2 chưa thống nhất.

## 4. Những phụ thuộc hiện không được giả định là đã có

Snapshot peer được kiểm tra nằm trong file quyết định, phải kiểm tra lại trước P8:

- Inventory có Batch UUID trong DB nhưng chưa công khai qua query hiện tại;
  mã Batch chưa có unique constraint, không dùng làm định danh mạnh một cách ngầm định.
- Catalog implementation có proto khác dev; GetProductVariant không trực tiếp
  trả parent Product ID; GetProduct public có danh sách SKU Active.
- Web có caller QR, nhưng nullable/verification/status chưa đầy đủ.
- Gateway mới có cấu hình; thiếu route Traceability và Identity contract hoàn chỉnh.
- Chưa thấy remote implementation riêng cho Identity, Fulfillment, Procurement,
  Audit; các service skeleton khác không được coi là provider thật.

Không vì các thiếu sót này mà mở direct DB access, tin client header, tự lấy
ngày/chứng nhận từ input không kiểm chứng hoặc chuyển mode thật sang fake.

## 5. Tiêu chí kiểm thử và truy vết yêu cầu

| Nhóm | Yêu cầu chính | Test trọng tâm |
| --- | --- | --- |
| Hồ sơ và quan hệ | FR-OCOP-11..14 | Product/SKU/Batch mismatch, unique binding, immutable history, version conflict |
| Nguồn và snapshot | FR-OCOP-05..07, phần tối thiểu 16 | Snapshot không đổi khi Supplier hiện tại đổi; DTO không lộ nội bộ |
| Chứng nhận | FR-OCOP-08..10, 17 | Scope, dates, manual vs external verification, revoke/review_due |
| Duyệt và quyền | FR-OCOP-18..23 | Required fields, self-approve, forged identity, stale approval, suspend |
| Public QR | FR-OCOP-01..04, 24 | 200/404/410, stable QR, no-store, không gọi provider mỗi scan |
| Độ tin cậy | NFR-04..09, 11 và contract chung | Transaction rollback, concurrency, idempotency, outbox recovery, privacy |

Các FR chỉ thực hiện một phần hoặc hoãn phải có nhãn PARTIAL/DEFERRED trong
test matrix chi tiết; không đánh dấu toàn FR PASS chỉ vì một scenario đã chạy.
Mục tiêu coverage theo DoD hiện tại tối thiểu 75%, đi kèm test chứng minh invariant;
coverage không thay thế kiểm thử transaction/concurrency hoặc provider thật.

## 6. Phân chia review và quản lý rủi ro

- Review thiết kế sau P0; review migration sau P2; review lát cắt admin sau P4;
  review public/QR sau P5; review độ tin cậy và bàn giao sau P6/P7.
- Không gom tất cả thành một diff lớn. Mỗi lát cắt có hành vi, tests và ghi chú
  giới hạn; không để implementation trung gian mở publish thiếu guard/outbox.
- Rủi ro chính: nhầm Batch do code không unique; source đổi sau approve; stale
  public state; role mapping giả; UI gắn badge sai; event consumer chưa có.
  Giảm rủi ro bằng stable identity gate, version/snapshot verification, read-time
  eligibility, identity fail-closed, contract matrix và tách R1/R2.
- Kế hoạch dùng thứ tự phụ thuộc, chưa cam kết ngày hoặc số ngày hoàn thành vì
  chưa có lịch nhân lực và thời điểm peer trả contract. Không gắn ngày giả vào P8.

## 7. Tracker hiện tại

| Hạng mục | Trạng thái | Điều kiện bắt đầu |
| --- | --- | --- |
| Quyết định nội bộ và kiểm tra code peer | DONE trong tài liệu trước; chưa runtime verify | Đã có snapshot nguồn |
| P0 Thiết kế chi tiết và contract | VERIFIED_R1 — thiết kế/contract và final matrix audit đã đối chiếu; OpenAPI full validation PASS | File kế hoạch này và quyết định hiện hành |
| P1 Khung chạy/config/test harness | VERIFIED — Docker/nonroot/runtime LOGIN/health/real HTTP; SIGTERM exit0 và overall-deadline tests PASS | Các ranh giới/config/mode P0 đã rõ |
| P2 Persistence/invariants | VERIFIED_R1 — migration/runtime grants/commands/dedup/workflow/public pointer/audit/outbox có PG evidence và final matrix | P0 schema và P1 harness |
| P3 Draft slice/auth | VERIFIED_R1 — signed sandbox identity, fixture, list/create/get/edit/new revision/pagination và failure-case matrix | P2 và identity fixture |
| P4 Review workflow | VERIFIED_R1 — signed HTTP + PG submit/independent decision/concurrency/rollback/clone contributor tests PASS | P3 và matrix quyền/trạng thái |
| P5 Publish/QR/public/suspend | VERIFIED_R1 — E2E HTTP, fixture source correction, atomic outbox, immutable history, public expiry/suspend/archive có evidence | P4 và transaction/outbox P2 |
| P6 Relay/expiry/operations | VERIFIED_R1 — PG lease/backoff/crash recovery/worker dedup; 3 tests broker thật; metrics/lifecycle/race PASS | Outbox/public eligibility P5 |
| P7 Review package R1 | VERIFIED_R1 — runbook/seed/smoke/Test/Race, Docker lifecycle/image, 79,0% coverage, schema validation, final audit và PR draft; CI remote NOT_RUN | P1..P6 có bằng chứng theo capability |
| P8 Peer integration R2 | NOT_DECIDED — user xác nhận nhóm chưa chốt P8 hoặc contract liên service; chưa bắt đầu | Nhóm chốt phạm vi/owner và X-01..X-06/X-09 theo từng peer |

Mỗi lần hoàn thành một phần, cập nhật tracker bằng commit/thay đổi và lệnh/kết
quả kiểm chứng. Không đổi TODO thành DONE dựa trên mô tả dự kiến trong kế hoạch.

Xác nhận từ người dùng ngày 2026-10-17: “chưa quyết định P8, nhóm chưa contrac
các service với nhau”. Vì vậy P8 là hướng tích hợp dự kiến để nhóm thảo luận,
chưa phải cam kết triển khai đã được các owner thống nhất. R1 có thể review độc
lập; chưa mở integration mode hoặc hiện thực adapter theo giả định contract.

# Traceability quyết định triển khai và đối chiếu code các service

Ngày rà soát: 2026-10-17. Nhánh làm việc: `feat/traceability-service`.
Nền: `dev@2291ef5a44dd3a6f7985297903bfec01b325b087`.

Người dùng đã giao quyền chọn phương án nội bộ theo đề xuất trong cuộc trao đổi,
và yêu cầu kiểm tra code service khác trước khi đặt câu hỏi cho team. Các mục
**CHỐT NỘI BỘ** bên dưới là quyết định cho phiên bản đầu của Traceability.
Các mục **CẦN PHỐI HỢP** là đề nghị đối với contract bên ngoài, chưa được xem
là cam kết của chủ service đó. Không thay code, protobuf hay cấu hình của peer.
Ở thời điểm lập snapshot quyết định này chưa triển khai code Traceability hoặc
chạy peer stack. Hiện backend R1 đã triển khai và có kiểm chứng local; xem
[đối chiếu cuối](../../../04_testing/traceability-service/04_final_r1_audit.md).
Chưa commit/push/PR hoặc kiểm chứng peer runtime. Ngày 2026-10-17 đã kiểm tra lại
danh sách nhánh GitHub: các implementation refs dưới đây chưa thay đổi.

## 1. Kết luận và cách sử dụng

Có thể bắt đầu thiết kế chi tiết và code một phiên bản chạy độc lập để review:
**tạo hồ sơ Batch → revision → gửi duyệt → duyệt → publish → quét QR → đình chỉ**.
Không cần chờ toàn bộ 18 service. Tích hợp thật chưa thể tuyên bố hoàn tất vì
contract định danh Batch còn thiếu, Identity chưa có implementation trên remote,
Gateway chưa route Traceability, và một số consumer chỉ mới được mô tả trong docs.

Đọc file này thay cho việc coi toàn bộ 14 mục trong readiness review là blocker.
Quyết định nội bộ ở đây cụ thể hóa các đề xuất cũ; không ngầm phê duyệt thay đổi
contract chung hay tuyên bố hoàn thành toàn EPIC 19.

## 2. Phương pháp và snapshot nhánh

Đã `git fetch origin --prune`, đối chiếu danh sách remote bằng `git ls-remote
--heads origin`, đọc trực tiếp blob qua `git show` và tìm symbol bằng `git grep`.
Không checkout các nhánh peer, không merge/cherry-pick mã của họ. Kết luận code
là static inspection tại các commit sau, chưa phải kiểm thử runtime:

| Nhánh remote | Commit được đọc | Vai trò |
| --- | --- | --- |
| `feature/catalog-service` | `db95c369bce3ed91f62b4d562c9825b1140ecf58` | Catalog đang có implementation |
| `feat/inventory-service` | `2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f` | Inventory implementation |
| `feature/ui-web` | `b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e` | Web public, API proxy, kiểu dữ liệu QR |
| `feature/mobile-ui` | `8e1994a7401a5c4759cfc24149182b5e2148b6be` | Kiểm tra caller Mobile |
| `feat/content-service` | `50f35e5f753ad67234501aa95e11c3013f553f42` | Content boundary, identity verifier |
| `feat/profile-service` | `7c1754d11490bd4cb747212e60d6630de256e06c` | Ví dụ auth nội bộ đã hiện thực |
| `feat/order-service` | `c40d06a461c99548312dbfb87545621b67a28901` | Kiểm tra nhu cầu peer và Gateway |
| `feat/order-service-spGW` | `c19bd2f8f2481f69742fc2c43f7d8999a83a4c20` | Cấu hình Kong bổ sung |

Nhánh `catalog-service@4587b76` chỉ ở checkpoint tài liệu, không phải nhánh
implementation `feature/catalog-service`. Remote chưa có `feat/traceability-service`;
nhánh Traceability hiện tại mới được tạo local theo yêu cầu trước đó.

## 3. Tám nhóm quyết định cho phiên bản đầu

### 3.1. Phạm vi — CHỐT NỘI BỘ

Làm backend REST, PostgreSQL và tài liệu/tests cho hồ sơ **cấp Batch**. Mỗi hồ sơ
gắn một Product, một SKU và một Batch của Inventory. Nhiều hộp cùng Batch có thể
dùng chung QR. Hồ sơ Product/SKU độc lập, QR từng hộp, chống hàng giả bằng serial,
genealogy nguyên liệu nhiều cấp, hẹn giờ publish và dashboard scan để sau.

Có dữ liệu nguồn, nhà sản xuất/quy trình và chứng nhận; nhân viên có quyền kiểm
tra thủ công dựa trên bằng chứng. Ghi `verification_method=MANUAL`, nguồn và thời
điểm; không diễn đạt như đã được EXT-10 xác minh. Không tự suy ra nguồn nguyên
liệu từ tên Supplier, mã SKU hay chứng nhận Catalog.

Phần tối thiểu của US-OCOP-06 được đưa vào scope này vì publish cần duyệt.
US-OCOP-05 nâng cao và tích hợp Procurement/EXT-10 tự động được hoãn. Không
đổi release plan của toàn nhóm; ghi rõ giới hạn khi review PR.

### 3.2. Vòng đời và revision — CHỐT NỘI BỘ

Tách trạng thái revision khỏi trạng thái vận hành của hồ sơ/QR:

- Revision: `DRAFT → IN_REVIEW → APPROVED → PUBLISHED → ARCHIVED`;
  `IN_REVIEW → REJECTED`. Nội dung sau submit được đóng băng; sửa tạo revision
  Draft mới, không ghi đè bản đã submit/approved/published.
- Hồ sơ: `ACTIVE`, `SUSPENDED`, `ARCHIVED`. Draft chưa có public pointer dù hồ sơ
  Active. Suspend chặn toàn bộ public pointer, không chỉ đánh dấu revision cũ.
- Một hồ sơ tối đa một Draft đang làm và một revision public hiện hành; có thể
  tiếp tục hiển thị bản cũ khi chuẩn bị bản mới.
- Approve không tự publish. Publish ngay theo lệnh hợp lệ, không hẹn giờ trong v1.
- Publish bản mới chuyển public pointer và lưu lịch sử bản cũ trong cùng transaction.
  Snapshot nội dung/bằng chứng cũ giữ nguyên; thay đổi trạng thái là metadata/audit.
- Reject bắt buộc lý do; có thể clone thành Draft mới để chỉnh sửa và duyệt lại.
- Suspend bắt buộc lý do. V1 chưa có restore: không được tạo/publish revision mới
  để lách đình chỉ; endpoint khôi phục được hoãn và phải thiết kế riêng trước khi mở.
- Archive giữ lịch sử và QR không tái sử dụng; không có xóa cứng dữ liệu đã phát hành.

| Lệnh | Guard chính | Tác động |
| --- | --- | --- |
| Create/Edit Draft | Quyền edit, hồ sơ không archive/suspend, expected version đúng | Lưu Draft và lịch sử thay đổi |
| Submit | Draft đủ dữ liệu, bằng chứng, quan hệ đối tượng đã xác minh trong mode tương ứng | Đóng băng nội dung và ghi submit |
| Approve | In review, reviewer không tham gia biên soạn revision, expected version đúng | Quyết định duyệt có actor/time |
| Reject | In review, quyền approve, reason không rỗng | Quyết định từ chối bất biến |
| Publish | Approved, hồ sơ Active, chứng nhận/verification còn hợp lệ, kiểm tra dependency đạt | Đổi public pointer + audit + outbox nguyên tử |
| Suspend | Có quyền suspend, reason không rỗng | Chặn public ngay + audit + outbox |

### 3.3. Quyền và identity — CHỐT NỘI BỘ, mapping ngoài CHƯA CHỐT

Danh mục permission nội bộ: `traceability.read_internal`, `.create`, `.edit`,
`.submit`, `.approve`, `.publish`, `.suspend`, `.archive`, `.read_evidence`.
Tên đầy đủ đều bắt đầu `traceability`; không suy rộng quyền read sang edit/publish.

Fixture role cho review độc lập: EDITOR có create/edit/submit/read_internal;
APPROVER có read_internal/approve/publish/suspend/archive/read_evidence.
EDITOR cần quyền evidence riêng nếu upload/xem bằng chứng. Ghi tập actor đã sửa
nội dung revision để reviewer không phải bất kỳ contributor nào của revision đó;
publisher được phép là reviewer. Không có ADMIN bypass điều kiện tự duyệt.
Đây là role **local/test**, không tự gán ACT-12/ACT-15 hay role Content cho production.

Public GET không cần đăng nhập. Admin phải qua verifier xác minh chữ ký, issuer,
audience, expiry và permission hoặc trusted gateway context được xác thực.
Không tin header `X-User-*` độc lập. Verifier chưa cấu hình phải từ chối admin.
Review sandbox có thể dùng token ký bằng khóa test riêng, chỉ bind loopback, không
commit secret và không chấp nhận token này khi đổi sang integration/production.

### 3.4. Dữ liệu, bằng chứng và tính nhất quán — CHỐT NỘI BỘ

Model dự kiến: `traceability_profiles`, `traceability_revisions`,
`revision_certifications`, `evidence_references`, `review_decisions`, `qr_bindings`,
`dependency_verifications`, `audit_entries` (local history), `outbox_events`,
`idempotency_records`. Inbox chỉ bổ sung khi có consumer thật trong scope.

UUID cho định danh nội bộ, revision tăng trong từng profile, không FK xuyên DB.
Batch binding bất biến sau cấp QR; sửa nhầm đối tượng phải suspend/retire binding
và lập hồ sơ/mã mới, không chuyển mã cũ sang Batch khác.

- Draft cho phép thiếu dữ liệu nhưng phải đúng kiểu/giới hạn và không chứa quan
  hệ đã biết là sai. Submit/publish yêu cầu Product/SKU/Batch xác minh, tên sản
  phẩm snapshot, producer/workshop, nguồn nguyên liệu, quy trình, evidence và
  chứng nhận OCOP được kiểm tra thủ công phù hợp phạm vi.
- Chứng nhận có type/number/issuer/scope/issued_on/valid_until nếu có, OCOP stars,
  verification method/time và evidence. Stars phải lấy từ chứng nhận phù hợp,
  không copy mặc định 4 sao. Không có ngày hết hạn không có nghĩa hiệu lực vô hạn:
  cần ghi loại hiệu lực `NO_EXPIRY_STATED` và `review_due_at` do reviewer xác nhận;
  đến hạn rà soát phải chặn biểu diễn là còn xác nhận cho tới khi review lại.
- Không có chứng nhận ATTP công khai thì trả trạng thái chưa có; không bịa mã
  hoặc lấy thông tin nhân sự VSATTP làm chứng nhận sản phẩm. Dữ liệu chưa xác
  nhận không được render như chứng nhận hợp lệ.
- Chứng nhận hết hạn/thu hồi hoặc đến hạn review làm public không còn đủ điều
  kiện trả successful-valid response; kiểm tra cả lúc đọc, không chỉ chờ cron.
- Ngày chứng nhận dùng ngày lịch Asia/Ho_Chi_Minh; `valid_until` bao gồm ngày đó,
  hết hiệu lực từ đầu ngày kế tiếp. Timestamps nghiệp vụ lưu UTC. NSX/HSD của
  Batch giữ semantics nguồn Inventory, không tự diễn giải lại HSD của họ.
- Nguồn/Producer/Supplier được snapshot theo revision; Supplier hiện tại thay đổi
  không sửa lịch sử. Contract, giá mua, PII và file gốc không nằm trong public DTO.
- Media v1 lưu reference/evidence metadata; chưa tự xây upload/object storage.
  Evidence nội bộ không trả qua public API. Public media phải được đánh dấu và
  duyệt riêng; không dùng arbitrary URL do client gửi để backend tự tải.
- Dùng optimistic version + DB transaction cho command. Publish đổi pointer,
  ghi local audit và outbox cùng transaction; một bước lỗi thì rollback tất cả.
  Cùng idempotency key/body replay cùng kết quả; key giống nhưng body khác → 409.
  Namespace key theo principal/action; giữ record suốt đời hồ sơ trong v1,
  không tự reuse key khi hết cache.

### 3.5. Định danh và xác minh Product–SKU–Batch

**CHỐT NỘI BỘ:** nhận `product_id` (UUID Catalog), `sku_code` và định danh Batch
authoritative. Tách port CatalogVerifier và BatchVerifier; kết quả giữ source,
source reference/version nếu có, checked_at, raw date semantics và snapshot hash.
Kiểm tra lại khi submit và publish; source thay đổi so với nội dung đã duyệt thì
không tự sửa snapshot để publish, trả conflict yêu cầu revision/duyệt lại.

Catalog hiện có đường kiểm tra một Product public chứa SKU cụ thể (mục 4.1).
Inventory hiện có đường lookup theo SKU + batch_code nhưng chưa có stable public
batch_id/uniqueness (mục 4.2). Không coi cặp code tạm thời là identity bất biến
của lô production; không đọc trực tiếp DB Inventory để bù thiếu contract.

Integration mode: chưa có Batch identity contract đủ chắc thì cho chuẩn bị Draft,
nhưng submit/publish cần xác minh phải từ chối `DEPENDENCY_CONTRACT_UNAVAILABLE`.
Sandbox dùng fixture stable Batch ID và mô phỏng cả mismatch/not-found/timeout;
mọi profile tạo bằng fixture được gắn sandbox, không được chuyển sang integration
hay export như hồ sơ nguồn gốc thật. Không tự downgrade lỗi provider sang fake.

### 3.6. QR — CHỐT NỘI BỘ; kích hoạt qua Packing hoãn

QR v1 cấp theo Batch, là token ngẫu nhiên ít nhất 128 bit, base64url không padding
hoặc hex; không chứa SKU/batch/UUID nội bộ, phù hợp SAFE_SEGMENT Web tối đa 160 ký tự.
Một Batch binding một QR canonical; nhiều hộp có thể in cùng QR. QR không chứng
minh một hộp là hàng chính hãng và không có chức năng chống sao chép trong v1.

Định dạng URL trang: `{PUBLIC_WEB_BASE_URL}/trace/{qr_code}` để khớp Web hiện tại.
Backend tra cứu: `GET /api/v1/trace/{qr_code}`. Không dùng URL do Inventory đang tự
ghép làm nguồn sở hữu QR. Chưa tự thêm alias `/batch/{batch_code}` có thể nhập nhằng.

Tạo mã ở Draft có thể phục vụ preview nội bộ nhưng chưa public; publish hợp lệ
kích hoạt public. QR giữ nguyên khi đổi revision cùng Batch. QR đình chỉ/archive
không được tái cấp. Tự động kích hoạt theo PackingCompleted là integration giai
đoạn sau, phải thống nhất Fulfillment trước khi bật; không gọi ngược Fulfillment
đồng bộ mỗi lần khách quét.

### 3.7. API và event — hướng API chốt nội bộ, consumer contract còn mở

Giữ public path Web đang gọi. Các admin endpoint dự kiến dưới
`/api/v1/admin/trace/records`:

| Method/path suffix | Chức năng |
| --- | --- |
| GET /, POST / | Danh sách theo quyền; tạo hồ sơ Draft |
| GET /{id} | Chi tiết nội bộ và revision hiện hành |
| POST /{id}/revisions | Tạo revision Draft từ bản trước |
| PATCH /{id}/revisions/{revision} | Sửa Draft với expected version |
| POST /{id}/revisions/{revision}/submit | Gửi duyệt |
| POST /{id}/revisions/{revision}/approve hoặc /reject | Quyết định có lý do |
| POST /{id}/revisions/{revision}/publish | Công khai ngay |
| POST /{id}/suspend hoặc /archive | Chặn public/retire có lý do |

Public: 200 chỉ khi profile/revision/chứng nhận đủ điều kiện; 404 cho token không
tồn tại hoặc chưa từng công khai (không lộ Draft); 410 cho mã đã công khai nhưng
đình chỉ/archive/chứng nhận không còn đủ điều kiện. Response 410 chỉ có mã lý do
an toàn và thông điệp, không đính kèm nội dung cũ như vẫn hợp lệ. Public v1
`Cache-Control: no-store`; chưa có CDN cache để tránh stale sau suspend.
Trong phiên bản public v1, Batch hết HSD theo timestamp/semantics nguồn hoặc
được ghi nhận cách ly cũng trả 410 với reason riêng `BATCH_EXPIRED`/
`BATCH_RESTRICTED`; không xóa hồ sơ lịch sử và không gọi đây là chứng nhận hết hạn.
Batch hết tồn không tự chặn truy xuất nếu vẫn đủ điều kiện khác. Màn hình xem
lịch sử public với cảnh báo đầy đủ có thể bổ sung khi Web hỗ trợ, thay vì trả 200
cho UI hiện luôn gắn nhãn hợp lệ. Known source correction phải chặn publish hoặc
đưa hồ sơ công khai vào review/suspend, không tự ghi đè revision đã duyệt.
Không làm read-only GET phụ thuộc đồng bộ vào EXT-10/Catalog/Inventory; snapshot
có `verified_at`, còn việc thay đổi nguồn/thu hồi cần quy trình update/đình chỉ riêng.
Không tuyên bố snapshot có thể phát hiện ngay việc peer âm thầm sửa Batch.

Admin: 400 validation, 401 identity, 403 permission, 404 ngoài phạm vi/không có,
409 version/state/idempotency conflict, 503 dependency tạm lỗi hoặc contract chưa
sẵn sàng. Command thành công trả ID/revision/version và kết quả; không return 200
cho lệnh chưa được thực thi. Error envelope dùng `error_code`, `user_message`,
`request_id`, metadata an toàn, không trả SQL/stacktrace/secret.

Public DTO giữ tên các trường Web hiện có: qr_code, product_name, batch_code,
ocop_star, ocop_certificate_no, production_date, expiry_date, workshop_location,
artisan_name, food_safety_cert, raw_materials_origin, crafting_video_url.
Đồng thời cần thông tin `scope=BATCH`, revision, verification_method/verified_at,
certification status và trạng thái Batch. Trường chưa có không được điền nội dung
giả. Artisan/ATTP/media không bắt buộc ở domain nếu chưa có bằng chứng: dùng
nullable/structured status trong target contract, cần Web cập nhật trước khi
tuyên bố tương thích đầy đủ với interface hiện yêu cầu string (mục 4.3).

Event draft: topic `traceability.events.v1`, partition key profile_id, CloudEvents
1.0; types dự kiến `vn.omama.traceability.profile.published.v1`,
`vn.omama.traceability.profile.suspended.v1`, `.archived.v1`,
`vn.omama.traceability.certification.status.changed.v1`.
Payload gồm profile/revision/scope, public status, public QR reference, product_id,
sku_code, Batch reference sau khi chốt, occurred_at, aggregate_version và status
chứng nhận; không có Supplier contract, PII hoặc evidence file gốc.
Đây là **đề nghị contract**, chưa có Catalog/Content consumer xác nhận; không phát
đồng thời nhiều alias event cũ để đoán consumer. Audit theo `audit.events.v1` và
schema chung hiện có, cần owner Audit xác nhận mapping khi có implementation.

### 3.8. Runtime, review và tests — CHỐT NỘI BỘ

Go, Clean Architecture, REST `net/http`, PostgreSQL 16, cổng mặc định 8003;
dependency versions sẽ pin khi khởi tạo module, không copy generated proto cũ
trên dev nếu không khớp provider. Migration có version, up/down dành riêng test;
không tự chạy destructive down/reset trong lệnh start. DB riêng `om_traceability_db`,
runtime DB role riêng, cấu hình connection bằng env; compose của service có thể
chạy độc lập, không sửa/cướp container hoặc volume của peer.

Chế độ sandbox cho review dùng Catalog/Batch fixture rõ nhãn và issuer test.
Chế độ integration chỉ dùng adapter thật được cấu hình, thiếu verifier thì từ
chối admin, thiếu provider contract thì chặn command cần xác minh. Chưa mở
production mode cho đến khi identity/Batch/public contract được kiểm chứng.

Health: `/livez` cho process; `/readyz` kiểm tra DB/schema và cấu hình bắt buộc
theo mode. Kafka relay lỗi hiển thị backlog/degraded và retry; public đọc DB không
bị chặn chỉ vì Kafka/Audit consumer tạm ngừng. Không dùng LogPublisher đánh dấu
event PUBLISHED trong integration. Relay sandbox có thể disabled rõ ràng và
để outbox PENDING, không giả đã giao event.

Review cần README, env example không secret, migration, synthetic seed, OpenAPI,
event schema draft, dependency capability matrix và script smoke/test. Tests:

1. Unit cho lifecycle, required fields, certificate dates/scope và quyền.
2. PostgreSQL thật cho uniqueness/public pointer, immutable content, rollback
   publish+audit+outbox, CAS, concurrent publish và idempotency.
3. HTTP auth tests: token giả/expired/wrong issuer/audience, tự gán role header,
   self-approve, đọc chéo scope và public không lộ Draft/evidence.
4. Sandbox E2E hai actor: create → submit → approve → publish → QR → sửa tạo bản
   mới → suspend; public 404/410; snapshot Supplier cũ không bị overwrite.
5. Peer contract tests compile against exact provider proto; fixture pass không
   được tính là provider integration pass. Sau khi provider sẵn sàng mới chạy
   roundtrip thật và cập nhật capability matrix.
6. Outbox retry/duplicate/out-of-order, Kafka thật khi relay được bật; publisher
   crash sau gửi trước ack không được làm mất state, consumer phải chống lặp.

## 4. Code peer đã trả lời được gì

### 4.1. Catalog — CÓ NHÁNH, trả lời một phần

Nguồn immutable:
[Catalog public proto](https://github.com/hvduong2k5/DUT.PBL6/blob/db95c369bce3ed91f62b4d562c9825b1140ecf58/packages/proto/catalog/v1/catalog_public.proto),
[service implementation](https://github.com/hvduong2k5/DUT.PBL6/blob/db95c369bce3ed91f62b4d562c9825b1140ecf58/services/catalog-service/src/main/java/com/hvduong/catalog/application/service/impl/CatalogServiceImpl.java),
[SKU migration](https://github.com/hvduong2k5/DUT.PBL6/blob/db95c369bce3ed91f62b4d562c9825b1140ecf58/services/catalog-service/src/main/resources/db/migration/V3__create_product_variants.sql).

- `product_id` và variant ID là UUID; `sku_code` UNIQUE toàn Catalog ở DDL.
- Provider thực tế là `hvduong.catalog.v1.CatalogPublicService` (Java), khác
  `omamx.catalog.v1.CatalogService` trong proto trên dev. Không wire-compatible
  chỉ vì method có tên gần giống; codegen phải dùng đúng proto/package provider.
- `GetProduct(id_or_slug)` → `getPublicProductDetail` trả product_id, version,
  dữ liệu sản phẩm và danh sách **SKU ACTIVE** của Product public.
  Có thể kiểm tra một product_id đã biết chứa sku_code đã chọn, không cần sửa DB.
- `GetProductVariant(sku_code)` có thật nhưng response không có product_id;
  không phải contract direct SKU → parent Product như proto cũ trên dev.
- SKU suspended/archived không có trong public product detail. Lưu hồ sơ mới
  cho SKU lịch sử chưa có đường đọc rõ ràng đủ dùng. Snapshot đã public cần
  được giữ khi catalog ngừng bán, không tự mất QR vì Product bị ẩn.
- `ocop_star` và certificate number có trong Catalog nhưng không có evidence/
  manual verification; không dùng làm chứng nhận authoritative của Traceability.
- Tìm implementation Kafka chỉ thấy stock projection; Traceability consumer
  mới có trong `phan-tich.md`, chưa thấy listener xử lý publish/suspend.

**Không cần hỏi lại:** SKU uniqueness, Product ID type, có lookup Product public
chứa SKU hay không. **Cần phối hợp:** chọn provider proto làm contract chung,
lookup SKU/Product lịch sử và event badge khi publish/suspend/expire.

### 4.2. Inventory — CÓ NHÁNH, thiếu identity contract công khai

Nguồn:
[DDL](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/migrations/000001_init_inventory_schema.up.sql),
[repository](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/infrastructure/postgres/repository.go),
[gRPC handler](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/presentation/grpc/handler.go),
[proto](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/packages/proto/inventory/v1/inventory.proto).

- DB có Batch `id UUID PRIMARY KEY`, SKU FK, supplier_id, mfg_date, exp_date và
  status. **Không có UNIQUE batch_code hoặc UNIQUE(sku,batch_code)** trong migration.
- `GetBatchFEFODetails(sku_code)` đã hiện thực. Repo `GetBatchesBySKU` không lọc
  số lượng hoặc status, vì vậy code hiện trả được cả depleted/expired/quarantine;
  đây là static behavior, không phải retention guarantee được owner xác nhận.
- Response có batch_code, dates, remaining quantity, quality và QR field;
  không có Batch UUID, supplier_id, source version hay parent Product ID.
- `warehouse_id` có trong request proto nhưng handler hiện chỉ truyền SKU xuống
  use case; không được giả định đã có lookup lọc theo kho.
- Handler tự ghép `ocop_trace_qr_code = https://trace.omama.vn/batch/{batch_code}`;
  không có query Traceability hoặc actual QR binding. Đó không phải bằng chứng
  QR đã phát hành/approved. Field hiện chứa URL, không phải opaque code.
- QualityStatus map không biểu đạt mọi raw trạng thái độc lập (depleted kiểm tra
  trước expired/quarantine), không dùng quality enum đó để xác minh chứng nhận.

**Không cần hỏi lại:** Batch nằm ở Inventory, UUID tồn tại trong DB, có dates,
query hiện không loại lô hết tồn. **Cần phối hợp:** public lookup ổn định bằng
Batch UUID (hoặc bảo đảm uniqueness khác được owner chọn), trả SKU/dates/raw
status/version/warehouse nếu cần; retention và correction semantics; xử lý field
QR đang tự ghép. Traceability không tự thêm migration/endpoint vào Inventory.

### 4.3. Web và Mobile — có Web caller, chưa thấy Mobile Traceability

Nguồn:
[Web page](https://github.com/hvduong2k5/DUT.PBL6/blob/b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e/apps/web-user/src/app/trace/%5BqrCode%5D/trace-experience.tsx),
[proxy](https://github.com/hvduong2k5/DUT.PBL6/blob/b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e/apps/web-user/src/app/api/customer-core/%5B...path%5D/route.ts),
[DTO](https://github.com/hvduong2k5/DUT.PBL6/blob/b1ff2918c7297fc93d0cb9a15cc1d0a0ee46b72e/apps/web-user/src/lib/customer-core/types.ts).

- Trang thực tế `/trace/[qrCode]`, khác route story trong Use Case cũ.
- Browser gọi `/api/customer-core/trace/{code}`; Next proxy gọi upstream
  `{CUSTOMER_CORE_UPSTREAM_URL}/trace/{code}`, default base Mockoon `/api/v1`.
  Vì vậy backend `/api/v1/trace/{code}` phù hợp caller hiện tại.
- Proxy cho public GET không yêu cầu capability/login và `cache:no-store`;
  segment chấp nhận `[A-Za-z0-9._~-]`, tối đa 160 ký tự.
- DTO là flat fields liệt kê ở mục 3.7, nhiều string bắt buộc. Mọi 200 đều render
  “Hồ sơ hợp lệ”, sao và chứng nhận; 404 có UI riêng, 410 chỉ rơi vào generic error.
  Chưa có UI explicit manual verification/suspended/expired/missing certificate.
  Đây là lý do phải thống nhất semantics public, không chỉ khớp tên field.
- Không thấy Admin Traceability workspace trong các path được kiểm tra.
- Nhánh Flutter có catalog ocopStars nhưng tìm traceability/trace/QR không thấy
  feature tra nguồn gốc. GraphQL schema trên dev là thiết kế, chưa thấy Mobile BFF
  implementation trong các remote service trees; không coi đã có Mobile tích hợp.

**Không cần hỏi lại:** URL Web, API path, DTO hiện dùng, public access và cache.
**Cần phối hợp:** nullable certificate/artisan, verification/status UI, 410,
admin workspace và thời điểm Mobile/BFF thực sự dùng contract.

### 4.4. Identity và Gateway — chưa có Identity implementation, Gateway có cấu hình

Nguồn:
[Kong](https://github.com/hvduong2k5/DUT.PBL6/blob/c19bd2f8f2481f69742fc2c43f7d8999a83a4c20/services/api-gateway/config/kong.yml),
[Profile auth](https://github.com/hvduong2k5/DUT.PBL6/blob/7c1754d11490bd4cb747212e60d6630de256e06c/services/profile-service/internal/transport/http/middleware/auth.go),
[Content verifier](https://github.com/hvduong2k5/DUT.PBL6/blob/50f35e5f753ad67234501aa95e11c3013f553f42/services/content-service/internal/transport/http/middleware/auth.go).

- Không có remote branch riêng Identity/Auth. Đã rà named service trees trên
  toàn bộ remote refs, không thấy code ngoài `.gitkeep` cho auth/identity.
  Có identity proto/docs và UI mocks không đồng nghĩa có provider đang triển khai.
- Kong có route nhiều service, JWT plugin/local dev credential và global strip
  một số header. **Chưa có route `/api/v1/trace` hoặc admin Traceability**;
  không thấy đầy đủ trusted user/permission injection/JWKS theo HLD trong file này.
  Không đưa local JWT secret trong Kong vào tài liệu mới hoặc tái dùng làm secret.
- Profile hiện dùng server-side internal token để xác thực caller trước khi tin
  User-ID/Role; Content có Verifier interface, nil verifier từ chối.
  Đây là mẫu có thể tham khảo, không phải Identity contract chung đã được chốt.

**Cần phối hợp trước tích hợp thật:** owner Auth, issuer/audience/JWKS hoặc trusted
gateway protocol, permission mapping, thu hồi quyền và route Traceability.
Không yêu cầu Identity phải xong mới viết unit/domain/DB/HTTP tests sandbox.

### 4.5. Content — CÓ NHÁNH, chưa có Traceability integration

Code Content có workflow/versioned editorial và auth fail-closed. Không thấy
consumer Traceability hay contract lấy hồ sơ nguồn gốc trong implementation đã
kiểm tra. Không copy role `CONTENT_APPROVER` làm role Traceability mặc định.
Khi cần bài viết gắn hồ sơ, thống nhất public reference và cách xử lý suspend;
không để Content sửa/chứng nhận dữ kiện nguồn gốc.

### 4.6. Peer chưa có nhánh/code để trả lời thêm

Đối chiếu tất cả remote heads và named service trees (không tính `.gitkeep`):

| Peer | Kết quả remote tại thời điểm kiểm tra | Ảnh hưởng |
| --- | --- | --- |
| Fulfillment | Không có nhánh riêng/code implementation tìm thấy | Chưa chốt được in/kích hoạt QR, seal mapping; hoãn consumer v1 |
| Procurement | Không có nhánh riêng/code implementation tìm thấy | Chưa xác minh receipt→Batch; dùng evidence manual v1 |
| Audit | Không có nhánh riêng/code implementation tìm thấy | Có schema/docs để tạo outbox; chưa chứng minh ingestion/hash-chain thật |
| Analytics | Chỉ skeleton `.gitkeep`, không có nhánh riêng | Chưa có consumer scan; không thêm tracking PII v1 |
| Notification | Chỉ skeleton, không có nhánh riêng | Không bắt buộc dispatch để hoàn thành workflow v1 |
| Shipping | Chỉ skeleton, không có nhánh riêng | Không tạo dependency trực tiếp mới cho Traceability |
| Mobile BFF | Không thấy named service implementation/nhánh riêng | GraphQL mới là design contract, chưa chứng minh runtime caller |
| Media/EXT-10 adapter | Chưa thấy adapter riêng trong code Traceability (chưa tồn tại) | Chưa có credentials/provider contract; hoãn tích hợp thật |

“Không thấy” chỉ áp dụng remote repository/ref/path đã quét; không kết luận code
trên máy riêng, chưa push hoặc repository khác không tồn tại. User có thể bổ sung
nhánh/repo nếu team đang phát triển ở nơi khác. Không gửi tin cho thành viên team.

## 5. Danh sách câu hỏi còn cần team, đã loại phần code trả lời được

| ID | Gửi owner nào | Câu hỏi còn thiếu | Chặn gì? |
| --- | --- | --- | --- |
| X-01 | Inventory | Có cung cấp Batch UUID lookup, SKU/dates/raw status/version và cam kết retention/correction không? Nếu dùng code, uniqueness scope nào được bảo đảm? | Submit/publish integration thật; không chặn sandbox |
| X-02 | Inventory + Web/Traceability | Thay URL QR tự ghép bằng actual binding/lookup thế nào, có cần compatibility alias không? | Tích hợp QR xuyên service, không chặn core workflow |
| X-03 | Catalog | Chốt dùng hvduong CatalogPublicService mới hay omamx proto trên dev? Tra Product/SKU ngừng bán qua contract nào? | Adapter thật ổn định và historical read |
| X-04 | Identity/Gateway | Nhánh/repo nào triển khai? Issuer/audience/verification, permission mapping và route public/admin là gì? | Admin integration thật |
| X-05 | Web | Có cập nhật nullable/verification/status, missing cert và 410 để tránh render sai không? | UI tích hợp đầy đủ, không chặn backend tests |
| X-06 | Catalog + Content | Consumer publish/suspend/cert-status dùng topic/type/schema nào, version/order policy nào? | Badge/content projection thật |
| X-07 | Fulfillment | Có nhánh chưa push/repo khác? QR theo Batch và seal theo Package mapping/in/activation ra sao? | Auto activation, đã hoãn v1 |
| X-08 | Procurement | Có nhánh/repo khác? Receipt–nguyên liệu–Batch được cung cấp qua contract nào? | Auto provenance, đã hoãn v1 |
| X-09 | Audit | Có nhánh/repo khác? Audit event schema consumer thực nhận và outcome/ordering/PII policy? | Audit end-to-end thật |

Không hỏi team chọn cấu trúc folder, SQL schema nội bộ, revision algorithm,
CAS/idempotency, test strategy hay luật không tự duyệt đã chốt cho Traceability.
Role mapping toàn hệ thống và payload peer vẫn là trao đổi liên service.

## 6. Ánh xạ 14 mục rà soát ban đầu sang quyết định hiện tại

| TRC | Kết quả |
| --- | --- |
| 01 MVP | Chốt Batch MVP + duyệt/publish tối thiểu; không chờ Procurement giai đoạn 3 |
| 02 Lifecycle | Chốt revision riêng và suspension cấp profile; chưa restore v1 |
| 03 QR | Chốt opaque stable Batch QR, Web /trace; activation Packing hoãn, X-02/X-07 |
| 04 Auth | Chốt permission/self-approval/verifier; mapping/identity thật X-04 |
| 05 Persistence | Chốt snapshot/public pointer/CAS/transaction/audit/outbox; cần LLD/DDL trước code |
| 06 REST | Path public đã có caller; admin là contract nội bộ mới, Web behavior X-05 |
| 07 Public schema | Batch scope giảm conflict; nullable/manual/status vẫn cần Web cập nhật |
| 08 Identity peers | Code giải quyết một phần Catalog, Inventory UUID chưa công khai; X-01/X-03 |
| 09 GoodsReceived | Receipt/genealogy chưa có implementation; manual evidence và hoãn integration |
| 10 PackingCompleted | Chưa có provider; không giả kích hoạt/duyệt bằng packing event |
| 11 Events | Có CloudEvents/Audit/outbox conventions; Traceability consumer contract X-06/X-09 |
| 12 Certificates | Chốt manual/evidence/scope/date/review_due và public validity; EXT-10 hoãn |
| 13 Public/cache/media | Chốt DTO allowlist/no-store/evidence private; UI status X-05 |
| 14 Runtime/tests | Chốt Go/PG16/8003/sandbox-integration split; chưa đổi CI/shared infra |

## 7. Điều kiện chuyển sang implementation có thể review

Tiếp theo trong nhánh này: viết LLD/ERD/DDL, OpenAPI và test matrix theo các
quyết định đã chốt, rồi triển khai từng lát cắt. Không cần đợi trả lời X-06..09
để làm core. X-01/X-03/X-04 có thể mô phỏng cho review nhưng phải để integration
gate chưa đạt, không viết README tuyên bố kết nối thật khi chưa kiểm chứng.

Trạng thái hiện tại: quyết định và bằng chứng code đã ghi; LLD/DDL/OpenAPI/tests
và implementation Traceability chưa được tạo trong đợt này. Không có peer runtime
test hoặc test pass mới được tuyên bố. Tài liệu này đủ để reviewer đánh giá phạm
vi/quyết định và biết chính xác câu hỏi còn lại, chưa thay thế báo cáo nghiệm thu code.

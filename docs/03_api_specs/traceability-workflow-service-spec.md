# Đặc tả Traceability: hồ sơ theo lô, scope kho và tem QR

Version thiết kế: 0.4.0-draft · Đọc cùng [index](traceability-workflow-README.md). Checkpoint27/11/2026: phần backend Traceability đã được triển khai sau feature flag; [báo cáo kiểm thử và cách bật](../04_testing/traceability-service/09_grpc_transport_2026-11-27.md) phân biệt PostgreSQL/HTTP/TLS gRPC local đã kiểm với peer/UI chưa tích hợp. Contract vẫn draft, chưa promote stable.

## 1. Ranh giới và dữ liệu authoritative

Traceability sở hữu Profile, Revision, Certificate/Evidence metadata, contributor/decision, public pointer, QR token, local audit/outbox và eligibility. Không tạo lô, cộng tồn, đánh dấu đã giao, thanh toán hoặc phê duyệt thay provider. Không SQL/FK xuyên database. Public scan và internal resolver chỉ đọc snapshot Traceability, không gọi Inventory/Catalog/Identity trong lượt public/resolver.

| Dữ liệu | Nguồn | Quy tắc |
| --- | --- | --- |
| Product UUID ↔ SKU, version | CatalogService.GetTraceabilityBinding gRPC; GetProductVariant bootstrap khi chỉ biết SKU | Bootstrap không phụ thuộc Inventory SKU detail/public price; create/submit/publish vẫn kiểm binding chính xác |
| Batch UUID, SKU, mã lô, NSX/HSD, restricted/depleted, version, warehouse | Inventory BatchSource | Warehouse không tin từ UI/JWT resource claim; không dùng mã lô làm ID |
| Nhân viên, active status, permission + scope | Keycloak JWT + Identity gRPC `GetCurrentStaffGrants` | Chữ ký/issuer/audience/thời hạn hợp lệ và intersection quyền hiện hành |
| Nguồn nguyên liệu, cơ sở/quy trình, chứng nhận | Nội dung nhân viên nhập và reviewer kiểm thủ công | Hiển thị MANUAL; không diễn đạt là cơ quan bên ngoài đã xác minh tự động |
| Lô thuộc đơn | Fulfillment sealed proof qua Order | Traceability resolver không khẳng định quyền sở hữu đơn; Order chịu trách nhiệm trước gọi resolver |

Binding `(product_id, sku_code, source_namespace, batch_id)` bất biến. Unique DB hiện có `(source_namespace, batch_id)` giữ nguyên, một lô không có hai profile dù product/SKU caller khác. Namespace integration chỉ `inventory`; sandbox `fixture` dùng database/mode tách biệt, không đưa fixture lên integration. UUID canonical lowercase; SKU giữ nguyên case, không trim để biến input sai thành một mã khác.

## 2. Actor và kiểm permission/scope

Mỗi thao tác nội bộ yêu cầu `traceability.read_internal` cùng quyền hành động trên **cùng resource warehouse**. Có thể kết hợp grant GLOBAL/WAREHOUSE cho từng permission nếu cả hai đều bao phủ resource; không chỉ intersection text, không tạo cross-product giữa permission và scope của các grant khác nhau. Tuple hiệu lực phải hiện diện/được bao phủ trong token và RPC current grants Identity. Bỏ grant DEPARTMENT hoặc không biết, không nâng thành GLOBAL.

| Thao tác | Quyền thêm ngoài read_internal | Guard |
| --- | --- | --- |
| List/get/lookup/context theo lô; tải/in trên UI | Không | Filter quyền tại SQL; in còn yêu cầu public eligible |
| Create | create | Kho từ Inventory; full binding khớp Catalog/Inventory |
| Edit/new revision | edit | Đúng kho của profile; state/version/contributor policy |
| Submit | submit | Recheck source và kho; content đầy đủ |
| Approve/reject | approve | Reviewer không thuộc contributors; lý do bắt buộc |
| Publish | publish | Đã approved; recheck source/kho; không tự bỏ qua review |
| Suspend/archive | suspend/archive | Lý do bắt buộc; không cần live Inventory để dừng công khai khẩn cấp |
| Đọc evidence.reference | read_evidence | Thêm read_internal, scope đúng; thiếu quyền trả reference rỗng, không làm thay đổi dữ liệu lưu |

Một người vừa có edit vừa approve vẫn không tự duyệt nội dung mình đóng góp. Publisher có thể là reviewer. Không có role-name bypass. Token invalid → 401; verified identity thiếu permission/scope hoặc inactive employee → 403; Identity RPC gián đoạn/không có hoặc credential service sai →503; staff token invalid/actor forbidden theo ErrorInfo →401/403. Không fallback /users/me REST hoặc chỉ dùng token stale. Một actor có quyền read nhưng thiếu create không hiện CTA tạo mới; receipt đã thành công vẫn giữ kết quả.

Existing profile: scope dựa trên warehouse binding do server đã xác minh và lưu. Reads thông thường dùng binding đó, không gọi Inventory mỗi lần; binding là quyền của lô gốc, không là ảnh chụp vị trí hiện thời. Inventory phải bảo toàn ý nghĩa kho sở hữu trong đời UUID của candidate contract. Create/lookup/submit/publish xác minh nguồn trong budget 2 giây; source warehouse khác đã lưu → 409 WAREHOUSE_BINDING_CONFLICT, không tự chuyển scope. Việc giao chuyển kho/đa kho cùng UUID là feature ngoài scope; suspend/archive vẫn dùng binding cũ. Không khẳng định rollback được transfer phía provider.

List: tính tập warehouse thỏa read_internal, đưa vào WHERE trước pagination. GLOBAL được bao phủ mọi warehouse; chưa backfill chỉ GLOBAL được nhìn thấy. Scoped cursor ký HMAC, bind actor + effective authorization digest + sort boundary; đổi quyền/scope → 400 INVALID_CURSOR, client đọc trang đầu. Cursor ≤256 ký tự: payload chỉ chứa created_at/id, HMAC ký cả payload lẫn actor và authorization digest được server tính lại, không nhét toàn bộ grants vào token. Mỗi request vẫn resolve current grants. Detail tồn tại ngoài scope → 403, không body profile. Lookup by-batch kiểm kho từ Inventory trước cả kết quả empty; bản draft không lộ bằng lookup public.

## 3. State machine và command guards

Profile: ACTIVE, SUSPENDED, ARCHIVED; không restore ở phiên bản này. Revision: DRAFT → IN_REVIEW → APPROVED → PUBLISHED; IN_REVIEW → REJECTED. Khi publish revision mới, bản public cũ thành ARCHIVED về metadata, nội dung vẫn bất biến. Một working revision tối đa (DRAFT/IN_REVIEW/APPROVED), một public pointer tối đa.

| Command | Input/state | Kết quả transaction |
| --- | --- | --- |
| Create | Binding mới, content draft đúng shape | Profile ACTIVE version=1, revision=1 DRAFT, QR cấp một lần, audit, idempotency result |
| Edit | ACTIVE, DRAFT, expected_version khớp | Full replacement content, contributor union, version+1, audit/result |
| New revision | ACTIVE, from_revision tồn tại; nếu có working revision phải có reason | Archive working metadata, clone content/contributors; không đổi public pointer |
| Submit | ACTIVE, DRAFT, đủ content/certificate, source eligible | Freeze content và dependency snapshot, IN_REVIEW, version+1, audit/result |
| Approve/reject | ACTIVE, IN_REVIEW, actor độc lập, reason | Decision bất biến, APPROVED/REJECTED; không tự public |
| Publish | ACTIVE, APPROVED, cert/source/kho vẫn hợp lệ | Public pointer/revision + audit + outbox + result commit nguyên tử |
| Suspend/archive | State cho phép theo baseline, reason, version | Status mới, audit/outbox/result; QR không tái sử dụng |

Peer query trước transaction. Repository lock profile, kiểm lại expected_version và content/source hash trước write. SourceHash bỏ CheckedAt nhưng gồm warehouse_id mới. Source version/content/kho đổi sau review → 409, dùng new revision + submit/review lại; không sửa approved snapshot tại chỗ. Không thể đảm bảo source chưa đổi ngay sau network response; public biểu thị source_verified_at, không diễn đạt live verification.

## 4. Content và chứng nhận

Baseline field rules ở `internal/domain/model.go` và contract service v1 là tham chiếu cho nội dung; target không nới validation để dễ demo.

| Nhóm | Draft | Submit/publish |
| --- | --- | --- |
| Tên sản phẩm/producer/workshop/process | Có thể rỗng; giới hạn lần lượt 250/250/1000/8000 ký tự | Không rỗng; ít nhất một ingredient có name/origin |
| Collections | ingredients ≤20, certificates ≤20, evidence ≤50; array rỗng là [] | Ít nhất một OCOP hợp lệ; thiếu không được submit |
| Certificate | UUID duy nhất, type OCOP/ATTP/OTHER; number/issuer/scope/ref hợp lệ; evidence IDs tồn tại | Tất cả cert phải đúng scope Product/SKU/Batch, không revoked, đã issued; MANUAL verified_at không ở tương lai |
| OCOP | stars 1–5; non-OCOP không có stars | Không suy sao hoặc certificate number từ Catalog/marketing |
| Hạn chứng nhận | valid_until hoặc no_expiry_stated + review_due_at, không đồng thời hai kiểu | valid_until inclusive ngày Việt Nam; review_due_at hết ngay tại instant |
| NSX/HSD/mã lô | Không nhập bản sao authoritative trong content | Lấy từ Inventory snapshot; lô hết tồn không tự mất eligibility |
| Media/evidence | Giữ schema evidence; Media flag yêu cầu document:<fileId> cho chứng nhận OCOP | Public DTO không trả evidence nội bộ. Media upload-session/complete theo contract đã merge; validate trước submit/publish và durable bind sau commit |

Các optional artisan/ATTP/video hiển thị null/ẩn khi thiếu; không điền mẫu. Video URL legacy chỉ allowlist hiện có; UI mới không mở form nhập URL tùy ý; upload bản scan OCOP qua Media REST/direct storage theo guide 0.4.0. Evidence submit tham chiếu ID/reference theo baseline; quyền xem reference không tự cấp cho nhân viên kho. Với v2 edit, reference bị omit hoặc rỗng trên cùng evidence ID được backend phục hồi từ reference đang lưu trước validation; đây là giữ nội dung ẩn, không phải chấp nhận evidence mới thiếu reference. Thêm evidence mới phải có reference hợp lệ; full replacement xóa một evidence ID chỉ thành công khi không làm certificate tham chiếu ID mất tồn tại. UI không hiển thị hoặc bịa reference để vượt validation. Nghiệm thu T-DATA-04 bảo vệ vòng đọc redacted → edit → replay.

## 5. Dữ liệu và migration target

Migration `000002_warehouse_scope` bổ sung các thay đổi dưới đây; migration000001 giữ nguyên checksum:

- `traceability_profiles.warehouse_id UUID NULL`: server-derived; null chỉ dữ liệu legacy chưa backfill. Index `(warehouse_id, created_at, id)` phục vụ scoped pagination; unique namespace/batch và unique QR giữ nguyên.
- Snapshot mới chứa `warehouse_id` trong dependency JSON và hash. Không UPDATE JSON của published/submitted revision lịch sử. Backfill profile metadata ghi nguồn/version/time và audit migration riêng; profile version tăng để stale command bị conflict, QR và public revision không đổi.
- Approved/submitted legacy snapshot không có kho: không cho publish bằng v2; tạo revision mới và đi qua submit/review để có source hash mới. Public snapshot legacy vẫn đọc theo eligibility baseline; thêm scope không làm mất dữ liệu công khai hợp lệ.
- Runtime grants/DB guards hiện có phải được mở rộng trong migration mới cho field/index và schema checksum; giữ runtime role không có quyền tự sửa binding/source snapshot.
- Binding kho có thể từ null → verified warehouse một lần bằng migration/admin maintenance được kiểm soát, không phải PATCH client. Không thêm endpoint public “sửa kho”. Backfill idempotent theo profile ID; fail một profile giữ null và có report, không đoán default kho.

`service_metadata.warehouse_writes_required` mặc định false phục vụ rollout writer theo giai đoạn; chỉ table owner bật một lần bằng `cmd/migrate -require-warehouse-writes`. Trigger chặn INSERT thiếu kho sau khi bật, chặn đổi binding đã có và chặn runtime LOGIN backfill. API v2 chỉ khởi động khi switch này đã bật. `cmd/warehouse-backfill -profile-id UUID` đọc nguồn thật rồi backfill một hồ sơ, tăng version và ghi audit gồm batch version/time bằng maintenance actor hệ thống; không sửa revision/hash/public pointer.

Rollback: deploy code đọc cả legacy/new trước migration enforcement. Khi rollback code chỉ GLOBAL, tắt scoped UI/routes trước; không drop field/backfill hoặc sửa lại public snapshots để ép downgrade. Không bật scoped write nếu proto provider không có kho; health/capability báo thiếu, không silently dùng sandbox.

## 6. REST, header và tương thích

[Admin scoped draft](traceability-scoped-admin-draft.openapi.yaml) là full target cho `/api/v2/admin/trace/records` và workflow. Baseline v1 GLOBAL-only và error/header legacy giữ nguyên để không phá client; chỉ v2 mới được advertise WAREHOUSE. Common errors v2 `{code,message,requestId}`; client không phải đoán giữa hai error envelope. Public v1/v2 QR hiện có giữ wire, BFF mới normalize lỗi legacy sang envelope chung khi cần.

- v2 mutation dùng `Idempotency-Key`, 16–128 URL-safe ASCII `[A-Za-z0-9._~-]`. Không nhận key trong JSON; gửi nhiều header hoặc cả alias legacy → 400. v1 `X-Idempotency-Key` là legacy, không bắt client v2 tự map.
- V2 key namespace dùng `(actor, v2.<logical_action>, key)`, tách namespace action v1 đang có để không replay response khác version. Retry giữ nguyên API version. Fingerprint gồm resource/revision/client body đã normalize, không gồm requestId/traceparent; tính trước khi hydrate evidence.reference từ DB, không tính từ content hiện tại. Cùng key/body trả kết quả cũ nếu actor vẫn có quyền/scope; khác body →409 IDEMPOTENCY_CONFLICT. Với edit v2, omission/empty reference cùng evidence ID là marker preserve ở raw normalized request. Replay lookup sau resource authorization nhưng trước hydrate; request mới hydrate từ row được lock và expected_version khớp, rồi validate/write trong cùng transaction. Không dùng latest content ngoài lock để dựng fingerprint hoặc biến replay thành mutation mới. Không tự retry mutation bằng key mới. Đọc resource scope trước trả replay; redact theo quyền hiện tại.
- Body JSON tối đa256KiB, reject unknown/duplicate fields và trailing JSON; expected_version positive; version thay đổi do mọi command, không đồng nhất revision number.
- REST dùng `X-Request-Id`/response requestId thống nhất, tạo server-side khi vắng/không hợp lệ; traceparent truyền theo baseline. Không log bearer token/DSN/evidence reference. `Cache-Control: no-store` cho authenticated/public REST. Resolver gRPC dùng metadata x-request-id theo guide, không có HTTP response envelope/cache header.
- GET read không cần key. Empty collection 200, items=[], total=0; list total là số row trang hiện tại (không giả làm tổng DB), next_cursor null khi hết. Lookup max1; không scan tất cả profile từ frontend để tìm Batch.
- By-batch lookup dùng binding đầy đủ để hỗ trợ staff tạo nháp đúng sản phẩm; internal resolver dùng namespace+batch UUID+SKU, đối chiếu hồ sơ đã unique namespace/batch. Không bắt Order đoán product_id.

| Code v2 | HTTP | Retry / xử lý |
| --- | --- | --- |
| INVALID_INPUT / INVALID_CURSOR |400| Sửa input hoặc đọc lại trang đầu; không retry cùng input vô hạn |
| UNAUTHENTICATED |401| Reauthenticate; không hiện dữ liệu cũ |
| FORBIDDEN |403| Không retry tự động; quyền/scope/ownership insufficient |
| NOT_FOUND |404| Resource không tồn tại; collection rỗng vẫn200 |
| CONFLICT / WAREHOUSE_BINDING_CONFLICT |409| Reread; nếu source đổi thì revision/review mới; không ghi đè |
| IDEMPOTENCY_CONFLICT |409| Key dùng sai body; không coi command đã thành công |
| PROFILE_SUSPENDED / PROFILE_ARCHIVED / BATCH_EXPIRED / CERTIFICATE_* |410 public| Không trả valid record cũ; không tải/in tem mới |
| DEPENDENCY_UNAVAILABLE |503| Read retry có backoff; mutation chỉ cùng key/body khi retry; không làm nhập kho lại |

New integration draft giữ các code riêng của operation đã mô tả; full admin v2 dùng bộ code baseline chuẩn hóa này. Không gắn 422 cho input admin v2 trong khi implementation/reviewer dùng400. Các response/examples trong OpenAPI là authoritative cho wire; spec này mô tả rule không thể biểu diễn chỉ bằng schema.

## 7. Resolver, public và print

Resolver unary gRPC `BatchLinkResolver.ResolveBatchLinks` read-only chỉ cho service identity Order, credential riêng tối thiểu32 bytes, TLS/internal network, không route public wildcard. Budget1.5s, ≤100 bindings, message≤256KiB, response≤512KiB. Proto/status/presence và mapping Order REST theo [gRPC guide](traceability-workflow-grpc.md). Internal REST resolver0.2.2 chưa phát hành đã bỏ; không dual transport hoặc HTTP fallback. Duplicates được giữ số lượng/thứ tự output; có thể dedup query nhưng phải reconstruct theo input. Không audit mỗi read như mutation và không cần Idempotency-Key.

AVAILABLE khi BuildPublic đạt → link public. Chưa public/không tồn tại/SKU mismatch → NOT_AVAILABLE, qr_url=null. Từng public nay gone → NO_LONGER_PUBLIC và link cũ có thể dẫn410, không thông tin nội bộ. Corrupt snapshot cho riêng row → UNAVAILABLE, null. Database toàn bộ không đọc được →503, không giả trả tất cả NOT_AVAILABLE. Scope Identity không dùng cho credential service; Order phải kiểm ownership trước gọi.

Print dùng v2 scoped profile read và kiểm public200 hiện hành. Nếu draft mới tồn tại, in theo public_revision và batch_code/product_name của bản public, không theo working draft. Trạng thái hợp lệ có thể đổi sau kiểm tra; trang QR kiểm lại mỗi lần mở. UI tạo QR từ qr_url server trả, allowlist public web origin, không nhận arbitrary qr URL từ query. Payload chỉ URL, không token/id nội bộ/PII. QR đen trên trắng có quiet zone, không logo phủ module; chọn thư viện QR cục bộ, hỗ trợ SVG/PNG/print stylesheet, không gọi bên thứ ba. Chỉ đạt nghiệm thu in khi decode được ảnh và bản in/thử scan thực; preview đẹp chưa đủ.

Không có “activate tem”, serial, audit số bản in vật lý hoặc sync hồ sơ mỗi lần quét. Public expiry theo clock không phụ thuộc worker; restricted sau snapshot chưa được service biết cần suspend thủ công theo quy trình hiện hành. Không dùng Kafka packing để công bố.

## 8. Outbox và boundary lỗi

Giữ schema `services/traceability-service/contracts/events/traceability.event.schema.json` và topic service-local hiện có. Publish/status và audit/outbox transaction nguyên tử. At-least-once: giữ event ID/payload/key profile khi retry, không tạo business command lần hai; worker expiry idempotent, không tự restore/suspend profile thay chính sách. Consumer shared chưa ký nhận không coi delivered-to-broker là đã xử lý downstream. Không thêm field warehouse_id hoặc PII vào required event schema cũ trong scope này.

Receipt thành công → Traceability503 không rollback receipt. Resolver503 không ngăn Order đóng gói/giao hàng và không làm mất detail đơn. Broker down không ngăn đọc QR đã public; mutation publish commit local outbox hoặc rollback cả transaction khi ghi outbox thất bại, không chạy publish DB xong mới ghi outbox bên ngoài.

## 9. Cấu hình và điều kiện bắt đầu implement

Giữ các env trong `.env.example` cho DB/Keycloak/Identity/Catalog/Inventory/public web base/relay. Cấu hình mới đã implement: `TRACEABILITY_SCOPED_API_ENABLED=false`, `TRACEABILITY_CURSOR_HMAC_KEY` tối thiểu32bytes; `TRACEABILITY_ORDER_RESOLVER_ENABLED=false`, `TRACEABILITY_ORDER_RESOLVER_TOKEN` là credential riêng tối thiểu32bytes, không trùng Catalog/Inventory/Identity token. Hai feature bật độc lập, mặc định tắt; resolver có thể triển khai trước scoped admin. Secret đọc runtime, không ghi file báo cáo. Xem runbook trong [báo cáo backend](../04_testing/traceability-service/09_grpc_transport_2026-11-27.md).

Scope implementation bắt đầu trực tiếp từ target0.4.0-draft, không chờ owner khác phản hồi. Tự tái sử dụng hoặc bổ sung domain/repository/transport và contract fixtures trước provider; runtime integration chỉ đóng khi Inventory trả authoritative warehouse và Identity grants thật. [Test IDs và rollout](traceability-workflow-acceptance.md) là Definition of Done. Tài liệu không thay thế fixture bằng mock rồi ghi kết quả provider thật.

## 10. Sửa sau review code 25/11 — target0.2.2

[Review và rủi ro](traceability-workflow-review-2026-11-25.md) là bằng chứng cho các thay đổi sau, có hiệu lực khi khác diễn giải bản trước:

- Form sau receipt gọi `GET /api/v2/admin/trace/batches/{batch_id}/context?sku_code=...`. Traceability đọc BatchSource, kiểm quyền warehouse trước, rồi Catalog exact-SKU binding; trả Product/SKU/Batch/kho/mã lô/NSX/HSD/version/flags trong budget chung2s. Không dựa Inventory SKU detail chưa có provider hoặc public Catalog có thể yêu cầu giá. Đây là read, không tự tạo nháp hoặc có Idempotency-Key.
- Catalog provider dùng canonical `CatalogService.GetTraceabilityBinding` từ PR #175, x-internal-token theo server hiện có; trả version_fingerprint ánh xạ nguyên trạng vào catalog_version. Bootstrap SKU dùng GetProductVariant(channel UNSPECIFIED) trước verify; hidden/no-price không bị chặn. REST verifier cũ đã decommission, không khôi phục. Xem guide 0.4.0 về TLS/explicit private plaintext và port thật.
- [BFF adapter contract](traceability-workflow-bff-draft.openapi.yaml) định nghĩa đường cookie Identity → bearer → Kong cho trace v2 và inventory workbench/receipt. Web phải forward Origin đã kiểm tra cho mutation, không disable CSRF. Identity v2 phải guard ACTIVE EMPLOYEE, không chỉ authenticated. Trong OIDC mode dùng bearer server-trusted, không cần cookie adapter.
- Web thêm permission projection Traceability giữ permission/scopeType/scopeId từ permissionGrants. Không dùng projection role-derived ALL/ASSIGNED_ONLY để suy quyền kho; không để whitelist cũ loại các mã lowercase Traceability.
- Admin v1/v2 cùng tồn tại phải dùng application writer biết warehouse metadata. Triển khai mọi writer mới trước scoped traffic; migration bổ sung guard INSERT warehouse không null cho profile mới sau khi writers đã tương thích, vẫn giữ null trên row lịch sử. Không cho old writer đang chạy tạo null âm thầm. Rollback tắt scoped traffic trước; old binary không tương thích guard không được tiếp tục làm writer.
- Batch version baseline tăng mọi UPDATE, gồm thay đổi tồn; giữ strict source hash nên approved có thể409 khi stock đổi. Không tự bỏ batch_version/depleted khỏi hash để né re-review. UI hướng dẫn revision/submit/review lại, public cũ không bị sửa; ghi rủi ro vận hành này khi nghiệm thu.
- Order hiện consolidated một SKU/một line. Customer DTO mất canonical line ID; subresource trả thêm sku_code authoritative, Web giữ typed skuCode từ raw detail, map exact unique SKU có kiểm tra quantity rồi dùng canonical ID của subresource. Không parse skuLabel/tên, không join index; duplicate SKU/không chứng minh mapping → UNAVAILABLE. Không phải mở hỗ trợ duplicate-SKU order trong release này.
- Customer traceability tối đa1000line-batch rows và response≤512KiB; vượt giới hạn trả rootUNAVAILABLE/items[]/total0, không truncate. Resolver chunk≤100, concurrency≤2, budget toàn5s (mỗi call≤1.5s); chunk chưa hoàn thành giữ UNAVAILABLE. Không tăng timeout của Order detail để che lỗi.
- Text tem/SVG/HTML phải encode, không chèn raw markup từ product_name/batch_code. Public web origin phải ổn định/HTTPS ngoài sandbox; không thể thu hồi tem offline hoặc đảm bảo scan nếu bỏ domain.

Sandbox v2 fixtures phải có warehouse UUID explicit để test scope; thiếu field là capability unavailable, không gán default kho. Kết quả fixture không là bằng chứng Inventory/Identity thật. BFF proxy có thể nhận cookie session hoặc staff bearer đã verify, đều kiểm ACTIVE EMPLOYEE; browser mode cookie vẫn không được tiếp cận access token và vẫn phải qua Origin/CSRF.

## 11. Lịch sử amendment transport0.3.0 — đã thay bằng mục12

Bản0.3.0 từng đề xuất ba proto thay internal REST; Catalog proposal đã được thay bằng canonical Dương ở mục12. [Guide hiện hành](traceability-workflow-grpc.md) là nguồn dùng để triển khai. REST public/admin/customer và proxy BFF north-south giữ nguyên; OIDC discovery/JWKS giữ HTTPS. Events chỉ thông báo sau local commit, không thay kiểm nguồn/quyền đồng bộ. Version0.2.2 là lịch sử transport, không dùng làm phiếu giao việc hiện tại.

Runtime thêm TRACEABILITY_CATALOG_ADDR/CA_FILE, TRACEABILITY_IDENTITY_ADDR/CA_FILE/TOKEN; giữ Inventory TLS env. Token service Identity phải khác Catalog/Inventory/Order. Resolver listener GRPC_ADDR, GRPC_TLS_CERT_FILE/KEY_FILE bắt buộc ở integration, flag mặc định tắt. Hai listener drain chung trước khi đóng DB; sandbox chỉ loopback hoặc isolated-container explicit. Không mở insecure integration, không log employee JWT trong x-staff-authorization. Schema DB, backfill và nghiệp vụ không đổi.

## 12. Đồng bộ plan Dương — 0.4.0

[Guide hiện hành](traceability-workflow-grpc.md) thay proposal Catalog của 0.3.0 bằng canonical dev@ac5395c5 và bổ sung Media. Bật TRACEABILITY_MEDIA_ENABLED sau migration000003 + cấp lại runtime grants. OCOP evidence dùng document:<UUID>, ValidateMediaReference trước submit/publish; media_bind_jobs cùng publish transaction, worker sau commit BindMediaToEntity có lease/retry. Không sửa frozen content/hash/QR hoặc rollback publication đã commit khi bind tạm lỗi. Legacy references cần draft/review mới trước bật Media nếu không đúng fileId; không tự backfill. Public DTO vẫn không trả reference nội bộ. OCOP files PUBLIC_READ là policy Media baseline, không đồng nghĩa chỉ public sau QR publish; xem guide trước upload.

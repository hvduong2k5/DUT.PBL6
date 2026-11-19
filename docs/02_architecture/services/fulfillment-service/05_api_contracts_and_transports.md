# 05 — API, gRPC, events và compatibility

[OpenAPI candidate](../../../03_api_specs/fulfillment-service.candidate.openapi.yaml) · [Contract appendix](../../../03_api_specs/fulfillment-service/README.md). Không thay shared wire ở lượt thiết kế này.

## 1. Bản đồ contract và trạng thái

| Artifact | Vai trò | Status |
| --- | --- | --- |
| Shared Fulfillment v1 trên dev | AcceptTask/CancelTask/GetTaskDecision, media legacy, GetTaskRevision additive | Integration-target, provider Fulfillment chưa có |
| Shared Order v1 | Permit/validate, shipping snapshot | Có declaration; handler permit/shipping chưa thấy ở SHA Order đã audit |
| Packing completed v1 | Legacy wire đã tồn tại | Frozen; không tự phát cho cohort mới |
| Packing completed v2 | Package+parcel+task revision+generation+video | Shipping consumer có code, E2E còn pending |
| Fulfillment REST candidate | Staff queue/assignment/picks/checklist/blocks/seal/media/command query | Proposal tại docs/03_api_specs |
| Dependency proto candidate | Order packing snapshot, Inventory committed allocation/validation | Proposal riêng, không claim peer provider |

Mỗi service pin SHA/generator baseline, không regenerate từ proto khác nhánh rồi chép stub vào runtime chung. Candidate namespace `omamx.fulfillment.integration.candidate.v1` biểu thị proposal; khi ratify, RPC phải được đặt vào owning Order/Inventory proto bằng thay đổi additive được review, không deploy một “provider Fulfillment” giả quyền dữ liệu của peers.

## 2. REST conventions

Base path đề xuất `/api/v1/admin/fulfillment`, chỉ staff qua Gateway. Internal service proof qua gRPC listener riêng. JSON snake_case theo backend repo; BFF adapter map naming về UI khi cần. UTC RFC3339 timestamps; server vẫn dùng Asia/Ho_Chi_Minh chỉ ở display/expiry policy provider. UUID lower-case string, bigint JSON là integer có giới hạn an toàn cho frontend hoặc client parser giữ precision; contract candidate `version` giới hạn safe integer cho REST, gRPC int64 không đổi.

| Method/path | Permission | Key/version | Kết quả |
| --- | --- | --- | --- |
| GET `/tasks` | fulfillment.task.read | Cursor + warehouse scopes | Queue page + queue_as_of |
| GET `/tasks/{task_id}` | fulfillment.task.read | ETag decimal task version | Snapshot, checklist, picks, blocks, package, integration |
| POST `/tasks/{task_id}/assignments` | fulfillment.task.assign_self hoặc .assign_other | Idempotency-Key + If-Match | Current assignee/history |
| POST `/tasks/{task_id}/start` | fulfillment.task.work | Key + If-Match | IN_PROGRESS |
| POST `/tasks/{task_id}/unassign` | fulfillment.task.assign_self hoặc .assign_other | Key + If-Match + reason | READY nếu chưa có picking/checklist work |
| GET `/tasks/{task_id}/history` | fulfillment.task.read_history | Cursor theo version | Redacted append-only history |
| PUT `/tasks/{task_id}/picks/{allocation_id}` | fulfillment.task.work | Key + If-Match | Counted total, new task version |
| PUT `/tasks/{task_id}/checklist/{item_code}` | fulfillment.task.work | Key + If-Match | Per-item result, new version |
| POST `/tasks/{task_id}/blocks` | fulfillment.task.block | Key + If-Match | Created block, BLOCKED |
| POST `/tasks/{task_id}/blocks/{block_id}/resolve` | fulfillment.task.resolve_block | Key + If-Match | Block resolved, recomputed workflow |
| POST `/tasks/{task_id}/seal` | fulfillment.task.seal | Key + If-Match | Local completion receipt, integration PENDING |
| POST `/tasks/{task_id}/reopen` | fulfillment.task.reopen | Key + If-Match | 409 FEATURE_DISABLED MVP |
| POST `/tasks/{task_id}/media-sessions` | fulfillment.media.upload | Key; package binding |201 session URL+expiry, separate file version |
| POST `/media/{file_id}/complete` | fulfillment.media.upload | Key; immutable session |202 verification pending |
| GET `/media/{file_id}` | fulfillment.media.read_metadata | Authorized scope | File verification state, no bearer URL |
| POST `/media/{file_id}/access` | fulfillment.media.view | Purpose, case if needed; fresh access audit mỗi lần | Short GET URL after local audit; không cache replay URL |
| GET `/commands/{key}` | Original caller or recovery permission | action+order_id required | Original durable command outcome |

Browser actor IDs/warehouse in request are filters/assignment targets, never trusted identity. Assignment body omit assignee for self; assign other requires elevated permission+reason and target active membership validation. No public endpoint creates READY task directly or changes admission to CANCELLED.

## 3. Idempotency and optimistic concurrency

Key16–128 visible ASCII `0x21..0x7e`, scope `(authenticated caller, action, order_id,key)` matching Order3.2. Staff caller identity must include stable principal; group all endpoint path variations into a fixed action name. Body canonical hash includes target task/package, normalized input, expected_version from `If-Match`; excludes trace/request IDs, auth, key. Missing If-Match →428, stale new command →412; body không chứa second expected_version. Strong ETag là quoted decimal, ví dụ `"7"`.

Replay same key/hash returns original receipt even current task version changed. ETag returned with original receipt reflects original result version; client GET task for current ETag. `command_key/action/order_id` bind query, không cho actor khác đoán key để đọc outcome. Same key khác hash →409 IDEMPOTENCY_CONFLICT; no new effect. GET queries không ghi registry. Unauthorized replay vẫn bị từ chối; cache không bỏ qua current permission.

No outcome currently visible →404 command query nghĩa “chưa quan sát”, không bảo đảm original request không thể commit. Retry original command to serialize, poll with backoff. Nếu server biết registry response đã hết retention →410 COMMAND_RESPONSE_EXPIRED, vẫn giữ anti-reuse identity.

### 3.1. Canonical bytes và action registry

[Hash vectors](../../../03_api_specs/fulfillment-service/canonical_hash_vectors.json) cung cấp canonical UTF-8 và SHA256 để Go/Java/client cùng kiểm. Đây là interop proposal theo normalization Order3.2; cần chốt cùng G03/G05 trước runtime acceptance. Strings encode UTF-8, không HTML escaping, object keys sort ASCII, không newline/BOM/whitespace, không floating-point. Protobuf Timestamp normalize bằng nested integer `seconds`/`nanos`, omit nanos0 theo luật scalar defaults; không dùng một phía RFC3339 trong hash và phía kia epoch object. JSON transport của candidate proto vẫn dùng protobuf Timestamp RFC3339 bình thường; canonical hash model là lớp riêng.

REST task mutation hash model cụ thể: `{"action":<registry action>,"task_id":<UUID>,"expected_version":<decimal integer>,"body":<normalized request>}`. Path child identity phải được đưa vào body canonical khi endpoint có `allocation_id`, `item_code` hoặc `block_id`; nếu không, cùng key cho hai slot có thể bị replay sai. Giữ wire body nguyên shape, canonical normalized body thêm đúng tên path child field trước hash. Action names ổn định trong `x-command-action` của OpenAPI, ví dụ operation `SealPackingTask` dùng registry action `SealTask`.

CreateMediaSession hash model: `{"action":"CreateMediaSession","order_id":<UUID>,"task_id":<UUID>,"body":<normalized request>}`. CompleteMediaUpload: `{"action":"CompleteMediaUpload","order_id":<UUID>,"file_id":<UUID>,"body":{}}`. Order binding resolve từ task/file authority, không nhận caller tự khai. Không thêm expected task version nếu API không yêu cầu. Null primary evidence trong media-optional giữ JSON null; không tự đổi null thành UUID rỗng. Business retry giữ nguyên canonical model; các action dùng chung normalization cần interop tests trước bật.

## 4. gRPC provider semantics

`contract_version=order-3.2`; resource/command evidence như shared semantics. Accept/cancel keys/hashes canonical theo snake_case/default omission, khác REST DTO hash khi input khác. Audit/requesting_user fields chỉ context từ transport verified.

| RPC | Caller | Critical response |
| --- | --- | --- |
| AcceptTask | Fulfillment coordinator/worker service identity duy nhất trong MVP | TaskDecision current state/version + original command proof; no bool-only success |
| CancelTask | Order orchestrator only | CANCELLED tombstone proof hoặc REJECTED with ACCEPTED task |
| GetTaskDecision | Order; Inventory purpose=reversal-proof; recovery service | Current admission + optional original command, no write |
| GetTaskRevision | Order source adapter; Shipping verifier nếu có adapter riêng | Exact version/kind/time/digest/warehouse/generation |
| GetPackingVideoUrl | Care/Gateway delegated purpose hoặc staff authorization | Authorized has_video/status; short URL/checksum, not raw bucket policy |
| SubmitPackingEvidence | Legacy trusted adapter only; default disabled | Existing fields insufficient for full modern seal; do not bypass guards |

Legacy SubmitPackingEvidence không có parcel/package/version hoặc detailed line quantities. Không dùng nó làm full v2 seal API. Nếu cần support, adapter phải resolve unique admitted task, expected version server-side với conflict handling, full measured parcel/picks từ task, verified file link và staff identity; chỉ chuyển vào cùng Seal use case, không implement đường tắt. Nếu không đủ dữ liệu trả FAILED_PRECONDITION/LEGACY_ADAPTER_DISABLED.

Shared TaskRevision không có package/parcel fields. MVP Order adapter cần exact revision payload ngoài proto hiện tại để bind package cho GetShippingSnapshot; proposal additive `GetPackageRevision`/fields hoặc provider facade phải review trong G04. Chưa có authority đó thì không suy package từ latest task hoặc caller event rồi đánh accepted.

GetTaskDecision command filters resolve original caller namespace theo fixed action profile: AcceptTask thuộc `fulfillment-service`, CancelTask thuộc `order-service`; query identity Inventory/Order không thay original caller scope. Service identity dùng logical stable service ID, không token ID/pod name. Nếu mở nhiều caller cho cùng action, cần additive caller selector hoặc authoritative disambiguation; không chọn tùy tiện một registry row cùng key.

## 5. Error model

REST envelope: `code,message,request_id,retryable,details` và optional command/order references; nội dung không chứa PII/token/object key cho unauthorized caller. `details` chỉ typed missing conditions/version, không SQL/stack. Thêm Retry-After cho429/503 khi biết interval.

| Tình huống | HTTP | gRPC | Retry policy |
| --- | --- | --- | --- |
| Syntax/key/measurement invalid |400|INVALID_ARGUMENT|Sửa input, command mới nếu chưa dispatch |
| Missing/invalid JWT/service identity |401|UNAUTHENTICATED|Refresh credentials; preserve business key |
| Forbidden warehouse/assignee/media purpose |403|PERMISSION_DENIED|Không tự nâng quyền |
| Not visible resource/command |404|NOT_FOUND|Không suy terminal absence |
| Key hash conflict |409|ALREADY_EXISTS|Giữ original input; investigate |
| State/required checklist/media feature conflict |409|FAILED_PRECONDITION|Read guards, resolve issue |
| Stale If-Match |412|FAILED_PRECONDITION hoặc ABORTED theo handler|GET rồi user quyết định command mới |
| Missing If-Match |428|INVALID_ARGUMENT|Bổ sung concurrency precondition |
| Rate/worker saturation |429|RESOURCE_EXHAUSTED|Backoff + Retry-After |
| Peer missing/unimplemented/temporary outage |503|UNAVAILABLE/UNIMPLEMENTED|Fail closed, same command identity |
| Outcome not known after deadline |504 hoặc transport timeout|DEADLINE_EXCEEDED|Query/retry original, no new key |

Khác biệt quan trọng: business rejection của admission theo Order3.2 là gRPC OK + CommandEvidence.REJECTED, không map tự động mọi gRPC OK thành task accepted. REST staff endpoints trả lỗi typed khi local business command chưa áp dụng; command registry vẫn giữ original outcome nếu durable rejection.

## 6. Event envelope and versioning

Topic `fulfillment.events.v2`, partition key=`order_id`, source=`https://omama.vn/services/fulfillment-service`, type=`vn.omama.fulfillment.packing.completed.v2`, subject=`order_id:<order_id>` theo Shipping consumer. **Không đổi sang `order:<id>` chỉ để giống Order v3**; hai contract có subject convention khác nhau. Stable CloudEvent UUID/time/payload/traceparent qua retry.

Data v2 bắt buộc order_id/package_id/packer_staff_id/seal_code/parcel/video object key/checksum/scanned_batch_codes/packed_at/task_id/resource_version/ready_generation. `resource_version` là version PACKING_SEALED task revision, không admission/Order version. `scanned_batch_codes` là summary sorted unique từ validated picks; bằng chứng quantities đầy đủ vẫn ở immutable task revision. Vì event đang mang staff_id và object key, phân quyền topic là bắt buộc; không thêm address/phone/access URL.

V1 giữ nguyên. Không phát cả v1 và v2 cho cùng package; không publish paid facts để Inventory legacy trừ thêm. Event schema chấp nhận extra fields ở vài cấp không có nghĩa consumer thực thi chúng. Mọi field mới phải có owner/consumer agreement hoặc version mới; file snapshot trong docs chỉ để kiểm sample, không là shared authority thứ hai.

## 7. Examples và verification

[Examples](../../../03_api_specs/fulfillment-service/examples.json) có task/receipt/rejection, sealed event và dependency inputs cùng IDs liên kết. [Source schema snapshot](../../../03_api_specs/fulfillment-service/packing_completed.v2.baseline.schema.json) pin từ dev để validate. Verifier kiểm spec, samples và cross-field bindings; hành vi identity/concurrency/peer authority cần provider integration tests trong test plan.

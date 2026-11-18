# MS-03 Traceability Service — Rà soát trước triển khai

Ngày rà soát: 2026-10-17. Nhánh: `feat/traceability-service`.
Base: `origin/dev`, commit `2291ef5a44dd3a6f7985297903bfec01b325b087`.

**Cập nhật:** Người dùng đã giao quyền chốt phương án nội bộ. Xem
[Quyết định và đối chiếu code peer](./01_decisions_and_peer_code_review.md) để biết
các quyết định hiện hành, commit đã kiểm tra và câu hỏi còn lại. Những câu ghi
"đề xuất/chưa chốt" bên dưới là trạng thái của vòng rà soát ban đầu; phần nội bộ
đã được chốt trong tài liệu mới, phần contract liên service vẫn chờ phối hợp.

## Kết luận

### Bổ sung sau vòng rà soát tổng quan hệ thống

Ngày 2026-10-17, đối chiếu thêm toàn bộ nhóm Epic, Use Case, kế hoạch phát hành,
Domain/Context/Service Boundary, HLD, API/event catalog, GraphQL và kế hoạch đội ngũ.
**14 mục bên dưới là danh sách cần phân loại, không phải 14 blocker phải hỏi team.**

- HLD 2.2, 3.5 và System Design 10 đã quy định Gateway kiểm tra JWT bằng JWKS
  cache, loại header giả và chỉ tin context từ nguồn nội bộ được xác thực. TRC-04
  cần chốt permission/actor và contract triển khai cụ thể, không thiết kế lại Auth.
- HLD 6.1 và Event Catalog đã quy định Audit qua `audit.events.v1`; HLD 5.2 đã có
  outbox, retry/DLQ, trace context. TRC-11/14 kế thừa các nguyên tắc này; chi tiết
  retry còn khác giữa tài liệu nên cần chuẩn hóa cấu hình, không coi là chưa có gì.
- Service Boundary 8.2 cho phép REST internal non-critical read (timeout 5s),
  trong khi HLD 3.1 định hướng query qua gRPC và bảng MS-03 chưa có gRPC peer.
  TRC-08 là thiếu contract đọc dữ liệu và lệch mô tả giữa tài liệu; không kết luận
  Traceability bị cấm mọi cách đọc đồng bộ.
- Use Case 4.9 đã có frontend route `/story/{skuCode}?batch={batchNumber}`.
  Đây là URL trang web, không phải REST endpoint; phải tách hai loại URL khi
  giải quyết TRC-06. GraphQL `OCOPTraceability` cũng có required fields theo Batch,
  nên TRC-07 cần đối chiếu Mobile BFF ngoài OpenAPI.
- Procurement thuộc giai đoạn 3; Traceability thuộc MVP v1.2. TRC-09 không tự
  trở thành điều kiện chặn mọi công việc MVP: phải chốt cách nhập/xác nhận nguồn
  có bằng chứng trước khi Procurement có thật, và giữ adapter cho tích hợp sau.
- HLD 5.2.4 mô tả PackingCompleted mang `batch_code`, `mfg_date`, `exp_date`;
  JSON schema hiện chỉ có `scanned_batch_codes`. TRC-10 là lệch payload giữa
  thiết kế và schema, không phải mọi tài liệu đều thiếu thông tin Batch.
- Revision, schema nội bộ, concurrency, transaction và test là phần người phụ
  trách Traceability có thể thiết kế. Chỉ cần phối hợp khi thay contract công khai
  hoặc trách nhiệm của service khác. Quyết định MVP/quyền duyệt vẫn cần được ghi rõ.

Xem thêm [Ghi chú tổng quan hệ thống](../../00_system_reading_notes.md).

**Chưa đủ rõ để triển khai đầy đủ hoặc tích hợp thật.** Đã có yêu cầu nghiệp vụ,
ranh giới sở hữu và nhiều acceptance scenarios; còn mâu thuẫn phạm vi, vòng đời,
QR và contract. Có thể bắt đầu thiết kế chi tiết và contract tests sau khi ghi
nhận quyết định MVP. Báo cáo này không thay thế đặc tả và không tự phê duyệt các
đề xuất bên dưới. Chưa triển khai mã dịch vụ.

## Nguồn đối chiếu

- [EPIC 19](../../../01_requirements/epics/EPIC_19_OCOP_Traceability.md): phạm vi,
  US-OCOP-01..06, FR-OCOP-01..24, lifecycle, mục 8 còn mở.
- [Service Boundary, MS-03](../../service_boundary.md): ownership, REST, events.
- [Bounded Context, BC-03](../../bounded_context.md): aggregate, invariants.
- [HLD, bảng MS-03](../../high_level_design.md): REST và quan hệ bất đồng bộ.
- [OpenAPI D2C](../../../03_api_specs/openapi_d2c.yaml): `/api/v1/trace/{qr_code}`,
  schema `OCOPTraceability`.
- [Event Catalog](../../../03_api_specs/02_kafka_event_catalog.md).
- [PackingCompleted schema](../../../../packages/events/schemas/fulfillment/v1/packing_completed.event.json).
- [GoodsReceived schema](../../../../packages/events/schemas/procurement/v1/goods_received.event.json).
- [Inventory proto](../../../../packages/proto/inventory/v1/inventory.proto),
  [Catalog proto](../../../../packages/proto/catalog/v1/catalog.proto).

## Những phần đã rõ

- MS-03, BC-03 sở hữu hồ sơ nguồn gốc, chứng nhận, revision và liên kết QR.
- Public read không yêu cầu đăng nhập, chỉ được trả dữ liệu đã công khai.
- Inventory sở hữu Batch/NSX/HSD/số lượng; Catalog sở hữu Product/SKU;
  Procurement sở hữu Supplier; Content sở hữu editorial; storage sở hữu file.
- Snapshot theo Batch phải giữ lịch sử; revision công khai bất biến.
- Xác minh bởi EXT-10, phê duyệt nội bộ và công khai là các việc khác nhau.
- QR không được tái sử dụng âm thầm cho Batch khác; thao tác quan trọng có audit.
- PostgreSQL là lựa chọn thiết kế hiện có; HLD chỉ định cổng 8003.

## Các điểm cần giải quyết trước implementation

| ID | Bằng chứng / thiếu sót | Quyết định hoặc đầu ra cần có |
| --- | --- | --- |
| TRC-01 | Epic mục 2 đưa US-01..04 vào MVP, US-06 sang giai đoạn 2; nhưng dữ liệu MVP phải được duyệt trước khi công khai. | Chốt đưa workflow duyệt/công khai tối thiểu vào MVP hay cung cấp quy trình được kiểm soát khác. Không được bỏ duyệt chỉ để chạy demo. |
| TRC-02 | Epic dùng `IN_REVIEW` cùng REJECTED/SUSPENDED/ARCHIVED; BC dùng `PENDING_REVIEW` và chỉ bốn trạng thái; Boundary gộp approve + publish. | Một bảng chuyển trạng thái duy nhất, quyền/guard mỗi transition; quy trình sửa bản bị từ chối, thay bản công khai, đình chỉ và khôi phục. |
| TRC-03 | Epic cho QR cấp Product/SKU/Batch; BC yêu cầu mỗi QR gắn Batch, Boundary ghi một QR mỗi Batch; HLD kích hoạt theo PackingCompleted. | Phạm vi QR MVP, cấp phát/in/kích hoạt, phân biệt seal theo đơn với QR theo lô; batch chứa nhiều đơn và đơn chứa nhiều batch; quy trình mã in sai. |
| TRC-04 | Epic để mở actor ACT-12/ACT-15 và quyền tự duyệt. | Ma trận create/edit/submit/approve/publish/suspend/read-internal, identity verifier từ Auth/Gateway, quyền tự duyệt. Không tin header role do client tự gửi. |
| TRC-05 | Chưa có schema triển khai cho revision/snapshot/public pointer; dữ liệu bắt buộc từng scope chưa chốt. | ERD/DDL, định danh, uniqueness theo scope, một revision công khai hiện hành; optimistic concurrency; transaction publish + audit + outbox; quy tắc snapshot và hiệu chỉnh nguồn. |
| TRC-06 | Boundary dùng `/trace/qr/{qr_code}` và `/trace/batch/{batch_code}`; OpenAPI/HLD dùng `/trace/{qr_code}`. OpenAPI chỉ có public 200/404. | OpenAPI service đầy đủ public/admin: payload, validation, pagination, concurrency token, idempotency, errors và trạng thái QR bị đình chỉ. |
| TRC-07 | OpenAPI bắt buộc batch_code, production_date, expiry_date, ocop_star trên mọi response; Epic cho hồ sơ cấp Product và dữ liệu chưa xác minh. | Schema theo scope, tách trạng thái chứng nhận/xác minh khỏi nội dung; không tạo batch/date/rating giả để thỏa required fields. |
| TRC-08 | HLD ghi MS-03 không gọi gRPC; Epic yêu cầu kiểm tra Product–SKU–Batch. Inventory proto chỉ liệt kê batch theo SKU/kho, chưa có lookup batch ổn định riêng; Catalog có SKU → product_id nhưng chưa giải quyết mọi scope. | Chọn projection qua event hay query đồng bộ được thống nhất kiến trúc; contract kiểm tra quan hệ/nguồn, batch_code unique trong phạm vi nào, timeout/stale/missing behavior. Không đọc DB service khác. |
| TRC-09 | GoodsReceived không có batch_id/batch_code/NSX/HSD và không có quan hệ nguyên liệu → lô thành phẩm. | Mapping receipt → batch, provenance nguyên liệu → thành phẩm và thời điểm đủ dữ liệu; receipt chỉ gợi ý draft, không tự chứng minh xuất xứ lô. |
| TRC-10 | PackingCompleted có seal_code + scanned_batch_codes, không có QR hoặc mapping seal → từng SKU/batch/đơn vị. | Contract activation và trường hợp nhiều batch; nhận packing event không được tự phê duyệt nguồn gốc. |
| TRC-11 | Boundary dùng topic `ocop.traceability.published`, `procurement.goods.received`; event catalog dùng `*.events.v1`. HLD phát BatchQrActivated; BC thêm QRStoryScanned. Chưa có outbound JSON schema Traceability trong packages. | Chốt topic/type/version/partition key/payload/consumer; publish, activation, suspension và certificate expiry cập nhật badge/cache ra sao; inbox dedupe, outbox retry, DLQ/replay và out-of-order policy. |
| TRC-12 | Epic mục 8 để mở danh mục chứng nhận, quyền hạn EXT-10, media và chính sách hết hạn. | Trường bắt buộc, scope certificate, expiry date/timezone, revoke, provider timeout/conflict/staleness; có thể dùng xác minh thủ công có bằng chứng nhưng phải ghi rõ, không giả xác minh EXT-10. |
| TRC-13 | Đình chỉ phải có hiệu lực công khai nhưng chưa có TTL/cache invalidation; media nội bộ/công khai chưa có contract. | Public DTO allowlist, media access policy, cache/ETag/invalidation khi suspend/expire, rate limit; scan count không làm chậm public read hoặc thu PII ngoài phạm vi. |
| TRC-14 | Chưa có service test plan/runbook/migrations riêng ở base. Infra init chưa tạo DB Traceability; Compose nền không có Kafka. CI chỉ ls và nhắm main/develop thay vì dev. | Chốt runtime/toolchain, config/secrets, DB naming/role, migrations, Kafka stack, health/readiness, graceful shutdown; integration tests + CI cho dev; runbook chạy độc lập. |

## Đề xuất phạm vi khởi đầu — chưa phải quyết định đã chốt

1. Giữ US-01..04 và thêm phần workflow tối thiểu từ US-06; tạm hoãn lịch publish,
   tích hợp EXT-10 và genealogy nguyên liệu nhiều cấp nếu được ghi rõ trong scope.
2. Dùng lifecycle đầy đủ từ Epic; approve và publish riêng, publish ngay khi được
   yêu cầu hợp lệ. Mỗi hồ sơ có revision nháp và con trỏ bản công khai riêng.
3. Ưu tiên QR cấp Batch cho demo xuyên suốt; nếu giới hạn scope này thì cập nhật
   acceptance criteria Product/SKU, không tuyên bố đã hoàn thành toàn Epic.
4. Public API theo path OpenAPI hiện tại để giảm ảnh hưởng caller; chuẩn hóa lại
   Boundary sau khi quyết định. Public read dùng snapshot nội bộ đã duyệt.
5. Đặt integration ports riêng cho Catalog, Inventory, identity và provider.
   Fake chỉ phục vụ test/demo có nhãn; tích hợp thật chưa có contract phải từ chối
   thao tác cần xác minh, không tự coi dữ liệu người nhập là nguồn authoritative.

## Checklist tài liệu cần có để bắt đầu lập trình

- [ ] Scope MVP và quyết định TRC-01..04 được ghi nhận.
- [ ] Domain model, transition matrix, ERD/DDL và transaction/concurrency design.
- [ ] OpenAPI public/admin, outbound event schemas và dependency contracts.
- [ ] Chính sách certificate/provider/media/public cache và xử lý đình chỉ.
- [ ] Test matrix truy về FR/US và hướng dẫn chạy stack độc lập.

Các test tối thiểu: public không lộ draft/internal evidence; role giả bị từ chối;
Product–SKU–Batch sai bị chặn; sửa published tạo revision mới; stale approve và
concurrent publish không ghi đè; QR không đổi Batch; suspend chặn nội dung cả khi
có cache; certificate hết hạn không còn biểu diễn hợp lệ; duplicate/out-of-order
event không kích hoạt sai; publish rollback không để audit/outbox lệch trạng thái;
provider lỗi không tạo xác minh mới; supplier đổi không sửa snapshot lịch sử.

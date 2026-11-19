# 09 — Quyết định, evidence và contract gaps

> **Implementation checkpoint30/10/2026:** G01 đã có code foundation và real PostgreSQL tests theo [evidence](../../../04_testing/fulfillment-service/03_implementation_verification.md); các mô tả “runtime chưa có” ở audit baseline giữ nghĩa lịch sử. Peer/provider gates còn pending, không được tự suy thành acceptance từ codegen/test fixtures.

30/10/2026. [Source manifest](source_manifest.json) pin commit/blob paths đã đọc; nhánh có thể thay đổi sau audit. [ADRs](../../adr/fulfillment-service/README.md). Tất cả acceptance dưới đây **PENDING**, trừ khi báo cáo testing ghi rõ chỉ static verification.

## 1. Kết quả audit liên quan

- `dev@7f22226` chứa MVP contract PR#34, chưa chứa runtime các service. Fulfillment shared proto đã có GetTaskRevision additive tại dev.
- `feat/order-service@5479e19` có Order/Core/persistence/SePay/Kafka/auth và source-generated stubs; transport gRPC chưa implement permit/validate/GetShippingSnapshot. PR#32 còn mở tại audit.
- `feature/identity-service@aa4b714`, `feat/profile-service@a36afa2` có principal/Keycloak và Profile; PR#37/#18 còn mở. Staff Fulfillment audience/ABAC chưa được chứng minh.
- `feat/inventory-service@2b9f179` có reserve/release/FEFO/worker, consumer paid.v1 gọi commit-stock; chưa thấy terminal3.2 và allocation read provider cần cho packing.
- `feat/shipping-service@69b5e0d` có v2 quote/revision/event và packing-v2 consumer; v2 validates nonempty video key, SHA256, parcel và task/generation. Chưa có Fulfillment producer thật.
- `feat/care-service@94d8731` có ticket/media platform; return/refund application/Order ownership còn gap theo README. Không dùng Care role strings tự khai làm approval.

Các kết luận dựa source read và mô tả PR; không chạy lại toàn peer suites ở lượt thiết kế. Không gán production-ready hoặc live carrier accepted.

## 2. Gap registry với owner/acceptance

| ID | Gap / quyết định đề xuất | Owner cần review | Điều kiện đóng |
| --- | --- | --- | --- |
| G01 | Fulfillment runtime chưa có; dựng module+DB+query trước admission | Fulfillment | Slice1–3 code+real PostgreSQL race tests |
| G02 | EPIC media optional phase2 vs Shipping v2 required video; integrated-v2 pin VIDEO_REQUIRED | Product, Fulfillment, Shipping | Policy sign-off + actual video VERIFY + v2 consumer test; optional cohort vẫn disabled |
| G03 | Order permit/online validation/packing snapshot providers | Order | Read/permit handlers actual, negative ACL/hold/expiry, pinned snapshots test |
| G04 | Candidate task.accepted.v1 discovery cần ratify; exact seal revision lacks package/parcel fields, Core ingress codec/signature and GetShippingSnapshot | Order, Fulfillment, Shipping | Accepted fact + additive package proof contract + both revisions applied + snapshot correct package/digest/generation/recipient |
| G05 | Inventory terminal3.2 + committed allocation detail/validation; legacy paid deduction | Inventory, Order | Finalize/query/release races; no double stock effect; batch quantities/quality checkpoints |
| G06 | FR-PACK-20 reopen sau seal thiếu remote create/stop fencing | Product, Order, Shipping, Fulfillment | Approved rework wire + in-flight shipment race/recovery tests; endpoint remains disabled |
| G07 | Staff audience/warehouse membership, service auth, delegation permission profile | Identity, Gateway, Fulfillment | Forged header/body denied, dedicated audience, warehouse/assignment tests |
| G08 | Batch quarantine race và hard-stop task ACCEPTED chưa có global protocol | Inventory, Order, Fulfillment | Accept checkpoint semantics+recall procedure, hoặc implement fenced stop nếu business yêu cầu strict cutover |
| G09 | S3 immutable upload/verified final key, scanner và retention policy | Platform, Fulfillment | Object replay/overwrite/cleanup/scan tests, private bucket policy, capacity review |
| G10 | Care case entitlement/media delegation; customer video ownership | Care, Identity, Order | Authoritative access contract + negative IDOR tests; customer access disabled tới acceptance |
| G11 | Audit codec/hash checkpoint và async export | Platform/Audit, Fulfillment | Local append-only atomicity + schema validation; sink/checkpoint tests khi có runtime |
| G12 | Parcel actual vượt checkout estimate/fee tolerance | Product, Order, Shipping | Pin tolerance and re-quote/review procedure; no silent price/carrier changes |
| G13 | Shared peer contract files chưa có đầy đủ trong current checkout | Integration owner | Consume dev baseline at pinned SHA khi implement; không regenerate từ nhánh cũ |

Không tự điền tên assignee/team đã đồng ý. Owner ở bảng là bounded-context responsibility đề xuất, named assignee/PR/evidence được bổ sung khi nhận việc.

## 3. ADR summary

| ADR | Lựa chọn | Hệ quả |
| --- | --- | --- |
| 001 | Admission terminal tách packing lifecycle; DB guard/order | Task BLOCKED/COMPLETED không là cancellation proof |
| 002 | One task/order + one package MVP; immutable exact source revisions | Rework/split phải mở protocol mới, giữ originals |
| 003 | Private verified evidence, integrated-v2 video-required cohort | Không fake video để thỏa schema; object final key immutable |
| 004 | Candidate contracts riêng docs, peer-backed adapters và explicit gates | Static docs không đổi shared runtime authority |

## 4. Các phương án bị loại

- Chỉ nghe paid và tạo packing task: bỏ qua stock finalization/hold/generation, xung đột Order3.2.
- Redis lock duy nhất theo Order: lock expiry/connection loss không bảo vệ durable uniqueness; DB guard dễ chứng minh hơn.
- CancelTask từ ACCEPTED trả CANCELLED local: Inventory có thể reverse trong khi hàng đang gói; sai terminal semantics.
- Lấy FEFO batches mới nhất thay allocation: thay physical resource đã commit và có thể trừ/tạo stock sai.
- Dùng GetOrderDetail/customer token như privileged packing snapshot: dư PII và thiếu đúng immutable policy/allocation authority.
- Empty video/checksum all-zero khi không quay: qua kiểu dữ liệu không chứng minh evidence thật; Shipping runtime còn reject key rỗng.
- Ghi đè revision/event/primary media sau khi hoàn tất: phá replay, khiếu nại và package proof.
- Gọi Order/Shipping bên trong SQL Tx: remote effect không rollback và giữ locks qua network.

## 5. Tiêu chí integration-ready

G01–05/G07–09/G12–13 đóng cho scope integrated-v2; G06/G10/customer access/COD/returns có thể giữ disabled nếu mô tả đúng demo scope. G11 local audit atomicity bắt buộc, remote Audit sink readiness là capability riêng. Product phải chấp nhận giới hạn G08 checkpoint/recall, không im lặng bỏ negative test.

Evidence record cần exact SHAs, configs/gates, environment, test IDs, commands, observed results và unresolved risks. Contract compile xanh không thay provider acceptance; runtime unit xanh không thay cross-service stack; test carrier không thay production carrier credentials/operations acceptance.

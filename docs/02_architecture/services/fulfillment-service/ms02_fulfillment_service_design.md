# MS-02 — Master design map

Phiên bản 1.0, 30/10/2026, DESIGN CANDIDATE. Entry point: [README](README.md). Tài liệu này điều hướng, không lặp các chương LLD.

## 1. Mục tiêu và định nghĩa hoàn thành nghiệp vụ

Nhân viên phải nhận đúng task đủ điều kiện; lấy đúng SKU/batch/quantity; kiểm checklist theo policy đã pin; xác nhận đúng một kiện hợp lệ; lưu bằng chứng có quyền truy cập; bàn giao dữ liệu để Order và Shipping tự quyết định phần việc của họ. Thành công cục bộ không được hiển thị như vận đơn đã tồn tại.

Phân biệt `admission=ACCEPTED`, `workflow=COMPLETED`, Order `PACKED` và Shipment `CREATED`. Bốn giá trị có owner, thời điểm commit và recovery riêng.

## 2. Thứ tự ưu tiên khi nguồn mâu thuẫn

| Nội dung | Nguồn được dùng | Khi chưa thống nhất |
| --- | --- | --- |
| Giá trị nghiệp vụ/phạm vi US | EPIC10, FR, backlog, NFR | Ghi conflict trong G02/G06, không tự xóa scenario |
| Wire đã merge | `dev` tại SHA trong manifest | Additive proposal riêng; không sửa tag/required v1/v2 |
| Admission semantics | `packages/contracts/order-3.2.md` §2–5 | Không nới terminal protocol để vừa runtime legacy |
| Provider đang làm được gì | Source peer pin SHA + test evidence | `PENDING`, không suy từ generated code |
| Local Fulfillment target | Các chương01–10 và ADR | Đổi quyết định bằng ADR + update matrix/test |
| Tham số vận hành | Deployment/runbook | Giá trị đề xuất, phải đo trước claim SLA |

`service_boundary.md` có mô tả cũ “video mandatory before PACKED”; EPIC10 mới cho media theo policy. Shipping v2 hiện yêu cầu video. Chọn policy riêng cho integrated-v2, giữ FR media-optional ở domain và ghi gate G02. Tên state `SEALED` trong parcel và `COMPLETED` trong task không thay Order `PACKED`.

## 3. Traceability đầy đủ

| User story / FR | Thiết kế | Test acceptance |
| --- | --- | --- |
| US-PACK-01; FR-PACK-01/02/03 | 00,01,02,04 §1–3; barrier và queue | FUL-T01–08,17,27 |
| US-PACK-02; FR-PACK-04/05/19 | 01,04 §4; authoritative allocation | FUL-T09–12,28 |
| US-PACK-03; FR-PACK-06/07 | 02,04 §5; checklist version | FUL-T13–15 |
| US-PACK-04; FR-PACK-08/09 | 07; upload verification và policy | FUL-T18–22 |
| US-PACK-05; FR-PACK-10/11 | 03,07; immutable evidence/version | FUL-T20–23 |
| US-PACK-06; FR-PACK-12/13/14 | 05,07; ABAC và audit | FUL-T23–25 |
| US-PACK-07; FR-PACK-15/16 | 04 §3,06; deadline revisions | FUL-T26–27 |
| US-PACK-08; FR-PACK-17/18 | 03,04 §6; atomic seal | FUL-T15–17,29–34 |
| US-PACK-08 reopen; FR-PACK-20 | 04 §9,08 §4; gated coordinated reopen | FUL-T35 |
| NFR-04/05/08/09 | 05,07; service identity, object isolation | FUL-T23–25,36 |
| NFR-06 | 02,03,06; locks/version/terminal barriers | FUL-T01–08,29–34,37 |
| NFR-07/11 | 06,07,runbook; audit/outbox/telemetry | FUL-T30–34,38–40 |
| NFR-01/03/10 | 00,03,06,deployment; bounded queues/scaling | FUL-T41–42 |

## 4. Quyết định cần peer review trước code tích hợp

1. Order cung cấp permit thật, immutable packing snapshot không PII và endpoint ingest proof; gắn task/package vào shipping snapshot mà không tạo vòng lặp chờ Shipping.
2. Inventory cung cấp allocation COMMITTED theo đúng reservation, batch eligibility và quy tắc expiry/timezone; stock đã commit trước packing, seal không trừ lần hai.
3. Shipping xác nhận consumer v2, nguyên tắc một shipment/package và readiness retry khi Order chưa apply seal.
4. Identity/Gateway thống nhất quyền staff/warehouse và audience Fulfillment; không tái sử dụng audience `order-api` hoặc tin role trong body.
5. Care xác nhận media access delegation theo case/actor; chức năng return/refund hiện chưa hoàn chỉnh nên không suy ra approval authority đã có.
6. Nhóm sản phẩm chốt integrated-v2 bắt buộc video và hoãn reopen sau seal; hai quyết định này cần thể hiện trong demo/DoD.

## 5. Cấu trúc source dự kiến

```text
services/fulfillment-service/
  cmd/server/                 # HTTP/gRPC/process lifecycle
  cmd/worker/                 # có thể deploy riêng, dùng cùng application ports
  internal/domain/            # admission, task, parcel, policy, evidence
  internal/application/       # use cases, ports, transaction coordinator
  internal/transport/         # HTTP, gRPC, Kafka ingress DTO mapping
  internal/infrastructure/    # PostgreSQL, Kafka, S3, peer RPC, auth
  internal/worker/            # ready, outbox, media, reconcile, SLA
  internal/gen/               # generated only, toolchain pin
  migrations/                 # expand/validate/contract theo slice
  deploy/                     # compose/K8s/metrics theo quy ước repo
```

Domain không import driver/transport. Repository nhận transaction context, không tự commit giữa use case. Ports trả typed outcomes gồm `observed_at/version/contract_version`; không dùng `bool success` cho remote proof. Chọn Go cùng Order/Shipping/Profile để tái dùng cách vận hành; không copy package nghiệp vụ giữa các DB. Phiên bản thư viện pin theo repo lúc implement, không coi module hiện tại là cam kết dependency tương lai.

## 6. Checklist review bộ thiết kế

- Mỗi invariant có owner, transaction và test; mỗi side effect có key/query/recovery.
- Tất cả pending peer RPC được đánh dấu; static verification không được đổi `PENDING` thành `ACCEPTED`.
- Reject/replay/UNKNOWN không tạo task/package hoặc trừ kho thêm lần nữa.
- Media-optional, multi-package, accepted-stop/reopen có giới hạn và đường migration rõ.
- Giữ nguyên file source đang sửa dở của Order; bộ này chỉ thêm docs/contract candidate/verifier.

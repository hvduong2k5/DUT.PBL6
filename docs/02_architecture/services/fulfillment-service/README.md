# MS-02 Fulfillment Service — bộ thiết kế 1.0

> **Coding checkpoint30/10/2026:** [Service runtime](../../../../services/fulfillment-service/README.md) đã có implementation foundation/workflow/admission/media adapters/messaging trên `feat/fulfillment-service`. [Evidence](../../../04_testing/fulfillment-service/03_implementation_verification.md) phân biệt PostgreSQL/Kafka/gRPC tests với owning-peer acceptance còn pending. Các đoạn “chưa có runtime” bên dưới mô tả baseline lúc soạn thiết kế; checkpoint này là nguồn tiến độ mới hơn, không tự đóng integration gates.

Ngày: **30/10/2026**. Trạng thái: **DESIGN CANDIDATE — chưa có runtime Fulfillment, chưa nghiệm thu liên service**. Ngôn ngữ nghiệp vụ: tiếng Việt; identifiers dùng tiếng Anh theo repo. Baseline giao tiếp: Order 3.2 và MVP contracts đã merge vào `dev` tại `7f22226`.

Fulfillment sở hữu công việc lấy hàng/kiểm tra/đóng gói, bằng chứng, kiện hàng và quyết định nhận/hủy trước khi nhận task. Order sở hữu eligibility, payment và lifecycle tổng; Inventory sở hữu allocation/số dư/lô; Shipping sở hữu vận đơn. Một service không đọc DB của service khác.

## 1. Cách đọc và nguồn chuẩn

| File | Nội dung, độc giả |
| --- | --- |
| [Master index](ms02_fulfillment_service_design.md) | Bản đồ quyết định, source of truth, traceability và checklist review |
| [00 — HLD](00_fulfillment_service_hld.md) | Boundary, modules, topology, sequence và trade-offs |
| [01 — Domain](01_domain_and_invariants.md) | Aggregate, thuật ngữ, invariants FUL-I01–20 và phạm vi MVP |
| [02 — Lifecycle](02_state_machine_and_lifecycle.md) | Admission barrier, generation, workflow, block, parcel và UNKNOWN |
| [03 — Persistence](03_database_and_persistence.md) | ERD, transaction, locks, schema, retention và restore |
| [04 — Use cases](04_usecases_and_orchestration.md) | Pseudocode nhận/hủy, assignment, picking, checklist, seal, media, reopen |
| [05 — Contracts](05_api_contracts_and_transports.md) | REST/gRPC, auth, error/idempotency, schema/version và compatibility |
| [06 — Messaging/worker](06_messaging_workers_and_recovery.md) | Inbox/outbox, delivery, durable jobs, replay và recovery |
| [07 — Media/security](07_media_security_and_authorization.md) | Upload/verify/view, RBAC/ABAC, privacy và audit |
| [08 — Integration/extension](08_integration_and_extensibility.md) | Provider matrix, codec Order, Shipping v2 và mở rộng đa kiện/đa kênh |
| [09 — Decisions/gaps](09_decisions_and_contract_gaps.md) | Evidence hiện tại, capability gates, các thay đổi peer cần review |
| [10 — Roadmap](10_implementation_roadmap.md) | Vertical slices, DoD, thứ tự PR và rollout/rollback |
| [DDL candidate](fulfillment_persistence.candidate.sql) | SQL tham chiếu, không phải migration đã triển khai |
| [Draw.io nhiều trang](fulfillment_service_architecture_all_in_one.drawio) | Overview, barrier, packing và ERD; Mermaid trong các chương là diễn giải chuẩn |
| [Source manifest](source_manifest.json) | Git SHA, blob/hash và đường dẫn nguồn đã đối chiếu |

Các artifact đi kèm đặt đúng nhóm tài liệu của repo:

- [OpenAPI candidate](../../../03_api_specs/fulfillment-service.candidate.openapi.yaml) và [phụ lục contract](../../../03_api_specs/fulfillment-service/README.md).
- [Proto dependency candidate](../../../03_api_specs/fulfillment-service/fulfillment-dependencies.candidate.proto): đề xuất bổ sung, không tự thay `packages/proto`.
- [ADRs](../../adr/fulfillment-service/README.md).
- [Test/acceptance plan](../../../04_testing/fulfillment-service/01_acceptance_and_verification_plan.md), [static verifier](../../../04_testing/fulfillment-service/verify_design.py).
- [Deployment plan](../../../05_deployment/fulfillment_service_deployment.md), [runbook](../../../06_operations/fulfillment_service_runbook.md).

## 2. Phân biệt ba mức bằng chứng

1. **Yêu cầu**: [EPIC 10](../../../01_requirements/epics/EPIC_10_Packing.md), FR-10/11, FR-PACK-01–20, US-PACK-01–08 và NFR-04–11.
2. **Thiết kế/contract**: bộ này chốt lựa chọn để review và code; shared contract trên `dev` là baseline tương thích. File `candidate` không đồng nghĩa provider đã đồng ý.
3. **Implementation/acceptance**: chưa có runtime Fulfillment. Các nhánh peer có code không đồng nghĩa full E2E PASS. Chỉ điền acceptance sau khi chạy đúng test và pin SHA hai phía.

Checkout đang làm trên nhiều nhánh. Checkout hiện tại của workspace không có toàn bộ artifact MVP trên `dev`; vì thế phụ lục giữ bản snapshot schema cần kiểm tra và manifest pin nguồn. Không copy nguyên implementation peer vào nhánh này, không sửa wire cũ.

## 3. Phạm vi triển khai được đề xuất

**MVP domain:** một Order, một warehouse, tối đa một task được ACCEPTED trong toàn lịch sử Order; một package được seal, integer quantity/gram/cm. Hỗ trợ nhiều SKU và nhiều batch trên một SKU. Admission và packing là hai state machine khác nhau. Các UUID là định danh nội bộ; `order_number`, barcode, seal code chỉ là nhãn nghiệp vụ.

**MVP integrated-v2:** ngoài MVP domain phải có video VERIFIED, checksum, parcel, `task_id/resource_version/ready_generation`, Order permit/packing snapshot, Inventory allocation authority, Order source adapter và Shipping v2 đã nghiệm thu. Đây là phạm vi hẹp hơn khả năng “media optional” của EPIC, cần review, không sửa requirement ngầm.

**Chưa bật:** reopen sau seal; hard stop task đã ACCEPTED; split shipment/multi-warehouse; COD/guest/marketplace/B2B nếu Order chưa bật; thay allocation trực tiếp; giao hàng/hoàn tiền/hoàn kho tự động. Các extension có đường mở rộng ở chương08, không dựng success stub để bỏ gate.

## 4. Các quy tắc phải nhớ

- `fulfillment_ready.v3` đánh thức worker xin permit; `paid` không tự tạo task.
- `AcceptTask` và `CancelTask` serialize theo **Order**, kể cả lúc task chưa tồn tại. CANCELLED là tombstone bền; ACCEPTED không biến lại thành CANCELLED.
- Local `BLOCKED`, timeout hoặc task `NOT_FOUND` không phải bằng chứng kho được phép reverse.
- DB transaction không bao remote RPC/Kafka/S3. Commit UNKNOWN phải phục hồi bằng cùng key, không tạo identity mới.
- Seal thành công lưu package + exact revision + audit + outbox cùng transaction. Order/Shipping nhận bất đồng bộ; UI hiển thị riêng trạng thái bàn giao.
- Video, PII người nhận và token không đi vào log/business event tùy ý. Bucket private, URL có hạn và kiểm quyền trước cấp.

## 5. Kiểm tra tài liệu

```powershell
python docs/04_testing/fulfillment-service/verify_design.py
```

Lệnh chỉ xác minh artifact tĩnh: liên kết, OpenAPI/examples, proto candidate nếu có compiler, schema/event, SQL parse nếu có parser và manifest. Không chứng minh concurrency, carrier, S3 hoặc peer runtime. Báo cáo kết quả thực chạy nằm trong thư mục testing.

# ADR-001 — Pure Core và legacy cohort

- Ngày: 17/10/2026
- Trạng thái: Accepted — implemented domain foundations; integration chưa active.
- Scope: PR #1 theo [Roadmap18](../../services/order-service/18_order_implementation_roadmap.md), không phải GitHub PR đã publish.

## Bối cảnh và gap

`internal/domain/model.go` hiện chứa DTO Order/Quote, HTTP-oriented errors và structural StateAllowed dùng bởi sandbox. Snapshot sandbox là Quote và không có immutable snapshot v1/decision aggregate. Thay trực tiếp DTO hoặc nới transition table sandbox để hỗ trợ Care cancellation mới sẽ kích hoạt semantics mới khi persistence/authority chưa có.

## Quyết định

Thêm [internal/domain/core](../../../../services/order-service/internal/domain/core/) cho target domain. Package này không import parent legacy domain, SQL driver, Redis/Kafka/SePay SDK hoặc transport. Giữ API/state shape của sandbox; application routes chưa gọi target aggregate. Không có hai active writers xử lý cùng Order/effect. `domain.MaxMoney` alias `core.MaxOrderAmount` để dùng một commercial bound; receipt ledger không dùng bound này để loại overpayment trong target Payment.

Core dùng immutable values, coded Violation không kèm HTTP status, explicit DecisionContext và typed snapshot/evidence. Model payload có JSON tags vì schema là persisted domain document; tags không cấp public API exposure. Current owner lưu riêng original owner-at-placement; public owner getter trả defensive copy. Verified claim/reconstitution mapping sẽ được thêm cùng application/persistence use case, không tạo generic owner setter.

## Phương án không chọn

- Thay Quote DTO bằng Snapshot v1 ngay: breaking sandbox cipher/API và migrations.
- Cho Core gọi repository/provider: trộn authority/transaction/network vào pure rules.
- Viết registry/framework cho mọi workflow trước: chưa chứng minh workflow cụ thể, trái vertical-slice roadmap.

## Hệ quả và phần còn mở

PR2 cần target repositories/reconstitution và projection adapter rõ cohort; legacy DTO không trở thành domain authority của target. Chưa chuyển traffic hoặc tuyên bố API target đã implement. Snapshot at-rest encryption, DB immutability/constraints và rollback/commit recovery vẫn thuộc persistence slice. Các implementation gaps được giữ ở [ADR/gaps07](../../services/order-service/07_decisions_and_contract_gaps.md).

## Kiểm chứng

Domain tests bảo vệ mutation/copy boundaries; compile/vet toàn module kiểm compatibility. Xem [verification](../../../04_testing/order-service/04_order_core_foundations.md), không suy integration PASS từ unit tests.

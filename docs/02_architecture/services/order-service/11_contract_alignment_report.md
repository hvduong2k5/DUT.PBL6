# 11 — Báo cáo đồng bộ HLD/LLD với wire contracts

Ngày 15/10/2026 · Nhánh `feat/order-service` · Baseline thiết kế 3.2 · Trạng thái contract: **proposed**.

## 1. Kết quả

Đã cụ thể hóa thiết kế 3.2 thành shared protobuf, OpenAPI target và event schemas/examples. [Semantics profile order-3.2](../../../../packages/contracts/order-3.2.md) quy định identity/key/hash, replay, terminal proof, expiry, barrier và migration. Các capability này cần service owners review và triển khai trước khi bật cohort mới.

Đợt này có regenerated Go và thay đổi verification/CI. Application business logic, database migrations và provider implementations chưa được nâng lên 3.2. Runtime vẫn là sandbox 3.1; contract/build PASS không đồng nghĩa integration PASS.

## 2. Thay đổi theo artifact

| Artifact | Đã cập nhật | Lý do |
| --- | --- | --- |
| common/v1/workflow.proto | CommandEvidence, DecisionReference, FulfillmentPermit | Tách kết quả command gốc khỏi current resource; proof có thể xác minh thay vì bool/header tự khai |
| inventory/v1/inventory.proto | Profile/hash/digest/context; response outcome; expected resource/version; approval/barrier proof; reversal ID/batch trace | Order xác minh đúng stock và phục hồi lost ACK, không suy gRPC OK là COMMITTED |
| fulfillment/v1/fulfillment.proto | AcceptTask, CancelTask, GetTaskDecision và generation/evidence | Serialize nhận task với hủy kể cả lúc chưa có task; tombstone chống delayed ready |
| order/v1/order.proto | GetFulfillmentPermit, ValidateFulfillmentPermit; operation stage/timestamps; cancellation/projection fields | Permit được cấp và kiểm ở authority, query theo dõi cancellation/compensation độc lập |
| catalog/v1/catalog.proto | GetCheckoutSnapshot với coherent metadata/price/validity | Không ghép nhiều revision thành quote |
| shipping/v1/shipping.proto | Profile/hash và quote ID/version/validity | Confirmation kiểm terms cho cả chuyển khoản và COD |
| promotion/v1/promotion.proto | Hold query/finalize/release protocol, fee/hash/evidence | Voucher feature gated có terminal semantics; freeship không vượt phí thực |
| Order OpenAPI 3.2 | Stage/projections/allowed_actions/Cancellation, errors và examples; provider auth gated | UI biết đang nhận yêu cầu, đã đặt, đang hủy hay cần đối soát |
| Order OpenAPI sandbox 3.1 | Snapshot giữ nguyên nội dung bản HEAD, chỉ khác khả năng newline trên Windows | Sandbox regression tiếp tục kiểm đúng runtime hiện hữu |
| Events Order v3 | 8 schemas + 8 example files; checker còn kiểm biến thể COD | Required proof/generation/ordinal, privacy và trạng thái hợp lệ |
| Codegen + CI | Go regenerated; static schema/negative checks + compile proto; workflow triggers mở rộng | Phát hiện drift khi sửa shared packages; snapshot runtime không che thiếu capability target |

Không đổi field tag/type hoặc chữ ký RPC cũ. Các field/RPC thêm mới là additive về wire. Caller opt-in 3.2 vẫn phải reject thiếu `contract_version=order-3.2`/proof; không dựa vào tính additive để suy semantic compatibility.

## 3. Vì sao chuyển event target từ v2 sang v3?

V2 đã có runtime phát payload và dùng `additionalProperties=false`, nhưng thiếu các trường bắt buộc mới. Thêm required proof/generation/ordinal vào cùng schema sẽ làm payload cũ fail; mở schema tùy tiện cũng không cho consumer biết capability nào được bảo đảm. Vì vậy v1/v2 giữ nguyên và target dùng `order.events.v3`/types `.v3`.

Đây là nâng version transport; các quyết định nghiệp vụ vẫn là baseline thiết kế 3.2. Đã cập nhật HLD hệ thống, service boundary, HLD Order, LLD contracts và event catalog để tránh hai luồng target cùng tồn tại. Chương 08/09 giữ lịch sử 3.1 và review tại SHAs cũ.

Rollout consumer v3 trước producer; route cohort rõ trên Order. Không dual-trigger stock deduction hoặc tạo task từ paid cho Order mới. Rollback vẫn phải chạy recovery cho resources đã tạo và giữ ledger/registry/tombstone.

## 4. Kiểm chứng thực tế

Baseline so sánh: HEAD `7ed06a48d2e1c0bd18698fcdfc9a490601f4e08d`; descriptor baseline được compile từ proto tại commit này, bằng cùng tools image. [Kết quả static](../../../04_testing/order-service/design-contracts-verification.json) ghi phạm vi và số lượng.

| Kiểm tra | Kết quả/phạm vi |
| --- | --- |
| OpenAPI validation | PASS: target 3.2 (21 paths), snapshot 3.1 (23 paths), dependency sandbox (20 paths) |
| OpenAPI examples | PASS: 4 examples/biến thể; reject SUCCEEDED thiếu Order và ACCEPTED lộ planned Order |
| Event schemas | PASS: 8 Draft-07 schemas; 9 valid examples gồm COD; 32 invalid cases bị reject |
| Negative event cases | PII trực tiếp, thiếu ordinal, sai version, paid khi stock chưa commit, thiếu decision proof, ready generation=0, COD paid giả, refund amount/currency sai |
| Compile shared proto | PASS: tất cả shared proto và imports |
| Compatibility descriptor | PASS: 298 field cũ giữ tag/name/type/presence; enum values và RPC signatures giữ nguyên |
| Go compile | PASS: `go test ./... -run '^$'` compile các package và test packages; không chạy test cases |
| Domain unit tests | PASS: `go test ./internal/domain`; không yêu cầu provider/infrastructure |
| Codegen drift | PASS: 7 generated Go files có hash trước/sau regeneration giống nhau; không chỉnh tay generated Go |
| Tài liệu/sơ đồ | PASS: 164 liên kết nội bộ trong 17 tài liệu; render thành công 12 Mermaid diagrams |

Compatibility check này kiểm cấu trúc wire, không chứng minh ý nghĩa nghiệp vụ tương thích. Không chạy provider integration/race/full sandbox release suite trong đợt contract. Các invariants serialize DB/remote effect cần tests với provider thật, không thể chứng minh bằng JSON Schema.

## 5. Cách kiểm lại

Từ repository root, Docker Desktop đang chạy và tools image đã build:

```powershell
python -m pip install -r services/order-service/scripts/requirements-verification.txt
python packages/contracts/verify_order_design.py --compile-proto
```

`--descriptor` và `--baseline-descriptor` nhận FileDescriptorSet đã compile khi cần so sánh wire với một commit cụ thể. Runtime checker `services/order-service/scripts/verify-contracts.py` vẫn kiểm snapshot 3.1 và payload v2 thật của sandbox; CI chạy thêm checker thiết kế 3.2, không đổi nhãn kết quả sandbox thành bằng chứng cho target.

## 6. Các việc còn phải làm trước tích hợp

1. Inventory thống nhất và implement profile terminal/tombstone/hash/expiry/reversal; Order bổ sung typed adapter và validators cho mọi path.
2. Fulfillment và Order implement permit online validation cùng accept/cancel/query/generation registry, kiểm race nhận task với hủy.
3. Catalog/Shipping implement coherent snapshot/quote validity; Order database và response mapping bổ sung projections/proof.
4. Consumer v3 và producer/outbox mapping phải kiểm payload thực tế; không dùng example fixtures thay runtime evidence.
5. Care approval/completion, payment provider, voucher/loyalty và claim vẫn feature gated cho đến có giao thức xác minh và tests tương ứng.
6. QA chạy provider fault suite: expiry/finalize, lost ACK, late reserve, replay/hash conflict, cancel/accept, reversal twice và rollback cohort.

G01–G13/R01–R08 trong [ADR/gaps](07_decisions_and_contract_gaps.md) tiếp tục ghi implementation/evidence còn mở. Đầu ra đợt này là bộ wire proposal có thể review và generate/build, làm đầu vào cho triển khai từng service.

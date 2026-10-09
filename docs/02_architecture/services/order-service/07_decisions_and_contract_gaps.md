> **Implementation 3.1 — 09/10/2026:** [08_implementation_and_acceptance.md](08_implementation_and_acceptance.md) là nguồn hành vi hiện tại. Acceptance tạo checkout operation, chưa tạo Order; PAID và outbox chờ terminal stock/voucher outcome. Các đoạn v3.0 khác mô tả candidate cần đọc cùng cập nhật này.

# 07 — Quyết định thiết kế, xung đột nguồn và integration gaps

[Chỉ mục](README.md) · [Trước: Testing](06_testing_and_failure_recovery.md)

## 1. Nhật ký quyết định v3.0

Các quyết định dưới đây là **đề xuất thiết kế được tài liệu hóa**, không phải ADR đã được các service khác phê duyệt. Mã OD dùng để tham chiếu review/implementation.

| ID | Vấn đề nguồn v2.0 / repo | Quyết định và lý do | Trade-off / điều kiện |
| --- | --- | --- | --- |
| OD-01 | README hệ thống có payment-service; boundary MS-04 chứa payment | Payment component trong MS-04, use case và financial ledger tách nghiệp vụ | chưa tách process; tách sau phải ADR/contracts mới |
| OD-02 | reserve dùng order ID rồi mới sinh ID; DB commit sau giữ kho chưa có nhật ký | sinh ID/T0 durable trước RPC; journal từng acquisition | thêm ghi SQL; có DRAFT chưa thành công phải expire/clean có retention |
| OD-03 | Redis key PROCESSING/replay là chống trùng chính | durable dedup key/hash/result, Redis chỉ cache | thêm DB load; giữ tombstone lâu đủ retry |
| OD-04 | trả thừa tiền vẫn PAID dù ghi exact-match | auto allocation chỉ exact; thiếu/thừa/multi-receipt vào review | giảm tự động hóa để không âm thầm đổi chính sách tài chính |
| OD-05 | OrderPaid trực tiếp kích cả commit stock và packing; TTL có thể nhả trước consume | atomic Inventory finalize acknowledgement rồi paid; ready riêng sau voucher | thêm protocol/latency; blocker ở proto/consumer hiện tại |
| OD-06 | COD bị gộp chờ thanh toán/PAID trước packing | CONFIRMED_COD + UNPAID, eligibility + committed stock | cần enum/ready event mới; không activate bằng legacy consumers |
| OD-07 | refund là Order status; thất bại quay PROCESSING | return/case/refund độc lập; financial allocation/budget | API nhiều projection, cần UI rõ trạng thái |
| OD-08 | claim dựa phone và customer ID; Profile cache SLA 5 ms | trusted verifier, owner update conditional, authoritative Profile read | Guest claim chặn khi chưa verifier; không hứa SLA mock |
| OD-09 | free ship bị ép discount <= subtotal và final>=shipping | merchandise/shipping discount tách, arithmetic explicit | Promotion phải nhận fee/cap để finalize đúng |
| OD-10 | SHIPPED từ ShipmentCreated; loyalty PAID/completed không nhất quán | dispatched mới SHIPPED, completed policy+case barrier; Promotion sở hữu điểm | completion auto off nếu chưa barrier; 7 ngày cần policy chốt |
| OD-11 | hai listener cùng port; tên RPC giả; payload khác schema | ports/config rõ; actual packages là contract hiện tại; candidate/gap ghi riêng | shared contracts chưa được sửa theo docs |
| OD-12 | tự nhận production-ready 10/10, p95 140 ms chưa chạy | thiết kế candidate, SLO mục tiêu, explicit evidence plan | chỉ hoàn thành docs, chưa chứng minh runtime |

## 2. Hợp đồng hiện có khác thiết kế ở đâu?

| ID / Mức | Gap cụ thể | Đơn vị cần bổ sung | Tiêu chí đóng / feature bị chặn |
| --- | --- | --- | --- |
| G01 / blocker | Inventory không có finalize/query reservation/release-by-order/tombstone/reversal | Inventory + Order | atomic terminal outcome và T08/09/13/14 pass; checkout finalize/COD/cancel committed chưa integration-ready |
| G02 / blocker khi voucher | Promotion không có query/consume/release-by-order; free ship input thiếu fee/cap | Promotion + Order | hold terminal idempotent, expiry conflict và reversal rõ; T07/15 pass |
| G03 / blocker checkout | Catalog validation chưa trả canonical full line snapshot + version nhất quán; Shipping fee thiếu quote ID/expiry | Catalog + Shipping | dùng response snapshot coherent và quote valid policy; T05 pass |
| G04 / blocker | order.proto thiếu finalizing/COD/delivery-failed/admin-cancel/checkout-failed, version và projection fields | Order + Gateway/clients | additive enum/fields không đổi numbers cũ; v1 projection có test; REST candidate chính thức hóa |
| G05 / blocker | schema paid/cancelled v1 required string customer/address; thiếu COD/ready/version/compensation semantics | Order + downstream consumers | schema v2 + migration; guest/POS/free text redaction và duplicate business effect tests |
| G06 / blocker Guest | Guest proof issuer/verifier và lookup tối thiểu chưa có; Profile verifier chưa cấu hình | Identity/Profile/Order | end-to-end verified ownership; false/replay/expired denied; claim ack và revoke policy |
| G07 / blocker packing/cancel | Fulfillment thiếu ready-generation authorization/cancel barrier/query task | Fulfillment + Order | acceptance/cancel race terminal outcome, stale ready không tạo task; T15/16 pass |
| G08 / blocker payment | Provider callback authenticity/finality/refund query contract chưa chọn/kiểm chứng | Payment component + integration owner | sandbox raw-signature/replay/receiver receipt + settlement query; T11/12/21 pass |
| G09 / blocker completion | Care active-case eligibility/version/barrier và allowed completion policy chưa có | Care + Order + Promotion | completion không vượt active case khi Kafka lag; loyalty once; auto-completion giữ off trước đó |
| G10 / feature-specific | marketplace payload/store reference, POS offline sale key và cash/card evidence chưa đủ | Channel + Order | T25; schema external payment info và store ACL |
| G11 / blocker auth | mapping trusted Identity user→Profile customer, staff scope chưa có trong Order implementation | Identity/Profile/Gateway + Order | auth integration T01; không lấy sub trực tiếp hoặc trust arbitrary headers |
| G12 / implementation | financial outbox/inbox, immutability enforcement và cross-table total/allocation invariant phải được hiện thực | Order + DB | candidate DDL triển khai thành migration; receipt/unmatched và money invariant tests |
| G13 / feature-specific | B2B tax/credit/deposit, gift text/privacy, multiple address policies | Commerce owner + related services | scope/policy review và versioned snapshot/contract trước release |

“Đóng gap” cần contract file/migration/adapter và bằng chứng test, không chỉ thêm method name vào tài liệu. Feature không phụ thuộc gap mở có thể phát triển độc lập; không bật checkout/fulfillment tiền thật trước blocker tương ứng.

## 3. Những giá trị phải được chủ nghiệp vụ chốt

| Policy | Default đề xuất | Vì sao cần chốt |
| --- | --- | --- |
| Hold trả trước | 15 phút; local expiry không vượt Inventory/Promotion | ảnh hưởng late receipts và khả năng giữ hàng |
| Quote trước xác nhận | 5 phút, revalidate lúc confirm | thay đổi phí/giá không âm thầm charge |
| Commercial confirmation | 15 phút từ canonical snapshot T1, lấy min remote expiries | không kéo dài hold do RPC chậm; quote preview hết hạn khác payment window |
| Paid cancellation | Customer tạo review; Manager approve + fulfillment barrier | tiền đã thu và sản xuất có thể đã bắt đầu |
| Receipt thiếu/thừa/nhiều khoản | không tự allocate MVP | cần quy tắc settlement/refund rõ |
| COD eligibility/completion | policy carrier/risk, chưa đối soát thì không auto-complete mặc định | delivery chưa chứng minh tiền về |
| Return/dispute window | 7 ngày default, Care quyết định scope | không dùng duration như quy định pháp lý đã chốt |
| Loyalty | Promotion quyết định earn/reverse, không hard-code 1% | refunds/Guest claim/late completion phải cùng ledger |
| Retention | response dedup 7 ngày, tombstone 30 ngày; ledger/PII theo policy riêng | phải cân bằng replay/privacy/audit, không tự xóa nghĩa vụ |
| SLA | từng stage/policy version; chưa đặt số giờ fulfillment cụ thể | cần năng lực thực tế và lịch vận hành kho |

## 4. Lộ trình triển khai có dependency

1. Chốt policies/payment provider/trusted identity; bổ sung Inventory terminal protocol và Catalog/Shipping canonical quote.
2. Domain + migration + repositories/UnitOfWork + durable dedup/saga/outbox/inbox; chạy money/unique/atomic tests.
3. Cart/quote/checkout trả trước + webhook/reconciliation + timeout/compensation; fault tests trước UI success integration.
4. Promotion finalization và ready/cancel barrier; sau đó COD + queue/SLA + shipping projection.
5. Guest verifier/claim ack; Care/refund budget/provider query; completion/loyalty theo barrier.
6. Marketplace/POS/B2B/gifting theo releases; load/restore/observability và go-live evidence.

## 5. Tham chiếu kỹ thuật đã kiểm tra

- [PostgreSQL 16 — transaction isolation](https://www.postgresql.org/docs/16/transaction-iso.html): đọc/khóa và re-evaluate UPDATE là nền cho guard cục bộ, không tạo distributed atomicity.
- [Apache Kafka — design](https://kafka.apache.org/41/design/design/): partition ordering và delivery semantics không tự bảo đảm transaction với PostgreSQL.
- [CloudEvents 1.0.2](https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md): envelope và extensions; traceparent là yêu cầu dự án, không tự là core required attribute.

Các SQL/RPC/protocol trong v3.0 là suy luận thiết kế từ yêu cầu và giới hạn repo, không phải mô tả implementation đã được nguồn bên ngoài xác nhận.

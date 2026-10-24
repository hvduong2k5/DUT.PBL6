# 07 — ADR, implementation gaps và rollout, baseline 3.2

[Chỉ mục](README.md) · [HLD](00_order_service_hld.md) · [Cart/Checkout](11_cart_checkout_design.md)

Ngày13/10/2026. Các lựa chọn dưới đây là **quyết định cho bản thiết kế Order mục tiêu**, chưa phải xác nhận các nhóm provider đã triển khai/đồng ý contract. Baseline này thay candidate3.0 trong LLD01–07; runtime3.1 và report09 vẫn là lịch sử có SHA.

## 1. ADR

Các quyết định phát sinh khi implement được ghi riêng tại [Order implementation ADRs](../../adr/order-service/README.md). PR1 gặp gap digest Core12 khác Inventory profile; ADR002 đã sửa design/fixtures theo wire profile, không tự đóng real-provider gate. [Evidence Core foundations](../../../04_testing/order-service/04_order_core_foundations.md) chỉ chứng minh phạm vi local tests được nêu.

| ID | Quyết định | Lý do | Trade-off / phương án khác |
| --- | --- | --- | --- |
| OD01 | Payment component trong Order | một owner receipt/allocation/refund execution | code/module discipline; tách service cần migration |
| OD02 | operation-first, planned UUID trước effect, public Order sau place | recovery và không tạo Order giả khi validation fail | thêm resource/polling; durable DRAFT cũng hợp lệ nếu semantics rõ |
| OD03 | PG registry/hash, Redis cache | không mất dedup khi cache mất | thêm SQL/storage/retention |
| OD04 | exact-only auto allocation MVP | policy tài chính dễ kiểm chứng | under/over cần review; auto split tương lai có ledger/policy |
| OD05 | gRPC finalize từ worker trước PAID; Kafka paid là fact | explicit terminal proof, giảm vòng chờ | thêm call/dependency; async command/reply hợp lệ nhưng chưa chọn |
| OD06 | ready riêng + Fulfillment accept/cancel barrier | COD và event trễ không bypass cancellation | thêm handshake/query/generation |
| OD07 | Inventory tự expire lease; Order điều phối business compensation | không giữ stock vô hạn khi Order outage | race phải serialize remote; sửa luật cũ “cấm Inventory tự nhả” |
| OD08 | separate lifecycle/payment/stock/refund | không đánh đồng money với stock/delivery | response/UI nhiều projection |
| OD09 | refund độc lập restock; reverse cần proof | không cộng kho hàng chưa trả/đã đóng gói | cần approval/barrier và ledger trace |
| OD10 | coherent snapshot + min expiry | không charge stale terms hoặc quá hạn hold | dependency contract bổ sung; cửa sổ ngắn có thể fail checkout |
| OD11 | capability/version rollout theo cohort | wire additive không đồng nghĩa semantic compatible | vận hành drain legacy orders |
| OD12 | evidence phân tầng; gates theo feature | tránh simulator PASS thành production-ready | rollout chậm hơn nhưng kết luận có căn cứ |
| OD13 | completion decision serialize tại Care | local event projection không thấy case đang bay | thêm barrier; auto-completion tắt khi thiếu |
| OD14 | privacy/auth ở provider và consumer | header/reference không phải bằng chứng quyền | service identity/ACL/cost vận hành |
| OD15 | Runtime13 tách business/workflow/job states, execution budget và reconciliation budget | UNKNOWN không mất nghĩa vụ khi hết retry hoặc manual review | thêm obligation scanner, policy registry và operational ownership |
| OD16 | Stale worker chỉ append observation; current owner validate/apply, intent query fallback | Mất lease không đồng nghĩa external effect chưa xảy ra | provider cần authoritative query/proof, tăng evidence storage |
| OD17 | Persistence17 dùng shared generic outbox, stream head progression và consumer guards | Unmatched financial facts không cần Order; lease không fence Kafka inflight | Late duplicate vẫn có thể tới, consumer/recovery contract bắt buộc |
| OD18 | READ COMMITTED + locks/constraints, commit UNKNOWN lookup và restore-before-mutation | Bảo vệ identity qua SQL ACK loss/PITR | Thêm recovery mode, migration/grant và restore gates |

## 2. Phân loại gap

Thứ tự triển khai/gates dùng [Roadmap18](18_order_implementation_roadmap.md). Shipping Quote là dependency trước checkout; manual reconciliation/refund và operational recovery là gate trước tiền thật. Roadmap không đóng các implementation/evidence gaps trong bảng dưới.

**Cập nhật lựa chọn provider 16/10/2026:** người dùng đã chọn SePay + VietQR. [Chương 09](09_payment_sepay_vietqr_design.md) là thiết kế chi tiết để review. Phần “chọn provider” của G08 bên dưới đã có quyết định; receiver/bank capability, adapter, identity mapping, migration và live tests vẫn còn mở. Automatic refund mặc định tắt đến khi xác nhận tài khoản đủ điều kiện; không coi sandbox refund PASS là live capability.

- DEFECT: code trái semantics đã cam kết, như release COMMITTED báo released.
- SPEC_GAP: thiếu quy tắc cho timeout/expiry/race, cần hợp đồng rõ.
- DESIGN_CHANGE: Order chọn giao thức mới, phải thống nhất provider; không kết luận provider “sai HLD cũ”.
- IMPLEMENTATION_GAP: target đã viết nhưng runtime/adapter chưa có.
- EVIDENCE_GAP: chưa có kiểm chứng với provider thật.

## 3. Baseline hiện tại → target

| ID / loại | Hiện tại có bằng chứng | Cần làm / owner | Gate |
| --- | --- | --- | --- |
| G01 DESIGN_CHANGE/IMPLEMENTATION | Shared Inventory proto có 4 terminal RPC + proof proposal 3.2; simulator trả outcome cũ; Inventory#11 chỉ RPC gốc + paidv1 consumer | Inventory+Order implement atomic terminal/query/tombstone/digest/version/reversal | T08/09/13/14/28 |
| G02 IMPLEMENTATION | Order app gọi HTTP simulator | Order typed Inventory port/gRPC adapter/strict outcome validator | T27 + provider suite |
| G03 DEFECT | Inventory reserve replay/release/consumer lỗi như review09 | Inventory sửa outcome/hash/offset/expiry/auth/config | T03/12/13/17/28/29 |
| G04 DESIGN_CHANGE | v2 schemas đóng băng; v3 proof schemas + examples đã có proposal | Order+downstream pin versions/cohort/drain, ngừng deduction từ paid | T30 |
| G05 IMPLEMENTATION | simulator barrier; real proto permit/accept/cancel/query đã có proposal | Fulfillment+Order implement và kiểm race/generation/online permit | T16/24 |
| G06 IMPLEMENTATION | Catalog snapshot RPC/Shipping quote fields đã có proposal; provider chưa chứng minh | provider coherent snapshot ID/version/validity, acceptance terms | T05/31 |
| G07 FEATURE | Voucher/loyalty disabled | Promotion terminal hold/finalize/release/earn/reverse, fee cap | T06/07/15/23 |
| G08 INTEGRATION | provider ledger sandbox | Payment owner chọn provider thật/auth/finality/statement/refund | T11/12/21 |
| G09 FEATURE | Guest session có; claim/verifier chưa | Profile/Identity proof/query/apply ACK, mapping trusted | T01/02/20 |
| G10 IMPLEMENTATION | Care barrier simulator | Care+Order serialized completion/refund approvals | T21/23 |
| G11 FEATURE | POS/marketplace/B2B/gifting chưa bật | Channel+Order và business owner policy/IDs/evidence | T25 |
| G12 EVIDENCE | local sandbox reports có; provider integration chưa | QA+service owners real dependency fault suite | chương06 |
| G13 DEPLOYMENT | chưa full restore/HA evidence | Ops readiness/Kafka/auth/PITR/remote reconcile | T26/29 |
| G14 DESIGN/IMPLEMENTATION | Cart/Checkout11 đã đề xuất projection/selection/quote claim/cleanup, runtime còn tối giản | Order+FE implement candidate DTO, SQL constraints, Redis guard/CAS, PLACE+Intent và real-provider tests | CC01–29 trong chương11 |
| G15 DESIGN/CONTRACT | Order Core12 chốt OC invariants, snapshot v1, command matrix, multi-hold và query/event semantics ở mức đề xuất | Review provider proofs/digests, Product/Care hold/cancel/completion policy, route/cohort và schema evolution | OC-T01–24 trong chương12 |
| G16 DESIGN/OPERATIONS | [Runtime13](13_order_saga_worker_runtime.md) đề xuất registry21 job kinds, state mapping, stale-proof path và execution/reconciliation policies | Order+Ops review concurrency/weights, provider query capabilities, recovery roles/thresholds; triển khai và fault tests riêng | SW-T01–24; chưa đóng integration/evidence gaps |
| G17 DESIGN/PERSISTENCE/MESSAGING | [Persistence17](17_order_persistence_messaging_design.md), DDL42 bảng và Payment event candidate cụ thể hóa target | Order+Ops+Finance review physical mapping/roles, consumer contracts, apply migrations riêng và chạy real DB/Kafka/provider fault tests | PM-T01–20; static parse/schema PASS không đóng deployment/evidence gaps |

## 4. Runtime3.1 còn khác target3.2

**PR2 checkpoint:** [evidence persistence](../../../04_testing/order-service/05_persistence_foundation.md) ghi migration004/target repositories và real SQL transaction gates đã đạt trong scope; [ADR004–006](../../adr/order-service/README.md) giữ các gap multi-entry history, full financial schema, grants/rewrap và target runtime wiring. Real sandbox regression PASS không tự đóng real provider G01/G02/G05/G08 hoặc target publisher/inbox G17.

Tracking PR1→PR2: pure Core chưa có target repository/reconstitution/current-owner claim mapping hoặc atomic milestone writer. Gap này được ghi tại [ADR001](../../adr/order-service/adr_001_pure_core_and_legacy_cohort.md) và ADR003; không mở route target trước khi persistence/application slice có evidence tương ứng.

**Cập nhật coding17/10/2026:** pure Core package đã có trong PR1 scope; target application/repository/worker wiring vẫn chưa chuyển cohort. Không đánh dấu G15/G16/G17 hoặc R01–R06 đã đóng từ domain unit/race tests. Target Core và legacy DTO chia sẻ commercial amount bound; legacy routes/state semantics được giữ.

| ID | Chênh lệch code cần sửa sau đợt thiết kế | Priority |
| --- | --- | --- |
| R01 | placement compensation chỉ loại COMMITTED, chưa whitelist terminal/identity | P1 integrity |
| R02 | COD finalize/cancel validators thiếu identity/digest/version đồng bộ | P1 integrity |
| R03 | reversal simulator biểu diễn RELEASED, target cần reversal proof/REVERSED projection rõ | P1 integration |
| R04 | chưa có per-provider typed production adapter và capability validation | P1 integration |
| R05 | backoff deterministic0.05*2^n cap3.2s, target full jitter cap30s | P2 reliability |
| R06 | COD PLACE bỏ qua expiry guard của quote; target cần canonical validity ở confirmation | P1 commerce |
| R07 | hard hold/ready permits target cần handshake chứng minh ở real Fulfillment | P1 integration |
| R08 | Shared proto/OpenAPI target đã bổ sung proof/projections; DB và response mapping runtime chưa có đầy đủ | P1 integration |

Không biến việc viết docs3.2 thành đã sửa code hoặc đóng gaps.

## 5. Policy defaults và rollout

Defaults thiết kế: preview5m, hold<=15m theo remote, minimum remaining30s, exact-only, một kho/địa chỉ/VND, completion window đề xuất7d, response retention7d/registry tối thiểu30d không reuse tự động. Business owner cần chốt provider/COD eligibility/return window/refund authority/retention trước bật feature; không diễn giải duration thành quy định pháp lý.

Thứ tự:
1. Review OD05/06/07 và message semantics với Inventory/Fulfillment/Payment owners.
2. Pin proto/schema và implement provider atomic methods + negative tests.
3. Implement Order ports/validators/recovery; close R01–R08 theo slice.
4. Real integration fault suite; deploy consumers v3 trước producer, explicit cohort cho legacy/new orders (v1/v2 giữ cho cohort cũ).
5. Canary, query/reconcile backlog, gate rollback/drain; rồi bật COD/refund/completion theo dependencies.
6. Feature extensions sau, không dùng simulator để bypass capability.

Rollback giữ ledger/tombstone/key; không reset operation hoặc gửi paidv1 của new-flow Order để “tương thích”. Báo cáo trình bày liên service ở chương10.

Ngày 15/10/2026 đã cụ thể hóa wire proposal và kiểm static/build; báo cáo alignment lịch sử đã được bỏ khỏi thư mục khi sắp xếp tài liệu. Những G/R liên quan provider implementation, database và adapter vẫn mở; không coi compile/schema PASS là đóng integration gate.

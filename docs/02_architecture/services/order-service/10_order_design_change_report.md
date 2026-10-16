# Báo cáo thay đổi thiết kế Order Service: HLD ban đầu → baseline 3.2

**Hoàn thiện ngày 14/10/2026 · Nhánh feat/order-service · Phạm vi: thiết kế và tài liệu, chưa thay code runtime/contracts generated.**

## 1. Thông điệp để trình bày với các service

Order vẫn điều phối đơn hàng và Inventory vẫn sở hữu tồn kho. Thay đổi chính là Order phải biết kết quả chốt hàng trước khi xác nhận đơn trả trước; Fulfillment chỉ bắt đầu qua ready/cancel barrier. Receipt luôn ghi nhận độc lập, kể cả khi kho hết hạn. Giao thức mới chọn gRPC commands từ durable workers và Kafka facts, giữ khả năng retry/query sau crash.

Không kết luận Inventory “làm sai HLD” chỉ vì chưa có FinalizeReservation. HLD gốc cho Inventory nghe OrderPaid; đây là **thay đổi thiết kế liên service** cần cùng thực hiện. Những lỗi replay/offset/response sai là loại khác, không được trộn với yêu cầu nâng cấp kiến trúc.

## 2. Nguồn và quá trình

| Mốc | Vai trò / giới hạn |
| --- | --- |
| HLD hệ thống trước sửa tại7ed06a4 | reserve sync, paid trước downstream stock commit; có khác biệt event/command giữa tài liệu |
| LLD2.0 tại6853015 | thêm domain/schema/CAS/outbox; sequence có lỗi ordering và claim performance thiếu evidence |
| LLD candidate3.0 | đề xuất finalize trước paid, các trục state và terminal protocol, còn mâu thuẫn với implementation |
| Sandbox3.1 tại7ed06a4 | operation-first, ledger và simulator có trạng thái; real adapters chưa hoàn tất |
| Baseline3.2 đợt này | HLD/LLD thống nhất, ownership/race/protocol/rollout rõ, gaps có owner/evidence |

Nguồn lịch sử: [HLD trước sửa](https://github.com/hvduong2k5/DUT.PBL6/blob/7ed06a48d2e1c0bd18698fcdfc9a490601f4e08d/docs/02_architecture/high_level_design.md), [LLD2.0](https://github.com/hvduong2k5/DUT.PBL6/blob/6853015/docs/02_architecture/services/order-service/04_usecases_and_saga_orchestration.md), [PR Inventory#11](https://github.com/hvduong2k5/DUT.PBL6/pull/11), [PR docs#17](https://github.com/hvduong2k5/DUT.PBL6/pull/17).

## 3. Luồng trước và sau

### HLD trước

```text
Checkout → reserve → PENDING_PAYMENT/QR
Receipt → PAID + OrderPaid
                 ├→ Inventory commit stock
                 └→ Fulfillment packing task
Cancel/timeout → Order ra lệnh release
```

Ưu: ít bước acknowledgement, downstream độc lập, dễ trình bày. Thiếu: không có proof kho đã commit trước packing, expiry/payment race chưa quy định đủ; system_design dùng Kafka command trong khi HLD dùng paid event. HLD walkthrough có durable draft/saga trước reserve; lỗi “ghi saga sau reserve” xuất hiện ở LLD2.0, không gán lỗi đó cho mọi tài liệu HLD.

### Baseline3.2

```text
Quote → durable checkout operation → 202
Worker canonical validation → resource intents → reserve
    → Order PENDING_PAYMENT + operation SUCCEEDED
Verified receipt → PAYMENT_FINALIZING + durable job
    → Inventory finalize (query/retry nếu ACK mất)
    → Promotion finalize nếu bật
    → PAID + outbox
    → FulfillmentReady(generation)
    → Order permit + Fulfillment accept/cancel barrier
    → PROCESSING → PACKED → SHIPPED → DELIVERED
    → completion decision + settlement policy → COMPLETED
```

COD: resource finalize → CONFIRMED_COD/UNPAID → ready; payment settlement cập nhật trục financial riêng. Under/over/late/unmatched receipt giữ ledger và reconciliation. Refund execution không tự restock.

## 4. Danh mục thay đổi và trade-off

| ID | HLD/LLD cũ | Baseline3.2 | Vì sao đổi | Đánh đổi |
| --- | --- | --- | --- | --- |
| C01 | HTTP chờ chuỗi checkout; public DRAFT không rõ | operation durable trước, Order sau placement | crash/retry không mất intent; validation fail không tạo đơn giả | thêm operation API/polling |
| C02 | LLD2 reserve trước sinh ID/ghi saga | planned UUID + intent trước side effect | remote effect luôn có correlation/recovery | thêm SQL writes |
| C03 | Redis dedup chính | PG scoped key/hash/registry, cache phụ | cache loss không tạo đơn lặp | storage/retention policy |
| C04 | PAID kích trừ kho và packing | finalize stock trước PAID; ready riêng | loại vòng phụ thuộc, chặn premature fulfillment | thêm terminal RPC/dependency latency |
| C05 | một PAID hàm ý nhiều thứ | receipt/payment/stock/lifecycle/fulfillment độc lập | COD/late payment/refund biểu diễn đúng | UI/API nhiều projection |
| C06 | “Order độc quyền timeout, Inventory không tự nhả” | Inventory expire lease atomic; Order điều phối cancellation | stock không treo khi Order outage, authority rõ | remote race protocol bắt buộc |
| C07 | timeout dễ thành failed | UNKNOWN obligation + query/stable retry | timeout không chứng minh rollback | manual review/runbook cần có |
| C08 | release boolean/no-op không phân biệt | actual resource state/digest/version; reversal riêng | không đóng compensation khi stock committed/unknown | provider phải bổ sung outcome |
| C09 | cancel khi local chưa PROCESSING | remote task barrier + generation/tombstone | event lag không làm hủy đơn đã bắt đầu task | thêm accept/cancel/query handshake |
| C10 | receipt gắn chặt Order; rule overpaid mâu thuẫn | receipt độc lập, exact-only allocation MVP | giữ sự thật tiền, policy rõ | ít tự động hóa, có đối soát |
| C11 | generic webhook HMAC/VietQR | adapter theo provider contract + statements | QR image không chứng minh receipt/finality | phải chọn/kiểm provider thật |
| C12 | ship/voucher có thể chạy trước input chuẩn; local+15m | coherent snapshot, dependencies theo dữ liệu, min remote expiry | không stale price/fee/hold window | có thể từ chối quote gần hết hạn |
| C13 | refund/return thành một Order state | refund obligation/approval/budget độc lập stock | chống hoàn hai lần hoặc restock hàng chưa về | ledger và phối hợp Care |
| C14 | shipment created thành shipped, delivered đủ complete | dispatched checkpoint; Care completion decision/settlement | không báo giao/hoàn tất sớm | thêm provider proof, completion gated |
| C15 | nhiều bản proto/schema trong docs | link wire nguồn + target semantics + support matrix | giảm contract drift, thấy rõ simulator vs real | phải quản trị version/cohort |
| C16 | p95 140/150ms mô tả như cam kết | tách acceptance/completion/finalization, workload/tests | đánh giá bằng evidence | phải xác nhận lại NFR measurement scope |
| C17 | Payment process/Finance refund owner không thống nhất | Payment module MS-04 thực thi, Care approve, Finance accounting | tránh hai executor tiền | module coupling, tách sau cần migration |
| C18 | feature rộng coi như sẵn sàng | core slice và explicit feature gates | MVP có phạm vi chứng minh được | POS/voucher/B2B triển khai theo đợt |

Các thay đổi C04/C06/C09/C17 cần được nhóm liên quan thống nhất vì thay đổi authority hoặc trigger. Chọn tên PAID có thể khác hệ thống khác: ở đây giữ nghĩa chặt của Order3.1, trong khi receipt phản ánh tiền ngay khi nhận.

## 5. Vì sao chọn gRPC finalize, không giữ paid-trigger?

Ba phương án đã cân nhắc:

| Phương án | Đánh giá |
| --- | --- |
| PAID fan-out tới Inventory và Fulfillment như HLD | đơn giản, nhưng chưa có stock/ready gate và expiry race |
| Async finalize command + terminal reply, rồi ready | có thể đúng; thêm command topic/reply/correlation/recovery/lag handling |
| Worker gRPC finalize/query + Kafka facts | chọn cho MVP: một attempt bounded, explicit outcome, tận dụng proto hiện có; workflow vẫn async với HTTP |

Không tuyên bố sync an toàn hơn async trong mọi hệ thống. Điều cần giữ là terminal outcome và fulfillment gate. Nếu chọn command/reply về sau, các invariants/key/query/tombstone không được bỏ.

## 6. Kịch bản giải thích với nhóm

**Khách trả sát hạn:**14:59 tiền được verified,15:00 kho expire trước finalize tới. Receipt vẫn tồn tại, không ready; Order reconciliation/refund review. Không báo “chưa nhận tiền” và không hồi sinh reservation đã bán cho khách khác.

**Finalize thành công, mạng mất reply:**Inventory đã COMMITTED. Worker giữ UNKNOWN, query/retry key cũ lấy outcome và tiếp tục PAID. Không release vì timeout.

**Cancel tới trước reserve:**chưa có reservation row. Inventory tạo order-key tombstone atomic; reserve trễ cùng planned ID bị reject. NOT_FOUND/query đơn thuần không đủ để đóng compensation.

**Ready bị trễ:**Order đã yêu cầu cancel. Fulfillment serialize task accept với cancel: cancel thắng thì event/permit cũ không tạo task; accept thắng thì cancel bị từ chối và Care xử lý. Order không tự suy từ status local.

**Refund callback mất:**budget đã giữ; query provider cùng key, không tạo refund mới. Tiền hoàn không tự cộng tồn. Hàng đã giao cần return inspection.

## 7. Từng service cần phối hợp gì?

| Service | Cần thống nhất/triển khai | Điều họ nhận được |
| --- | --- | --- |
| Inventory | atomic finalize/expire/release; query; tombstone trước reserve; key/hash; digest/version/proof; reversal | single command authority, không đoán ý nghĩa paid/cancel facts |
| Fulfillment | ready generation; permit validation; serialized accept/cancel/query; task evidence | không nhận việc từ đơn thiếu stock hoặc đã cancel |
| Catalog | coherent canonical batch snapshot/version/validity | không bị Order ghép metadata/price khác revision |
| Shipping | quote ID/validity và trusted checkpoints/COD evidence | Order không tự suy “created=shipped” |
| Promotion | hold/finalize/query/release, fee cap; earn/reverse policy | budget/voucher/loyalty vẫn thuộc Promotion |
| Profile/Identity | authenticated mapping/address ownership; claim proof và apply ACK | Order không dùng user ID thay customer hoặc tự claim bằng phone |
| Care | approval scope/budget authority, completion barrier/post-complete policy | case/refund/Order state tách rõ |
| Finance | receipt/refund/reconciliation semantics, accounting ownership | không có refund executor thứ hai |
| Notification/Analytics | consume v3; phân biệt received money/confirmed order/ready | không hiểu mọi receipt là đơn đã sẵn sàng |
| Gateway/clients | operation202/status polling, explicit UI states/errors | retry an toàn, không báo thành công sớm |

Mỗi provider cần trả lời: signature/version nào, terminal decision đặt ở đâu, replay/hash conflict ra sao, timeout query bằng gì, caller quyền nào, evidence/test nào đóng gate. Không yêu cầu đổi database của họ theo schema Order.

## 8. Những gì đã sửa trong repository

- high_level_design.md: thay các luồng/order interaction matrix/checkout/compensation/contracts/walkthrough; dẫn về HLD Order chuẩn và wire files thay copied schemas.
- system_design.md và service_boundary.md: cập nhật phần Order–Inventory–Fulfillment/Payment có liên quan, đánh dấu phạm vi mô tả lịch sử còn lại.
- 00_order_service_hld.md: HLD riêng làm nguồn chuẩn.
- 01–07 LLD: domain, lifecycle/barriers, schema/transactions, use cases, target contracts, fault tests, ADR/gap ownership.
- README và master: một baseline3.2, phân biệt runtime3.1; master là navigation map để tránh sao chép nhiều bộ nội dung.
- 08 implementation: giữ snapshot3.1 và thêm banner về baseline mục tiêu mới.
- 09 review: giữ findings/SHA như báo cáo lịch sử.
- 10 báo cáo này: giải thích before/after/trade-offs/ownership/rollout.

Mermaid trong HLD/LLD là sơ đồ chuẩn3.2. Draw.io/PNG cũ là reference lịch sử chưa regenerate; không dùng chúng làm nguồn quyết định luồng3.2. Các ảnh người dùng đang xóa hoặc thư mục diagram chưa track được giữ theo trạng thái workspace.

Đợt thiết kế ngày 14/10 chưa sửa runtime/wire files. Ngày 15/10 đã bổ sung shared proto, OpenAPI target, event schemas v3, regenerated Go và kiểm tra static/build theo [báo cáo đồng bộ contract](11_contract_alignment_report.md). Application logic/migrations và provider implementations vẫn chưa đổi; G/R trong chương 07 tiếp tục ghi phần còn mở. OpenAPI sandbox 3.1 và events v2 giữ làm snapshot kiểm runtime.

## 9. Thứ tự phối hợp và tiêu chí hoàn tất

1. Review OD05/06/07/09 với Inventory/Fulfillment/Payment, chốt caller authority và semantics paid/ready.
2. Wire proposal và producer/provider contract tests; không coi method name đã có là capability ready.
3. Provider implementation; Order typed adapter + validators + recovery.
4. Real integration fault suite với SHAs hai phía/ledger assertions; giữ disabled khi chưa đủ.
5. V3 consumers trước producers, explicit cohort/drain legacy Orders; không dual-trigger effect.
6. Canary/observability/restore-runbook; rollout features theo gate.

Một thiết kế “tốt” ở đây là: invariant có guard/owner; side effect có durable intent; unknown có query; replay không duplicate; conflict có policy; runtime gap có owner; test có oracle. Không dùng số bảng/states hoặc simulator latency để chấm production-ready.

## 10. Kiểm chứng tài liệu

Kiểm tra ngày 14/10/2026 trên 16 tài liệu Markdown (HLD hệ thống, system design, service boundary và bộ Order):

- 152 liên kết nội bộ được kiểm tra đường dẫn/anchor: không còn lỗi.
- Các khối code Markdown đóng/mở đầy đủ.
- Render thành công toàn bộ 12 sơ đồ Mermaid bằng Mermaid CLI 11.12; đã sửa lỗi dấu chấm phẩy trong sequence diagram và anchor mục lục cũ.
- Đối chiếu HLD/LLD theo các invariant: stock trước PAID, receipt độc lập lifecycle, COD không giả PAID, timeout không phải failure terminal, cancellation có remote barrier, refund không tự restock.

Các kiểm tra này xác nhận cấu trúc tài liệu và tính nhất quán của thiết kế; không chứng minh correctness của implementation. Runtime 3.1 evidence vẫn ở chương 08/report kiểm thử; đợt này không chạy hay tạo evidence integration thật với các provider.

## 11. Nội dung trình bày với các service khác

Có thể mở đầu buổi review như sau:

> HLD đầu đã xác định các service và saga, nhưng một số quy tắc xử lý timeout, hết hạn và tranh chấp chưa đủ chi tiết. Bản 3.2 giữ ownership của các service, đồng thời làm rõ khi nào Order được xác nhận, ai có quyền thay đổi tồn kho và bằng chứng nào cho phép tiến bước. Thay đổi lớn nhất là xác nhận stock trước PAID và phát FulfillmentReady riêng. Đổi lại, Order cần worker/recovery và các provider cần contract query, idempotency, terminal outcome rõ ràng. Đây là thiết kế mục tiêu để phối hợp triển khai; không phải tuyên bố mọi provider đã hỗ trợ.

Sau đó đi lần lượt ba tình huống: checkout thành công; đã nhận tiền nhưng finalize mất phản hồi; hủy đơn đồng thời Fulfillment nhận task. Với mỗi tình huống, chỉ ra durable intent nằm ở đâu, service nào quyết định kết quả và caller phục hồi bằng API nào. Kết thúc bằng bảng phối hợp ở mục 7 và các gate ở mục 9.

| Thuật ngữ | Cách giải thích khi trình bày |
| --- | --- |
| Durable intent | Ghi bền vững việc cần làm trước khi gọi service khác, để crash vẫn phục hồi được |
| Terminal outcome | Kết quả cuối có bằng chứng từ service sở hữu tài nguyên; timeout chưa phải kết quả cuối |
| Idempotency key | Cùng một ý định được retry với cùng key; không phát sinh thêm effect |
| Tombstone | Dấu hủy được lưu kể cả khi reserve chưa đến, chặn request đến trễ |
| Barrier | Điểm phối hợp có quyết định được lưu bền vững, phân xử nhận task với hủy task |
| Compensation | Thao tác nghiệp vụ bù lại effect đã xảy ra; không phải rollback transaction xuyên service |
| Contract gap | Capability/semantics còn cần thống nhất hoặc triển khai; không mặc nhiên là lỗi của provider |

## Bổ sung phạm vi MVP 1 ngày 15/10/2026

Xem [quyết định mới và trade-offs](13_mvp_delivery_decisions.md). Báo cáo này ghi baseline trước khi thu hẹp MVP; các kết quả kiểm chứng cũ không chứng minh SePay hay refund manual đã triển khai. Shared wire giữ tương thích; cần bổ sung API/payment attempt và manual proof trước implementation.

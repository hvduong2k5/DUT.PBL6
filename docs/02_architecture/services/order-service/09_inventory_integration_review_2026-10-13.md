# Review Inventory và tương thích Order 3.1 — 13/10/2026

> Review lịch sử tại các SHA bên dưới. Sau review, bộ thiết kế mục tiêu đã thống nhất thành [3.2](00_order_service_hld.md); xem [báo cáo thay đổi](10_order_design_change_report.md). Findings trong đây chưa được coi đã sửa chỉ vì tài liệu thiết kế đổi.

## Phạm vi và kết luận

Đã chuyển checkout sang `feat/order-service`, HEAD `7ed06a4`, theo yêu cầu. Bản chính dùng để đánh giá Order là implementation 3.1, đặc biệt chương 08; không lấy bản thiết kế cũ trên nhánh Profile làm chuẩn.

Nguồn:
- [PR #11](https://github.com/hvduong2k5/DUT.PBL6/pull/11), đang open, head `2b9f179`. Hai commit trực tiếp của Inventory: `77815bb` (domain/database/use cases), `2b9f179` (gRPC/Kafka/outbox/workers). Mô tả PR còn nói 76 tests của giai đoạn đầu, không đại diện đầy đủ head mới.
- [PR #17](https://github.com/hvduong2k5/DUT.PBL6/pull/17), draft/open, head `e23a919`, bổ sung hai tài liệu; không triển khai HTTP server.
- [Order implementation 3.1](08_implementation_and_acceptance.md), [gap register](07_decisions_and_contract_gaps.md), code app, simulator và protobuf tại HEAD.

**Kết luận: lõi Inventory có nền tảng tốt, nhưng chưa nên chấp nhận tích hợp Order thật. Order 3.1 có thiết kế an toàn hơn flow OrderPaid cũ; cần nâng Inventory và adapter Order để thống nhất giao thức, không hạ thiết kế Order về PAID trước khi xác nhận kho.** Các khoảng trống đã được Order ghi nhận; có simulator và generated proto không có nghĩa Inventory thật đã hỗ trợ.

Đây là review tập trung vào lifecycle tồn kho, contract và recovery giữa hai service, không phải audit toàn bộ 221 file của PR #11.

## 1. Ưu điểm Inventory

### Làm rõ baseline HLD và nguồn phát sinh gap

HLD hiện có contract thật, không phải dự án thiếu thiết kế chung. [HLD:473](../../high_level_design.md) quy định Inventory consume OrderPaid để trừ kho; [system design:278](../../system_design.md) mô tả commit deduction bằng Kafka command sau Paid; [service boundary:399](../../service_boundary.md) phát order.stock.commit sau PAID. Ngay các tài liệu nền này đã khác nhau về trigger event/command, dù cùng chọn trừ kho bất đồng bộ sau thanh toán.

Order OD-05 đổi thứ tự thành finalize kho rồi mới PAID, và thêm fulfillment-ready. Chương 07 nói rõ đó là đề xuất chưa được các service khác phê duyệt. Commit Order 7ed06a4 bổ sung bốn RPC terminal vào shared proto, nhưng Inventory PR #11 chưa có chúng. Vì vậy, thiếu FinalizeReservation trên Inventory so với Order 3.1 là dependency của thay đổi kiến trúc phía Order, không đủ căn cứ gọi Inventory vi phạm HLD gốc.

Cần phân biệt ba loại: (1) implementation defect, như offset skip và response release sai; (2) contract chưa quy định đủ failure semantics, như race expiry/finalize và timeout UNKNOWN; (3) thay đổi kiến trúc mới chưa đồng bộ, như finalize-before-PAID và ready event. Loại (3) cần ADR chung, cập nhật HLD/contracts và migration đồng bộ trước khi dùng làm chuẩn bắt buộc. Nhận xét “Order 3.1 an toàn hơn” là đánh giá kỹ thuật có điều kiện, không có nghĩa mọi service phải mặc nhiên theo thiết kế riêng của Order.

- Domain/use case/repository/transport tách riêng; FEFO và entity có test riêng, thuận lợi thay transport.
- Reservation, stock và outbox được ghi cùng transaction; lỗi lưu outbox làm rollback mutation.
- PostgreSQL có row locks, CHECK physical >= reserved >= 0, unique order_id, allocation theo batch; có cơ sở chống bán vượt tồn.
- Gom SKU trùng và sort trước giữ khóa; commit/release đọc lại reservation có khóa trong transaction.
- Commit đã COMMITTED trả no-op, hạn chế trừ kho hai lần khi redelivery.
- Có cleanup reservation, xử lý hạn lô, outbox retry, panic recovery và shutdown.
- Commit thứ hai bổ sung đáng kể transport/workers/tests. Không nên chỉ đọc mô tả PR đầu rồi kết luận service mới có domain.

## 2. Findings cần sửa trước tích hợp

### INV-01 — P1: consumer có thể bỏ mất nghiệp vụ khi offset sau thành công

Nguồn: [internal/presentation/kafka/consumer.go:279](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/presentation/kafka/consumer.go#L279).

Khi xử lý offset N lỗi DB/lock, vòng lặp continue và FetchMessage tiếp. Nếu N+1 cùng partition thành công, CommitMessages(N+1) xác nhận luôn các offset trước đó. N không tự được retry chỉ vì code không commit riêng N. Hệ quả: đơn đã trả tiền nhưng không trừ kho, sau đó reservation bị nhả.

Đây là suy luận trực tiếp từ control flow và [semantics CommitMessages của kafka-go](https://pkg.go.dev/github.com/segmentio/kafka-go#Reader.CommitMessages). Sửa bằng retry message đang giữ trước khi vượt offset, hoặc durable inbox/retry queue trước ACK; poison message phải có quarantine bền vững. Test cần N lỗi, N+1 thành công, restart cùng group, kiểm stock cuối cùng.

### INV-02 — P1: paid đến sau release bị coi là xử lý thành công

Nguồn: [internal/presentation/kafka/consumer.go:177](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/presentation/kafka/consumer.go#L177) và [internal/application/usecase/commit_stock.go:64](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/application/usecase/commit_stock.go#L64).

Commit trả ErrReservationAlreadyProcessed cho RELEASED/CANCELLED/EXPIRED; consumer nuốt lỗi này rồi ghi idempotency, trả nil. Ví dụ tiền vào gần hết hạn, Kafka lag, cleanup đã nhả hàng; paid event tới sau sẽ ACK mà không có stock commit hoặc failure outcome gửi Order.

Chỉ COMMITTED mới là duplicate thành công. Các terminal state khác phải tạo conflict/reconciliation outcome bền vững. Không để Fulfillment coi việc nhận tiền là bằng chứng hàng đã được chốt.

### INV-03 — P1: replay reserve trả SUCCESS dù không còn giữ hàng hoặc payload đã đổi

Nguồn: [internal/application/usecase/reserve_stock.go:89](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/application/usecase/reserve_stock.go#L89) và [internal/presentation/grpc/handler.go:116](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/presentation/grpc/handler.go#L116).

Use case trả bất cứ reservation hiện có theo order_id, không so payload/status; handler luôn trả SUCCESS. Replay sau cleanup có thể được báo giữ kho thành công dù reservation RELEASED. Cùng order_id nhưng đổi SKU/quantity cũng nhận kết quả cũ, không có conflict. idempotency_key trong proto không được chuyển vào use case.

Cần canonical request hash + stable key + terminal outcome. Replay payload giống trả cùng resource/outcome; payload khác trả conflict; terminal reservation không được trình bày như hold đang active.

### INV-04 — P1: release COMMITTED báo is_released=true

Nguồn: [internal/application/usecase/release_reservation.go:87](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/application/usecase/release_reservation.go#L87) và [internal/presentation/grpc/handler.go:150](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/presentation/grpc/handler.go#L150).

Use case gặp COMMITTED trả 0,nil; handler trả is_released=true. Không có restock nhưng caller có thể đóng compensation như đã hoàn tất. Khi đồng thời release, biến totalRestored còn được tính trước transaction nên response cũng có thể nói đã hoàn lượng hàng mà transaction chỉ no-op.

Trả actual state và lượng thực sự thay đổi trong transaction; COMMITTED cần reversal riêng và bằng chứng cho phép. Không suy diễn successful transport thành successful release.

### INV-05 — P1: expiry chưa là guard atomic của finalize

Nguồn: [internal/domain/entity/reservation.go:109](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/domain/entity/reservation.go#L109), commit_stock.go và cleanup_worker.go.

Commit chỉ kiểm PENDING, không kiểm expires_at. Reservation quá hạn nhưng cleanup 60 giây chưa chạy vẫn commit được. Cleanup lại dùng Release(), ghi RELEASED thay vì EXPIRED. DB lock ngăn hai mutation đồng thời nhưng không định nghĩa ai được thắng theo deadline.

Chốt policy tại Inventory: dưới reservation row lock, kiểm DB time và trạng thái; finalize chỉ PENDING còn hạn, cleanup chỉ expire PENDING đã đến hạn. Outcome persisted phải phân biệt EXPIRED/RELEASED/COMMITTED. Receipt đến đúng hạn nhưng finalize tới trễ cần policy explicit, không tự bỏ qua deadline.

### INV-06 — P1 khi triển khai thật: thiếu Kafka vẫn đánh dấu outbox published

Nguồn: [cmd/server/main.go:129](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/cmd/server/main.go#L129), [internal/infrastructure/kafka/producer.go:51](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/infrastructure/kafka/producer.go#L51), outbox_publisher.go.

Không đặt KAFKA_BROKERS thì server dùng LogEventPublisher trả nil; worker MarkPublished dù chưa gửi broker. Không có consumer paid trong cấu hình này. Khi cấu hình lại Kafka, những outbox đã published không được tự gửi lại.

Chế độ production phải fail startup/readiness khi thiếu broker; log-only phải là sandbox explicit và không xác nhận delivery thật.

### INV-07 — P2: token Redis lock thuộc key, không thuộc lần acquire

Nguồn: [internal/infrastructure/redis/lock.go:22](https://github.com/hvduong2k5/DUT.PBL6/blob/2b9f1790079ef4d1433a91d77ce3c57ac8f3c37f/services/inventory-service/internal/infrastructure/redis/lock.go#L22).

Token lưu trong sync.Map theo key dùng chung service. A giữ khóa quá 3 giây; B cùng process acquire key sau expiry và ghi token B; A cleanup đọc token hiện tại B và có thể unlock khóa B. Lua compare-token không giải quyết khi truyền nhầm token chủ sở hữu. Đây là rủi ro khóa, chưa đủ để kết luận oversell vì PostgreSQL vẫn có row lock/constraints.

API lock nên trả lease handle/token cho từng invocation; Release dùng đúng handle. Mã hiện dùng một Redis client SET NX, không phải triển khai quorum Redlock như tên gọi gợi ý.

### INV-08 — P2: trường contract chưa có semantics thực thi

Reserve không sử dụng ttl_minutes, preferred_warehouse_id, channel; TTL luôn 15 phút. Stock query bỏ warehouse_id. Schema inventory_items dùng sku làm PK nên chưa hỗ trợ cùng SKU nhiều kho. Release bỏ release_reason/audit_context và không kiểm cặp order_id/reservation_id khi cả hai được truyền.

MVP một kho/TTL cố định có thể chấp nhận, nhưng phải reject giá trị không hỗ trợ hoặc công bố giới hạn; không âm thầm bỏ qua yêu cầu kho khác. Cần caller authorization: server hiện đăng ký recovery interceptors, chưa thấy auth interceptor/TLS trong main; việc bảo vệ bằng hạ tầng bên ngoài chưa được xác minh.

## 3. Đối chiếu Order 3.1

| Nội dung | Order 3.1 | Inventory PR #11 | Đánh giá |
| --- | --- | --- | --- |
| Transport thực chạy | App gọi HTTP /inventory/* của dependency simulator | gRPC 4 RPC | Chưa có production adapter nối hai bên |
| Reserve | cần order_id, resource ID, RESERVED, expiry | SUCCESS không phản ánh lifecycle | Phải normalize và kiểm actual state |
| Finalize | receipt → PAYMENT_FINALIZING → kho COMMITTED → PAID | nghe OrderPaid v1 rồi mới commit | Vòng phụ thuộc nếu nối nguyên trạng |
| Query recovery | terminal query là contract thiết kế | chưa có RPC query reservation | Không giải quyết timeout rõ ràng |
| Release by order | compensation cần terminal outcome/tombstone | release no-op cả khi COMMITTED; không tombstone cho order chưa tồn tại | Không bảo đảm cancel thắng reserve đến muộn |
| Reversal | cancel committed/COD cần reversal có điều kiện | chưa có use case/RPC tương ứng | COD và paid cancellation bị chặn |
| Events | order paid/ready v2 | filter paid v1 | Đổi topic/type đơn thuần không sửa được semantic |
| Expiry | tin remote terminal result; không PAID khi kho thất bại | deadline không được kiểm ở commit | Cần atomic protocol |
| Stock model | projection order.stock_status | physical/reserved + batch allocations | Hợp về ownership, phải thống nhất tên trạng thái |
| Evidence | sandbox và provider simulator | unit/mock tests | Chưa có bằng chứng Order + Inventory thật |

Nguồn Order: [placement.go](../../../../services/order-service/internal/app/placement.go), [jobs.go](../../../../services/order-service/internal/app/jobs.go), [base.go](../../../../services/order-service/internal/app/base.go), [simulator gRPC](../../../../services/order-service/internal/simulator/grpc.go), [shared proto](../../../../packages/proto/inventory/v1/inventory.proto).

**Ưu điểm Order nên giữ:** accepted operation durable trước side effect; planned_order_id/stable keys; lease epoch; receipt/stock/payment tách nhau; PAID có điều kiện; ready riêng; COD UNPAID; refund không tự restock; nghĩa vụ UNKNOWN không bị xóa. Đây là cải tiến đúng so với bản cũ.

**Nhược điểm Order cần nâng cấp:**
1. Production Inventory port/adapter còn thiếu: hiện s.Call dùng Dependencies HTTP, không phải client gRPC Inventory thật. Generated protobuf và simulator RPC không thay adapter này.
2. COMPENSATE trong placement.go chỉ loại COMMITTED rồi đánh FAILED với mọi state khác, kể cả RESERVED/UNKNOWN/chuỗi rỗng. Cần whitelist RELEASED/EXPIRED hoặc tombstone đã xác minh; kiểm order_id/reservation_id. HTTP 200 không được đóng nghĩa vụ khi payload chưa terminal.
3. COD_FINALIZE kiểm state COMMITTED nhưng không kiểm order_id/resource identity như nhánh payment FINALIZE. CANCEL cũng chưa kiểm r.OrderID trước cập nhật local stock. Chuẩn hóa validator chung cho response của mọi operation.
4. ReservationOutcome.state là string, thiếu version/decision time/reason/operation outcome. Cần enum và quy tắc tương thích; UNKNOWN thuộc quan sát caller khi timeout, không phải terminal persisted state.
5. Chương 07 và các chương v3.0 giữ mô tả gap cũ, chương 08 override. Ví dụ proto trên nhánh Order đã có 4 RPC mới, nhưng gap vẫn ghi “không có”; cần ghi rõ “đã có candidate/simulator, Inventory PR #11 chưa implement”.
6. Retry implementation backoff 0.05*2^n, cap 3.2 giây, không jitter; tài liệu chương 04 đề xuất 0.5 giây, cap 30 giây, full jitter. Cần thống nhất và cấu hình theo stage/deadline; tránh retry storm khi Inventory outage.

## 4. Nâng cấp giao thức đề xuất

Giữ flow Order 3.1:

```text
checkout operation durable
  → ReserveStock(all items, stable key)
  → PENDING_PAYMENT
  → verified receipt → PAYMENT_FINALIZING
  → FinalizeReservation atomic trong Inventory
  → COMMITTED → PAID + outbox
  → fulfillment-ready khi đủ mọi điều kiện
```

Nếu timeout: giữ obligation, query/retry cùng key, không suy diễn thất bại hay tạo hold mới. Nếu EXPIRED/RELEASED: giữ receipt, reconciliation/refund review, không ready.

Các quyết định cần đưa thành ADR chung Order–Inventory:
- Inventory sở hữu reservation/expiry và stock ledger; Order sở hữu quyết định thương mại và saga. Cleanup Inventory được tự expire hold, không tự refund/cancel Order.
- Duy nhất finalize command tạo commit; paid event trở thành fact hậu điều kiện. Bỏ legacy consumer trừ kho hoặc chuyển sang kiểm chứng/no-op cho COMMITTED; không duy trì hai đường có semantics khác.
- Chuẩn hóa PENDING của Inventory thành RESERVED tại adapter hoặc enum chung; SUCCESS chỉ là kết quả command.
- Finalize/release/query/reversal trả order_id, reservation_id, lifecycle state, version, expires_at, decision_at và reason. Key/hash có uniqueness, retention và conflict behavior.
- Release-by-order trước reserve phải tạo cancellation tombstone. Reserve đến muộn cùng planned_order_id không được tạo hold mới; NOT_FOUND đơn thuần không chứng minh không có request đang bay.
- Reversal cần fulfillment barrier và approval/policy được xác minh, không chỉ tin chuỗi approval do caller gửi. Phân biệt reversal chưa xuất hàng với return đã giao cần kiểm định; giữ ledger truy vết batch và ngăn cộng hai lần.
- Chốt MVP một kho; reject warehouse khác thay vì im lặng. FEFO thuộc Inventory, Order không chọn batch hoặc tự tính lại stock.
- Sổ tồn hiện tại: reserve tăng reserved; commit giảm cả physical và reserved; release giảm reserved. Không mô tả committed_quantity như cột có thật. Nếu cần tồn đã bán nhưng chưa giao, thêm ledger/projection rõ nghĩa ở phase riêng.

## 5. Kế hoạch và tiêu chí nghiệm thu

| Ưu tiên | Việc | Owner | Hoàn tất khi |
| --- | --- | --- | --- |
| P1 | Sửa consumer offset/lifecycle, reserve replay, release outcome | Inventory | regression cases dưới đây pass |
| P1 | ADR + proto terminal protocol + expiry/tombstone | Order + Inventory | contract thống nhất, server thật implement |
| P1 | Typed Inventory port + gRPC adapter, strict outcome validation | Order | suite chạy với PR #11 đã nâng cấp, không simulator |
| P1 | Chuyển paid thành fact, rollout consumer v1/v2 | Hai bên + Fulfillment | không vòng chờ, không double commit, không packing trước ready |
| P2 | Đồng bộ PR #17 với migration/proto thật | Inventory docs | không gọi schema giả là snapshot đã deploy |
| P2 | Metrics/alerts/replay/lock-handle/retry policy | Hai bên | vận hành có runbook và evidence |

Test bắt buộc trước integration release:
1. N lỗi transient rồi N+1 cùng partition thành công; không mất nghiệp vụ N.
2. Paid tới sau expiry/release; không ACK như committed thành công, không fulfillment.
3. Finalize đua cleanup/cancel ở deadline; chỉ một terminal outcome, stock invariant giữ.
4. Reserve timeout nhưng remote đã commit; replay cùng key không giữ hai lần.
5. Cùng key/order đổi items → conflict; replay RELEASED không báo active reserve.
6. Release COMMITTED không báo đã hoàn kho; reversal chỉ một lần.
7. Cancel trước reserve đến muộn; tombstone chặn hold mới.
8. Order crash sau remote finalize nhưng trước local PAID; resume/query hoàn tất đúng một paid/ready effect.
9. COD reserve/finalize/cancel đúng UNPAID, không dựa paid event.
10. Payload success nhưng state UNKNOWN/RESERVED/rỗng hoặc sai order → không đóng compensation/COD/cancel.
11. Thiếu Kafka ở production không startup thành công rồi xóa nghĩa vụ outbox.
12. Contract test dùng Inventory thật và PostgreSQL/Redis/Kafka, thêm concurrent same-SKU và multi-SKU FEFO.

## 6. PR #17 và giới hạn kiểm chứng

PR #17 có ích như phác thảo flow ngắn gọn, nhưng không khớp PR #11: REST confirm so với gRPC/Kafka; inventory_reservations/inventory_movements so với stock_reservations/allocations/adjustments; RESERVED/CONFIRMED so với PENDING/COMMITTED; thiếu batch FEFO và outbox. Không dùng tài liệu này làm implementation contract hiện tại. Cần sửa dựa migration/proto hoặc ghi rõ đó là phương án thay thế chưa implement.

Review dựa trên mã tại các SHA đã nêu; chưa chạy integration Order–Inventory thật. Không thay đổi code nghiệp vụ hay tự ghi review lên GitHub.

Đã chạy `go test -count=1 ./...` trên bản export Inventory head `2b9f179` bằng container Docker và `GOTOOLCHAIN=auto` (Go 1.25.0): tất cả 11 package có tests PASS; 4 package không có tests. Lần thử Go 1.22 bị từ chối vì go.mod hiện yêu cầu 1.25.0, nên mô tả PR “Go 1.22” cũng cần cập nhật. Chưa chạy race/load hay infrastructure integration; unit/mock PASS không phủ nhận các finding về offset, lifecycle và contract ở trên.


# 17 — Order Persistence và Messaging

**Phiên bản:** `order-persistence-messaging-1.0-review` · **Ngày:** 17/10/2026 · **Trạng thái:** Đề xuất để review; chưa là migration hoặc delivery/recovery implementation đã nghiệm thu.

## 1. Data ownership, authority và quyết định chung

Đặc tả persistence/messaging chung cho [Payment09](09_payment_sepay_vietqr_design.md), [Checkout11](11_cart_checkout_design.md), [Order Core12](12_order_core_design.md), [Worker13](13_order_saga_worker_runtime.md). Các chương nghiệp vụ giữ guards/proof requirements; chương này thống nhất physical mapping, transactions, outbox/inbox và recovery. Khi bảng tổng quát ở [Persistence03](03_database_and_persistence.md) hoặc [Contracts05](05_api_contracts_and_transports.md) chưa phản ánh amendment, dùng chi tiết tại đây cho cohort mới; không tự thay contract/runtime baseline.

| Dữ liệu | Authority | Vai trò local |
| --- | --- | --- |
| Accepted checkout, snapshot, Order decisions | Order PostgreSQL | Durable input, current root, history và obligations |
| Receipt/allocation/refund budget | Payment module trong Order PostgreSQL | Canonical financial records; provider/bank evidence được xác minh |
| Reservation/commit/release/reversal | Inventory | Order lưu command identity/proof/projection, không tự đặt state thay provider |
| Fulfillment accept/cancel barrier | Fulfillment | Order lưu generation/decision, cấp permit theo current guards |
| Shipment/settlement, Care decision | Service tương ứng | Source projections và trusted decision references |
| Cart, quote preview | Redis | Admission input/CAS; accepted operation không phụ thuộc TTL sau T0 |
| Event delivery | PostgreSQL outbox và Kafka ACK | Kafka không là nguồn chuẩn của Order snapshot hoặc remote effect |

Các quyết định của bản này:

1. PostgreSQL16 là dialect candidate; READ COMMITTED + explicit locks/unique constraints là default.
2. Dùng shared outbox có aggregate identity NOT NULL; Order, receipt và refund là các stream riêng. order_id chỉ là optional association.
3. Publisher claim **stream**, replay head chưa xác nhận trước next head. PostgreSQL epoch chỉ fence DB writes. Chấp nhận duplicate/late records và bắt buộc consumer guard; chưa chọn Kafka transactional producer fencing hoặc CDC.
4. Inbox disposition và business/deferred responsibility commit atomic trước khi offset eligible.
5. DDL ở [candidate schema](order_persistence_messaging.candidate.sql) dành cho empty isolated schema `order_pm_review`, không thay [schema sandbox](order_schema_candidate.sql) hay chạy vào database hiện có.

## 2. Unified PostgreSQL schema

### 2.1 Kiểu dữ liệu và quy ước

PK nội bộ là application-generated UUID; provider ID là TEXT và có unique theo connection/scope, không ép numeric webhook ID và canonical transaction UUID thành cùng identity. Tiền VND dùng BIGINT dương cho receipt/refund, nonnegative cho thành phần giá; Order final >0 và ≤10^12. Kiểm overflow trước parse/multiply/sum; không dùng float. Receipt không bị giới hạn bằng giá trị Order để vẫn lưu khoản chuyển thừa.

Thời gian TIMESTAMPTZ, lưu instant UTC, hiển thị theo timezone policy. Version/epoch BIGINT không âm; business version bắt đầu1, readiness generation bắt đầu0. Hash canonical SHA-256 là lowercase hex64; HMAC lookup có key dùng cùng kiểu biểu diễn nhưng khác purpose/key. BYTEA cipher là versioned AEAD envelope có key ID/nonce/AAD; không lưu secrets hoặc raw PII trong JSON policy/headers.

JSONB chỉ dùng cấu trúc mở có schema/version như policy/deadlines/headers; identities, amount, status, deadlines, lease và query keys dùng typed columns. Internal snapshot chỉ có một nguồn chuẩn tại orders.snapshot_cipher/hash/version. order_items là immutable projection của snapshot, được kiểm consistency lúc PLACE; không refresh Catalog để viết lại đơn cũ.

### 2.2 Catalogue và physical mapping

DDL candidate là nguồn chính xác cho cột, kiểu, NOT NULL, PK/FK/UNIQUE/CHECK/index. Bảng dưới chỉ định ownership và invariants ngoài DDL:

| Owner | Physical tables | Invariant cần transaction/validator |
| --- | --- | --- |
| Checkout | checkout_operations, idempotency_records, checkout_quote_consumptions, checkout_submission_guards | Input/quote/scope/hash association đúng; một consumed quote, guard chỉ takeover sau safe failure |
| Order Core | orders, order_items, order_holds, order_cancellations, command_registry, order_milestones, order_history | Snapshot/items/totals nhất quán, transition/approval đúng, history cùng root version |
| Payment configuration | payment_provider_connections, payment_receiver_profiles | Profile version bất biến về receiver terms; accepting_new_payments là control flag |
| Payment money | payment_intents, receipts, allocations, refunds, refund_execution_attempts | Allocation exact eligibility, tổng budget refund không vượt receipt, no second send UNKNOWN |
| Payment evidence/access | payment_provider_observations, payment_observation_conflicts, payment_evidence, guest_payment_grants | Canonical receipt identity, immutable evidence, scoped grant/revocation |
| Payment recovery | payment_order_projections, reconciliation_cases, payment_sync_runs, payment_sync_checkpoints | Projection cùng financial decision, continuous coverage, case có nguồn và owner |
| Workflow runtime | workflows, jobs, external_effects, external_attempts, external_observations, external_observation_dispositions, runtime_cases | Same key/hash, fenced apply, stale evidence retained, UNKNOWN có recovery plan |
| Cart cleanup | cart_cleanup_results | SQL result map Lua proof/marker; không ghi APPLIED chỉ vì request đã gửi |
| Messaging | event_streams, outbox_events, inbox_events, inbox_deliveries, deferred_events, message_quarantine, source_projections | Stream append/progress atomic, inbox+business atomic, prerequisites/version không bị bypass |
| Audit | audit_records | Append-only actor/reason/evidence; không thay decision authority |

**Mapping amendment:** jobs là physical runtime queue cho workflow của mọi aggregate, không có NOT NULL order_id. workflows giữ aggregate_type/id và nullable Order FK. payment_work_items, checkout worker fields và order_workflows/resource_outcomes trong các chương trước là logical roles được map vào jobs/workflows/external_effects; không tạo hai executor cùng sở hữu một effect. payment_sync_runs dùng lease của job, không thêm lease độc lập. Legacy cohort có adapter riêng trong giai đoạn migrate.

Riêng OUTBOX_PUBLISH là logical registry entry được schedule/lease trực tiếp bằng event_streams: không tạo thêm LEASED jobs row để cạnh tranh với stream lease. PostgreSQL stream epoch là ownership authority của publisher; Worker13 cung cấp timing/fairness/delivery-budget semantics. Như vậy không có hai lease độc lập quyết định ai được advance cùng stream.

`orders.operation_id UNIQUE` là canonical link operation→Order; không duy trì thêm writable checkout_operations.order_id. API operation order_id được project bằng link này. `command_registry` là physical mapping của order_command_registry; `order_cancellations` map cancellation workflow control, runtime progress nằm ở workflows. Ownership history/snapshot amendments tương lai chưa bật, không thêm bảng placeholder chưa có semantics.

Receipt **không cần Order**: receipts liên kết connection/receiver; allocations tạo quan hệ với Order khi đủ guard. refunds.order_id nullable, receipt_id luôn NOT NULL. Workflow payment unmatched không được tạo Order giả. Event financial có stream RECEIPT hoặc REFUND với internal UUID, order_id có thể NULL.

## 3. Constraints, indexes, FK và migration

### 3.1 Database enforcement và application enforcement

| Invariant | Enforcement |
| --- | --- |
| Một quote_id tối đa một accepted operation | checkout_quote_consumptions PK quote_id, ghi cùng T0 |
| Một operation tối đa một Order, đúng planned ID đã dùng acquisition | orders.operation_id UNIQUE và composite FK(operation_id,id) → checkout_operations(id,planned_order_id) |
| Một VietQR intent/Order, unique payment code theo connection | payment_intents unique order_id và (connection_id,payment_code) |
| Canonical receipt chỉ ghi một lần | receipts unique(connection_id,transaction_id), hash conflict path riêng |
| MVP một allocation/receipt/Order | allocations receipt_id UNIQUE, order_id UNIQUE |
| Một active cancellation/Order | Partial unique index order_cancellations(order_id) WHERE active |
| First paid/completed một lần | order_milestones PK(order_id,kind) + guards |
| Một remote action identity | external_effects unique(provider,scope,action,stable_key), hash immutable qua runtime |
| Event identity/position | Outbox event_id PK; unique(type,id,version,ordinal) và (stream_id,sequence), mọi identity NOT NULL |
| Repeated delivery không reapply | Inbox unique(consumer_contract,source,event_id) + hash check trong transaction |
| Cross-row refund budget và exact amount | Lock receipt parent, tính toàn bộ reserved/settled obligations rồi ghi atomic; không dùng cross-table CHECK |

PostgreSQL UNIQUE thông thường cho phép nhiều NULL; do đó outbox identity không dùng nullable order_id. CHECK phải đi cùng NOT NULL cho mandatory values; cross-table arithmetic thuộc transaction/validator, không giả được bảo vệ bởi row CHECK. Xem [PostgreSQL constraints](https://www.postgresql.org/docs/16/ddl-constraints.html).

Mọi FK trong candidate là cùng database/schema, ON DELETE mặc định NO ACTION; không cascading xóa ledger. Inventory reservation_id, warehouse, customer identity, Fulfillment task và approval refs là external references, không cross-database FK. Polymorphic workflow/stream aggregate association được verify bằng typed module transaction; không khai generic aggregate_id là FK orders. Composite FK outbox→stream ngăn mismatch stream identity; receiver+connection FK ngăn cross-connection binding.

Indexes phục vụ owner keyset, warehouse/status/SLA queue, pending payment expiry, receipt lookup, open refund budgets, lane/run_after, expired lease, due obligations, stream head, deferred/quarantine due. PK/UNIQUE tự có supporting index; không tạo bản trùng. FK child indexes bổ sung theo measured access/delete/archive plan trước deploy. Đánh giá EXPLAIN với volume dự kiến; không tự partition bảng làm mất global uniqueness event/receipt.

### 3.2 Immutability và quyền ghi

Candidate có triggers chặn UPDATE/DELETE history, milestones, receipts, allocations, quote consumption, immutable item và attempt/observation/audit evidence. Mutable validation tách external_observation_dispositions; financial correction lưu fact/evidence mới, không sửa receipt gốc. Outbox trigger giữ nguyên identity/payload/headers/time, chỉ publish fields đổi. Snapshot trigger giữ commercial identity/hash/version/amount; cipher rewrap chỉ bởi audited maintenance path và plaintext logical hash không đổi.

Least-privilege deployment phải tách migration owner, app writer, append-only evidence writer, publisher và archival maintenance roles. Triggers không bảo vệ trước privileged DBA hoặc ciphertext rewrite bởi owner; không tuyên bố tamper-proof. Các enum state chưa cố định provider-specific được pin bởi input/policy schema và service guard; migration triển khai có thể thêm CHECK sau khi provider mapping đã nghiệm thu.

Publisher chỉ SELECT event bytes và UPDATE publish metadata/stream ownership/progress, không UPDATE payload hoặc DELETE outbox. Stale evidence writer chỉ INSERT observations/dispositions ban đầu, không sửa validated disposition/root. Business writer chỉ được UPDATE các operational columns của effect/job; stable key/hash/input identity không được sửa qua recovery command. Maintenance owner/grants phải được review trong deploy migration, không dùng schema owner làm runtime account.

### 3.3 Expand → backfill → validate → switch

1. Inventory schema hiện có, cohorts và unresolved keys; không chạy CREATE schema candidate để thay database live.
2. Expand nullable fields/new tables/indexes, giữ legacy reader/writer. Backfill connection theo provider/environment/cohort; không trộn webhook aliases với API transaction UUID.
3. Tạo unique connection+transaction identity sau khi reconcile duplicates; validate FK/CHECK rồi mới NOT NULL/switch. CREATE INDEX CONCURRENTLY là phase riêng ngoài transaction khi áp dụng; giới hạn backfill batch/lock time.
4. Backfill workflow/effect identities từ durable evidence, pin original keys/hash. Không suy terminal outcome từ Order status; record thiếu association vào manual recovery.
5. Một cohort chỉ một writer/commit authority. Drain hoặc fence legacy claim trước chuyển work sang jobs; tránh hai queue cùng chạy. Existing lease không được copy thành quyền sở hữu mới mà không invalidate owner cũ và reconcile inflight requests.
6. Consumers hỗ trợ contract mới trước producer; canary theo cohort, so ledger/count/hash, có rollback routing. Rollback không xóa receipt, registry, outbox hoặc remote obligations. Contract breaking cần version/cohort mới.

## 4. Transaction boundaries và isolation

Default READ COMMITTED, explicit row locks, expected versions và unique constraints. READ COMMITTED không làm một chuỗi nhiều SELECT trở thành snapshot nhất quán; phải lock invariant parent hoặc dùng coherent provider proof. SERIALIZABLE chỉ dùng khi đã chứng minh invariant không đủ bảo vệ bằng constraints/locks và có retry contract; không bật toàn database theo mặc định. Tham khảo [PostgreSQL isolation](https://www.postgresql.org/docs/16/transaction-iso.html).

| Tx | Atomic writes / authority |
| --- | --- |
| T0 acceptance | Checkout operation/input + idempotency + SQL quote consumption + submission guard + workflow/job + audit |
| Tcanonical | Accepted canonical terms + fixed commercial deadline + acquisition effect intent/key/hash + next job |
| Tattempt | Current lease validation + immutable dispatch attempt trước external send |
| Tobserve | Append authenticated evidence + initial validation disposition; stale worker được append, không mutate Order |
| Tapply | Current owner/guard + validated proof projection + workflow/job disposition + next obligation |
| T3 PLACE | Order/items/snapshot + VIETQR intent hoặc COD ready work + history/placed outbox + cleanup work + consumed guard + operation success |
| Treceipt | Canonical receipt nếu mới; eligible allocation + FINALIZING/history/job hoặc durable case; financial outbox khi contract đã bật |
| Tpaid | Proof/current guards + FIRST_PAID + PAID/history/outbox + ready intent |
| Tcontrol/Tresume | Hold/cancel/resume control + generation/registry/history/outbox + external work intent |
| Trefund | Receipt lock + verified approval + reserved budget + immutable attempt intent; settlement có proof ở transaction kế |
| Tappend | Stream sequence allocation + immutable event append cùng owning business transaction |
| Tpublish | Current stream lease/epoch + head ACK metadata + published head advance atomic |
| Tconsume | Inbox disposition + business effect/history/outbox hoặc deferred/quarantine+responsibility + delivery disposition |

Không RPC/Redis/Kafka send bên trong SQL transaction. `synchronous_commit=on` là baseline writer proposal; HA/RPO phụ thuộc replication/failover contract được Ops nghiệm thu, không coi local WAL ACK là zero data loss sau mọi failover.

### 4.1 Redis → SQL acceptance

Redis chỉ đọc/xác thực quote bytes/hash, owner, incarnation/revision/selection bằng guard của Checkout11. Copy accepted snapshot vào SQL. T0 unique quote consumption là authority dùng quote, bất kể client dùng key khác hay Redis marker chưa được ghi. Một quote_id có tối đa một accepted operation.

Nếu SQL commit thành công nhưng Redis update/cleanup thất bại, operation vẫn accepted; không trả business failure để người dùng tạo key mới. Marker Redis nếu có chỉ advisory. Nếu Redis lỗi trước admission và không có replay SQL hợp lệ thì không accept mới. Quote hết hạn sau T0 không ảnh hưởng input durable; commercial terms vẫn theo deadline riêng. Mutation sau Redis admission nhưng trước SQL commit vẫn là race được chấp nhận theo Checkout11, không hứa transaction xuyên hai hệ thống.

## 5. Lock ordering, concurrency và commit UNKNOWN

### 5.1 Thứ tự lock chuẩn

Trong một application transaction: **Order root theo UUID → checkout operation/identity control nếu liên quan → intent/holds/cancellation/command controls theo ID → receipts theo ID → refunds → workflows → external_effects → jobs → event_streams theo ID**. Các row cùng nhóm sort ID; writer cần nhiều Orders sort trước. Publisher chỉ lock stream/outbox, không giữ stream lock rồi quay lại Order. Inbox insert/dedup lock trước business path được phép chỉ khi không có business writer nào lấy inbox lock; replay claim phải commit riêng trước business apply. Không lấy job lock rồi Order trong cùng transaction.

Pre-placement không có Order để lock: checkout operation là parent. PLACE lock operation rồi INSERT Order mới chưa visible; không lock một existing Order sau operation. Nếu Order đã tồn tại, đọc kết quả canonical và replay, không chạy ngược hierarchy. Unmatched receipt path lock receipt khi cần và không quay lại lấy Order; linking transaction mới lấy Order trước receipt. Refund không có Order thì receipt là invariant parent. Claim/heartbeat chỉ job hoặc stream, không giữ locks khi network.

Unique-key lock cũng có thể deadlock: inserts theo thứ tự cố định, giới hạn transaction size, retry SQLSTATE. Không dùng một SELECT row không tồn tại để bảo vệ acquisition identity; dùng unique registry/parent row protocol. Decision clock dùng clock_timestamp() sau locks; transaction-start now() không thay decision clock khi đã chờ lâu.

### 5.2 Failure handling

| Failure | Hành vi bắt buộc |
| --- | --- |
| 40P01 deadlock | Rollback và retry toàn SQL transaction bounded, same identity/input |
| 40001 serialization | Rollback và chạy lại transaction từ đầu, re-read/guard; không chỉ retry statement cuối |
| 23505 unique | Rollback, lookup constraint-specific business identity; same hash replay, khác hash conflict |
| Lock timeout/query cancel | Rollback/verify session state; retry bounded nếu còn deadline, không giả business reject |
| COMMIT ACK mất / connection lost | Commit outcome UNKNOWN; lookup authoritative primary bằng identity trước retry/compensation |
| DB unavailable khi lookup | Giữ UNKNOWN; trả acceptance unknown hoặc giữ durable recovery work, không sinh ID/key mới |
| Remote timeout | Durable effect UNKNOWN, query original key/resource theo Worker13 |

Default SQL retry max3 attempts trong request/stage budget, full jitter25–250ms; hết budget trả retryable/UNKNOWN theo actual phase. Retry không phát lại remote call trong transaction closure. Unique-conflict lookup có thể cần chờ transaction cạnh tranh xong; đọc replica có lag không được dùng kết luận chưa commit.

SQLSTATE mapping tham chiếu [PostgreSQL error codes](https://www.postgresql.org/docs/16/errcodes-appendix.html); xử lý theo code và constraint identity, không parse localized error message.

| Điểm UNKNOWN | Lookup identity |
| --- | --- |
| T0 | principal scope+action+idempotency key; đồng thời quote consumption/submission guard nếu cần |
| PLACE | operation_id → orders.operation_id; VIETQR kiểm intent liên quan |
| Receipt | connection_id+canonical transaction_id, allocation/milestone |
| Paid/control/refund | command registry, Order version/milestone, approved refund operation_key |
| Inbox | consumer_contract+source+event_id, payload hash và durable disposition |
| Publish | Stream head + immutable event ID; ACK chưa durable thì replay same head |

Lookup absent một lần trong khi transaction cũ còn unresolved không chứng minh rollback. Retry cùng unique identities sẽ serialize/no-op/conflict; không compensate cho đến khi local outcome và external effect đủ evidence.

## 6. Transactional Outbox schema và append

`event_streams` có aggregate_type/id NOT NULL, routing/topic/key pin, last_sequence, published_sequence, lease owner/until/epoch và blocked state. `outbox_events` có event_id, stream identity, business aggregate_version/ordinal, stream_sequence, immutable payload bytes/hash/schema/headers/time, optional order_id và publish metadata. Candidate FK bảo đảm event thuộc đúng stream.

Order stream: aggregate_type=ORDER, aggregate_id=Order UUID, key=order_id, topic=order.events.v3. Receipt stream: RECEIPT/internal receipt UUID, key=receipt_id. Refund stream: REFUND/internal refund UUID, key=refund_id. Không dùng NULL order_id như stream identity; không hứa ordering giữa ba stream dù có cùng Order association.

Append trong owning business transaction: lock/create stream theo unique identity, increment last_sequence cho mỗi event theo deterministic ordinal, insert immutable rows, commit cùng business changes. Nếu transaction rollback sequence allocation cũng rollback. last_sequence là contiguous **internal event sequence**, không phải Order version. Business versions có thể không phát event; consumer không đợi mọi số version đều xuất hiện.

Schema payload v3 hiện đóng; không thêm stream_sequence vào JSON đang có. Sequence/routing metadata có thể ở pinned Kafka headers cho contract consumer đã review, nhưng correctness vẫn theo event identity/resource prerequisite; không ép consumer cũ hiểu headers mới. Payload persisted là bytes chính xác đã validate; retry không dựng lại payload/time từ current Order.

Publisher fields PENDING/PUBLISHED/BLOCKED không sửa event fact. Poison outbound event chặn stream, tạo case; không skip sequence hoặc phát bản sửa cùng event ID. Sửa schema/adapter cần correction/replay plan có audit; superseding event phải có identity mới và consumer contract rõ.

## 7. Publisher ownership, ordering và failover

### 7.1 Protocol được chọn

1. Claim stream ACTIVE có backlog, đến run_after, lease free/expired bằng SKIP LOCKED; epoch tăng, lease default10s theo Worker13. Không claim từng event tùy ý.
2. Chỉ đọc head `stream_sequence=published_sequence+1`. Một outstanding send/head/stream; không prefetch-send next head khi ACK trước chưa commit. Validate payload/schema và routing đã pin.
3. Commit claim; send immutable head ngoài SQL. Kafka ACK callback mang stream_id/head/event_id/epoch, không chỉ job ID.
4. ACK → transaction CAS current owner/epoch/lease, đúng head/event → mark PUBLISHED + partition/offset/time + advance published_sequence atomic.
5. Chỉ sau commit Tpublish mới gửi head tiếp. ACK/commit outcome UNKNOWN → lookup progress, chưa confirmed thì replay head cũ.
6. Mất lease: ngừng send mới, đóng producer local best-effort; không còn quyền mark/advance. Owner mới luôn bắt đầu head chưa confirmed. Stream BLOCKED không phát successors, có alert và recovery owner.

Kafka producer baseline đề xuất: acks=all, enable.idempotence=true, retries>0, max.in.flight.requests.per.connection=1 cho publisher ban đầu. Config adapter Go phải map đúng semantics/client support và test. Idempotence trong producer session không dedup arbitrary application replay giữa producer instances; stable event ID và inbox vẫn bắt buộc. Xem [Kafka producer configuration](https://kafka.apache.org/40/configuration/producer-configs/).

### 7.2 Điều được bảo đảm và giới hạn

| Tình huống | Hành vi |
| --- | --- |
| A gửi v10 rồi mất lease trước ACK | B replay head v10, chưa được nhảy v11 |
| B ACK/commit v10, sau đó gửi v11 | DB progress hợp lệ; A không có quyền advance |
| Request v10 cũ của A đến Kafka sau v11 | Có thể xuất hiện v10 duplicate/late; consumer dedup và không đảo state |
| A có ACK nhưng SQL Tpublish chưa commit | Owner kế replay same ID; không tự tin rằng broker chưa nhận |
| Head missing/corrupt hoặc schema unsupported | Block stream/case; không tự increment published_sequence |

**Không cam kết broker log luôn tăng version tuyệt đối qua failover.** PostgreSQL epoch chỉ fence local state. Protocol bảo vệ head progression, còn late requests được xử lý bằng consumer business ordering/dedup/prerequisite. Nếu consumer bắt buộc raw log strict order ngay cả duplicate late, capability đó chưa được đáp ứng: cần design/version riêng cho Kafka transactional producer với stable transactional.id theo shard và broker fencing, hoặc CDC đã kiểm chứng. Không gọi enable.idempotence là producer ownership fencing.

Publisher workers vẫn dùng delivery budget/fairness; hết một cycle không mất event. Leases/stream TTL không purge payload. Routing partition count/hash strategy pin theo cohort; tăng partition count có thể chuyển key sang partition khác nên phải drain/migrate routing có barrier, không hot-change rồi dựa vào order cũ.

event_streams lưu cycle_started_at/cycle_attempts/run_after/error để E_DELIVERY sống qua restart; increment dispatch intent dưới current stream epoch trước send. Hết10 attempts/15m cycle thì alert/backoff rồi cycle mới cùng head ID, không advance head hoặc reset identity. Broker/client delivery timeout phải bounded và pin trong adapter config; application timeout/mất lease không chứng minh client buffer/request đã hết khả năng gửi. Heartbeat/progress và stale callback vẫn theo fencing ở trên.

## 8. Inbox, deferred, quarantine và replay

### 8.1 Consume transaction

Consumer_contract là logical handler contract/version, ổn định qua rebalance và deployment; không dùng instance UUID. Hai handler khác nghiệp vụ có namespace riêng. Restart consumer-group không đổi namespace để né dedup. Validate source ACL, type/schema, identity, subject/key equality, payload hash/size trước business apply.

1. Nhận record; auto-commit tắt. Compute canonical envelope hash theo contract, loại transport offset/redelivery metadata; persist raw evidence encrypted. Không silently normalize hai payload khác thành cùng hash.
2. Trong SQL insert/dedup inbox identity. Same identity/hash đã có disposition → no-op business, ghi delivery disposition nếu đây là offset mới. Same ID khác hash → giữ original inbox, thêm quarantine conflict và alert.
3. Event mới: kiểm prerequisite và current source/resource versions dưới business locks. Đủ → business changes/history/outbox + inbox APPLIED. Thiếu → inbox DEFERRED + raw payload + prerequisite + workflow/job trách nhiệm replay. Invalid/unsupported → QUARANTINED + case/retained bytes. Tất cả cùng commit.
4. Commit SQL xong mới đưa offset vào eligible set. Commit outcome UNKNOWN → lookup, chưa rõ thì không commit offset.

Inbox APPLIED không được ghi trước business commit. Inbox source ID không parse được/envelope hỏng dùng synthetic quarantine identity theo cluster/topic/partition/offset + payload hash; không tạo event_id nghiệp vụ giả. inbox_deliveries có thể trỏ NULL inbox khi chỉ có durable quarantine; transaction phải tạo quarantine record tương ứng.

### 8.2 Offset và rebalance

Mỗi topic-partition chỉ commit **next offset** sau contiguous prefix mọi delivered record trước đó đã có durable disposition. Không lấy MAX(offset completed) khi đang xử lý song song. Nếu có offset100 chưa durable, offset101 xong vẫn không commit102. DEFERRED hoặc QUARANTINED đủ payload/job/case durable được tính là disposition, không giữ partition vô hạn vì business prerequisite.

Lúc partition revoke: ngừng dispatch mới, drain bounded pending SQL; commit chỉ eligible prefix trong current assignment, lỗi commit vì generation cũ thì để redelivery. Transaction SQL đang bay có thể commit sau revoke: current handler không được commit offset cũ, owner mới dedup qua database unique constraint; không rollback giả business effect đã committed. Consumer poll/heartbeat chạy trong supported threading model, pause partition khi DB/backlog không nhận thêm; không gọi KafkaConsumer tùy ý từ nhiều thread. Tham khảo [KafkaConsumer API](https://kafka.apache.org/40/javadoc/org/apache/kafka/clients/consumer/KafkaConsumer.html).

### 8.3 Business ordering và replay

Event version12 đến trước prerequisite11: lưu DEFERRED và prerequisite key, không apply state cần11. Khi prerequisite được apply, enqueue/coalesce replay cùng inbox identity. Version chỉ tăng không đủ chứng minh đã có mọi fact cần thiết; không skip tất cả lower version khi chúng mang effect/resource obligation chưa xét. Không đợi mọi integer version nếu schema không hứa contiguous event sequence.

Source version cũ không undo newer resource state, nhưng financial/effect evidence vẫn phải được lưu và reconcile. Different sources/receipt/refund streams không có total order; query authoritative resource khi thiếu/contradictory evidence. Deferred age quá threshold mở case và query work; không tự synthesize missing event hoặc set APPLIED để dọn hàng.

Replay claim ở transaction riêng, sau đó transaction business re-lock inbox/current aggregate, revalidate proof/schema/permissions và chuyển DEFERRED→APPLIED atomic. Hai replayers chỉ một apply, job DONE không phải authority. Quarantine re-drive cần audited decision và handler/schema support; cùng payload ID giữ identity. Payload sửa phải là correction có identity/provenance mới theo source authority, không overwrite original quarantine.

DLQ là diagnostic delivery tùy chọn, **PostgreSQL quarantine mới giữ responsibility** trước commit offset. DLQ publish intent cùng transaction, publish lỗi không mất original bytes; production không chỉ log rồi ACK. Independent recovery scanner phát hiện deferred thiếu job, quarantine thiếu owner và offset delivery inconsistency.

## 9. Kafka topics, consumers, schemas và evolution

| Topic / status | Key và facts | Consumer contract / effect |
| --- | --- | --- |
| order.events.v3 — target đã có schemas | Order UUID; placed/paid/ready/cancelled/completed/audit theo pinned types | Fulfillment chỉ nhận ready để xin permit/barrier; Notification/Accounting theo subscription riêng; paid không deduct stock |
| payment.events.v1 — candidate, chưa enable | RECEIPT/REFUND UUID; receipt_recorded/refund_settled facts gồm unmatched money | [JSON Schema candidate](../../../03_api_specs/order-payment-events.candidate.schema.json) và [fixtures](../../../03_api_specs/order-payment-events.candidate.examples.json); cần Finance/Accounting sign-off semantics/privacy/dedup |
| Provider Inventory/Fulfillment/Shipping topics | Key/version theo source contract | Order lifecycle/source projections; chỉ subscribe khi ACL/schema/association được nghiệm thu |
| order.messaging.quarantine.v1 — optional proposal | Quarantine UUID; redacted diagnostic reference | Ops tooling; raw financial/PII evidence chỉ qua scoped DB/object access |

Shared physical outbox **không** có nghĩa nhét Payment facts vào Order v3 schema. Candidate có hai CloudEvents types `vn.omama.payment.receipt_recorded.v1` và `vn.omama.payment.refund_settled.v1`, source Order-service, subject receipt/refund UUID. Validator bắt buộc subject/key/aggregate_id bằng receipt_id hoặc refund_id; JSON Schema không tự kiểm cross-field equality. Required order_id cho phép null, amount là positive int64 VND; consumer phải parse integer chính xác, không IEEE754 làm tròn. Event không chứa bank account, nội dung chuyển khoản hoặc customer/actor PII.

receipt_recorded phát một lần khi canonical receipt commit, **không** có nghĩa đã allocate hoặc PAID. order_id chỉ association đã được xác minh tại thời điểm append, có thể null rồi match sau; không rewrite event cũ hoặc phát lại receipt_recorded với ID mới để báo match. refund_settled chỉ phát khi settlement đã verify và budget/projection terminal commit; SUBMITTED/UNKNOWN không được phát settlement. Receipt/refund decisions và outbox cùng transaction. Nếu subscription cần allocation/correction facts, thêm schema/version riêng; hai types này chưa đại diện full Finance ledger integration.

Payment schema/consumer/Finance approval vẫn là explicit activation gate. Unmatched receipt luôn được lưu và có durable case/work; khi chưa enable publish capability, không mất tiền hoặc tạo Order giả. Acceptance phải kiểm event path khi schema đã được phê duyệt/bật.

Mỗi logical subscriber pin event types/schema hashes, consumer_contract namespace, source ACL, prerequisites và replay semantics. Exact group names/partitions theo deployment manifest, không invent group đã tồn tại. Unsupported major/type/version → quarantine+alert, không parse sang DTO gần giống; current closed v3 additions phải qua schema version review. Compatible migration deploy readers trước writers, cohort/drain rõ; không dual-publish legacy paid cho effect consumers của new-flow Orders.

Production durability baseline proposal: replication factor3, min.insync.replicas2, producer acks=all, unclean leader election disabled; local single-broker sandbox không được gọi tương đương. Topic retention khởi điểm14d, cleanup.policy=delete cho business facts, **không dùng compact để thay event history**. Các số cần capacity/recovery review và actual cluster/client version pin trước rollout; Kafka retention không quyết định ledger retention. Nếu consumer lag gần retention horizon, alert/backfill trước mất coverage, không reset latest tùy ý. Tham chiếu [Kafka topic-level configuration](https://kafka.apache.org/40/configuration/topic-level-configs/).

## 10. Encryption, retention, backup và restore

### 10.1 Retention classes

| Nhóm | Policy target |
| --- | --- |
| Checkout response |7d như contract; sau đó410/reference, không reuse identity |
| Idempotency/consumption/remote keys | Tối thiểu30d và vượt toàn replay/inflight horizon; nếu chưa chứng minh horizon thì không auto-purge identity |
| Money, allocations, refund/effect evidence | Không đặt TTL tự xóa trong MVP; retention theo Finance/business policy cần sign-off, unresolved không purge |
| Outbox payload đã publish | Hot tối thiểu30d proposal, sau đó archive verified immutable payload+identity/hash/position; event chưa confirm/block không purge |
| Inbox/quarantine/deferred | Giữ identities vượt mọi replay horizon; deferred/quarantine unresolved không purge; raw evidence archive theo privacy policy |
| Job/attempt/history/snapshot | Terminal có thể archive sau dependency/reference checks; không xóa nghĩa vụ hoặc snapshot còn cần phục vụ Order |
| Redis cart/quote/markers | Theo Checkout11; không rút marker retention ngắn hơn replay policy |

Archive phải verify count/hash/readability và FK/reference plan trước hot delete; identity tombstone/dedup lookup vẫn dùng được. Immutable triggers/NO ACTION là guard mặc định; archive dùng reviewed maintenance migration có role/audit và restore test, không disable trigger trong request path. Key rotation/rewrap giữ logical hash, decrypt keys đủ cho retention; crypto-erasure chỉ sau kiểm obligations và privacy/Finance policy, không xóa ledger để dọn PII.

### 10.2 Backup/restore là correctness protocol

Baseline proposal Ops: encrypted backups + WAL/PITR, RPO≤5m/RTO≤60m làm mục tiêu đo, không là kết quả đã đạt; financial data loss có thể đòi hỏi tighter HA contract. Restore drill phải có Kafka/provider state mới hơn backup.

1. Đóng admission/mutation workers/publisher theo affected cohort; vô hiệu old deployment credentials/consumer assignments/producer senders trước mở restored state. DB epoch bị rollback không fence được process cũ còn sống.
2. Restore vào isolated environment, kiểm schema/keys/cipher/ledger integrity. Không tự mở worker vì healthcheck DB đã xanh.
3. Thu thập external Inventory/Fulfillment outcomes, SePay/account/bank coverage, refund evidence và Kafka retained/archive events sau recovery point. Giữ original effect keys/IDs; reconstruct association mất trong backup bằng evidence có audit, không chạy Reserve/Refund như mới.
4. Phục hồi dedup và local outcomes từ trusted evidence/replay trong recovery mode: inbound event không tự cấp quyền phát financial/stock effect. Phân biệt Order facts đã phát với authoritative remote state; không lấy OrderPaid thay Inventory proof.
5. Reconcile outbox đã được Kafka nhận nhưng DB mark lost: stable ID replay có thể xảy ra; consumers phải dedup. Nếu dedup cũng lost, recovery guard phải kiểm business identity/milestones/current resources trước effects. External downstream có side effects phải có recovery/idempotency contract riêng; restore local không bảo đảm thay họ.
6. Lập unresolved cases/coverage gaps, xác minh balances, stock/barrier decisions và routing. Chỉ mở từng capability sau operator sign-off theo evidence, không reset offsets về earliest/latest rồi mở mọi mutation.

## 11. Observability và runbooks

| Signal / fault | Hành động |
| --- | --- |
| SQL40P01/40001/lock waits/commit UNKNOWN age | Trace identity, kiểm hierarchy/hotspot, lookup primary; không tạo key mới |
| Stream head blocked/oldest unpublished/publish ACK uncertainty | Ngừng successors cùng stream; replay head ID sau kiểm current ownership |
| Stale publisher callback | Reject progress write, đếm metric; expected late record phải dedup được |
| Inbox hash conflict/unsupported schema | Quarantine durable, alert publisher/contract owner, không overwrite original |
| Deferred age/prerequisite gap | Query authority, repair/replay cùng identity, không force APPLIED |
| Consumer lag/offset commit failure/rebalance storm | Pause/backpressure, bảo vệ eligible prefix, tăng capacity trong dependency quota |
| Unmatched receipt/refund UNKNOWN | Payment case/assignee + continued reconciliation, budget không release |
| Archive/decrypt/restore coverage fail | Chặn purge/mutation capability, giữ evidence và escalations |

Logs/traces: operation/order/workflow/job/effect/event/stream ID, epoch, schema/policy version và reason code; không log raw cipher plaintext/token/account PII. Metrics labels bounded, không order/event UUID. Recovery audit phải có actor, scope, reason, before/after references, evidence và next action. Readiness phân biệt DB/Kafka availability, contract capability và coverage; không mark outbox published khi chỉ log event.

## 12. Acceptance, review resolution và rollout gates

### 12.1 Fault-test oracles

| ID | Scenario | Oracle bắt buộc |
| --- | --- | --- |
| PM-T01 | PostgreSQL COMMIT thành công nhưng client mất ACK | Lookup same identity, đúng một operation/Order/intent; không compensation giả |
| PM-T02 | Hai checkout cùng quote, khác idempotency key; Redis marker lỗi | SQL quote PK chỉ một accepted operation, bên thua không Reserve |
| PM-T03 | Worker crash sau remote effect trước proof | Durable intent query ra effect, no duplicate acquisition/refund |
| PM-T04 | Kafka ACK thành công, mark published chưa commit | Replay same immutable ID/bytes, consumer no-op duplicate |
| PM-T05 | Old publisher gửi sau mất lease | Không advance SQL; owner mới replay head trước next; late duplicate không undo state |
| PM-T06 | Version12 tới trước prerequisite11 | DEFERRED+work durable; replay khi đủ, không chờ integer version không được contract hứa |
| PM-T07 | Consumer rebalance khi SQL đang commit | Offset chỉ eligible prefix/current assignment; redelivery dedup effect |
| PM-T08 | Cùng event ID, payload khác | Original giữ nguyên; conflict quarantine+alert, không business reapply |
| PM-T09 | Restore DB cũ rồi replay Kafka | Fence old deployment; reconcile/dedup reconstruction trước mutation, không second money/stock effect |
| PM-T10 | Receipt không có Order vẫn lưu và phát event | Receipt identity duy nhất; RECEIPT stream NOT NULL identity, order_id NULL; validated Payment schema gate |
| PM-T11 | Deadlock40P01 hoặc serialization40001 | Whole-tx bounded retry cùng identity, không replay remote trong closure |
| PM-T12 | Hai publisher/two events cùng version khác ordinal | Head progression atomic, deterministic sequence; không next trước confirmed head |
| PM-T13 | Offset100 chưa durable,101 đã commit | Không commit102; durable handoff100 rồi mới advance prefix |
| PM-T14 | Poison/unsupported schema hoặc deferred không có job | Quarantine/deferred responsibility atomic, scanner sửa orphan, không partition block im lặng |
| PM-T15 | Concurrent refund budget + extra/late receipt | Receipt lock bảo vệ total, money giữ đủ, no auto allocate/refund sai |
| PM-T16 | Snapshot/evidence/outbox identity bị UPDATE | DB guard reject; audited rewrap chỉ giữ logical identity |
| PM-T17 | Purge trước dedup horizon hoặc nghĩa vụ unresolved | Reject archival plan; identity/query/evidence còn phục hồi được |
| PM-T18 | Routing/partition count đổi khi stream còn active | Gate migration/drain, không dựa vào old partition ordering |
| PM-T19 | Unique conflict lookup DB tạm unavailable | Giữ UNKNOWN, retry lookup primary, không new identity |
| PM-T20 | Mất cả dedup lẫn milestones sau PITR | Recovery mode dùng external evidence/identity, không tự mở side effects |

### 12.2 Resolution và thứ tự triển khai

| Review | Resolution trong bản này |
| --- | --- |
| Publisher lease không đủ ordering | Mục6–7: stream head protocol, giới hạn late duplicates, consumer guards; không tuyên bố broker fencing |
| Redis/SQL atomicity | Mục4.1: Redis admission khác SQL acceptance, quote-consumption PK quyết định |
| Commit UNKNOWN | Mục5: SQLSTATE handling và identity lookup cho từng phase |
| Nullable Order outbox | Shared generic stream identity NOT NULL, receipt/refund stream riêng |
| Inbox atomic/offset/replay | Mục8: business/disposition atomic, contiguous eligible offsets, deferred recovery |
| Physical schema | DDL candidate có typed columns, FK/constraints/indexes/immutability; chưa là upgrade migration |
| Crash/restore | Mục10: deployment fencing, dedup reconstruction, authoritative reconciliation trước mutation |

Triển khai theo thứ tự: schema/constraints → transaction/recovery adapters → stream publisher → inbox/replay → operational/fault gates. Phê duyệt kiến trúc không tự phê duyệt activation: cần PostgreSQL DDL execution/constraint tests, Kafka multi-replica failover/rebalance tests, provider real proofs, Payment unmatched-event schema/Finance review, capacity/retention/restore acceptance. Các oracle trên chưa được chạy chỉ vì đã viết tài liệu.

### 12.3 Kiểm chứng tài liệu trong lượt review

DDL được parse bằng PostgreSQL parser:42 tables,23 explicit indexes,3 PL/pgSQL functions,12 triggers; kiểm PK và FK/index/trigger target references. Các function bodies cũng parse được. Payment JSON Schema candidate hợp lệ Draft7;3 fixtures hợp lệ,12 structural negative cases bị từ chối;3 identity-mismatch cases cần semantic guard được kiểm riêng để thể hiện giới hạn JSON Schema.236 local links trong bộ Order+API README hợp lệ, Markdown fences cân bằng. Không thay application code hoặc Order v3 schemas.

Docker daemon hiện không khả dụng trong lượt kiểm tra này; chưa execute candidate vào PostgreSQL16 hoặc chạy concurrent/Kafka/provider fault tests. Static parse không chứng minh FK creation/trigger execution, transaction behavior, ordering failover hay restore correctness. Candidate vẫn ở trạng thái review và các deployment gates ở trên còn hiệu lực.

# Data model, state machine và failure recovery

Đây là thiết kế logic ứng viên, chưa chạy SQL/migration. UUID cho identity nội bộ, `timestamptz` UTC, version bigint tăng đơn điệu, enum/check constraints rõ ràng. Payload nhạy cảm dùng AEAD ciphertext + key ID; không JSONB plaintext chung. UUID/order ID từ producer giữ đúng contract, không tự ép mọi legacy ID thành UUID.

## 1. Bảng và constraints bắt buộc

| Table | Fields/keys chính | Constraint/index / ý nghĩa |
|---|---|---|
| `inbox_events` | source, event_id, type, schema/cohort, fingerprint, broker topic/partition/offset, disposition, received_at | PK `(source,event_id)`; unique broker location theo consumer subscription; exact event bytes digest, never overwrite original; raw payload nếu cần replay phải encrypted và TTL |
| `event_conflicts` | original event ref, conflicting fingerprint, reason, first_seen, count | Unique original ref+conflicting hash; không đưa raw conflicting payload vào log; collision không ACK mất trước khi lưu evidence |
| `notification_requests` | id, producer_id, canonical_intent_key, purpose, classification, audience, recipient_key, source ref/version, context_ref, request_status, template/policy pin, created_at, expires_at, version | Unique canonical_intent_key; source/time search; request/recipient scope immutable; status mô tả orchestration, không read |
| `idempotency_records` | producer_id, command_kind, key, semantic_fingerprint, request_id, original response ref, retention_until | PK producer+command_kind+key; same key conflict 409; tombstone không purge sớm hơn replay window |
| `notification_deliveries` | id, request_id, channel, endpoint_key, delivery_status, encrypted destination/content, template_version, policy_version, provider/account, stable_send_key, provider_message_id, not_before, expires_at, attempts_started, next_run_at, lease_owner/until/epoch, version | Unique request+channel+endpoint_key; partial unique provider/account/message_id khi có; stable_send_key unique theo provider/account; due partial index; no duplicate delivery khi template đổi |
| `delivery_attempts` | id, delivery_id, attempt_no, lease_epoch, prepared_at, started_at, finished_at, result, provider_ref, safe_error_code | Unique delivery+attempt_no; append history; không lưu secret/raw error/recipient; mọi external call có prepared identity trước |
| `provider_receipts` | provider/account/event_id, fingerprint, message_id, normalized status/time, correlation_state, encrypted raw optional | Unique provider/account/event_id; unmatched index; raw TTL; stable collision evidence riêng không overwrite |
| `notification_templates` | template_key, version, channel, locale, audience, variable_schema_hash, content_hash, approval_ref | Immutable version PK; body files checksum verified; active mapping thuộc versioned policy, không sửa historical version |
| `routing_policies` | policy_id/version, source/type/cohort/purpose mapping, allowed audience/classification, channel order, template pins, expiry/retry policy | Immutable version; only approved enabled policy; no caller-selected arbitrary route |
| `contact_suppressions` | issuer-scoped contact HMAC, channel, reason, source evidence, active_since, expires_at, policy/version | HMAC domain-separated; hard bounce/complaint not same marketing opt-out; no plaintext searchable address |
| `audit_outbox` | id, subject, action, actor, safe metadata, occurred_at, bytes/hash, state, lease/epoch, attempts | Stable bytes/id; transactional with protected mutation; publisher does no network inside transaction |

N1 chỉ tạo foundation subset cần cho intake; N2 additive deliveries/attempts/callbacks, không tạo mọi future table ngay. Future preferences/endpoints/inbox_read/alerts/escalation tables thuộc N6/N7 entry design. Retention tasks có checkpoint/range, batch bound, không full-table destructive sweep.

## 2. Identity và dedup

Ba mức độc lập:

1. **Transport**: `(producer source,event ID)` cho Kafka; `(authenticated producer,command kind,idempotency key)` cho API. Kafka offset bổ sung giúp poison-pill không lặp vô hạn; source/event ID vẫn chống republish sang offset mới.
2. **Business intent**: canonical encoding `(source namespace, source event ID hoặc owner intent ID, purpose, audience, stable recipient key)`. Không bao gồm template version/policy version/channel nên re-render/route update không tạo business notification mới. Tuy nhiên đổi audience/recipient phải source-authorized; không dùng body caller làm authority.
3. **Delivery**: request+channel+endpoint key và stable provider idempotency key. Send key không thay theo attempt. Intentional resend có command ID/sequence mới và liên kết parent; retry không phải resend.

Canonical serialization length-delimited hoặc canonical JSON, SHA256 cho public event bytes, keyed HMAC khi fingerprint liên quan OTP/contact. Không lower-case local-part Email một cách mù; normalization phải theo contact owner, domain case normalization tách biệt. Key rotation không làm mất dedup: lưu key version và dual lookup/tombstone trong rotation window.

Cùng ID + khác body: quarantine collision, giữ original, alert, không “last write wins”. Kafka rebalance/concurrent replicas dùng unique constraints + transaction, không check-then-insert trong RAM. Cùng business intent từ hai nguồn event+command cần owner-supplied mapping; nếu không có, chỉ bật một route.

## 3. Intake transaction và Kafka offset

1. Validate auth/schema/producer/purpose/size ngoài DB; no network lookup trong SQL transaction.
2. Begin TX → insert inbox/idempotency identity → resolve duplicate/conflict → create request `ACCEPTED` hoặc `WAITING_CONTEXT`, frozen route/template choice, due obligation → audit metadata → commit.
3. Chỉ ACK HTTP/Kafka sau commit. DB commit outcome unknown: retry/query bằng stable key, không ACK giả.
4. Context resolver chạy sau durable intake; persist permitted snapshot và delivery trong một TX khác với request version check; retry resolver không tạo thêm delivery.
5. Poison/unsupported records: lưu quarantine disposition + masked reason + source location trước ACK; DLQ publish qua durable outbox sau. Nếu DB down thì pause/retry, không bỏ record và không commit offset N+1 khi N chưa durable. Xử lý tuần tự theo partition ở MVP; parallel chỉ sau contiguous-ack tracker tests.

Kafka consumer retry lỗi persistence tách khỏi retry gửi Email. Broker retry không được gọi provider trực tiếp. Không để email provider chậm chặn partition hàng giờ sau khi đã có durable job.

## 4. States chính xác

Request lifecycle: `ACCEPTED -> WAITING_CONTEXT -> READY -> PROCESSING -> COMPLETED | PARTIAL | FAILED | SUPPRESSED | EXPIRED`. Bypass WAITING_CONTEXT nếu đã có trusted context. COMPLETED nghĩa mọi delivery đã đạt outcome theo policy (có thể chỉ provider-accepted); API luôn expose child delivery statuses để không diễn giải là delivered/read. Request state derived/recomputed trong transaction từ deliveries, không dùng counter RAM.

Delivery scheduling: `PENDING -> LEASED -> SENDING`; `WAITING_RETRY -> LEASED`; outcomes `PROVIDER_ACCEPTED`, `DELIVERED`, `UNKNOWN`, `FAILED_PERMANENT`, `RETRY_EXHAUSTED`, `SUPPRESSED`, `EXPIRED`. Bounce/complaint là dispositions riêng và timestamps, không ép thành thứ tự số tuyến tính.

| Tình huống | State/action |
|---|---|
| Validation/policy/contact reject trước send | SUPPRESSED hoặc FAILED_PERMANENT, reason cụ thể |
| Chưa đến not_before hoặc shared quota hết | Để PENDING/WAITING_RETRY, persist next_run_at; không tăng send attempt |
| Hết expires_at trước dispatch | EXPIRED; không gửi dù còn retry budget |
| Provider trả accepted + message ID | PROVIDER_ACCEPTED; chờ callback/query, không gọi là delivered |
| Provider xác nhận rejected, retry-safe, chưa nhận message | WAITING_RETRY với bounded backoff |
| Timeout/reset/5xx không chắc đã nhận | UNKNOWN; reconciliation, không retry mù |
| Callback delivered hợp lệ | DELIVERED và delivered_at, không read |
| Callback hard bounce/complaint | Record disposition + suppression theo policy; complaint có thể đến sau delivered, giữ cả hai facts |
| Retry budget hết với known rejection | RETRY_EXHAUSTED + durable DLQ obligation |
| UNKNOWN không có khả năng xác minh | Manual review gate, không tự biến thành FAILED/RETRY_EXHAUSTED |

Callback state transitions phải theo provider semantics: late accepted sau delivered chỉ thêm evidence, không downgrade; delivered sau UNKNOWN có thể reconcile; delivered/bounce mâu thuẫn giữ facts và flag manual review, không chọn timestamp mới nhất một cách mù. Expiry ngăn gửi mới, không xóa bằng chứng late provider delivery sau lần gửi hợp lệ.

## 5. Lease và crash windows

- Claim due rows bằng `FOR UPDATE SKIP LOCKED` trong transaction ngắn; set worker UUID, lease_until theo DB clock, increment epoch. Hai workers không claim cùng live lease.
- Persist attempt PREPARED và stable key trước network. Chuyển SENDING bằng CAS owner+epoch+unexpired lease rồi thực hiện send; timeout ngắn hơn lease với margin. Renew bounded nếu cần; mất lease trước send → không gửi.
- Provider call ngoài SQL TX. Completion UPDATE phải kiểm owner/epoch, không overwrite kết quả callback đã terminal; callback và completion serialize bằng row lock/version.
- Crash trước SENDING: lease hết có thể reclaim nếu chứng minh chưa dispatch. Crash ở SENDING/khoảng chưa biết request đã rời process: chuyển UNKNOWN; **fencing DB không ngăn được provider duplicate**.
- Nếu provider idempotency có documented retention còn hiệu lực: reconciliation/resubmit cùng stable key có thể an toàn theo contract. Hết window hoặc provider không hỗ trợ/query không kết luận: giữ UNKNOWN, manual review. Không đổi provider/fallback để “thoát timeout”.
- Callback tới trước response: receipt lưu ngay; correlate sau khi stable key/message ID ghi vào DB. Never overwrite terminal outcome với late sender response. SQL commit unknown sau provider success xử lý qua stable key + callback/query, không send lại.

## 6. Retry/backpressure defaults ứng viên

Đề xuất tối đa 5 external send attempts tổng cộng, backoff sau known safe rejection: 5s, 30s, 2m, 10m + bounded jitter, Retry-After được tôn trọng trong expiry. Đây là policy version candidate để giải quyết tài liệu 3-vs-5 lần; không áp cho OTP nếu vượt deadline. Resolver có retry budget/expiry riêng, không tăng send attempts.

HTTP 429 hoặc 5xx chỉ retry nếu adapter provider xác nhận semantics an toàn; không mặc định mọi 5xx đều “chưa gửi”. Hard invalid-address/auth/permission/template lỗi không retry mãi; provider auth lỗi mở circuit/pause queue và alert, không đánh dấu toàn bộ recipients invalid.

Bound worker count, per-provider quota, per-recipient anti-spam, queue cap/oldest age. Intake vượt cap trả 429/503 trước acceptance; đã accepted thì persist deferred state, không drop. Không ngủ trong SQL lock hoặc giữ goroutine theo mỗi delayed message. Priority có fair scheduling để transactional không starvation và security không unlimited bypass quota.

## 7. Privacy, retention và replay

Encryption AAD ràng buộc request/delivery/purpose; wrong AAD/key fail closed. Destination HMAC để suppression/dedup, masked view cho admin. Plaintext chỉ trong bộ nhớ cần render/send; không claim Go bảo đảm zeroization mọi buffer. HTTP/body tracing phải tắt ở sensitive paths.

Retention **đề xuất, chưa chính sách production**: OTP secret purge ngay sau accepted/terminal hoặc expiry, callback/raw payload ≤ 7 ngày, transactional encrypted context ≤ 30 ngày, metadata/dedup tombstones ≥ 90 ngày **và không ngắn hơn** broker retention + producer replay + provider idempotency window đã chốt. Nếu các window chưa rõ: không bật purge dedup. Security purge payload tách khỏi identity tombstone; replay sau purge trả `PAYLOAD_EXPIRED`, không hydrate dữ liệu người nhận mới âm thầm.

Manual replay chỉ sau authorization + expected_version + reason + idempotent command. Revalidate purpose/expiry/contact permission/template pin/suppression và khả năng safe send. Không chạy replay hàng loạt bằng SQL UPDATE bỏ audit. Retention/legal hold cụ thể và quyền xem payload phải chốt trước production; raw payload inspection API ngoài MVP.

# ADR-005 — SQL retry và commit UNKNOWN recovery

- Ngày: 17/10/2026
- Trạng thái: Accepted — implemented/tested tại PostgreSQL boundary.

## Bối cảnh và gap

Sandbox gọi Begin/Commit trực tiếp; COMMIT ACK mất trả ACCEPTANCE_UNKNOWN nhưng không có transaction runner/recovery boundary dùng chung. Unique-conflict, serialization/deadlock rollback và ambiguous network outcome cần xử lý khác nhau để tránh tạo business identity hoặc compensation mới.

## Quyết định

- Runner dùng READ COMMITTED, max3 whole-transaction attempts trong tổng3s, lock_timeout2s/statement_timeout3s. Jitter `[0,25ms*2^retry_index)`; không retry một statement trong aborted transaction. Callback chỉ SQL/local domain/crypto, không RPC/Redis/Kafka hoặc tự Commit/Rollback.
- SQL40P01/40001 là known rollback, retry toàn closure với business IDs pin ngoài closure. SQL23505 không retry mù: rollback rồi tra identity/constraint. Definitive COMMIT class23 và ErrTxCommitRollback không gọi unknown success.
- Error khác ở COMMIT giữ ErrCommitUnknown và cause, không tự chạy closure lại. Primary lookup bounded2s dùng context không bị cancellation của request cũ nhưng giữ request context values. Lookup absent/unavailable không chứng minh rollback; giữ UNKNOWN và yêu cầu retry cùng scope/key. Không dùng replica lag làm negative proof.
- T0 acquire advisory key lock theo trusted scope/action/key để serialize identity chưa có row, rồi unique registry/quote/fingerprint constraints làm authority. Hash collision chỉ giảm concurrency. Một quote khác key bị QUOTE_ALREADY_USED; fingerprint active/consumed khác quote bị CHECKOUT_ALREADY_SUBMITTED.
- Shared CHECKOUT registry không đổi namespace để bypass legacy identity. Replay recheck current Order scope, hash, response retention; terminal hết7d trả identity reference cùng ErrReplayExpired, không cho reuse key. Pending operation không bị xóa vì preview/response TTL.
- PLACE recovery tra operation→Order và decrypt/verify canonical root/intent state. Không compensate vì DB client báo mất ACK. Epoch/fence kiểm ở SQL writes dưới row lock; thiếu row là stale lease, dependency/SQL failure không được đổi thành business reject.

## Hệ quả

Runner không giải quyết remote UNKNOWN, provider idempotency hoặc full Saga compensation; PR3 cung cấp phần đó. Advisory key lock + audit chain lock được đo contention sau, không bỏ unique constraints để tối ưu. Thời gian request có thể gồm recovery lookup sau execution budget; API timeout/status mapping là application slice, không đồng nhất HTTP deadline với business expiry.

## Kiểm chứng

Tests thật trên PostgreSQL: concurrent12 same-key calls, different keys same quote, exact one durable operation; injected SQL40001/40P01 với partial writes rollback; two-transaction real deadlock; retry exhaustion3. Commit wrapper thực hiện COMMIT thật rồi trả EOF để kiểm ambiguity boundary; không tuyên bố đã drop network packet. Test cả rollback-before-EOF và committed-but-primary-pool-unavailable, sau kết nối lại replay đúng identity. Xem [evidence](../../../04_testing/order-service/05_persistence_foundation.md).

# ADR-006 — Encrypted hydration và atomic PLACE

- Ngày: 17/10/2026
- Trạng thái: Accepted — PR2 local persistence implemented; real provider/target worker chưa active.

## Bối cảnh và gap

Core PR1 là immutable values, chưa có reconstitution API hoặc cách gắn snapshot/hash với indexed SQL columns. Legacy cipher dùng AAD order-v3.1, không có key ID. PLACE thiết kế yêu cầu Intent/history/outbox/cleanup cùng commit nhưng runtime3.1 chưa có đủ physical records. PR2 cần test boundary này trước khi PR3 có actual Inventory executor.

## Quyết định

1. Thêm StateRecord schema1 và Restore trong Core, nhận immutable Snapshot cùng current owner/state/version/decision time/resource/allocation/milestone. Validate invariant và reject unsupported lifecycle/schema, không mở setter arbitrary status. Completion/shipping/refund projection extensions cần cập nhật explicit khi feature được triển khai.
2. Cipher AES256-GCM envelope có version/key ID/nonce và purpose-bound AAD. Accepted input bind operation+original scope; canonical payload bind operation; snapshot/state bind Order; follow-up jobs bind job ID. Reader có keyring, writer chọn current key. Không cho legacy decryption fallback hoặc plaintext state.
3. LoadOrder decrypt+validate snapshot logical hash, StateRecord và indexed owner/state/version/money/resource/time/item projections. Column mismatch hoặc unknown schema fail closed; không đọc Catalog/Profile để sửa historical snapshot. Public exposure phải qua redacted application view, không trả root record nguyên bản.
4. T3 lock operation/control receiver/workflow/job, DB decision clock sau locks; canonical input match original quote/cart/selection/hash/owner/method. Final placed_at được chốt tại transaction, không lấy timestamp client. Core.Place kiểm proof association/expiry/method; real auth/query proof thuộc typed port PR3, không được tuyên bố từ local fixture.
5. T3 ghi Order/items + VIETQR PaymentIntent (COD không có) + history/PLACED milestone + immutable placed.v3 outbox/stream + operation/workflow/job result + durable cleanup/initial COD-ready jobs + audit. Lease được kiểm lại tại fenced job apply. Không network trong SQL transaction. Full cleanup Lua, readiness generation và executors thuộc slices sau; jobs giữ references tới immutable accepted input/snapshot.
6. Payment code đúng Payment09: CSPRNG OMP+12 uppercase alphanumeric, unique(connection,code), collision retry trước commit. Không derive code từ Order UUID hoặc tự dùng prefix khác. ACK mất chỉ lookup intent đã commit, không generate code mới cho Order đã placed.
7. placed event payload/schema pin dùng actual v3 shape, no PII. schema_hash là SHA256 file bytes normalize CRLF→LF để checkout Windows/Linux không tự làm lệch pin; runtime test so pin với contract file. PostgreSQL UUID aggregate và TEXT partition_key bind explicit cast, tránh ambiguous prepared parameter type mà static SQL parse không phát hiện.

## Hệ quả và giới hạn

Repository Accept/Place là internal boundary, chưa register target HTTP endpoint. Tests chuẩn bị canonical payload và leased ORDER_PLACE job qua fixture SQL; đây không phải worker reserve/canonicalization đã implement. Accepted input cần application admission/ownership trước repository; evidence structs không là public trusted DTO.

Shared audit chain giữ existing lock/format để tương thích; khi nối runtime, injected audit key phải khớp chain verifier/key của cohort còn active. Separation/rotation key ID cho audit và non-privileged grants là deployment gate riêng. Keyring reader test không chứng minh đã re-encrypt stored ciphertext; existing immutable snapshot trigger vẫn giữ nguyên.

## Kiểm chứng

PG faults tại Intent/outbox insert rollback toàn T3, missing/stale/expired lease không place; actual COMMIT rồi EOF phục hồi đúng Order. VIETQR/COD placement/load/replay, SQL immutable triggers, corrupted projections fail closed; emitted placed payloads validate theo actual v3 schema. Xem [PR2 evidence](../../../04_testing/order-service/05_persistence_foundation.md).

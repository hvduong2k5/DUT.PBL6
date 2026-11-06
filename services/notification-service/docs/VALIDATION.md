# Verification và acceptance plan

**Hiện tại: runtime tests/build/containers/provider sends NOT RUN** vì chỉ có tài liệu. Test matrix dưới đây là yêu cầu tương lai, không kết quả đã đạt. Mọi evidence mới ghi trong `docs/verification/` (metadata nhỏ, không raw secrets/log dumps), kèm commit/toolchain/config/mode và skipped cases.

## Layers / commands dự kiến

1. Domain/usecase unit tests: clock, fake ports, table-driven state transitions. Chạy `go test ./internal/domain/... ./internal/usecase/...` khi package tồn tại.
2. HTTP/contract: actual httptest requests/responses + version-pinned JSON Schema/OpenAPI validation; không chỉ kiểm fixtures tự viết khớp một implementation sai.
3. Real PostgreSQL: migrations fresh/reapply checksum, unique/locking/rollback, concurrent replicas, leases, retention/tombstones. Không thay bằng mocks cho concurrency guarantees.
4. Real Kafka: manual commits, crash before/after DB commit, rebalance, poison/collision, backlog restart; distinct isolated topics/group cho test, producer ACL failure nếu environment hỗ trợ. Test fake broker không đủ nghiệm thu offset semantics.
5. Redis optional: flush/down/restart không mất request; shared quota failure không bypass safety.
6. Fake/capture email server: deterministic accepted/rejected/unknown/late callback, content assertions, zero real recipients. Browser/real email không cần ở local foundation.
7. Release checks trong service: `go test ./...`, `go vet ./...`, `go build ./...`; race `go test -race -count=1 ./...` trên toolchain hỗ trợ, có thể dùng isolated Linux container nếu Windows thiếu compiler. Không cài nặng hoặc đổi host config ngầm.
8. N4 tạo `scripts/verify.ps1` + shell equivalent/Compose runner có explicit test env; thiếu DB/broker phải fail required integration gate, không PASS toàn task sau skipped tests. Cleanup chỉ resources do runner tạo, không `docker system prune`, không shared volume delete.

## Acceptance matrix bắt buộc

| Test ID | Trigger / expected result | Loại evidence |
|---|---|---|
| V01 | POST durable intake → 202 chỉ sau commit; DB down/commit unknown không trả accepted giả | HTTP + real DB fault |
| V02 | 50 concurrent same-key same-body → một request/delivery, original response identity; changed body →409 | Real DB, count/assert constraints |
| V03 | Same event ID/different body → original unchanged + conflict ledger; no send | Kafka/DB |
| V04 | Same event+recipient+purpose sau template deployment → không delivery mới | Unit + DB |
| V05 | Crash sau DB commit trước Kafka ACK → redelivery không duplicate; before commit → không mất | Real Kafka/restart |
| V06 | Partition record N persistence fail → N+1 không commit làm mất N; rebalance replay safe | Real Kafka offsets |
| V07 | Poison/unsupported type/version → durable quarantine trước ACK, DLQ publisher retry giữ ID | Contract + broker down |
| V08 | Hai workers claim một delivery → một live lease; stale owner update bị từ chối | Real DB race |
| V09 | Crash sau provider accepted trước DB update → UNKNOWN, same key query/reconcile; không new key/send mù | Fault fake provider + restart |
| V10 | Provider không idempotency/query + timeout → manual review, không fallback hoặc retry | Adapter/usecase |
| V11 | Known rejection/429 Retry-After, permanent failure, expired OTP, exhausted safe retries | Fake clock/provider |
| V12 | Callback duplicate, tampered signature, wrong account, oversized body, old replay timestamp, conflicting same ID | HTTP + DB |
| V13 | Callback đến trước send response; late accepted sau delivered; bounce/complaint sau delivered | Concurrent transitions, history preserved |
| V14 | Không auth→401; sai service/purpose→403; foreign request→404; stale replay→409; authorizer fail trước repo/provider | HTTP + spy ports |
| V15 | Valid JWT nhưng inactive employee/mất permission/resource scope → deny; spoofed X-User-Role không cấp quyền | Auth integration khi G08 chốt |
| V16 | Missing/invalid contact; Guest đúng Order, buyer khác gift recipient; revoked contact giữa intake/send | Resolver fixtures + owner contract tests |
| V17 | Gift recipient mail không chứa giá, buyer address/PII trong HTML/text/subject/vars lưu | Negative data assertions |
| V18 | Template missing vars/locale, HTML/script injection, email header CRLF, unapproved link | Renderer/HTTP |
| V19 | DB dump/log/trace/DLQ/audit không plaintext synthetic secret/contact; wrong key/AAD fail | Canary secret scan + real ciphertext checks |
| V20 | Identity OTP expired/resend/cancel challenge; IdP-owned flow không double send | Owner fixtures rồi real sandbox gate |
| V21 | Invoice khác order/legal entity/recipient, arbitrary URL, expired link → reject/no fetch | Artifact/resolver tests |
| V22 | Marketing no consent suppressed; opt-out khác hard-bounce; mandatory exception chỉ approved policy | Policy matrix |
| V23 | Replay command duplicate/unauthorized/UNKNOWN/payload-purged/terminal delivered bị xử lý đúng | HTTP + audit atomicity |
| V24 | Redis flush/down, broker down, restart sau backlog → DB request giữ nguyên, không mất/duplicate logical intents | Isolated Compose |
| V25 | Quota/queue cap, bounded goroutines, fairness, graceful shutdown, slow provider không chặn source order | Load/fault trace metadata |
| V26 | Retention purge ciphertext vẫn giữ tombstone đủ replay window; key rotation không tạo dedup mới | DB + clock |
| V27 | Fresh migration, repeated checksum mismatch, rollback failure, DB schema version readiness | Migration integration |
| V28 | Real provider accepted rồi callback delivered/bounce đúng correlation | **Chỉ** sandbox owner-authorized, provider-side evidence; không thay bằng fake |

N5 bổ sung fallback không gửi đồng thời khi UNKNOWN, ordering/coalescing không notification cũ sau mới; N6 endpoint stolen token/multi-device/logout/notification center read/deep link auth; N7 role revocation/resource visibility/ack≠resolved/escalation dedup. Detailed tests viết khi phase đó mở.

## Nghiệm thu theo mức

- **DESIGN READY:** pin sources, paths/links hợp lệ, contradictions/gates rõ, scope riêng, model có next step cụ thể.
- **LOCAL CORE VERIFIED:** V01–V19/V22–V27 liên quan core pass thực; OTP/invoice chỉ fixtures khi gates chưa chốt; no production delivery claim.
- **INTEGRATION VERIFIED:** event fixture từ producer đúng SHA + owner resolver/auth contract + real local broker/DB, actual API payloads verified. Không suy ra deploy/provider acceptance.
- **PROVIDER VERIFIED:** V28 và owner-authorized Email content/receipt evidence; accepted, delivered, read báo riêng. Một Email đến inbox không chứng minh toàn bộ retries/duplicates.
- **FULL EMAIL MVP:** toàn US-NOTI-01 scenarios Order/OTP/invoice/bounce/gift accepted hoặc approved reduced scope được ghi rõ, N4 DoD hoàn tất, blockers không bị đổi thành DONE.

## Report format

Mỗi checkpoint ghi: branch/HEAD, changed paths, exact commands, exit code + pass/fail/skip, environment/mode, fake-vs-real, failures pre-existing vs introduced, artifact metadata nếu cần, remaining gates, next action. Không tái dùng số test/benchmark của Content/Profile/Order làm kết quả Notification. Không commit recipient inbox screenshots, OTPs, tokens, .env, full raw provider callbacks hoặc container dumps.

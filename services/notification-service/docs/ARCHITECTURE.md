# Kiến trúc Go — thiết kế đề xuất

## Stack và lựa chọn

- Module riêng `github.com/omamx/notification-service`, Go, HTTP Chi v5, pgx v5, Zerolog, Prometheus, kafka-go; theo họ thư viện Go hiện có. Go 1.22.0 là baseline repo để tham khảo, không phải yêu cầu cài bản cũ. N0 phải ghi toolchain hiện có, policy support/security và pin version trước N1; không tự nâng sibling modules.
- PostgreSQL private `notification_db` là source of truth: inbox, requests, deliveries, attempts, policies/templates, callback receipts, jobs, audit outbox. Không foreign key hoặc query sang DB service khác.
- Redis giữ compatibility với MS-17 architecture (`notif:queue:*`), nhưng chỉ lưu ID/hint/limiter, không là bản duy nhất của request hoặc OTP. PostgreSQL scheduler có thể khôi phục dispatch sau Redis flush. Redis outage được degraded và rate-limit-sensitive dispatch pause; không dùng cache để quyết định consent.
- Một binary hỗ trợ `api`, `worker`, `all` cho local; có thể scale API/worker riêng. Không thêm gRPC vì chưa có consumer contract yêu cầu. Không thêm Elasticsearch, BullMQ, Temporal hoặc broker thứ hai.

## Luồng thành phần

```text
Owner event / trusted dispatch command
  -> auth + producer/schema/cohort adapter
  -> Intake usecase
  -> PostgreSQL: inbox + request + delivery/job (+ audit) COMMIT
  -> Kafka offset ack / HTTP 202

DB due-job scan -> optional Redis wake-up -> Worker claim (lease + epoch)
  -> authorized recipient/content resolver [network outside transaction]
  -> freeze permitted destination/template/policy snapshot
  -> consent/suppression/expiry gate -> safe renderer
  -> persist dispatch attempt/key -> Provider.Send [outside transaction]
  -> fenced result update / UNKNOWN + reconciliation

Provider callback -> verify raw-body signature -> durable callback receipt
  -> correlate delivery -> locked transition + suppression + audit
  -> outbox publisher -> approved audit/status topic

Authenticated operator -> scoped query / replay command -> same usecases
```

Không commit Kafka offset trước durable disposition. Receipt và business intent được phân biệt: malformed/unsupported records có quarantine disposition bền vững nhưng không tạo delivery. Sender không gọi ngược để rollback Order.

## Layout dự kiến, chưa tạo source

```text
services/notification-service/
  AGENTS.md, README.md, go.mod, go.sum
  cmd/notification/main.go
  internal/
    config/                         # validated startup, feature gates
    domain/                         # request/delivery states, policy, errors
    ports/                          # Store/Clock/Provider/Auth/Resolver/Crypto
    usecase/                        # intake/dispatch/callback/replay/query
    adapters/
      postgres/                     # tx, constraints, leasing, persistence
      kafka/                        # versioned decoders, manual commits, outbox
      redis/                        # optional hint queue, shared quota
      provider/fake/                # explicit synthetic evidence
      provider/email/               # chosen provider only after decision
      identity/                     # verifier / current permissions
      recipient/                    # owner contract adapters, no direct DB
      security/                     # AEAD encryption, HMAC identifiers
    transport/http/                 # Chi handlers, DTO, auth middleware
    worker/                         # bounded scheduler, recovery, retention
  migrations/                       # service-only additive SQL + checksums
  templates/email/vi-VN/            # immutable versions and variable schema
  contracts/                        # candidate OpenAPI/JSON schema/fixtures
  tests/{contract,integration,e2e}/
  scripts/                          # bounded local verification
  deploy/local/                     # isolated Compose, fake capture only
  docs/                             # all design/progress/evidence here
```

`src/.gitkeep` chỉ xóa khi tạo Go structure và ghi rõ trong diff. Không tạo generic shared library trong packages để tiện reuse; lấy ý tưởng/pattern, giữ module độc lập.

## Ports cần định nghĩa trước adapter

| Port | Hợp đồng hành vi |
|---|---|
| `Store` / `Tx` | Atomic intake + dedup; scoped read; claim/renew/fenced completion; persist callback/audit; không network trong callback transaction |
| `Clock`, `IDGenerator` | Deterministic test; DB clock cho lease, UTC persistence; event time không quyết định auth/lease |
| `RecipientResolver` | Resolve(ref, source, purpose, audience, revision) → authorized contact snapshot, minimal variables, provenance, expiry; unavailable khác unauthorized |
| `PolicyEvaluator` | Versioned route/classification/permission/consent/suppression, deny default; mandatory chỉ từ policy đã duyệt |
| `TemplateRenderer` | Pinned version + strict variable schema → subject/text/HTML; no arbitrary execution, remote includes hoặc filesystem templates do caller chọn |
| `ChannelProvider` | Send(message, stableKey) → accepted/rejected/unknown + sanitized code/provider ID; capabilities về idempotency/query/callback; không bool success đơn giản |
| `DeliveryReconciler` | Query bằng provider ID hoặc stable key nếu provider hỗ trợ; unknown không có lookup → manual review |
| `CallbackVerifier` | Provider-specific raw bytes/time/key/signature, verified event ID + normalized disposition; raw content không được log |
| `Authenticator` / `Authorizer` | Trusted service identity hoặc verified user; per-purpose producer ACL, operation/resource scope; usecase kiểm tra trước DB/provider |
| `SecretCipher` | AEAD + key ID + AAD binding request/purpose/recipient; rotation/decrypt failure explicit; fake chỉ tests |
| `AuditOutbox` | Metadata audit obligation trong transaction; stable ID khi publish; broker down không mất audit |

Provider HTTP adapter phải có timeout, bounded response, TLS verification, fixed allowlisted base URL; không nhận arbitrary URL từ command. API error mapping nằm ở transport, domain không phụ thuộc HTTP.

## Triển khai local và lifecycle

- Service logical HTTP port 8017; local bind đề xuất `127.0.0.1:18017 -> 8017`, kiểm port trước dùng. DB/broker/Redis private network, project name riêng `pbl6-notification-local`, volumes có prefix riêng, không fixed global `container_name`.
- Compose và config chỉ dưới service; không chỉnh stack dev chung. Database migration là lệnh/job explicit, advisory lock + checksum ledger, không mỗi replica tự migrate khi boot. Test migrations chỉ disposable DB/schema; không áp dụng shared/production DB từ task này.
- Startup: validate mode + allowlists + keys → connect DB/schema check → API health → workers theo gates. `APP_MODE=local`, provider=fake, event routes disabled là default. Unknown mode/provider hoặc real credential trong fake mode cần fail startup theo config validation.
- `/health/live` chỉ process; `/health/ready` DB/schema + dependencies bắt buộc cho mode. Broker down ở worker → not ready/degraded + metric; API-only vẫn có thể nhận command nếu backlog dưới quota và durable store OK. Redis optional down → degraded; không bypass hard quota khi shared limiter unavailable.
- `/metrics` internal-only. No PII labels, recipient ID, order ID, full error strings hoặc request ID trong metric labels. Log request/event/delivery ID đã được kiểm tra độ dài; không serialized payload.
- Graceful shutdown: stop fetching/claiming → finish bounded in-flight attempts hoặc để lease/UNKNOWN recovery → flush safe telemetry → close DB/broker. Rebalance/lost ownership không commit offset hoặc update stale lease.
- Secrets dùng runtime secret files/env qua injection, never examples/commits; private key/PII encryption material không print. No production external sending flag chỉ dựa vào presence của API key.

## Observability / vận hành

Metrics: intake by source/outcome, inbox duplicates/conflicts, oldest pending age, queue depth, policy suppression reason, due/leased/unknown deliveries, attempt result, callback invalid/duplicate/unmatched, retry exhaustion, outbox age, dependency health, rate-limit delay, purge outcome. Dashboard phải tách “persisted”, “attempted”, “provider accepted”, “delivered”, “read”.

Alert candidate: oldest transactional due > 5 phút, UNKNOWN > 0 kéo dài, callback verification spike, DB unavailable, audit outbox backlog, key-decrypt failures. Ngưỡng là đề xuất cần load/SLO review; không gọi các số này production SLO. Runbook N4 cần pause/resume sends, drain, replay một record, reconcile UNKNOWN, key rotation và restart sau outage; tất cả scope cụ thể, không replay toàn DLQ tự động.

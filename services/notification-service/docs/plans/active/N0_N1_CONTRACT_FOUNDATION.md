# N0/N1 — contract ratification và Go foundation

Status 2026-11-06: N0 source audit/design complete; ratification/open contracts pending; N1 NOT STARTED. File này là detailed current/next phase plan, không authorization viết code từ yêu cầu planning-only.

## Goal / Why

Biến EPIC 28 và patterns đang có thành service foundation có thể test, không đặt sai cohort, gửi sai người hoặc tạo duplicate. N0 giải quyết ranh giới; N1 xây durable intake, chưa bật provider thật. Một module build được nhưng chưa đúng owner contracts không coi là tích hợp hoàn tất.

## Dependencies và prior decisions

- Đọc AGENTS, CURRENT_STATE, MASTER_PLAN, SOURCE_AUDIT rồi DECISIONS D01–D12.
- Base dev a24492b; service scaffold chỉ `src/.gitkeep` trước task. Go modules/profile/order pattern được tham khảo, không kéo source branch khác.
- Chốt implementation authority và G01. Chọn local-only core nếu real provider/owner contracts chưa sẵn; ghi gate OPEN, không hỏi lại những thứ user đã chốt.
- G02/G03/G08 phải có approved adapter contract trước integration, không chặn test ports bằng synthetic fixtures.

## Scope

In: candidate contracts/examples, Go module/layout, config validation, health, DB pool/lifecycle, migration runner, domain IDs/errors/request states, authorization ports, intake usecase + HTTP adapter, idempotency/inbox atomicity, narrow tests và handoff.

Out: real provider, OTP generator, invoice generation, full delivery worker/callback engine (N2), production auth wiring (G08), other-service source/shared contract edits, root CI/Compose, UI, Push/SMS/alerts, merge/push.

## Architecture/contracts affected

Only new Notification candidate contracts. Existing event schemas không sửa. Internal POST/GET proposed ở CONTRACTS; store/domain/HTTP DTO độc lập. DB foundation phải để N2 thêm deliveries/attempts bằng migration additive; không cần build future device/preference/alert tables.

## Implementation steps sau khi được giao

| Step | Cụ thể cần làm | Expected files dưới service | Acceptance |
|---|---|---|---|
| N0.1 | Verify branch/HEAD/status, service-only allowlist; đối chiếu pin bằng git show, không fetch mặc định | docs/CURRENT_STATE.md | No unrelated changes; baseline drift recorded |
| N0.2 | Chốt G01 toolchain/module/dependency pins; chọn G11 full Email vs incremental acceptance, recipient model G03; ghi alternatives/result | docs/DECISIONS.md | Proposed vs approved tách rõ; no unsupported Go/tool install ngầm |
| N0.3 | Tạo candidate OpenAPI internal intake/status + safe error schema; event allowlist/schema manifest cho Order v3, one source fixture và duplicate/conflict fixtures | contracts/notification.openapi.yaml, contracts/events/manifest.json, contracts/examples/ | Match source fixture type/source/key/data; unique IDs per event; no PII/secrets; no shared docs edits |
| N0.4 | Chốt semantic fingerprint, limits, fake principal/resolver contracts; define expected tests trước adapter | contracts/README.md, tests/contract/ | 202/200/409/401/403/404/422 semantics, missing owner contracts disabled |
| N1.1 | Create module and composition root; only necessary direct deps, go.sum, local .gitignore; remove empty src marker if appropriate | go.mod/go.sum, cmd/notification/main.go | go build, no sibling imports/upgrades |
| N1.2 | Config struct, explicit modes/gates, bounded timeouts/body/backlog; validation fail-fast; no secret dumps | internal/config/ | Unit tests invalid production/fake combos, missing keys, invalid URLs/limits |
| N1.3 | Domain types/errors, canonical intent/fingerprint, status transitions; interfaces Clock/Auth/Store/Resolver/Cipher | internal/domain/, internal/ports/ | Pure tests no network; locale/audience/purpose constraints; no HTTP/pgx imports in domain |
| N1.4 | DB foundation migrations: inbox/conflict/idempotency/request/audit obligations; checksum + advisory lock; dedicated local DB | migrations/, internal/adapters/postgres/ | Fresh schema + concurrent insert/rollback/commit-retry tests; no auto shared migration |
| N1.5 | Intake transaction, same key/body result, collision ledger; authorization first; context pending job persisted | internal/usecase/intake.go | Duplicate races one request; no 202 before commit; context missing no send |
| N1.6 | Chi internal intake/status endpoints, typed errors/request IDs, body/media limits; injected trusted service auth local tests | internal/transport/http/ | HTTP/OpenAPI tests + foreign ID 404; public headers never auth |
| N1.7 | Liveness/readiness/metrics, signal shutdown, isolated local Compose design if resources approved; no external sends | cmd/, deploy/local/ | DB outage ready false; live true; no secret metrics/log fields |
| N1.8 | Run focused tests then module vet/test/build and real DB concurrency; inspect diff; update evidence/current phase | tests/integration/, docs/verification/, docs/CURRENT_STATE.md | Required tests run, skipped explicit; only service paths; no push |

Kafka consumer adapter/offset behavior can be added in N2 once durable intake exists; N1 defines inbox semantics and fixtures, does not falsely claim broker integration. N1 HTTP foundation uses context refs, no sensitive raw OTP persistence until cipher/owner contract is ratified.

## API/data/security checklist

- Trusted service identity bound to allowed source/purpose/audience; raw body cannot choose authenticated producer. 401 vs 403 semantics preserved.
- Idempotency unique scope includes authenticated producer + command kind; canonical business intent spans retry transport where mapping is approved.
- Transaction commits request + recoverable obligation; audit protected operations same TX. Validation 4xx doesn't create half-request.
- Request GET belongs to producer/resource scope; no contact/payload returned by foundation API. Role checks in usecases precede repository access.
- No real recipient data/OTP in contract examples, stdout or test snapshots. Default fake adapters cannot send network traffic.

## UI/UX states

Không có UI trong N0/N1. API consumers cần phân biệt accepted/pending, context unavailable, suppressed, expired, forbidden, conflict, rate limit/dependency outage. Không hiển thị “email đã gửi” từ 202. Future UI contract ở N3/N6, không sửa Mobile/Web tại đây.

## Risks / failure modes

1. Old catalog trộn versions → pinned manifest và disabled unknown types.
2. Optional Profile email bị coi verified/Guest ownership → use resolver port + G03.
3. Toolchain cũ/dependency install nặng → inspect local và chốt G01 trước generate module.
4. Unit mock bỏ sót races → real DB V02/V03/V27 bắt buộc trước N1 DONE.
5. Required integration tests skipped do Docker absent → ghi BLOCKED/NOT RUN, không sửa host Docker hay bỏ tests.
6. Writer scope lọt root docs/other services → git diff allowlist ở checkpoint.

## Validation / DoD

Chạy từ service khi files tồn tại: `go test ./internal/domain/... ./internal/usecase/... ./internal/transport/http/...`; sau đó `go test ./...`, `go vet ./...`, `go build ./...`. Real DB env `NOTIFICATION_TEST_DATABASE_URL` chỉ database/schema disposable của task; test harness must require explicit value. Race test trên supported environment. Không chạy Flutter hay full monorepo suites cho service foundation.

N0 DONE khi contract ratification cho current slice rõ, unresolved gates assigned/deferred, machine-readable candidate contract/fixture validation pass. N1 DONE khi foundation build/test/vet, atomic/concurrency/auth/status tests pass, readiness/shutdown verified, no real sends, handoff updated. N2 plan được mở sau khi N1 DoD được kiểm, không tiến khi DB tests còn giả định.

## Completed / blockers / exact next action

Completed this task: source audit, branch isolation, specification/architecture/data/reliability/API proposals, roadmap/test matrix và handoff prompt.

Blockers: chưa giao implementation; G01–G12 chưa ratify theo phạm vi tương ứng. Không có runtime failure vì chưa có runtime.

Next: user/model review DECISIONS, nhận implementation authority; thực hiện N0.1–N0.4 rồi N1.1. Đừng tạo branch mới nếu đang ở đúng nhánh. Mọi local milestone commit chỉ intentional service files; không push kể cả khi low quota/context. Current handoff ở docs/CURRENT_STATE.md.

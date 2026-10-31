# Identity / Auth — sai khác contract và kế hoạch chốt triển khai

> Mục đích: biến các mô tả đang mâu thuẫn thành quyết định có thể triển khai.  
> Phạm vi chỉ là Identity/Auth MS-16 và các service/contract bị ảnh hưởng; đây không phải migration hay source implementation.

## 1. Quy ước nguồn ưu tiên

| Loại | Nguồn ưu tiên để code | Không được suy diễn thành contract chạy được |
|---|---|---|
| gRPC | [`packages/proto/identity/v1/identity.proto`](../../packages/proto/identity/v1/identity.proto) | sơ đồ/service boundary cũ ghi tên RPC khác |
| Customer REST | [`customer-stable.openapi.yaml`](../../docs/03_api_specs/customer-stable.openapi.yaml) | `openapi_d2c.yaml` khi khác stable; planned draft |
| Admin REST | `admin-stable.openapi.yaml` nếu route tồn tại | `admin-draft-planned.openapi.yaml` (`x-lifecycle: draft`) |
| Event payload | schema trong `packages/events` sau khi được thêm và version hóa | bảng topic/kiến trúc chỉ mới là ý định |
| Business behavior | EPIC/NFR và ADR đã chốt | ví dụ hoặc comment tài liệu không có quyết định cuối |

## 2. Ma trận điểm cần chốt

| Ưu tiên | Sai khác / thiếu | Bằng chứng | Rủi ro | Quyết định/đầu ra cần có |
|---|---|---|---|---|
| P0 | Tên runtime | Folder là `auth-service`; kiến trúc/proto/topic dùng `identity-service` | DB, deployment, Kafka source, tracing và routing bị tách đôi | Chọn một canonical service id; đổi folder hoặc ghi mapping duy nhất trước khi phát artifact |
| P0 | RPC xác thực thường | `service_boundary.md` ghi `ValidateToken`/`CheckPermission`; proto chỉ có JWKS và specialized permission | Consumer codegen/lời gọi runtime không thể build hoặc tạo bottleneck | Giữ local JWT validation tại Gateway; sửa doc legacy. Chỉ thêm RPC mới khi ADR nêu lý do, caller và SLO |
| P0 | Token TTL | BC-16: access 1 giờ; customer stable và legacy D2C: 900 giây/15 phút | client refresh sai, exposure sau revoke quá dài/ngắn | Chọn access TTL, refresh TTL, clock skew; cập nhật OpenAPI, HLD, config và test cùng lúc |
| P0 | Password hash/session vocabulary | HLD nhắc Bcrypt + whitelist Redis; Profile design nhắc Argon2id; BC-16/service boundary nhắc blacklist/session | thiết kế secret storage và incident revoke trái nhau | Chọn thuật toán/parameter, pepper/KMS policy; quyết định allowlist/denylist/session store, token rotation, replay detection và TTL |
| P0 | Response Auth chứa dữ liệu domain khác | stable `login/register/me` trả name/avatar/tier/loyalty; Identity không owner các trường này | duplicate data, login fragile hoặc stale response | Chọn BFF composition, async read model, hoặc tách response. Viết SLO/fallback và ownership field-by-field |
| P0 | Event `identity.events.v1` chưa có schema | API Kafka catalog nêu topic/event; `packages/events` không có identity schema/topic matrix | Profile/Marketing không thể deserialize ổn định; PII leak | Thêm CloudEvents JSON schema versioned cho UserRegistered/Deactivated/PasswordChanged, examples, compatibility, outbox and consumer idempotency |
| P0 | Không có implementation | `services/auth-service` chỉ có Markdown và `src` trống | Không có endpoint/RPC/event nào để verify | Khởi tạo service, build, config, migration/test theo quyết định đã chốt; không gắn nhãn implemented trước đó |
| P1 | REST public không nhất quán | stable login dùng `identifier`; legacy dùng phone; stable recovery 3 bước; service boundary dùng forgot/reset | frontend/codegen/QA chọn nhầm route/payload | Deprecate hoặc annotate legacy spec; stable becomes one client source; define identifier normalization and account enumeration behavior |
| P1 | Refresh auth shape | stable yêu cầu bearer/cookie **và** body refresh token | refresh token có thể bị gửi/lưu không an toàn, semantics mơ hồ | Chọn cookie-only hoặc body token model, CSRF/rotation/reuse policy; bỏ scheme không dùng |
| P1 | Guest context và guest order claim | requirement có, public contract chưa có | Checkout/Order tự tạo identity logic | Định nghĩa guest token/grant, ownership proof, route owner và Phase-2 lifecycle; không auto-create user |
| P1 | Recovery → Notification | recovery stable cần email flow; chưa có Notification command/event contract | leak recovery secret hoặc gửi không idempotent | Thiết kế private command/outbox hoặc Notification API; chỉ truyền opaque message reference, TTL/rate-limit, never recovery secret in broad event |
| P1 | IAM admin chỉ draft | `/api/v1/admin/access/**` đang `draft` | FE/admin implement trước khi policy audit/scope chốt | Chốt stable contract + authz model before implementation; document version/optimistic concurrency and error semantics |
| P1 | Revocation propagation | Role/account changes phải xử lý active session, nhưng TTL/cache policy chưa chốt | nhân viên bị revoke vẫn thao tác được | Đặt maximum propagation time, token/session version/introspection rule, Gateway cache invalidation and high-risk recheck policy |
| P2 | Documentation availability statement | service boundary nói all services sync verify; HLD nói JWKS local | diễn giải outage sai | Cập nhật: issue/login/JWKS refresh bị ảnh hưởng; request có JWT hợp lệ có thể tiếp tục tới khi JWKS cache/expiry policy yêu cầu refresh |

## 3. Contract sau khi chốt: ranh giới khuyến nghị

```text
Browser / Admin UI
        │ REST (stable OpenAPI)
        ▼
API Gateway ── cache JWKS ── gRPC GetJwksPublicKey ──► Identity
    │ validates JWT and sanitizes trusted identity headers
    ├──► Identity: register, login, refresh, logout, recovery
    ├──► Profile + Promotion: only if /auth response composition is explicitly chosen
    └──► downstream service: normal operation with trusted context

Finance / Order high-risk ── gRPC CheckSpecializedPermission ──► Identity

Identity DB ── transactional outbox ──► identity.events.v1 ──► Profile / Marketing
Identity ── audit event ──► audit.events.v1 ──► Audit Service
```

Điều này không cho phép downstream service tin header `X-User-*` do internet tự gửi; Gateway phải loại/ghi đè headers đó. Dịch vụ nhận cần chỉ tin traffic nội bộ đã xác thực mTLS/network policy.

## 4. Kế hoạch triển khai theo thứ tự an toàn

### Giai đoạn 0 — Chốt ADR, không code trước

1. Chọn tên canonical `identity-service` hoặc `auth-service`; quyết định database/schema/service discovery/topic source/telemetry name theo một tên.
2. Chốt token profile: issuer, audience, signing algorithm, `kid`, access/refresh TTL, claims, skew, refresh rotation, reuse/replay handling, storage and revoke model.
3. Chọn password hash algorithm/parameters và policy password; không kế thừa cả Bcrypt lẫn Argon2id mà không migration plan.
4. Chốt response ownership/compose pattern cho `login`, `register`, `me`; xác định behavior khi Profile event chưa được xử lý.
5. Chốt guest and admin IAM routes/lifecycle; bỏ/đánh dấu deprecated contract legacy.

**Exit criteria:** ADR được duyệt, OpenAPI/proto/event docs phản ánh cùng một truth và có owners cho Gateway, Identity, Profile, Promotion, Order, Notification, Audit.

### Giai đoạn 1 — Core service và dữ liệu

1. Khởi tạo module build/test/config; secrets đọc từ secret manager/environment, không commit private key.
2. Tạo Flyway migration riêng cho account, social identity, role/permission/assignment, refresh/recovery/session/revocation/audit-outbox theo ADR.
3. Áp database constraints: normalized unique email/phone/identifier, state transition, unique social provider subject, active assignment invariant, timestamps/version optimistic concurrency.
4. Tạo Redis key naming/TTL cho rate-limit/failed login/session/revocation và procedure khi Redis unavailable. PostgreSQL vẫn là source of truth.

**Exit criteria:** migration chạy trên PostgreSQL test từ database trống; rollback/backup policy được nêu; secret scan và test persistence pass.

### Giai đoạn 2 — Customer auth public API

1. Implement stable `register`, `login`, `logout`, `refresh`, recovery request/verify/reset với validation/error mapping đúng OpenAPI.
2. Bảo đảm idempotency cho endpoint có `Idempotency-Key`: cùng key + cùng semantic request trả replay safe; khác payload trả conflict theo policy.
3. Thêm neutral recovery response, brute-force/rate-limit, 5 failed attempts lock, no secret logging, reset proof one-time/expiry, revoke all sessions on reset.
4. Nếu auth response compose, implement chosen pattern with timeout/error policy; nếu không, sửa OpenAPI trước khi FE phụ thuộc.

**Exit criteria:** OpenAPI contract tests, negative security tests, integration PostgreSQL+Redis và frontend mock examples đều pass.

### Giai đoạn 3 — JWKS, Gateway và high-risk authorization

1. Implement đúng hai RPC hiện có: `GetJwksPublicKey`, `CheckSpecializedPermission`; generate code từ package proto, không fork message locally.
2. Gateway cache JWKS, validate issuer/audience/algorithm/expiry/key id locally, rotate keys with overlap; test stale cache, unknown `kid`, restart and Identity unavailable.
3. Định nghĩa `action` và `context_attributes` cho every high-risk call; Finance/Order gửi `AuditContext`, enforce timeout/fail-closed policy theo hành động.

**Exit criteria:** không service thường nào gọi `ValidateToken`; gateway/header spoofing, key rotation và decision-deny scenarios có automated test.

### Giai đoạn 4 — Event, Profile bootstrap và audit

1. Thêm `packages/events/identity/` schema versioned và đưa `identity.events.v1` vào package catalog; review PII field-by-field.
2. Lưu business update + outbox trong một transaction; publisher idempotent/retry-safe. Publish only after commit.
3. Profile consumer tạo profile idempotent theo `user_id`; test duplicate/out-of-order/replay/DLQ. Quyết định register success khi Profile lag.
4. Emit audit evidence cho register, login success/failure, logout, recovery, reset, account status, role/permission/assignment changes. Audit async, not login critical path.

**Exit criteria:** CloudEvents validation/backward compatibility, outbox retry, consumer dedupe, PII review và audit correlation tests pass.

### Giai đoạn 5 — IAM admin và Phase 2

1. Nâng `/api/v1/admin/access/**` từ draft lên stable chỉ sau scope taxonomy, approval, last-admin protection, state machine, concurrency/error mappings và audit design được chốt.
2. Implement social login/linking and guest-order claim only khi provider and ownership proof contracts được duyệt.

## 5. Bộ test acceptance tối thiểu

| Nhóm | Case bắt buộc |
|---|---|
| Register/login | duplicate normalized email/phone, idempotent replay, wrong credential, fifth failure locks, locked/disabled account, successful login resets counter theo policy |
| Token/session | valid/expired/wrong issuer/wrong audience/unknown kid signature; refresh rotation; refresh replay; logout; password reset; disable/role revoke propagation |
| Recovery | unknown email vẫn nhận neutral 202, rate limit, expired/used recovery token returns 410, reset proof cannot become access token, all sessions invalidated |
| Gateway/JWKS | startup cache, key rotation overlap, stale cache, Identity unavailable, forged external `X-User-*` headers dropped |
| Specialized permission | allowed/denied, malformed context, threshold boundary, missing audit context policy, Identity timeout fails closed for high risk |
| Profile/event | duplicate/out-of-order registration delivery, Profile unavailable, event schema compatibility, no password/refresh/recovery secret/PII leakage |
| IAM admin | last admin protection, inactive/expired role assignment, scope/action difference, concurrent version conflict, audit evidence and session impact |
| Resilience | PostgreSQL/Redis/Kafka outage, retry/DLQ/restart, time skew and key-cache expiry |

PII ở hàng event bao gồm phone/email đầy đủ, address, password/token/recovery proof; các secret này không được xuất hiện trong event.

## 6. Tài liệu cần đồng bộ khi hoàn thành quyết định

- `docs/02_architecture/service_boundary.md`: loại hoặc version hóa rõ `ValidateToken`/`CheckPermission`, đồng bộ direct-call model với HLD.
- `docs/02_architecture/bounded_context.md` và `high_level_design.md`: đồng bộ access TTL, hash algorithm, session/revoke terms.
- `docs/03_api_specs/customer-stable.openapi.yaml`, `openapi_d2c.yaml`, planned drafts: source-of-truth label, identifier/recovery/refresh/cookie behavior, composition response.
- `docs/03_api_specs/01_grpc_contracts.md` và `packages/proto/identity/v1/identity.proto`: giữ exact same RPC/message semantics.
- `docs/03_api_specs/02_kafka_event_catalog.md` và `packages/events/README.md`: cùng topic, partition key, publisher/consumer và event schema path.
- Profile/Order/Finance/Gateway/Notification/Audit design docs: update integration behavior and ownership after ADR.

## 7. Liên kết báo cáo tổng quan

Xem [phan-tich-identity.md](phan-tich-identity.md) để có phạm vi sở hữu, API/RPC hiện có, event và các service bị ảnh hưởng.

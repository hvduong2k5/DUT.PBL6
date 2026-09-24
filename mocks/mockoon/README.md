# O Ma Web MVP — Mockoon

## Customer Profile & Address Book

- Environment: `oma-customer-profile-mvp.json`
- Port: `4011`
- Base URL: `http://127.0.0.1:4011/api/v1`
- UI: D2C-012 `/account/profile`, D2C-013 `/account/addresses`
- Data/API source: `../../docs/ui_web/CUSTOMER_PROFILE_DATA_API_MATRIX.md`

```powershell
npx @mockoon/cli validate --data .\mocks\mockoon\oma-customer-profile-mvp.json
npx @mockoon/cli start --data .\mocks\mockoon\oma-customer-profile-mvp.json
```

| Capability | Method | Route |
| --- | --- | --- |
| `PROFILE-C01` | GET | `/customers/me/profile` |
| `PROFILE-C02` | PATCH | `/customers/me/profile` |
| `ADDRESS-C01` | GET | `/customers/me/addresses` |
| `ADDRESS-C02` | POST | `/customers/me/addresses` |
| `ADDRESS-C03` | PATCH | `/customers/me/addresses/:addressId` |
| `ADDRESS-C04` | DELETE | `/customers/me/addresses/:addressId` |
| `ADDRESS-C05` | PUT | `/customers/me/addresses/:addressId/default` |

Chọn state bằng `X-Mock-Scenario`; danh sách scenario nằm trong `CUSTOMER_PROFILE_DATA_API_MATRIX.md`. Environment là mock stateless: UI giữ state sau mutation để mô phỏng tương tác trong một phiên phát triển.

## Authentication MVP

Mockoon environment phục vụ phát triển UI cho D2C-009 Login, D2C-010 Registration và D2C-011 Recovery.

## Files

- `oma-auth-mvp.json`: Mockoon environment, port mặc định `4010`, API prefix `/api/v1`.
- `../../docs/ui_web/DATA_API_MATRIX.md`: nguồn mô tả data, capability và error semantics.
- `../../docs/ui_web/SCOPE_TRACEABILITY.md`: phạm vi và quyết định Product đã duyệt.

## Start

### Mockoon Desktop

1. Mở Mockoon.
2. Chọn **Open environment**.
3. Chọn `mocks/mockoon/oma-auth-mvp.json`.
4. Start environment **O Ma - Authentication MVP**.
5. Kiểm tra `GET http://127.0.0.1:4010/api/v1/health`.

### Mockoon CLI

```powershell
npx @mockoon/cli validate --data .\mocks\mockoon\oma-auth-mvp.json
npx @mockoon/cli start --data .\mocks\mockoon\oma-auth-mvp.json
```

Base URL:

```text
http://127.0.0.1:4010/api/v1
```

## Route inventory

| Capability | Method | Route | Default response |
| --- | --- | --- | --- |
| Health | GET | `/health` | `200` |
| `AUTH-C01` | POST | `/auth/login` | Login success + mock session cookie |
| `AUTH-C02` | GET | `/auth/me` | Authenticated customer summary |
| `AUTH-C03` | POST | `/auth/logout` | `204`, clear session |
| `AUTH-C04` | POST | `/auth/register` | `201 PENDING_VERIFICATION` |
| `AUTH-C05` | POST | `/auth/registration-verifications/confirm` | `200 ACTIVE` + mock session cookie |
| `AUTH-C06` | POST | `/auth/registration-verifications/:verificationId/resend` | `202` accepted |
| `AUTH-C07/C10` | POST | `/auth/recovery-requests` | `202` neutral accepted |
| `AUTH-C08` | POST | `/auth/recovery-requests/verify` | `200` reset proof |
| `AUTH-C09` | POST | `/auth/recovery-requests/reset` | `204`, revoke/clear session |

## Scenario selection

Không truyền `X-Mock-Scenario` thì route trả happy path mặc định. Để ép một state, truyền header:

```text
X-Mock-Scenario: <scenario-name>
```

### Login

| Scenario | HTTP | Mục đích UI |
| --- | --- | --- |
| `login-invalid` | 401 | Sai email/password, generic error |
| `login-unavailable` | 403 | Account không khả dụng |
| `login-validation` | 422 | Field validation |
| `login-rate-limited` | 429 | Retry/rate-limit state |
| `login-server-error` | 500 | Generic server error |
| `login-slow` | 200 sau 3 giây | Loading và duplicate-submit protection |
| `login-remembered` | 200 | Cookie mock 30 ngày |

### Current session / Logout

| Route | Scenario | HTTP |
| --- | --- | --- |
| `/auth/me` | `session-expired` | 401 |
| `/auth/me` | `me-slow` | 200 sau 3 giây |
| `/auth/logout` | `logout-server-error` | 500 |
| `/auth/logout` | `logout-slow` | 204 sau 3 giây |

### Registration

| Scenario | HTTP | Mục đích UI |
| --- | --- | --- |
| `register-unavailable` | 409 | Guidance login/recovery |
| `register-password-validation` | 422 | Toàn bộ password checklist |
| `register-validation` | 422 | displayName/email/terms errors |
| `register-rate-limited` | 429 | Submit throttling |
| `register-delivery-failed` | 503 | Email provider failure |
| `register-server-error` | 500 | Generic server error |
| `register-slow` | 201 sau 3 giây | Loading/duplicate submit |

### Registration magic link

| Route | Scenario | HTTP |
| --- | --- | --- |
| Confirm | `registration-link-invalid` | 422 |
| Confirm | `registration-link-expired` | 410 |
| Confirm | `registration-link-used` | 409 |
| Confirm | `registration-link-server-error` | 500 |
| Confirm | `registration-link-slow` | 200 sau 3 giây |
| Resend | `registration-resend-rate-limited` | 429 |
| Resend | `registration-resend-delivery-failed` | 503 |
| Resend | `registration-resend-invalid` | 404 |
| Resend | `registration-resend-slow` | 202 sau 3 giây |

### Recovery

| Route | Scenario | HTTP |
| --- | --- | --- |
| Request | `recovery-request-unknown` | 202, giống response email tồn tại |
| Request | `recovery-rate-limited` | 429 |
| Request | `recovery-delivery-failed` | 503 |
| Request | `recovery-server-error` | 500 |
| Request | `recovery-slow` | 202 sau 3 giây |
| Verify link | `recovery-link-invalid` | 422 |
| Verify link | `recovery-link-expired` | 410 |
| Verify link | `recovery-link-used` | 409 |
| Verify link | `recovery-verify-server-error` | 500 |
| Verify link | `recovery-verify-slow` | 200 sau 3 giây |
| Reset | `recovery-reset-validation` | 422 password checklist |
| Reset | `recovery-reset-proof-invalid` | 422 |
| Reset | `recovery-reset-proof-expired` | 410 |
| Reset | `recovery-reset-proof-used` | 409 |
| Reset | `recovery-reset-server-error` | 500 |
| Reset | `recovery-reset-slow` | 204 sau 3 giây |

## Quick checks

Health:

```powershell
curl.exe http://127.0.0.1:4010/api/v1/health
```

Default login success:

```powershell
curl.exe -i -X POST http://127.0.0.1:4010/api/v1/auth/login `
  -H "Content-Type: application/json" `
  -d '{"email":"customer.active@example.com","password":"OmaHeritage#2026","rememberMe":false}'
```

Force invalid login:

```powershell
curl.exe -i -X POST http://127.0.0.1:4010/api/v1/auth/login `
  -H "Content-Type: application/json" `
  -H "X-Mock-Scenario: login-invalid" `
  -d '{"email":"customer.active@example.com","password":"wrong"}'
```

Force password validation errors:

```powershell
curl.exe -i -X POST http://127.0.0.1:4010/api/v1/auth/register `
  -H "Content-Type: application/json" `
  -H "X-Mock-Scenario: register-password-validation" `
  -d '{"displayName":"Khach Test","email":"new.customer@example.com","password":"weak","termsVersion":"2026-09"}'
```

## Frontend integration

Frontend chỉ cấu hình base URL tại shared API client, ví dụ:

```text
AUTH_API_BASE_URL=http://127.0.0.1:4010/api/v1
```

Khuyến nghị dùng Next.js rewrite/BFF cùng origin trong quá trình phát triển. Mockoon đang bật automatic CORS để test các request không phụ thuộc cookie, nhưng cơ chế credentialed cookie cross-origin không nên được coi là mô phỏng production hoàn chỉnh.

`X-Mock-Scenario` chỉ được sử dụng trong local/test configuration. Không gửi header này tới API Gateway production.

## Quan hệ với Backend

Backend có thể dùng route, payload, status và error code trong environment này để triển khai song song, nhưng Mockoon **không phải contract cuối cùng** và không mô tả database, hashing, token generation, Redis, CSRF hoặc authorization internals.

Quy trình đồng bộ đúng:

1. Frontend chứng minh UI flow bằng Mockoon.
2. Backend và Architecture review `DATA_API_MATRIX.md` cùng behavior của mock.
3. Các bên chốt OpenAPI contract.
4. Backend implement theo OpenAPI và bổ sung contract tests.
5. Mockoon được cập nhật theo OpenAPI/contract đã duyệt.
6. Frontend đổi base URL sang API Gateway và chạy lại contract/E2E tests.

Nếu Mockoon và OpenAPI khác nhau, OpenAPI đã được review là nguồn sự thật ưu tiên.

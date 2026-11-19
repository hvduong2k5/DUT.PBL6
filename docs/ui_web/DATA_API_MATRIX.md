# UI Web Data & API Matrix

> Trạng thái: Baseline v0.3 — OpenAPI candidate proposed; chờ Backend/Architecture/Security review
> Ngày lập: 2026-09-24
> Vertical slice: Authentication MVP
> Liên quan: `SCOPE_TRACEABILITY.md`, `../03_api_specs/authentication-mvp.openapi.yaml`

## 1. Cách sử dụng tài liệu

Tài liệu này mô tả dữ liệu mà UI thực sự cần và các capability API đã được mô phỏng để phát triển D2C-009/010/011. Từ implementation thực tế, candidate đã được biểu diễn bằng OpenAPI tại `docs/03_api_specs/authentication-mvp.openapi.yaml`; nó chỉ trở thành contract được duyệt sau Backend, Architecture và Security review.

Quy ước trạng thái field:

- **Required by requirement**: có thể truy ra requirement/acceptance criteria.
- **Derived for flow**: cần để nối UI flow nhưng không phải business rule mới.
- **DECIDED**: Product đã chốt cho Authentication MVP.
- **OPEN**: Backend/Architecture vẫn cần chốt chi tiết kỹ thuật trước API Contract.
- **Server-only**: UI không được tự tính hoặc tin cậy giá trị phía client.

## 2. Nguyên tắc tích hợp frontend

```text
Page / Component
      ↓
Feature hook / form action
      ↓
Auth service
      ↓
Shared API client
      ↓
Mockoon hoặc API Gateway
```

- Component không biết base URL của Mockoon.
- Mockoon và API Gateway phải đi qua cùng `AuthService` interface.
- Secret, access token và refresh token không được log hoặc đưa vào fixture.
- Browser dùng opaque `__Host-` secure session cookie qua BFF/API Gateway; không lưu access/refresh token trong `localStorage` hoặc `sessionStorage`.
- Response login/recovery phải tránh tiết lộ account tồn tại khi policy không cho phép.
- Client validation phục vụ UX; Backend vẫn là nơi thực thi validation và security policy.

## 3. UI data inventory

### 3.1. D2C-009 — Login

| UI data | Trạng thái | Nguồn/sử dụng | Ghi chú |
| --- | --- | --- | --- |
| `email` | Required; **DECIDED** | User input | Identifier duy nhất của Auth MVP; normalize/validate nhất quán với Backend. |
| `password` | Required; **DECIDED** | User input | Không persist sau submit; không log. |
| `rememberMe` | **DECIDED** | Checkbox trong design | Boolean, mặc định `false`. |
| `returnUrl` | Derived for flow | Router/query/session context | Chỉ cho internal allowlisted path. |
| `checkoutContext`/cart identity | Derived for flow | Cart store/session | Không đưa toàn bộ cart vào login payload. |
| `submitState` | Derived for UI | Client state | `idle/submitting/succeeded/failed`. |
| `fieldErrors` | Derived for UI | Client/API validation | Map theo stable field key. |
| `authErrorCode` | Derived for UI | API error | Copy hiển thị phải an toàn, không enumeration. |
| `currentCustomer` | Required by flow | Session/current-user API | Chỉ trả dữ liệu header/routing tối thiểu. Profile đầy đủ thuộc EPIC 02. |

### 3.2. D2C-010 — Registration

| UI data | Trạng thái | Nguồn/sử dụng | Ghi chú |
| --- | --- | --- | --- |
| `email` | Required; **DECIDED** | User input | Unique identifier của Auth MVP. |
| `password` | Required; **DECIDED** | User input | Áp dụng policy tại Mục 4.1 ở cả client và server. |
| `passwordConfirmation` | Derived for UI | Client-only | Không gửi API. |
| `displayName` | Required; **DECIDED** | User input | Dữ liệu tối thiểu để hiển thị account summary. |
| `dateOfBirth` | Out of scope | Design có trường ngày sinh | Không gửi Auth API. |
| `preferences` | Out of scope hiện tại | Design có sở thích | Thuộc personalization/profile; không đưa vào Auth contract mặc định. |
| `termsVersion` | Required; **DECIDED** | UI config/request | Server ghi version và thời điểm chấp nhận; không tin timestamp từ client. |
| `verificationId` | Required for pending flow | API response | Opaque, không chứa PII có thể đọc. |
| `verificationToken` | Required from email link | Deep link | Single-use; không persist sau exchange. |
| `registrationStatus` | Required for flow | API response | Registration tạo `PENDING_VERIFICATION`; confirm thành công tạo account `ACTIVE` và session. |

### 3.3. D2C-011 — Recovery

| UI data | Trạng thái | Nguồn/sử dụng | Ghi chú |
| --- | --- | --- | --- |
| `email` | Required; **DECIDED** | User input | Request luôn trả thông điệp trung tính dù email không tồn tại. |
| `recoveryToken` | Required from email link | Deep link | Token ngẫu nhiên, single-use, hết hạn sau 15 phút; backend chỉ lưu hash. |
| `resetProof` | Derived for flow | API response | Proof ngắn hạn sau khi exchange recovery token; chỉ dùng để reset. |
| `newPassword` | Required; **DECIDED** | User input | Áp dụng cùng password policy; không log/persist. |
| `newPasswordConfirmation` | Derived for UI | Client-only | Không gửi API. |
| `resendAvailableAt` | Server-only | API response | Cooldown baseline 60 giây; server là nguồn thời gian tin cậy. |
| `remainingAttempts` | Không trả cho UI | Server state | Tránh lộ rule/risk signal; UI chỉ nhận `RATE_LIMITED` và `retryAfter`. |

### 3.4. Approved password validation policy

Password đăng ký và password mới khi recovery phải thỏa tất cả điều kiện:

| Rule code | Quy tắc đã chốt |
| --- | --- |
| `PASSWORD_TOO_SHORT` | Ít nhất 15 Unicode code point. |
| `PASSWORD_TOO_LONG` | Không quá 128 Unicode code point. |
| `PASSWORD_MISSING_UPPERCASE` | Có ít nhất một ký tự thuộc Unicode uppercase letter, tương đương nhóm `\p{Lu}`. |
| `PASSWORD_MISSING_LOWERCASE` | Có ít nhất một ký tự thuộc Unicode lowercase letter, tương đương nhóm `\p{Ll}`. |
| `PASSWORD_MISSING_DIGIT` | Có ít nhất một chữ số thập phân, tương đương nhóm `\p{Nd}`. |
| `PASSWORD_MISSING_SPECIAL` | Có ít nhất một ký tự punctuation/symbol, tương đương `\p{P}` hoặc `\p{S}`. Khoảng trắng không được tính là ký tự đặc biệt. |
| `PASSWORD_COMPROMISED` | Backend từ chối password nằm trong blocklist phổ biến/rò rỉ. |

Quy tắc triển khai:

- Client hiển thị checklist realtime nhưng Backend phải validate lại toàn bộ.
- Không trim hoặc âm thầm thay đổi password người dùng nhập; khoảng trắng được bảo toàn.
- Cho phép paste, browser autofill và password manager.
- `passwordConfirmation`/`newPasswordConfirmation` chỉ được so sánh tại client và không gửi API.
- Không log password, confirmation hoặc giá trị field vào analytics/error telemetry.
- Yêu cầu composition là quyết định Product của dự án; không được ghi chú là yêu cầu từ NIST.

## 4. API capability matrix

| Capability ID | UI action | Candidate endpoint | Candidate request | UI cần từ response | Auth | Release |
| --- | --- | --- | --- | --- | --- | --- |
| `AUTH-C01` | Submit Login | `POST /api/v1/auth/login` | `email`, `password`, `rememberMe` | Session cookie; customer summary; error code ổn định | Public | MVP |
| `AUTH-C02` | Bootstrap/check session | `GET /api/v1/auth/me` | Không có body | Customer summary, session status, permissions tối thiểu nếu cần | Session | MVP |
| `AUTH-C03` | Logout | `POST /api/v1/auth/logout` | Không có body hoặc CSRF metadata theo kiến trúc | Success idempotent; session bị vô hiệu | Session | MVP |
| `AUTH-C04` | Submit Registration | `POST /api/v1/auth/register` | `displayName`, `email`, `password`, `termsVersion` | `PENDING_VERIFICATION`; opaque verification reference | Public | MVP |
| `AUTH-C05` | Confirm Registration Magic Link | `POST /api/v1/auth/registration-verifications/confirm` | `verificationToken` | Account `ACTIVE`, session cookie, customer summary | Public + opaque proof | MVP |
| `AUTH-C06` | Resend Registration Verification | `POST /api/v1/auth/registration-verifications/{verificationId}/resend` | Không có body | Neutral acknowledgement, next resend time nếu được phép | Public + opaque proof | Conditional MVP |
| `AUTH-C07` | Request Recovery | `POST /api/v1/auth/recovery-requests` | `email` | Neutral acknowledgement; không xác nhận account tồn tại | Public | MVP |
| `AUTH-C08` | Exchange Recovery Magic Link | `POST /api/v1/auth/recovery-requests/verify` | `recoveryToken` | Short-lived `resetProof` | Public + opaque proof | MVP |
| `AUTH-C09` | Reset Password | `POST /api/v1/auth/recovery-requests/reset` | `resetProof`, `newPassword` | Completion; revoke toàn bộ session; không auto-login | Public + reset proof | MVP |
| `AUTH-C10` | Send Recovery Again | dùng lại `POST /api/v1/auth/recovery-requests` | `email` | Cùng neutral acknowledgement; rate-limit metadata khi bị chặn | Public | MVP |
| `AUTH-C11` | Continue Guest Checkout | Không phải Auth endpoint | Không có | Router chuyển về checkout; cart context còn nguyên | Guest session | MVP |

### Ghi chú về endpoint candidate

- Backend có thể đề xuất resource naming khác; UI chỉ phụ thuộc vào `AuthService`, không phụ thuộc trực tiếp path candidate.
- `GET /auth/me` chỉ trả summary phục vụ xác định session/header/routing. Hồ sơ chi tiết phải dùng API EPIC 02.
- Exchange token và reset được tách để URL token không được dùng trực tiếp như credential dài hạn; `resetProof` có thời hạn ngắn và capability giới hạn.
- Logout nên idempotent để UI có thể dọn local state ngay cả khi session phía server đã hết hạn.

## 5. Candidate data shapes phục vụ Mockoon

Các shape dưới đây đủ để dựng mock theo quyết định Product; tên endpoint, HTTP status và security header vẫn chờ Backend/Architecture review.

### 5.1. Customer summary

```json
{
  "id": "CUS-001",
  "displayName": "Khách Tri Kỷ",
  "status": "ACTIVE"
}
```

`email`, `phone`, loyalty tier và profile detail không trả mặc định nếu UI header không dùng.

### 5.2. Login success

```json
{
  "authenticated": true,
  "customer": {
    "id": "CUS-001",
    "displayName": "Khách Tri Kỷ",
    "status": "ACTIVE"
  }
}
```

Session không nằm trong response body. Mockoon cần mô phỏng `Set-Cookie` với giá trị giả rõ ràng; API client không được đọc cookie bằng JavaScript.

### 5.3. Registration result

```json
{
  "status": "PENDING_VERIFICATION",
  "verificationId": "REGV-001",
  "delivery": {
    "channel": "EMAIL",
    "maskedDestination": "u***@example.com"
  },
  "verificationExpiresAt": "2026-09-25T10:00:00Z",
  "resendAvailableAt": "2026-09-24T10:01:00Z"
}
```

Confirm magic link thành công:

```json
{
  "status": "ACTIVE",
  "authenticated": true,
  "customer": {
    "id": "CUS-002",
    "displayName": "Khách Tri Kỷ",
    "status": "ACTIVE"
  }
}
```

### 5.4. Neutral recovery acknowledgement

```json
{
  "accepted": true,
  "messageCode": "RECOVERY_REQUEST_ACCEPTED"
}
```

Response, HTTP status và thời gian phản hồi không được khác biệt có thể quan sát theo việc email có tồn tại hay không.

### 5.5. Error envelope candidate

```json
{
  "code": "VALIDATION_ERROR",
  "message": "Request is invalid",
  "errors": [
    {
      "field": "email",
      "code": "INVALID_FORMAT",
      "message": "Thông tin chưa đúng định dạng"
    }
  ],
  "requestId": "REQ-MOCK-001"
}
```

Quy tắc UI:

- UI xử lý theo `code`/`errors[].code`, không parse `message`.
- Message từ mock là example; production copy có thể được map ở frontend.
- `requestId` dùng cho hỗ trợ/debug, không chứa PII.

## 6. Stable error codes UI cần phân biệt

| Code candidate | HTTP candidate | UI behavior | Có hiển thị chi tiết? |
| --- | --- | --- | --- |
| `VALIDATION_ERROR` | 400/422 — cần thống nhất | Map field error và focus field đầu tiên | Có, ở mức field an toàn |
| `AUTHENTICATION_FAILED` | 401 | Generic login error, cho phép recovery | Không tiết lộ identifier tồn tại |
| `SESSION_EXPIRED` | 401 | Clear authenticated state, lưu return path hợp lệ, về Login | Không |
| `ACCOUNT_UNAVAILABLE` | 403 | Hiển thị guidance chung/support path | Không tiết lộ trạng thái nội bộ |
| `REGISTRATION_NOT_AVAILABLE` | 409 | Registration guidance chung: login hoặc recovery | Không khẳng định email đã tồn tại |
| `VERIFICATION_INVALID` | 400/422 | Giữ user ở verification step | Có, nhưng không lộ dữ liệu account |
| `VERIFICATION_EXPIRED` | 410 hoặc 422 — cần thống nhất | Cho resend/restart theo policy | Có |
| `VERIFICATION_ALREADY_USED` | 409 | Điều hướng theo trạng thái an toàn | Hạn chế |
| `RECOVERY_REQUEST_INVALID` | 400/404 — cần thống nhất | Yêu cầu khởi tạo recovery mới | Generic |
| `RATE_LIMITED` | 429 | Disable submit/resend đến `retryAfter` nếu được trả | Có thời gian chờ, không có rule nội bộ |
| `NOTIFICATION_DELIVERY_FAILED` | 502/503 — cần thống nhất | Retry/reselect channel nếu policy cho phép | Generic |
| `SERVICE_UNAVAILABLE` | 503 | Error state và retry | Generic |

HTTP status còn ghi “cần thống nhất” phải được chốt trong API Contract review.

## 7. Mockoon scenario matrix

### 7.1. `AUTH-C01` Login

| Scenario | Expected response | UI assertion |
| --- | --- | --- |
| `login-success` | 200 + authenticated customer summary | Redirect đúng `returnUrl`; cart giữ nguyên |
| `login-invalid` | 401 `AUTHENTICATION_FAILED` | Generic error; không tạo session |
| `login-restricted` | 403 `ACCOUNT_UNAVAILABLE` | Guidance an toàn; không redirect protected |
| `login-validation` | 400/422 `VALIDATION_ERROR` | Error map đúng field |
| `login-rate-limited` | 429 `RATE_LIMITED` | Submit bị giới hạn theo metadata |
| `login-server-error` | 500 | Generic retry state |
| `login-slow` | 200 sau delay khoảng 3 giây | Loading; chống duplicate submit |

### 7.2. `AUTH-C04..C06` Registration

| Scenario | Expected response | UI assertion |
| --- | --- | --- |
| `register-pending-verification` | 201 `PENDING_VERIFICATION` | Hiển thị trạng thái kiểm tra email |
| `register-unavailable` | 409 `REGISTRATION_NOT_AVAILABLE` | Guidance chung login/recovery |
| `register-password-validation` | 400/422 `VALIDATION_ERROR` | Map checklist length/uppercase/lowercase/digit/special |
| `registration-link-success` | 200 `ACTIVE` + session cookie | Auto-login và redirect `returnUrl` hợp lệ |
| `registration-link-invalid` | 400/422 | Hiển thị link không hợp lệ, cho request lại |
| `registration-link-expired` | 410/422 | Hiển thị link hết hạn, cho resend |
| `registration-link-used` | 409 | Điều hướng an toàn tới Login/current session |
| `registration-delivery-failed` | 502/503 | Retry theo policy |
| `registration-rate-limited` | 429 | Disable resend/submit |
| `registration-slow` | Success sau delay | Loading; chống duplicate account submit |

### 7.3. `AUTH-C07..C10` Recovery

| Scenario | Expected response | UI assertion |
| --- | --- | --- |
| `recovery-request-existing` | Neutral accepted | Không xác nhận account tồn tại |
| `recovery-request-unknown` | Cùng neutral response/copy | Không enumeration |
| `recovery-verify-success` | Reset proof | Chuyển bước đặt credential mới |
| `recovery-link-invalid` | Invalid error | Không tạo reset proof; không lộ account data |
| `recovery-expired` | Expired error | Cho khởi tạo/resend theo policy |
| `recovery-already-used` | Used error | Không reset; yêu cầu flow mới |
| `recovery-reset-validation` | Password validation error | Map checklist length/uppercase/lowercase/digit/special |
| `recovery-reset-success` | Completion | Xóa dữ liệu nhạy cảm khỏi client state; về Login |
| `recovery-rate-limited` | 429 | Hiển thị thời gian chờ nếu API cung cấp |
| `recovery-delivery-failed` | 502/503 | Retry/guidance |
| `recovery-slow` | Response delay | Loading; chống duplicate submit/resend |

## 8. Mock data rules

- Dùng ID cố định như `CUS-001`, `REGV-001`, `RECV-001` để test tái hiện được.
- Dùng email giả thuộc domain `example.com`; không dùng dữ liệu thành viên thật.
- Credential fixture chỉ là selector cho scenario local, không mô phỏng password thật và không commit token có giá trị.
- Không dùng dynamic Faker cho các scenario business assertion; chỉ dùng khi test layout/volume.
- Masked destination phải nhất quán với identifier fixture nhưng không chứa PII thật.
- Delay chỉ bật ở scenario `*-slow`, không áp vào success mặc định.

## 9. Mapping capability → requirement

| Capability | Requirement source | Screen |
| --- | --- | --- |
| `AUTH-C01`, `C02`, `C03` | `US-AUTH-03`, `FR-AUTH-04`, `FR-AUTH-05`, `UC-AUTH-03/06`, `FR-01` | D2C-009, shared shell/protected route |
| `AUTH-C04`, `C05`, `C06` | `US-AUTH-02`, `FR-AUTH-03`, `UC-AUTH-02`, `FR-01` | D2C-010 |
| `AUTH-C07`, `C08`, `C09`, `C10` | `US-AUTH-05`, `FR-AUTH-06`, `UC-AUTH-05`, EPIC 28 dependency | D2C-011 |
| `AUTH-C11` | `US-AUTH-01`, `FR-AUTH-01`, `FR-AUTH-09`, `UC-AUTH-01` | D2C-009 ↔ D2C-006 |

## 10. Điều kiện phê duyệt API Contract Candidate

OpenAPI candidate hiện tại chỉ được đổi trạng thái sang Approved sau khi:

1. Backend/Architecture xác nhận BFF cookie flow, internal JWT/refresh rotation, CSRF và Redis revocation.
2. Backend xác nhận HTTP status, endpoint naming và error taxonomy.
3. Notification team xác nhận contract gửi email magic link và trusted frontend origin.
4. Frontend xử lý hoặc thống nhất các implementation gap ghi trong `AUTHENTICATION_MVP_CONTRACT_REVIEW.md`.
5. Security review account enumeration, rate limiting, reset proof, cookie flags và open redirect.
6. Mockoon, frontend types và contract tests được đồng bộ theo bản đã review.

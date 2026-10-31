# Authentication MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
> Phiên bản: `0.1.0-candidate`
> Ngày đề xuất: 2026-09-24
> OpenAPI: `authentication-mvp.openapi.yaml`

## 1. Điểm dừng của workflow hiện tại

Artefact này hoàn thành bước **đề xuất API Contract từ implementation UI thực tế**. Nó chưa đại diện cho cam kết của Backend, Architecture hay Security; không được dùng để tuyên bố API production đã sẵn sàng.

Phần chưa thực hiện:

- Backend/Architecture/Security review và phê duyệt.
- Backend implementation, persistence, session store và email provider.
- Contract test với API Gateway thật.
- Chuyển frontend khỏi Mockoon.

## 2. Nguồn hình thành contract

Contract được đối chiếu từ:

- `docs/ui_web/SCOPE_TRACEABILITY.md` — phạm vi và Product decisions.
- `docs/ui_web/DATA_API_MATRIX.md` — capability, data và error semantics ban đầu.
- `mocks/mockoon/oma-auth-mvp.json` — executable response examples.
- `apps/web-user/src/services/auth-service.ts` — request thật mà UI đang gửi.
- `apps/web-user/src/lib/auth/types.ts` — response thật mà UI đang đọc.
- Các flow `/login`, `/register`, `/account-recovery` — behavior thực tế.

## 3. Ranh giới public API và BFF

| Lớp | Path | Mục đích |
| --- | --- | --- |
| Browser → Next BFF | `/api/auth/*` | Same-origin boundary; browser không biết host Mockoon/Gateway. |
| BFF → API Gateway | `/api/v1/auth/*` | Contract candidate mô tả trong OpenAPI. |
| Local scenario selection | `X-Mock-Scenario` | Chỉ Mockoon/development; không thuộc production contract. |

`returnUrl`, `passwordConfirmation` và `newPasswordConfirmation` là client-only data, không được gửi tới Auth API.

## 4. Capability được đề xuất

| ID | Method và path | UI consumer | Success |
| --- | --- | --- | --- |
| `AUTH-C01` | `POST /auth/login` | Login form | `200` + opaque session cookie |
| `AUTH-C02` | `GET /auth/me` | Session bootstrap/shared shell | `200` customer summary |
| `AUTH-C03` | `POST /auth/logout` | Shared account shell | `204` + clear cookie |
| `AUTH-C04` | `POST /auth/register` | Registration form | `201 PENDING_VERIFICATION` |
| `AUTH-C05` | `POST /auth/registration-verifications/confirm` | Registration magic link | `200 ACTIVE` + session cookie |
| `AUTH-C06` | `POST /auth/registration-verifications/{verificationId}/resend` | Pending registration state | `202` acknowledgement |
| `AUTH-C07/C10` | `POST /auth/recovery-requests` | Recovery request/resend | neutral `202` |
| `AUTH-C08` | `POST /auth/recovery-requests/verify` | Recovery magic link | `200` short-lived reset proof |
| `AUTH-C09` | `POST /auth/recovery-requests/reset` | New password form | `204`, revoke all sessions |

## 5. Quyết định đã phản ánh trong contract

- Email là identifier duy nhất của Authentication MVP.
- Registration chỉ gửi `displayName`, `email`, `password`, `termsVersion`.
- Verification và recovery dùng email magic link, không dùng OTP/SMS/Zalo.
- Password mới dài 15–128 Unicode code point, có chữ hoa, chữ thường, số và punctuation/symbol; whitespace không được tính là special.
- Browser chỉ nhận opaque `__Host-oma_session` cookie; token nội bộ không xuất hiện trong response body.
- `rememberMe=false` đề xuất session tối đa 24 giờ; `true` tối đa 30 ngày.
- Registration confirmation auto-login; password reset không auto-login và thu hồi mọi session.
- Recovery request trả response trung tính cho cả email tồn tại và không tồn tại.
- Error envelope dùng stable `code`; frontend không parse `message`.
- Validation dùng `422`, expired capability dùng `410`, already-used/conflict dùng `409` trong candidate này.

## 6. Phát hiện từ implementation thực tế

### 6.1. Field UI đang thực sự dùng

- Login đọc `authenticated`, `customer` rồi redirect; session không đọc được bằng JavaScript.
- Registration pending đang dùng `delivery.maskedDestination` và `verificationId`.
- Registration confirmation chỉ cần biết thành công để redirect sau khi cookie được thiết lập.
- Recovery verify dùng `resetProof`; reset success chỉ cần `204`.
- UI map field error theo `errors[].field` và hiển thị copy an toàn theo `code`.

### 6.2. Field contract giữ lại nhưng UI chưa tiêu thụ đầy đủ

| Field/header | Lý do giữ | Việc cần làm sau review |
| --- | --- | --- |
| `verificationExpiresAt` | Product chốt verification link 24 giờ. | Dùng để hiển thị thời hạn thay cho copy hard-code. |
| `resendAvailableAt` | Server là nguồn cooldown 60 giây. | Khóa nút resend theo server time. |
| `Retry-After` | Chuẩn hóa rate-limit UX. | UI hiện chỉ hiển thị lỗi chung; cần countdown nếu team duyệt. |
| `GET /auth/me` | Cần cho session bootstrap/protected route. | Shared shell hiện chưa gọi khi khởi động. |
| `POST /auth/logout` | Thuộc MVP session lifecycle. | Chưa có account menu/logout UI trong ba màn Auth. |

Những mục trên là implementation gap cần theo dõi, không phải bằng chứng rằng contract đã sai.

## 7. Sai khác có chủ đích so với Mockoon hiện tại

- OpenAPI bổ sung `422` cho recovery request malformed email; Mockoon chưa có scenario tương ứng.
- OpenAPI đề xuất `Retry-After` response header cho `429`; Mockoon hiện chỉ có error body.
- `SERVICE_UNAVAILABLE` được tách khỏi `NOTIFICATION_DELIVERY_FAILED` để phân biệt service outage và email-provider failure.
- OpenAPI không chứa `/health` vì đây là operational endpoint, không phải UI capability.
- OpenAPI không chứa `X-Mock-Scenario` vì header này tuyệt đối không đi tới production Gateway.

Mockoon chỉ được đồng bộ theo các điểm trên sau khi contract được review; candidate không tự động ghi đè mock đang phục vụ UI.

## 8. Câu hỏi cần Backend/Architecture/Security chốt

| ID | Câu hỏi | Candidate hiện tại |
| --- | --- | --- |
| `AUTH-CR-01` | Public contract thuộc BFF hay Gateway; có giữ prefix `/api/v1` không? | OpenAPI mô tả Gateway; BFF giữ `/api/auth/*`. |
| `AUTH-CR-02` | Cookie được tạo/rotate tại Gateway hay Auth Service? | Gateway/BFF sở hữu browser cookie; token nội bộ không lộ. |
| `AUTH-CR-03` | CSRF policy cho POST sử dụng cookie là Origin check, CSRF token hay cả hai? | BFF đã same-origin Origin check; production cần Security chốt. |
| `AUTH-CR-04` | Logout khi cookie thiếu/hết hạn có luôn `204` không? | Có, idempotent. |
| `AUTH-CR-05` | Dùng `422/410/409` như candidate hay gom business error về `422`? | Giữ semantic status riêng. |
| `AUTH-CR-06` | `Retry-After` dùng seconds hay HTTP-date; có thêm body field không? | Integer seconds trong header, không lặp trong body. |
| `AUTH-CR-07` | `displayName` normalize/trim và giới hạn chính xác thế nào? | Candidate 1–100 ký tự; chưa chốt normalization. |
| `AUTH-CR-08` | `verificationId` trong URL có đủ entropy và có bị log như bearer capability không? | Opaque reference; token xác minh vẫn chỉ nằm trong magic link body exchange. |
| `AUTH-CR-09` | TTL reset proof sau exchange là bao lâu? | Mock hiện 5 phút; cần Security chốt. |
| `AUTH-CR-10` | Correlation ID dùng body `requestId`, header hay cả hai? | Body bắt buộc trong error; cần thống nhất với observability. |
| `AUTH-CR-11` | Neutral recovery timing được cân bằng ở Gateway hay Auth Service? | Backend phải làm response khó phân biệt; chưa quy định thuật toán. |
| `AUTH-CR-12` | `Clear-Site-Data` nên xóa phạm vi nào khi logout/reset? | Candidate chỉ `"cache"`; Security review bắt buộc. |

## 9. Tiêu chí để đổi trạng thái sang Approved

- Backend xác nhận endpoint ownership, schema và status/error taxonomy.
- Architecture xác nhận BFF/Gateway/session boundary.
- Security xác nhận cookie, CSRF, account enumeration, rate limiting, token/proof TTL và revocation.
- Notification team xác nhận email delivery handoff và trusted frontend magic-link origin.
- Frontend cập nhật các implementation gap đã được team chấp thuận.
- Mockoon và generated client/types được đồng bộ theo OpenAPI cuối cùng.
- Contract test chạy được trên API Gateway thật.

Cho đến khi hoàn tất các mục trên, file OpenAPI phải giữ hậu tố version `-candidate` và trạng thái **PROPOSED / NOT APPROVED**.

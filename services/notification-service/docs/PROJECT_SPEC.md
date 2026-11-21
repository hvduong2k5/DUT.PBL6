# Product specification — Notification / BC-17

Status: PROPOSED DESIGN, 2026-11-06. Nguồn nghiệp vụ: EPIC 28 tại commit trong SOURCE_AUDIT. User chọn Go làm hướng triển khai; chưa phê duyệt production provider hay mở rộng các service nguồn.

## Mục tiêu

Nhận một intent/event được phép, xác định recipient/audience và policy đã duyệt, tạo request bền vững, gửi qua provider, theo dõi attempt và bằng chứng callback. Lỗi Notification không làm thất bại/rollback giao dịch nguồn. Model khác có thể triển khai từng phase mà không cần tái dựng Mobile history.

Actors: service nguồn đáng tin; guest/registered recipient; operator được cấp quyền; provider callback. V1 không mở API “gửi tùy ý đến email bất kỳ” cho khách hàng.

## MVP đề xuất

1. Email xác nhận đặt Order và các transition đã chốt với Order owner; tách placed với paid, không nói “đã thanh toán” chỉ vì Order được tạo.
2. Email OTP theo yêu cầu/expiry của Identity, **chỉ sau khi thống nhất luồng với IdP**. Notification không sinh/verify OTP, không reset password.
3. Email invoice khi Finance có artifact và audience contract; không tự tạo hóa đơn hoặc link công khai.
4. Common reliability: inbox dedup, template immutable/versioned, policy classification, privacy, encrypted recipient/payload, attempts, UNKNOWN/reconciliation, retry/DLQ, audit, operational search, metrics.
5. Local demo dùng dữ liệu tổng hợp và fake/capture provider. Gọi là “LOCAL CORE VERIFIED”; chỉ gọi “US-NOTI-01 MVP accepted” khi Order + OTP + Invoice + gift audience scenarios có evidence, hoặc owner đã phê duyệt scope giảm bằng văn bản trong DECISIONS.

Thiếu owner contract không ngăn foundation/test-fixture, nhưng ngăn bật adapter và tuyên bố E2E hoàn tất.

## Giai đoạn sau

- SMS/Zalo: provider, fallback policy, contact verification, Shipment ordering/coalescing.
- Push Mobile/Web + Notification Center: authenticated endpoint registration, device ownership/revocation, inbox/read state, safe deep links, consent/quiet-hours/caps.
- Internal Alert: role/resource-scope routing, read/ack vs resolved, SLA escalation; source vẫn sở hữu điều kiện urgency/expiry/breach/resolution.
- Marketing: audience từ Marketing/Promotion; consent explicit default-deny, không lẫn nội dung quảng cáo trong Email giao dịch. Chưa bật ở MVP.

## Non-goals hiện tại

Không code Go trong tác vụ lập plan; không UI Flutter/Web/Admin; không sửa Order/Profile/Identity/Finance/Care/Gateway; không tạo OTP, payment/invoice engine hoặc scheduler tự quyết định business triggers; không subscription billing/marketing campaign engine; không tự cấp quyền nhân viên; không real sends/production infra; không publish shared contracts sang packages/docs gốc.

## Yêu cầu bất biến

- Durable acceptance khác provider acceptance, delivery và read. Notification ID không cấp quyền truy cập resource.
- Một business intent không tạo lại khi event redelivery, retry, template upgrade hoặc consumer restart. Intentional resend phải là command riêng, có reason và audit.
- Guest nhận Email khi nguồn chứng minh contact được phép cho đúng Order; không bắt buộc biến guest thành customer.
- Buyer/recipient là hai audience độc lập. Hide-price policy dùng whitelist biến template riêng, không chỉ CSS ẩn giá.
- Không hỗ trợ một phiên bản event bằng cách bỏ qua required fields hoặc bịa customer/email/source version.
- Marketing không consent → suppress; opt-out Marketing không tự chặn security/transactional, nhưng cũng không tự cho phép vượt hard bounce/complaint. Policy này phải được chốt riêng.
- Dữ liệu lưu/log/metrics tối thiểu; không đưa OTP hoặc raw payload vào Kafka/DLQ/audit chung.

## Traceability và nghiệm thu

| FR EPIC 28 | Hạng mục trong plan | Phase | Điều kiện chính |
|---|---|---|---|
| 01–02 | Intake, durable dedup/fingerprint | N1–N2 | Duplicate/concurrent/collision tests; commit trước ack |
| 03–04 | Template version, schema, safe render | N2 | Missing vars fail; HTML/header/link injection tests; history immutable |
| 05 | Classification + dispatch policy | N1–N2 | Unknown classification denied; producer không tự nâng priority/mandatory |
| 06 | Order Email | N3 | Đúng event/cohort/audience/contact; không giả thanh toán |
| 07 | OTP Email | N3 gated | Identity owns expiry/resend; encrypted secret, no late send, không double-send IdP |
| 08 | Invoice Email | N3 gated | Đúng artifact/Order/recipient; expired/unauthorized link rejected |
| 09 | Bounce/complaint suppression | N2–N3 | Verified callbacks; suppression scope và security exception explicit |
| 10 | Gift buyer/recipient | N2–N3 | Whitelist projection, recipient payload không chứa giá/buyer PII |
| 11–13 | SMS/Zalo + fallback + ordering | N5 | Chốt contact/ordering/fallback trước implementation |
| 14 | Delivery khác read | N2, N5–N6 | Không tạo read từ provider acceptance/delivery |
| 15–19 | Endpoints, Push, consent, center | N6 | Multi-device ownership, revoke, UI/API acceptance riêng |
| 20–24 | Internal Alert/current scope | N7; 24 cũng N6 | Role/resource recheck, ack không resolve nghiệp vụ |
| 25–27 | Delivery reliability/callback/non-rollback | N2–N4 | Fault injection, UNKNOWN, fencing, late callbacks |
| 28 | Audit | N1–N4 | Durable same-transaction metadata event, no sensitive values |
| 29 | Privacy/retention | N1–N4 | Encryption, purge/tombstone rules, no plaintext tests |
| 30–31 | Metrics/quota/backpressure | N2–N4 | Bound workers, durable deferred state, no drops |
| 32 | Authorized operations search | N3–N4 | Masking, scoped access, cursor limits, audit |

## Capacity / SLO assumptions cần xác nhận

Chưa có tải, ngân sách provider hoặc SLO được user giao. Thiết kế với bounded workers và backpressure; không hứa “100k users” hoặc SLA theo benchmark service khác. Mục tiêu local đề xuất để đo: 20 synthetic requests/s trong 5 phút với 2 workers, không mất/duplicate logical intent; p95 durable intake < 500ms trên cấu hình host được ghi. Đây là acceptance benchmark ứng viên, không phải cam kết production hay provider delivery latency. OTP deadline luôn do nguồn cấp, không tự mặc định thời hạn.

# Kế hoạch triển khai tiếp theo — identity-service (MS-16)

Trạng thái: kế hoạch triển khai đề xuất dựa trên cấu trúc scaffold và các yêu cầu Identity/Auth đã cung cấp. Đây chưa phải xác nhận rằng các bước đã được code hoặc kiểm thử.

## 1. Mục tiêu và nguyên tắc kiến trúc
Xây identity-service bằng Java + Spring Boot, MyBatis, PostgreSQL và Flyway; dùng Keycloak làm Identity Provider. Không xây một hệ thống credential/token thứ hai.

**Nguồn chân lý**
- Keycloak: credential/password policy, login OIDC, MFA/required actions, SSO session, refresh token, social identity, account enabled/disabled, signing keys và JWKS.
- identity-service + identity_db: ánh xạ principal nội bộ, điều phối provisioning/registration, lời mời nhân viên, quyền IAM chi tiết theo scope/thời hạn, policy quyết định quyền đặc biệt, audit và transactional outbox.
- Profile Service: tên, avatar, địa chỉ và hồ sơ nghiệp vụ.
- Order Service: đơn hàng, quyền sở hữu đơn, guest_order_grant, điều kiện hủy/claim đơn Guest.
- Cart/Checkout: guest_session và giỏ Guest.
- Domain services: vẫn là nơi thực thi authorization đối với tài nguyên/nghiệp vụ của chính chúng.

**Contract hiện được giả định**
- Tên chuẩn: identity-service; port dự kiến 8016.
- Gateway xác minh JWT bình thường bằng issuer/audience/signature/expiry và JWKS cache. Không gọi Identity đồng bộ cho mọi request.
- Chỉ các quyết định rủi ro cao mới gọi CheckSpecializedPermission qua gRPC.
- Proto hiện có GetJwksPublicKey và CheckSpecializedPermission; không code dựa trên ValidateToken hoặc CheckPermission nếu chưa được bổ sung proto và phê duyệt ADR.
- Các sự kiện Kafka chỉ được publish khi schema versioned trong packages/events đã được duyệt.

## 2. Giai đoạn 0 — Chốt các quyết định bắt buộc (P0, làm trước khi thêm nghiệp vụ)
**Công việc**
- [ ] Chốt một nguồn contract cho customer REST, admin REST, gRPC và Kafka; đánh dấu spec cũ là deprecated hoặc draft.
- [ ] Chốt luồng đăng nhập theo OIDC Authorization Code + PKCE. Không triển khai API nhận username/password rồi tự phát JWT, và không dùng OAuth Resource Owner Password Credentials.
- [ ] Quyết định frontend đi trực tiếp OIDC hay qua BFF. Nếu browser dùng cookie, chốt Secure, HttpOnly, SameSite, domain/path, CSRF và luồng refresh/logout.
- [ ] Chốt token profile: realm issuer, audience cho từng API, algorithm, kid, claim sub, claim user_id, access TTL, refresh TTL, clock skew, refresh/revoke policy.
- [ ] Giải quyết mâu thuẫn access-token TTL trong tài liệu (15 phút so với 1 giờ); cập nhật tất cả spec/config/test cùng lúc.
- [ ] Chốt mapping ổn định principal_id UUID ↔ Keycloak subject và claim user_id (ví dụ protocol mapper đọc app_user_id). Chỉ service account được cấp quyền sửa attribute bảo mật này.
- [ ] Chốt mô hình role: role coarse-grained trong Keycloak; role assignment có scope/thời hạn và quyền thao tác chi tiết trong PostgreSQL. Tránh quản lý trùng cùng một quyền ở hai nơi.
- [ ] Chốt lifecycle invitation/employee, account disable, role revoke và thời gian tối đa quyền bị thu hồi có hiệu lực.
- [ ] Chốt tên action_code, schema context_attributes, ngưỡng và fail-closed policy cho CheckSpecializedPermission.
- [ ] Chốt event schema CloudEvents cho identity.events.v1 và audit.events.v1, consumer, partition key, PII policy và retry/DLQ.
- [ ] Chọn rõ contract đăng ký: Keycloak xử lý self-registration/required action trực tiếp hay identity-service điều phối tạo user qua Admin REST API. Không triển khai hai luồng song song.

**Đầu ra / tiêu chí hoàn thành**
- ADR được phê duyệt.
- OpenAPI, proto, event catalog và configuration không còn mâu thuẫn ở các mục đã chốt.
- Có ma trận ownership và quyền truy cập cho Customer, Employee, Guest và service principal.

## 3. Giai đoạn 1 — Làm cho scaffold build và database đáng tin cậy (P0)
**Công việc**
- [ ] Kiểm tra build.gradle, Gradle Wrapper, Java toolchain và phiên bản Spring Boot theo chuẩn monorepo.
- [ ] Cập nhật MyBatis Spring Boot Starter lên phiên bản tương thích đã chốt (ví dụ 3.0.5 nếu phù hợp với cấu hình monorepo); không nâng phiên bản chỉ dựa trên tài liệu mà chưa build.
- [ ] Kiểm tra @MapperScan, package Mapper, mapper-locations, map-underscore-to-camel-case.
- [ ] Hoàn thiện IdentityPrincipalMapper, DTO/row model và XML; thêm các query nền tảng: insert principal, tìm theo principal_id, tìm theo (idp_issuer, idp_subject), cập nhật provisioning state/version.
- [ ] Thêm Mapper/XML cho registration operation, IAM, authorization decision, audit và outbox theo từng giai đoạn, không tạo toàn bộ repository cùng lúc.
- [ ] Chạy Flyway V1–V5 trên PostgreSQL sạch; xác minh constraints, index, FK và thứ tự migration.
- [ ] Thêm integration test MyBatis + PostgreSQL/Testcontainers. Không chỉ dựa vào mock Mapper.
- [ ] Bật transaction boundary rõ ràng ở service layer; mọi SQL thay đổi policy/role cần optimistic concurrency/version khi phù hợp.
- [ ] Bảo đảm migration V1–V5 bất biến sau khi áp dụng ở môi trường dùng chung; thay đổi tiếp theo dùng V6+.

**Tiêu chí hoàn thành**
- `./gradlew clean test` thành công.
- Migration chạy từ database trống thành công.
- Mapper kiểm thử được trên PostgreSQL thật/Testcontainers.
- Không có secret production mặc định trong application.yml; chỉ dùng env/secret manager.

## 4. Giai đoạn 2 — Thiết lập Keycloak và tích hợp nền tảng (P0)
**Công việc**
- [ ] Tạo realm cho ứng dụng và cấu hình issuer nhất quán giữa Keycloak, Gateway và service.
- [ ] Tạo client cho frontend theo Authorization Code + PKCE; quyết định public/confidential client theo loại ứng dụng.
- [ ] Tạo service account/client riêng cho identity-service Admin API; cấp quyền tối thiểu, không dùng master admin credential.
- [ ] Tạo audience/client scopes cho catalog-api, order-api, inventory-api, identity-api và các API thực tế cần xác thực.
- [ ] Thiết lập mapping claim ổn định user_id từ app_user_id và role/scope coarse-grained.
- [ ] Lưu client secret trong secret manager hoặc môi trường triển khai; không commit vào Git và không ghi log.
- [ ] Thiết lập key rotation/JWKS cache policy tại Gateway; xác định xử lý unknown kid và khi Keycloak/JWKS unavailable.
- [ ] Cấu hình các luồng required action, xác minh email/phone, reset password và MFA theo phạm vi MVP.

**Tiêu chí hoàn thành**
- Login qua OIDC hoạt động ở môi trường local/test.
- Access token hợp lệ có issuer, audience, expiry, sub và user_id đúng quy ước.
- Negative tests từ chối token sai issuer/audience, hết hạn, chữ ký sai hoặc kid không hợp lệ.
- identity-service chỉ truy cập Admin API bằng client credential riêng, quyền tối thiểu.

## 5. Giai đoạn 3 — Principal mapping và đăng ký idempotent (MVP đầu tiên)
**Công việc**
- [ ] Xây domain/service cho identity_principals và registration_operations bằng MyBatis.
- [ ] Thực hiện registration orchestration theo ADR: tạo local operation + principal_id ở trạng thái PENDING; gọi Keycloak Admin REST; gắn app_user_id; hoàn tất mapping; ghi outbox trong local transaction.
- [ ] Không lưu password, access token, refresh token, recovery token hoặc raw invitation token vào identity_db.
- [ ] Thực hiện idempotency: cùng key + cùng request fingerprint trả lại kết quả an toàn; cùng key nhưng khác fingerprint trả conflict.
- [ ] Thiết kế reconciliation job cho tình huống Keycloak tạo user thành công nhưng local DB update thất bại. Không tự động xóa user Keycloak như compensation mặc định.
- [ ] Chốt response registration: không giả định Profile đã được tạo đồng bộ bởi event. Chọn response identity-only hoặc composition/BFF có chính sách timeout/fallback.
- [ ] Tạo UserRegistered outbox row sau khi registration hoàn tất và theo schema event đã được duyệt.

**Tiêu chí hoàn thành**
- Đăng ký thành công và tạo mapping duy nhất.
- Duplicate normalized identifier được xử lý ổn định.
- Retry không tạo nhiều principal/user.
- Có integration tests cho các failure window giữa Keycloak và PostgreSQL.

## 6. Giai đoạn 4 — Authorization IAM nhân viên (P1)
**Công việc**
- [ ] Xây query/command cho iam_roles, iam_permissions, iam_role_permissions, iam_role_assignments.
- [ ] Chuẩn hóa permission code, ví dụ catalog.product.approve, inventory.stock.adjust, finance.refund.approve.
- [ ] Hỗ trợ GLOBAL và scoped assignment, valid_from, valid_until, trạng thái hiệu lực và version.
- [ ] Chỉ cho phép principal loại EMPLOYEE nhận assignment nhân viên.
- [ ] Thực thi kiểm tra người cấp quyền; ngăn tự nâng quyền và bảo vệ tài khoản System Administrator cuối cùng.
- [ ] Định nghĩa API admin stable trước khi mở frontend; những endpoint hiện ở admin-draft-planned.openapi.yaml chưa được coi là stable.
- [ ] Thay đổi account/role/permission/assignment phải ghi audit và áp dụng concurrency control.
- [ ] Định nghĩa chiến lược quyền bị thu hồi: TTL token, policy re-check cho thao tác rủi ro cao và cache invalidation nếu có.

**Tiêu chí hoàn thành**
- Assignment hết hạn hoặc chưa đến thời điểm bắt đầu không có hiệu lực.
- view không được suy ra thành update, approve hoặc export.
- Kiểm thử concurrency, inactive role, scoped resource mismatch và last-admin protection.

## 7. Giai đoạn 5 — Quyền đặc biệt qua gRPC (P1)
**Công việc**
- [ ] Generate gRPC code từ packages/proto/identity/v1/identity.proto; không tự viết request/response thay proto.
- [ ] Cài đặt CheckSpecializedPermission theo action, context_attributes và AuditContext đã chốt.
- [ ] Xây policy evaluator có schema được validate; không thực thi SQL, JavaScript, SpEL hoặc expression tùy ý lưu trong JSON.
- [ ] Ghi decision gồm actor, action, allow/deny, reason code, policy/version và context đã sanitize.
- [ ] Đặt timeout/deadline, giới hạn kích thước payload và phân loại lỗi chuẩn.
- [ ] Với tác vụ high-risk, timeout/unavailable phải fail closed trừ khi ADR được phê duyệt quy định khác.
- [ ] Xác thực caller nội bộ bằng TLS/mTLS hoặc service credential; không tin header từ internet.

**Tiêu chí hoàn thành**
- Test allow/deny, biên ngưỡng, context thiếu/sai, policy version, timeout và Identity unavailable.
- Quyết định và audit nhất quán trong transaction cục bộ khi cần.

## 8. Giai đoạn 6 — Outbox, Kafka và audit (P1)
**Công việc**
- [ ] Hoàn thiện outbox_events Mapper, repository và publisher worker.
- [ ] Chỉ tạo event theo schema versioned tại packages/events; envelope tuân CloudEvents 1.0 theo yêu cầu dự án.
- [ ] Ghi thay đổi DB, audit row và outbox row trong cùng transaction cục bộ khi phải đảm bảo phát event.
- [ ] Publisher dùng retry/backoff, trạng thái lỗi, xử lý crash và DLQ theo policy nền tảng.
- [ ] Consumer phải dedupe theo eventId; chấp nhận at-least-once delivery.
- [ ] Chỉ phát UserRegistered, UserDeactivated, PasswordChanged khi event semantics và consumer đã chốt.
- [ ] Audit login/security actions không chặn đồng bộ đường login; không chứa password, token, OTP, reset proof hoặc PII không cần thiết.
- [ ] Phân quyền Kafka principal riêng, TLS/SASL/ACL theo hạ tầng; không dùng user role trong JWT làm thay thế broker ACL.

**Tiêu chí hoàn thành**
- Outbox publisher restart/retry không làm mất event sau commit.
- Có kiểm thử trùng event, lỗi Kafka, retry, DLQ và schema compatibility.
- Payload và log vượt qua kiểm tra không rò rỉ secret.

## 9. Giai đoạn 7 — Account lifecycle, recovery và employee invitation (P1/P2)
**Công việc**
- [ ] Cấu hình/điều phối verify email/phone, forgot/reset password và logout bằng cơ chế Keycloak được chọn.
- [ ] Quyết định neutral response để tránh account enumeration và rate-limit cho recovery.
- [ ] Xây invitation workflow cho employee; token bí mật do Keycloak/flow chuyên dụng quản lý, DB Identity chỉ lưu metadata/trạng thái.
- [ ] Disable account hoặc thu hồi assignment phải có behavior xác định với session/token đang còn hiệu lực.
- [ ] Social login/linking (Google/Facebook/Zalo) chỉ triển khai sau khi chốt provider, identity-linking policy và chống account takeover.
- [ ] Không triển khai guest account. Guest session/cart thuộc Checkout; guest_order_grant thuộc Order.

**Tiêu chí hoàn thành**
- Reset password/disable/revoke có tác động được kiểm chứng đến session/quyền trong SLA.
- Invitation one-time/expiry, replay và revoke có tests.
- Social linking không tự link tài khoản chỉ vì email giống nhau nếu chưa xác minh chính sách.

## 10. Giai đoạn 8 — Security regression, vận hành và phát hành (P0 trước production)
**Test bắt buộc**
- JWT: expired, invalid signature, wrong issuer, wrong audience, unknown kid, key rotation.
- Registration: duplicate normalized email/phone, idempotency replay/conflict, Keycloak success + DB failure, reconciliation.
- IAM: assignment time window, scope mismatch, role inactive, concurrent update, last-admin protection.
- Specialized permission: allow/deny, threshold boundary, missing context, timeout/fail-closed.
- Outbox: crash after publish/before marking sent, duplicate delivery, unavailable broker, DLQ.
- Security: header spoofing, CSRF khi dùng cookie, brute-force/rate limit, secret/PII trong log, quyền Admin API tối thiểu.
- Persistence: migration từ database trống, constraints và transaction semantics với PostgreSQL integration tests.

**Vận hành**
- [ ] Health/readiness/liveness và metric cho Keycloak/Admin API, DB, Kafka, outbox lag, reconciliation và permission latency.
- [ ] Correlation/trace ID xuyên Gateway → Identity → Kafka/gRPC; không log token.
- [ ] Backup/restore identity_db, Keycloak DB, realm/client configuration và signing key policy.
- [ ] Secret rotation, TLS, least privilege, network policies và retention audit.
- [ ] Runbook cho Keycloak unavailable, JWKS unknown key, reconciliation backlog, Kafka outage và khôi phục database.

## 11. Trình tự PR đề xuất
1. PR-1: ADR và contract cleanup; build/version/config; Flyway/MyBatis integration tests.
2. PR-2: Keycloak realm/client setup bằng cấu hình có thể tái lập; issuer/audience/JWKS verification.
3. PR-3: Principal mapping + registration idempotency + reconciliation.
4. PR-4: Event schema/package + outbox publisher + Profile consumer contract test.
5. PR-5: IAM role/permission/assignment và admin API khi stable contract được duyệt.
6. PR-6: gRPC CheckSpecializedPermission và policy/decision audit.
7. PR-7: lifecycle, invitation/recovery/social login tùy ưu tiên.
8. PR-8: security regression, resilience, deployment, observability và runbooks.

## 12. Điều kiện MVP được xem là hoàn thành
- [ ] Keycloak là nguồn chân lý duy nhất cho credential/session/token.
- [ ] Gateway/service validation dựa vào issuer/audience/signature/expiry, không gọi Identity cho mọi request thường.
- [ ] Principal mapping và registration retry-safe; có reconciliation.
- [ ] IAM assignment có scope/thời hạn hoạt động đúng; admin operation có audit.
- [ ] CheckSpecializedPermission khớp proto và có fail-closed policy được duyệt.
- [ ] Outbox/event schema/versioning/idempotency hoạt động và không rò rỉ secret.
- [ ] Flyway/MyBatis, integration tests và runbooks đã được xác minh trên môi trường test.

## 13. Những việc chưa nên làm ngay
- Không code toàn bộ admin REST trước khi draft OpenAPI được chốt thành stable.
- Không dùng Keycloak Direct Grant hoặc tự phát hành JWT trong Identity.
- Không tạo bảng credential/session/recovery song song với Keycloak.
- Không làm cho mọi API gọi gRPC đến Identity để xác minh token.
- Không phát Kafka event tùy ý trước khi schema và consumer contract được duyệt.
- Không đưa Guest session hoặc quyền sở hữu đơn Guest vào Identity DB.

# Rà soát nguồn API Identity cho Mobile — 2026-11-04

## Kết luận và phạm vi

Yêu cầu hiện tại: đọc prompt_tinh_nang_login_mobile.txt, kiểm tra từng nhánh Git và thư mục Drive để tận dụng API có sẵn; chưa triển khai thêm code hoặc kích hoạt auth thật. Đây là rà soát source/contract, không phải kết quả chạy backend.

**Đã có API cho cả 8 endpoint trong prompt.** Nguồn tích hợp đề xuất là Identity Service tại `origin/dev` commit `a24492b63c364f93ad19d880798137e554898c3f`. Không có nhánh tên authservice, nhưng điều đó không đồng nghĩa thiếu backend. Nhận định nguồn chưa có trong checkpoint M6 trước đây đã lỗi thời sau lần fetch này.

Mobile hiện ở `feature/mobile-ui`, HEAD `0c0dd058ec7cb64dda1a0787e5d552ce584320db`; source ứng dụng vẫn là bản đã kiểm thử trước đó. Không checkout/merge nhánh khác. Chỉ tài liệu audit/handoff được sửa; bản nháp `docs/MOBILE_MVP_DISCUSSION_REFERENCE.md` không thuộc thay đổi này.

## Cách kiểm tra và độ bao phủ

- Đã `git fetch origin`, đối chiếu `git ls-remote --heads origin`: 25 nhánh remote tại thời điểm kiểm tra, không có authservice.
- Liệt kê file trên từng nhánh bằng Git objects; tìm route auth trong code Java/Go/TS/JS/Python, phân biệt frontend client/test/spec với backend implementation.
- So sánh tree/blob và diff để đọc các phiên bản Identity khác nhau, không đọc lặp file giống hệt trên nhiều nhánh. Đọc route/interface → controller → service/orchestrator → DTO → session/filter/security/config, các test liên quan và Gateway.
- Kiểm tra thêm 4 nhánh local: doc, feat/content-service, feature/mobile-ui trùng tip remote tương ứng; main local `09a6616` chỉ có khung auth-service `.gitkeep`.
- Không tuyên bố đã đọc từng dòng của toàn bộ repository: việc đọc từng file tập trung vào module auth và phụ thuộc cần thiết cho prompt.

| Nhánh remote | SHA ngắn | Kết quả phần auth |
|---|---|---|
| dev | a24492b | Có đủ endpoint prompt, native flow, cookie binding, Gateway; nguồn chính đề xuất |
| codex/fulfillment-order-shipping | 4c877d9 | Identity subtree giống hệt dev: deadec44c67d981a36d901ce799811283b3045dc |
| feature/ui-web | 47ca69b | Có endpoint prompt nhưng phiên bản Identity cũ hơn dev; callback/refresh khác |
| feature/catalog-service | 68a9fb4 | Có endpoint prompt, cùng thế hệ cũ hơn dev; thêm migration quyền Catalog |
| codex/identity-order-claims | 988ae33 | Identity cũ: profile/registration/claims; chưa có AuthenticationApi browser/session như prompt |
| codex/mvp-customer-order-shipping | 4e20f42 | Identity cũ, có cải tiến đăng ký/idempotency; chưa có đầy đủ browser/session |
| codex/web-identity-gateway | b3735cb | Identity cũ + tích hợp Web; chưa có đầy đủ browser/session |
| codex/order-inventory-integration | 273adc3 | Identity tree d85288aa: profile/registration, chưa đủ prompt |
| codex/ui-contract-gaps | 2d7737b | Cùng Identity tree d85288aa |
| feat/inventory-service | 5131c13 | Cùng Identity tree d85288aa |
| feat/shipping-service | a415bbf | Cùng Identity tree d85288aa |
| feat/traceability-service | 601dd11 | Cùng Identity tree d85288aa |
| backup/feature-ui-web-before-rebase | 11df2b6 | Web client/spec, backend auth-service chỉ khung |
| catalog-service | 4587b76 | Không có Identity implementation của prompt; auth-service khung |
| codex/order-keycloak-auth | c041774 | Consumer xác thực JWT của Order, không phải dịch vụ phát hành phiên/login |
| copilot/trin-khai-inventory-service | e23a919 | Không có Identity implementation của prompt; auth-service khung |
| doc | 12e42da | Tài liệu/khung auth-service |
| feat/care-service | 94d8731 | Không có Identity implementation của prompt; auth-service khung |
| feat/content-service | 50f35e5 | Không có Identity implementation của prompt; auth-service khung |
| feat/order-service | 5479e19 | Order/contract consumer; auth-service khung |
| feat/order-service-spGW | c19bd2f | Order/Gateway consumer; auth-service khung |
| feat/profile-service | a36afa2 | Profile/authorization consumer; auth-service khung |
| feature/mobile-ui | 0c0dd05 | Có Flutter Identity adapter; backend auth-service khung |
| integration/mvp-contracts | 9416090 | Contract/reference, không có Identity implementation của prompt |
| main | f0033b8 | Khung auth-service |

## API có thể tái sử dụng

Tất cả đường dẫn sau có prefix `/api/v1`. Source Java bên dưới nằm trong `services/identity-service/src/main/java/com/hvduong/identity/` tại dev đã pin.

| API | Source chính | Kết luận |
|---|---|---|
| GET /public/auth/login | api/account/AuthenticationApi.java; impl/AuthenticationController.java; application/account/impl/AuthenticationServiceImpl.java | Có PKCE S256/state, JSON authorizationUrl, chọn native theo redirect URI đã cấu hình |
| GET /public/auth/callback | Cùng các file trên; security/BffSessionAuthFilter.java; infrastructure/session/AuthStateStore.java | Có đổi code, state một lần/10 phút; bắt buộc cookie OMAMA_AUTH_STATE; native nhận token JSON |
| GET /public/auth/session | AuthenticationController/ServiceImpl; dto/response/SessionResponse.java | Có authenticated; public sessionId chỉ để đối chiếu, không thay cookie credential |
| GET /users/me | api/account/UserLifecycleApi.java; impl/UserLifecycleController.java; application/account/impl/UserLifecycleServiceImpl.java | Có principalType/provisioningStatus/roles/permissions, issuer+subject và user_id checks; không có firstName/lastName trong DTO hiện tại |
| POST /public/auth/tokens/refresh | AuthenticationController/ServiceImpl; BffSessionStore.java | Có cookie session refresh; native trả accessToken và credential sessionId, không cần gửi raw refreshToken |
| POST /public/auth/logout | AuthenticationController/ServiceImpl | Có xóa session/cookie và thử revoke Keycloak; lỗi revoke được server bắt/log nên HTTP 200 không chứng minh revoke upstream thành công |
| POST /public/customer-registrations | api/registration/RegistrationApi.java; impl/RegistrationController.java; application/registration/impl/RegistrationOrchestrator.java | Có tạo Keycloak user, DB principal/outbox, idempotency, fingerprint do server HMAC; cần đọc status |
| PUT /users/me/password | UserLifecycleApi/Controller/ServiceImpl; dto/request/ChangePasswordRequest.java | Có code; chưa sẵn sàng kết luận dùng qua Gateway/template hiện tại, xem blocker dưới |

Contract gần implementation nhất: `docs/03_api_specs/identity-service.openapi.yaml` tại dev. `authentication-mvp.openapi.yaml` vẫn thiên về BFF Web, callback mô tả 302 và có base/path cần kiểm tra trước codegen; không dùng riêng file này để suy ra native DTO. Không ghép contract của các commit/nhánh khác nhau thành một API giả định.

## Những thay đổi Mobile thực sự cần — chưa implement

1. **Cookie callback (chặn login):** IdentityApi chỉ đọc Set-Cookie `OMAMA_SESSION`, trong khi login trả `OMAMA_AUTH_STATE`. Repository truyền sessionId của bước login, không giữ cookie state. Filter backend dòng 46–54 từ chối callback thiếu cookie bằng 400 INVALID_AUTH_STATE. Cần cookie transaction riêng, scope đúng origin, chỉ tồn tại trong lần login, dọn khi hoàn tất/hủy/hết hạn. Không bỏ kiểm tra state để vượt lỗi.
2. **Refresh/restore/logout (chặn khi token hết hạn):** IdentityApi tự gắn Bearer vào mọi request có session. BffSessionAuthFilter bỏ xử lý cookie khi thấy Bearer; resource-server tiếp tục kiểm tra JWT, kể cả route public. Bearer hết hạn có thể chặn refresh trước controller, rồi Mobile xóa session khi gặp 401. Contract dev yêu cầu native refresh bằng cookie, không gửi Bearer hết hạn. Tách rõ credential của session endpoints và profile/commerce; sau refresh xác minh persona rồi mới công bố token. GET session không trả token mới, không thay credential bằng public sessionId.
3. **Đăng ký:** Mobile đang coi mọi success:true là đã đăng ký và mở login. Backend có thể trả 200 với status PENDING khi replay đang xử lý; lần đầu thành công trả ACTIVE, replay hoàn tất trả COMPLETED. Cần phân biệt các trạng thái, không báo thành công/mở login sớm. Giữ idempotency key cho cùng nội dung retry. requestFingerprint email của prompt không còn là authority: server tự tính HMAC, Mobile không cần tự xây thuật toán fingerprint mới.
4. **Validation đúng policy:** Flutter đang dùng chung 15–128 ký tự; DTO đăng ký dev yêu cầu 12–128, DTO đổi mật khẩu min 8. Cần xác nhận policy Keycloak cuối cùng rồi tách validation theo tác vụ, không tự hạ bảo mật chỉ để khớp một con số trong DTO.
5. **Giữ lại phần đã làm:** browser plugin/deep link, origin allowlist, PKCE/state checks, atomic secure storage, persona/Employee purchase guards, UI login/register/account/password và local mode. Không viết thêm AuthService hay làm lại toàn bộ UI.

Các file Mobile cần sửa khi được triển khai: `features/auth/data/identity_api.dart`, `identity_auth_repository.dart`, các test Identity hiện có; UI đăng ký/validation chỉ điều chỉnh theo trạng thái/policy đã xác minh. Hiện chưa sửa file Dart nào.

## Phần backend/config còn vướng

- **Đổi mật khẩu qua Gateway:** `services/api-gateway/config/kong.yml:41–53` có `/api/v1/users/me` nhưng chỉ POST/GET/OPTIONS. Chưa có route PUT `/api/v1/users/me/password` trong bảng route này. Cần owning team bổ sung route; không sửa backend từ nhánh Mobile.
- **Đổi mật khẩu và Keycloak:** UserLifecycleServiceImpl:459 xác minh mật khẩu hiện tại bằng grant_type=password, nhưng realm-export.template.json:26 tắt directAccessGrantsEnabled cho frontend-web. Theo template hiện tại luồng này mâu thuẫn; cần backend quyết định luồng xác minh phù hợp, không đề xuất bật password grant chỉ để chạy được. Chưa thấy đổi mật khẩu tự revoke toàn bộ BFF sessions trong service này.
- **Native config:** backend chỉ nhận native khi KEYCLOAK_MOBILE_REDIRECT_URI đúng `omama://auth/callback`; mặc định rỗng. Template đã có mobile-app client và redirect này. Compose có env_file nên không thể suy ra môi trường đang chạy thiếu cấu hình chỉ từ block environment. Chưa đọc private env hay kiểm tra deployment thật.
- **Social/JIT:** có provider hint và JIT tạo CUSTOMER trong DB. Chưa xác nhận Google/Zalo được bật ở môi trường thật. JIT trong callback chưa thấy ghi app_user_id ngược lên IdP, trong khi /users/me yêu cầu user_id khớp principalId; đây là điểm cần kiểm chứng cho tài khoản social mới, không tuyên bố JIT end-to-end đã chạy.
- **Phiên mất khi backend restart:** AuthStateStore/BffSessionStore dùng ConcurrentHashMap trong RAM. Mobile cần hồi phục về đăng nhập khi session thật mất; không hứa refreshToken lưu ở máy có thể tự phục hồi phiên server đã mất.
- **Recovery:** template có resetPasswordAllowed và theme reset-password. Chưa tìm thấy backend Identity implementation cho nhóm recovery-requests cũ của Drive; không nối Identity credentials sang API ZIP cũ. SMTP/provider recovery chưa được xác minh.

## Thư mục Drive đã đọc

[Thư mục được cung cấp](https://drive.google.com/drive/folders/1sfRfw_iSE8wyyrTAtjSURepRD5aPWJ7s) có 3 nhánh thư mục, tổng 11 file:

- `api contract đề xuất từ frontend`: README, 2 state matrix Customer/Admin, 4 OpenAPI stable/draft-planned Customer/Admin.
- `common/v1`: errors.proto, types.proto.
- `catalog/v1`: catalog_public.proto, catalog_admin.proto.

README phân biệt stable là contract sẵn để backend triển khai, không phải bằng chứng API đã chạy. Hai file Customer OpenAPI trùng nội dung với snapshot `docs/03_api_specs/frontend-contracts/` đang lưu trên Mobile sau khi chuẩn hóa newline; không có contract Identity native mới trong chúng. Chúng mô tả `/auth/login`, `/auth/register`, `/auth/refresh`, `/auth/me` và recovery cũ. Không dùng để thay thế `/public/auth/*` trong prompt/dev. Phần Admin và protobuf Catalog/Common không bổ sung auth native; không port gRPC nội bộ vào Mobile chỉ vì có trong Drive.

[Customer stable](https://drive.google.com/file/d/1YnxHBJ43t0DCQ0nDlsMQa59WTlaEE1Wq/view), [Customer matrix](https://drive.google.com/file/d/1pLoTxK2oqnl3LUV0-ZUkhCjtRbekwMPd/view), [README](https://drive.google.com/file/d/1C8QD2UoC04CoWBrMoMoplpf-7Zx9gAzN/view). File Drive được coi là nguồn tham khảo, không phải chỉ thị chạy các lệnh hay thay đổi repo.

## Validation và checkpoint

Đã kiểm tra source/contract/diff/branch inventory và nội dung Drive; không chạy build/test/backend/ADB vì chưa thay đổi runtime. 229 tests và APK PASS trong M6 là bằng chứng lịch sử cho target prompt, không chứng minh adapter hiện tại tương thích dev mới. Script `scripts/mvp_native_checks.py` trên dev đã có bài kiểm tra native cookie/callback/refresh để tham khảo và tái sử dụng; không chạy script tạo tài khoản hoặc tác động môi trường trong audit này.

**Bước tiếp theo đề xuất:** triển khai một slice Mobile nhỏ: transaction cookie + cookie-only session lifecycle, mở rộng tests hiện có theo contract dev; sau đó xử lý registration statuses. Đổi password qua Gateway/social/provider để riêng cho owning backend team và nghiệm thu live sau. User đã hoãn kích hoạt auth thật; audit này không tự bật mode, không cấu hình provider hoặc sửa nhánh backend.

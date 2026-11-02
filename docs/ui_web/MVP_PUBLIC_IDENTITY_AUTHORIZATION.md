# MVP Public Frontend — Identity và Authorization

Tài liệu này chốt Giai đoạn 8 cho Web Customer. Nó mô tả trust boundary của frontend/BFF; backend/gateway vẫn là authority xác thực token, ownership và quyền nghiệp vụ.

## 1. Nguồn đối chiếu

| Nguồn | Mốc |
| --- | --- |
| Identity source/docs | `feature/identity-service@bbb78b05b6dd4a142f009bf9f2f1cae5f7a6e9e8` |
| Public customer baseline | `2d7737b2b6112456cec3cdf9090833ceb38fcc3e` |
| Frontend implementation | `feature/ui-web` |

Identity quyết định OIDC/JWT được xác minh tại Gateway. Web hiện vẫn bọc auth runtime qua same-origin BFF và cookie HttpOnly; việc thay login proxy hiện tại bằng Authorization Code + PKCE cần contract/runtime riêng, không được frontend tự suy diễn trong giai đoạn này.

## 2. Trust boundary

Luồng request bảo vệ:

`Browser → same-origin BFF với OMAMA_SESSION → Identity xác minh principal → capability/permission backend → owning service`

Quy tắc bắt buộc:

- BFF không đọc hoặc chuyển tiếp `Authorization`, `X-User-ID`, `X-Role`, `X-Permission` do browser gửi.
- BFF chỉ đọc cookie HttpOnly `OMAMA_SESSION`; không nhận token, Role hoặc Permission do browser tự khai báo.
- BFF dùng cookie này để gọi `/public/auth/session` và `/users/me` của Identity. Response không hợp lệ, timeout hoặc provider lỗi đều fail closed.
- Customer BFF và Employee BFF xác minh lại `principalType` trước khi gọi owning service; UI redirect không phải kiểm soát bảo mật.
- Browser chỉ nhận projection cần cho UI; không nhận access/refresh token.

## 3. Principal và phân tách tài khoản

Session frontend tách các khái niệm sau:

| Trường | Ý nghĩa |
| --- | --- |
| `principalId` | ID định danh đã được Identity xác minh. |
| `principalType` | `CUSTOMER` hoặc `EMPLOYEE`; không phải permission. |
| `roles` | Nhóm nhiệm vụ được backend gán, có thể kèm scope. |
| `permissions` | Quyền nhân viên hiệu lực để điều khiển UI staff; backend vẫn kiểm tra lại. |
| `customerId` | Định danh Customer domain, chỉ tồn tại với principal `CUSTOMER`. |
| `customerAccess` | Chỉ `true` với tài khoản `CUSTOMER` đã xác minh. |

Quy chuẩn Phân lập Danh tính Doanh nghiệp (Enterprise Persona Separation):

- **Tách biệt tài khoản chuẩn hóa:** Nhân viên khi mua sắm cá nhân sử dụng tài khoản `CUSTOMER` độc lập (email cá nhân).
- **Tài khoản `EMPLOYEE` gọi API Customer:** luôn trả `403 EMPLOYEE_PURCHASE_NOT_ALLOWED`, kể cả payload cũ có `customer_id` hoặc role `CUSTOMER`.
- **Tài khoản `CUSTOMER` gọi API Employee:** luôn trả `403 STAFF_ACCESS_REQUIRED`.
- Employee không được tạo Cart, Checkout, Customer Order hoặc lưu Address; khi mua sắm cá nhân phải đăng nhập bằng tài khoản `CUSTOMER` độc lập.
- **Chính sách Offboarding:** Khi vô hiệu hóa tài khoản `EMPLOYEE`, tài khoản `CUSTOMER` cá nhân của họ hoàn toàn không bị ảnh hưởng.

## 4. Ma trận `401` và `403`

| Tình huống | HTTP | UI |
| --- | --- | --- |
| Không có access cookie | `401 AUTH_REQUIRED` | Chuyển tới đăng nhập với `returnUrl` nội bộ. |
| Token hết hạn/không hợp lệ | `401 AUTH_REQUIRED` | Xóa trạng thái session trên UI và yêu cầu đăng nhập lại. |
| Principal bị backend từ chối | `403 PRINCIPAL_FORBIDDEN` | Access-denied, không lặp đăng nhập vô hạn. |
| Employee gọi Customer BFF | `403 EMPLOYEE_PURCHASE_NOT_ALLOWED` | Chuyển sang không gian Employee; không hiển thị Customer actions. |
| Customer gọi Employee BFF | `403 STAFF_ACCESS_REQUIRED` | Chuyển về storefront; không hiển thị menu nội bộ. |
| Thiếu Customer capability | `403 CAPABILITY_FORBIDDEN` | Ẩn action và chặn deep link/direct request. |
| Identity/capability service lỗi | `503 IDENTITY_UNAVAILABLE` hoặc `CAPABILITY_UNAVAILABLE` | Khóa chức năng và cho phép thử lại read/session check. |

## 5. Đồng bộ UI và backend

- `hasRole` và `hasPermission` chỉ dùng projection principal đã xác minh để dựng navigation/staff UI.
- `hasCapability` chỉ dùng Customer capability projection sau khi `customerAccess=true`.
- UI ẩn action không phải là kiểm soát bảo mật.
- BFF gọi lại Identity/mapping/capability cho Cart, Checkout, Order, Payment, Profile/Address, B2B, Return và các Customer mutation được bảo vệ.
- Owning service vẫn phải kiểm tra ownership, scope, version, idempotency và policy động; BFF không thay thế các kiểm tra đó.
- Feature gate của Giai đoạn 9 chỉ kiểm soát phát hành capability và luôn đứng ngoài quyết định identity/authorization.

## 6. Kết quả task

| ID | Kết quả |
| --- | --- |
| `FE-AUTH-01` | Hoàn tất: chỉ dùng cookie HttpOnly `OMAMA_SESSION` để xác minh qua Identity; loại bỏ browser Authorization/Role/Permission tự khai báo. |
| `FE-AUTH-02` | Hoàn tất: session/guard/BFF phân biệt `401` và `403`. |
| `FE-AUTH-03` | Hoàn tất: Customer và Employee là hai principal tách biệt; Employee luôn bị chặn khỏi Customer BFF. |
| `FE-AUTH-04` | Hoàn tất: role/permission/capability điều khiển UI, BFF kiểm tra lại; owning service vẫn là authority cuối. |

## 7. Acceptance còn cần backend

- Gateway xác minh issuer, audience, signature, expiry và JWKS; frontend không tự giải mã token để cấp quyền.
- `/users/me` phải trả `principalId`, `principalType`, Role và Permission hiệu lực; frontend không suy đoán loại tài khoản từ Role.
- Tài khoản Customer và Employee runtime thật phải được kiểm tra theo cả hai chiều từ chối chéo.
- Permission/scope nhân viên phải được backend áp dụng lại trên mọi Employee API; frontend dùng same-origin `/api/admin/**` và layout `/admin/**`.
- Luồng OIDC Authorization Code + PKCE và refresh/revocation cần contract/runtime acceptance riêng trước production.

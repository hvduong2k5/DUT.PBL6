# MVP Public Frontend — Identity và Authorization

Tài liệu này chốt Giai đoạn 8 cho Web Customer. Nó mô tả trust boundary của frontend/BFF; backend/gateway vẫn là authority xác thực token, ownership và quyền nghiệp vụ.

## 1. Nguồn đối chiếu

| Nguồn | Mốc |
| --- | --- |
| Identity source/docs | `feature/identity-service@bbb78b05b6dd4a142f009bf9f2f1cae5f7a6e9e8` |
| Public customer baseline | `integration/mvp-contracts@9416090bb476406b4928f376c594cc1748671c49` |
| Frontend implementation | `feature/ui-web` |

Identity quyết định OIDC/JWT được xác minh tại Gateway. Web hiện vẫn bọc auth runtime qua same-origin BFF và cookie HttpOnly; việc thay login proxy hiện tại bằng Authorization Code + PKCE cần contract/runtime riêng, không được frontend tự suy diễn trong giai đoạn này.

## 2. Trust boundary

Luồng request bảo vệ:

`Browser → same-origin BFF → token từ HttpOnly cookie → Gateway/Identity xác minh → capability/ownership backend → owning service`

Quy tắc bắt buộc:

- BFF không đọc hoặc chuyển tiếp `Authorization`, `X-User-ID`, `X-Role`, `X-Permission` do browser gửi.
- BFF không chuyển nguyên header `Cookie` sang service khác; refresh token và cookie nội bộ không được rò sang provider.
- Access token chỉ được đọc server-side từ `oma_access_token` và chuyển thành Bearer cho gateway/provider được cấu hình.
- `/auth/me` phải chấp nhận token đã xác minh và trả principal projection. Response không hợp lệ, timeout hoặc provider lỗi đều fail closed.
- Browser chỉ nhận projection cần cho UI; không nhận access/refresh token.

## 3. Principal và Customer mapping

Session frontend tách các khái niệm sau:

| Trường | Ý nghĩa |
| --- | --- |
| `principalId` | ID định danh đã được Identity xác minh. |
| `principalType` | `CUSTOMER` hoặc `EMPLOYEE`; không phải permission. |
| `roles` | Nhóm nhiệm vụ được backend gán, có thể kèm scope. |
| `permissions` | Quyền nhân viên hiệu lực để điều khiển UI staff; backend vẫn kiểm tra lại. |
| `customerId` | Mapping tới Customer domain do backend trả. |
| `customerAccess` | Chỉ `true` khi có `customerId` đã xác minh. |

Quy tắc Employee kế thừa Customer:

- `CUSTOMER` có `customer_id`: được kiểm tra Customer capability bình thường.
- `EMPLOYEE` có `customer_id`: được dùng Customer feature tương ứng; backend vẫn kiểm tra ownership/capability.
- `EMPLOYEE` không có `customer_id`: session vẫn authenticated nhưng Customer feature trả `403 CUSTOMER_MAPPING_REQUIRED`.
- Chỉ có role `CUSTOMER`, hoặc role/permission do browser gửi, không tạo ra mapping Customer.

## 4. Ma trận `401` và `403`

| Tình huống | HTTP | UI |
| --- | --- | --- |
| Không có access cookie | `401 AUTH_REQUIRED` | Chuyển tới đăng nhập với `returnUrl` nội bộ. |
| Token hết hạn/không hợp lệ | `401 AUTH_REQUIRED` | Xóa trạng thái session trên UI và yêu cầu đăng nhập lại. |
| Principal bị backend từ chối | `403 PRINCIPAL_FORBIDDEN` | Access-denied, không lặp đăng nhập vô hạn. |
| Employee chưa có Customer mapping | `403 CUSTOMER_MAPPING_REQUIRED` | Giải thích chưa có hồ sơ Customer; không hiển thị Customer actions. |
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
| `FE-AUTH-01` | Hoàn tất: chỉ forward token server-side; loại bỏ browser Authorization/cookie forwarding. |
| `FE-AUTH-02` | Hoàn tất: session/guard/BFF phân biệt `401` và `403`. |
| `FE-AUTH-03` | Hoàn tất phía frontend: Employee fail closed nếu backend chưa trả `customer_id`. Runtime backend mapping vẫn cần E2E acceptance. |
| `FE-AUTH-04` | Hoàn tất: role/permission/capability điều khiển UI, BFF kiểm tra lại; owning service vẫn là authority cuối. |

## 7. Acceptance còn cần backend

- Gateway xác minh issuer, audience, signature, expiry và JWKS; frontend không tự giải mã token để cấp quyền.
- `/auth/me` hoặc endpoint session thay thế phải trả `principal_id`, `principal_type`, role, permission và `customer_id` khi mapping tồn tại.
- Employee có/không có Customer mapping phải được kiểm tra bằng tài khoản runtime thật.
- Permission/scope nhân viên phải được backend áp dụng lại trên mọi `/admin` API sau khi Web Admin được gộp vào unified Web.
- Luồng OIDC Authorization Code + PKCE và refresh/revocation cần contract/runtime acceptance riêng trước production.

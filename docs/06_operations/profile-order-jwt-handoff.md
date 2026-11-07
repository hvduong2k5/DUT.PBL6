# Handoff IAM: phần 3–4

Trạng thái: **contract tạm để phát triển consumer** theo lựa chọn 1K-A, 1H-A,
2-A, 3-A, 4-B + 4-A, 5-B, 6-B + 6-C và 7-B. Chưa phải xác nhận task 1–2
đã phát hành token hay API projection tương ứng. Không triển khai consumer
staff này lên dev chung trước khi issuer phát hành đúng contract.

## JWT gửi đến Profile và Order

Schema: [staff-token.v1.schema.json](../../packages/contracts/iam/staff-token.v1.schema.json).
Ví dụ **payload tổng hợp, không phải token đăng nhập**:

```json
{
  "iss": "https://identity.example.test/realms/omama",
  "aud": "omama-api", "sub": "idp-subject",
  "user_id": "11111111-1111-4111-8111-111111111111",
  "typ": "Bearer", "iat": 1794052800, "exp": 1794053700,
  "principal_type": "EMPLOYEE", "principal_status": "ACTIVE", "authz_version": 1,
  "permission_grants": [
    {"permissionCode": "EMPLOYEE_PROFILE_VIEW", "scopeType": "GLOBAL", "scopeId": null},
    {"permissionCode": "ORDER_VIEW", "scopeType": "WAREHOUSE", "scopeId": "HUE"}
  ]
}
```

- RS256, `kid`, issuer và audience cấu hình phải khớp; bắt buộc iat/exp, leeway
  token 5 giây. Tổng Authorization header tối đa 8192 byte, tối đa 128 grants.
- `ADMIN` là role của EMPLOYEE; không phải principal type và không bypass
  policy trong hai consumer. Issuer phải đưa permission–scope đã cấp vào token.
- `permission_grants` là array JSON, không phải chuỗi JSON. Mảng rỗng hợp lệ
  nhưng không mở API staff. Thiếu claim, sai version hoặc grant sai cấu trúc
  bị từ chối; không chuyển sang lấy quyền tươi từ `/users/me`.
- Mỗi grant giữ permission và scope **cùng nguồn cấp**. GLOBAL có scopeId
  null/rỗng/không có. Scope khác cần ID không rỗng, không khoảng trắng đầu/cuối,
  không ký tự điều khiển, tối đa 160 byte. Không kết hợp chéo role assignments.
- `validFrom`/`validUntil` tùy chọn, null hoặc RFC3339 có timezone. Khoảng hiệu lực
  là `[from, until)`; grant chưa đến hạn/hết hạn không cấp quyền. Khoảng sai
  hoặc trường grant không biết làm token không hợp lệ.
- HR MVP toàn doanh nghiệp: quyền EMPLOYEE_* cần GLOBAL. Department chỉ để
  phân loại/lọc, không tạo ranh giới phân quyền. Order nhận GLOBAL hoặc WAREHOUSE
  đúng ID kho đang lưu; MVP một kho dùng chung không tạo thêm kho.
- Credential service nội bộ giữ riêng. Claim `service` có giá trị không rỗng
  không được dùng làm staff token. Không nhận quyền từ header trình duyệt.

Task 1 sở hữu TTL chính thức, đồng bộ IAM → issuer, refresh và thu hồi session.
Không suy ra TTL sản phẩm từ ví dụ 15 phút ở trên. Token cũ giữ snapshot tới
exp hoặc validUntil sớm hơn; thay đổi role không vô hiệu hóa bearer cũ tức thì.
JWKS vẫn cần khi khởi động/đổi key; outage không tạo đường bypass.

Customer legacy vẫn kiểm persona qua Identity và resolve trạng thái Profile;
đây là phạm vi tương thích riêng, chưa chuyển sang contract staff. Identity BFF
hiện cũng kiểm principal ACTIVE trong DB trước proxy; do đó đường Web có thể
chặn account sớm hơn bearer trực tiếp. Task 1 cần thống nhất điều này cùng
session/refresh; không tuyên bố toàn hệ thống đã thuần stateless.
Menu và session UI hiện vẫn lấy projection `/api/auth/me` từ Identity; danh sách
quyền cấu hình tươi có thể ẩn menu trước khi bearer cũ hết hạn. Đây chỉ là UX,
không thay quyết định JWT của Order/Profile. Task 1–2 cần đưa snapshot quyền
của session vào projection phiên để UX và hạn token thống nhất; không suy ra
quyền API bằng cách ghép danh sách role/scope hiển thị trên UI.

Hai adapter cần sửa khi contract chính thức đổi:

- `services/profile-service/internal/infrastructure/security/staff_claims.go`
- `services/order-service/internal/app/staff_claims.go`

## Projection Order và đường đi Web

Web → `/api/admin/orders` → Identity `/api/v1/admin/order-api/orders` → Kong →
Order `/api/v1/admin/orders`, Accept `application/vnd.omama.admin-order+json`.
OpenAPI producer/proxy: [iam-consumer-read.openapi.yaml](../03_api_specs/iam-consumer-read.openapi.yaml).
Chỉ GET, opaque OMAMA_SESSION; Identity dùng bearer giữ phía server. Không đưa
token về JavaScript và không chuyển tiếp browser Authorization/header giả danh.
Web Admin dùng WEB_AUTH_MODE=identity-bff; standalone oidc chưa có adapter staff
tương ứng nên báo 503, không chuyển sang mock. Legacy mock chỉ cho phát triển local.
Web chuẩn hóa lỗi sang code/message/requestId/errors. Order trực tiếp vẫn giữ
legacy error envelope để tương thích, được khai báo riêng trong OpenAPI; chuẩn
hóa toàn bộ lỗi Order là việc tiếp theo, chưa thay ngầm contract cũ ở đây.

GET list/queue nhận `limit` 1..100, `status`, `cursor`; detail theo UUID.
List có `items`, `next_cursor`, `calculated_at`; detail có `order`, `lines`,
`history`, `subtotal`, `shipping_fee`, `discount`, `recipient`, `calculated_at`.
Recipient null nếu thiếu ORDER_SENSITIVE_VIEW ở kho của đơn. Lọc scope trước
LIMIT; cursor gắn actor + scope + filter + queue. Snapshot sản phẩm/người nhận
lấy từ Order; không dựng lại bằng Catalog/Profile hiện tại. Default API JSON
cũ vẫn giữ format; UI chủ động chọn projection bằng Accept.

UI chỉ thống kê trên các hàng đã tải; hỗ trợ tải tiếp. Trạng thái chưa biết
hiển thị UNKNOWN, không có deadline hiển thị chưa có SLA. Không giả số liệu,
deadline hoặc thành công khi upstream lỗi. Các thao tác rủi ro chưa có capability
vẫn đóng theo Care/Fulfillment/Finance hiện tại.

## Projection Access Management chờ task 2

Web consumer: `apps/web-user/src/features/employee/access/live.ts` và
`server/live-handler.ts`. Producer Identity **chưa có** sáu API projection này:

| GET dưới `/api/v1/admin/access` | Quyền producer phải kiểm | Data |
|---|---|---|
| `/employees` | EMPLOYEE_ACCOUNT_VIEW | EmployeeAccessList |
| `/employees/{id}` | EMPLOYEE_ACCOUNT_VIEW + ACCESS_REVIEW_VIEW | EmployeeAccessDetail |
| `/roles` | ROLE_VIEW | AdminRoleList |
| `/roles/{code}` | ROLE_VIEW | AdminRoleDetail |
| `/permissions` | PERMISSION_VIEW | AdminPermissionList |
| `/permissions/{code}` | PERMISSION_VIEW | AdminPermissionDetail |

Accept: `application/vnd.omama.iam-admin-read.v1+json`. Envelope bắt buộc:

```json
{"contractVersion":"iam-admin-read.v1","capabilities":{"accessReview":false},"data":{"items":[],"statusCounts":[],"calculatedAt":"2026-11-07T12:00:00Z"}}
```

Ví dụ là **role list**. Data đầy đủ theo các interface trong
`apps/web-user/src/features/employee/access/types.ts`; runtime decoder liệt kê
từng trường bắt buộc, chỉ xuất các trường contract, không phát sinh ngày/count
thay thế. Array tối đa 1000 phần tử, text tối đa 4096 ký tự; lớn hơn phải bàn
pagination/version mới. Không query/mutation trong v1. Producer phải redaction
review fields trước khi trả; consumer còn xóa effectivePermissionCodes và
permissions khỏi employee list nếu accessReview=false. Detail cần capability true.

Producer trả `X-IAM-Read-Contract: iam-admin-read.v1` kể cả 404 tài nguyên.
404 không có marker được xem là producer chưa sẵn sàng (503 phía Web). Sai
version/thiếu trường/enum lạ trả 502; không fallback sang mock. Quyền UI không
thay kiểm tra quyền producer. Chưa alias USER_VIEW thành EMPLOYEE_ACCOUNT_VIEW;
task 2 cần đăng ký/quyết định mã EMPLOYEE_ACCOUNT_VIEW và ACCESS_REVIEW_VIEW.

Trạng thái UI ACTIVE/INACTIVE/ARCHIVED và INVITED/PENDING_ACTIVATION/ACTIVE/
LOCKED/DISABLED cần projection có nghĩa nghiệp vụ từ task 2, không map tùy tiện
boolean `is_enabled` hay provisioning status. Quyền hiển thị là **cấu hình
hiện tại**, không phải quyền trên tất cả JWT chưa hết hạn. API CRUD cũ không
đủ assignment ID, nguồn quyền, impact và lifecycle để thay projection này.

## Ranh giới bàn giao và nghiệm thu

Phần này không sửa phát token, gán quyền, bảo vệ Admin cuối cùng, archive/delete
role hay MFA của task 1–2. Cần issuer chính thức + sáu projection + cấu hình
gateway/session, rồi chạy lại login → Profile/Order → Web bằng token thực.
Fixture có thể phát claims tổng hợp bằng mapper trong realm dùng một lần để
kiểm consumer; đó không chứng minh IAM → Keycloak sync đã hoàn thành.

Kiểm bắt buộc: token Customer/legacy employee không mở staff; ADMIN không bypass;
scope không ghép chéo; HR GLOBAL; che PII trước giải mã; cursor không đổi scope;
quyền cũ còn trên token cũ, token mới không có quyền bị từ chối; upstream lỗi
không hiển thị mock. Audit/approval nghiệp vụ đang có vẫn giữ nguyên; chưa có
pipeline audit thống nhất cho mọi quyết định allow/deny ở tất cả service.

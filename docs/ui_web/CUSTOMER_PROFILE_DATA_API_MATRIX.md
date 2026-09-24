# Customer Profile & Address Book — Data & API Matrix

> Trạng thái: Baseline v0.2 — OpenAPI candidate proposed; chờ Backend/Architecture/Security review
> Ngày lập: 2026-09-25
> Liên quan: `CUSTOMER_PROFILE_SCOPE_TRACEABILITY.md`, `../03_api_specs/customer-profile-mvp.openapi.yaml`

## 1. UI data inventory

### D2C-012 — Customer Profile

| Field | Yêu cầu | Ghi chú |
| --- | --- | --- |
| `customerId` | Server-only, read-only | Lấy từ session; client không được truyền ID Customer khác. |
| `fullName` | Bắt buộc, 2–100 ký tự | Không chỉ chứa khoảng trắng. |
| `email` | Bắt buộc, read-only | Đổi email ngoài scope. |
| `phone` | Tùy chọn, 8–15 chữ số sau normalize | UI cho phép khoảng trắng, `.`, `-`, `(`, `)` và `+` khi nhập. |
| `dateOfBirth` | Tùy chọn, `YYYY-MM-DD` | Không ở tương lai. |
| `gender` | Tùy chọn | `MALE/FEMALE/OTHER/PREFER_NOT_TO_SAY`. |
| `version` | Server-only | Dùng optimistic concurrency; update gửi `If-Match`. |
| `updatedAt` | Server-only | Hiển thị thời điểm cập nhật gần nhất. |

### D2C-013 — Address Book

| Field | Yêu cầu | Ghi chú |
| --- | --- | --- |
| `id` | Server-only | Opaque ID thuộc current Customer. |
| `label` | Bắt buộc, 1–50 ký tự | Ví dụ “Nhà riêng”, không phải tên người nhận. |
| `type` | Bắt buộc | `HOME/OFFICE/GIFT/OTHER`. |
| `recipientName` | Bắt buộc, 2–100 ký tự | Có thể khác tên profile. |
| `recipientPhone` | Bắt buộc | Validation phone như profile. |
| `provinceCode/name` | Bắt buộc | Code theo danh mục thống nhất với Checkout/Shipping. |
| `districtCode/name` | Tùy thuộc nguồn địa giới/provider | Candidate hỗ trợ để tương thích provider; không dùng làm identifier duy nhất. |
| `wardCode/name` | Bắt buộc | Code theo danh mục thống nhất. |
| `addressLine` | Bắt buộc, 5–200 ký tự | Số nhà, đường, tòa nhà… |
| `deliveryNote` | Tùy chọn, tối đa 500 ký tự | Không chứa chỉ dẫn bí mật/credential. |
| `isDefault` | Server-controlled | Tối đa một địa chỉ mặc định. |
| `version`, `updatedAt` | Server-only | Concurrency và hiển thị. |

## 2. API capability matrix

| ID | UI action | Candidate endpoint | Response chính |
| --- | --- | --- | --- |
| `PROFILE-C01` | Xem profile | `GET /api/v1/customers/me/profile` | `CustomerProfile` + `ETag` |
| `PROFILE-C02` | Cập nhật profile | `PATCH /api/v1/customers/me/profile` | Profile mới + `ETag` |
| `ADDRESS-C01` | Xem Address Book | `GET /api/v1/customers/me/addresses` | `items[]`, `total` |
| `ADDRESS-C02` | Thêm địa chỉ | `POST /api/v1/customers/me/addresses` | Address đã tạo |
| `ADDRESS-C03` | Sửa địa chỉ | `PATCH /api/v1/customers/me/addresses/{addressId}` | Address mới |
| `ADDRESS-C04` | Xóa địa chỉ | `DELETE /api/v1/customers/me/addresses/{addressId}` | Danh sách còn lại để đồng bộ default |
| `ADDRESS-C05` | Đặt mặc định | `PUT /api/v1/customers/me/addresses/{addressId}/default` | Danh sách mới với đúng một default |

Không có `customerId` trong path vì browser chỉ thao tác trên tài nguyên `me`. Backend vẫn phải derive subject từ session và kiểm tra ownership của `addressId`.

## 3. Candidate shapes

```json
{
  "id": "CUS-001",
  "fullName": "Tôn Thất Hoàng",
  "email": "hoang.tonthat@example.com",
  "emailVerified": true,
  "phone": "+84912345892",
  "dateOfBirth": "1985-11-18",
  "gender": "MALE",
  "version": 3,
  "updatedAt": "2026-09-25T08:30:00Z"
}
```

```json
{
  "id": "ADDR-001",
  "label": "Nhà riêng",
  "type": "HOME",
  "recipientName": "Tôn Thất Hoàng",
  "recipientPhone": "+84912345892",
  "province": { "code": "VN-TTH", "name": "Thành phố Huế" },
  "district": { "code": "HUE-CENTER", "name": "Khu vực trung tâm Huế" },
  "ward": { "code": "PHU-HOI", "name": "Phường Phú Hội" },
  "addressLine": "54 Lê Lợi",
  "deliveryNote": "Gọi trước khi giao.",
  "isDefault": true,
  "version": 2,
  "updatedAt": "2026-09-25T08:30:00Z"
}
```

## 4. Stable error codes UI cần phân biệt

| Code | HTTP candidate | UI behavior |
| --- | --- | --- |
| `VALIDATION_ERROR` | 422 | Map `errors[]` theo field và focus field đầu tiên. |
| `UNAUTHENTICATED` | 401 | Điều hướng Login với `returnUrl` nội bộ. |
| `FORBIDDEN` | 403 | Không hiển thị dữ liệu, hướng dẫn đăng nhập đúng tài khoản/hỗ trợ. |
| `ADDRESS_NOT_FOUND` | 404 | Gỡ item stale hoặc tải lại danh sách. |
| `VERSION_CONFLICT` | 409 | Báo dữ liệu vừa thay đổi và cho tải lại. |
| `ADDRESS_LIMIT_REACHED` | 409 | Không mở form thêm; hiển thị hướng dẫn xóa địa chỉ không dùng. Giới hạn cuối do Backend/Product chốt. |
| `DEFAULT_ADDRESS_REQUIRED` | 409 | Tải lại để phục hồi invariant; candidate API hiện tự chọn default thay thế. |
| `SERVICE_UNAVAILABLE` | 503 | Giữ dữ liệu form và cho retry. |

Error envelope dùng cùng cấu trúc Authentication MVP: `code`, `message`, `errors[]`, `requestId`.

## 5. Mock scenario matrix

| Capability | Scenario | Kết quả |
| --- | --- | --- |
| Profile load | `profile-empty-phone`, `profile-unauthenticated`, `profile-server-error`, `profile-slow` | Optional state, auth, retry, loading |
| Profile update | `profile-validation`, `profile-conflict`, `profile-server-error`, `profile-update-slow` | Field error, concurrency, retry, duplicate submit |
| Address list | `addresses-empty`, `addresses-unauthenticated`, `addresses-server-error`, `addresses-slow` | Empty/auth/retry/loading |
| Address create/update | `address-validation`, `address-limit`, `address-not-found`, `address-conflict`, `address-save-slow` | Form/error/concurrency |
| Address delete/default | `address-not-found`, `address-conflict`, `address-action-slow` | Stale/action loading |

`X-Mock-Scenario` chỉ dùng local/test và không được gửi tới production Gateway.

## 6. Backend/Architecture review points

- Profile thuộc Auth/Identity service hay Customer/Profile service và ownership dữ liệu giữa hai service.
- Cách đổi email/phone và trạng thái verified; không đưa vào contract slice này nếu chưa chốt.
- Chuẩn code đơn vị hành chính và compatibility với 3PL.
- Chính sách giới hạn số địa chỉ.
- `ETag/If-Match` hay version trong body cho optimistic concurrency.
- Audit/PII encryption, retention và data erasure policy.
- Checkout lấy Address trực tiếp hay nhận snapshot qua BFF/order command.

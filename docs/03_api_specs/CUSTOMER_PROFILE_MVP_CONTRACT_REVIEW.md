# Customer Profile & Address Book MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
> Phiên bản: `0.1.0-candidate`
> Ngày đề xuất: 2026-09-25
> OpenAPI: `customer-profile-mvp.openapi.yaml`

## 1. Điểm dừng của workflow

Artefact này hoàn thành bước **đề xuất API Contract từ UI đã được duyệt và implementation thực tế**. Nó chưa phải cam kết production của Backend/Architecture/Security.

Chưa thực hiện:

- Review và phê duyệt liên team.
- Backend persistence, authorization, PII protection và audit.
- Đồng bộ danh mục địa giới với Checkout/Shipping/3PL.
- Contract/E2E test với API Gateway thật.
- Chuyển frontend khỏi Mockoon.

## 2. Nguồn hình thành contract

- `docs/01_requirements/epics/EPIC_02_Customer_Profile.md` trên nhánh `doc`: `US-USER-01/02`, `FR-USER-01..05/10`.
- `docs/ui_web/CUSTOMER_PROFILE_SCOPE_TRACEABILITY.md`: phạm vi và quyết định MVP.
- `docs/ui_web/CUSTOMER_PROFILE_DATA_API_MATRIX.md`: field, capability và error semantics.
- D2C-012/D2C-013: visual reference đã được đối chiếu và UI đã được người dùng duyệt.
- `mocks/mockoon/oma-customer-profile-mvp.json`: executable examples.
- `apps/web-user/src/services/customer-service.ts` và `src/lib/customer/*`: payload UI thực sự gửi/đọc.

## 3. Ranh giới Browser, BFF và Gateway

| Lớp | Path | Trách nhiệm |
| --- | --- | --- |
| Browser → Next BFF | `/api/customer/*` | Same-origin, Origin check, ẩn upstream host. |
| BFF → API Gateway | `/api/v1/customers/me/*` | Contract candidate trong OpenAPI. |
| Mock local | `X-Mock-Scenario` | Development/test only, không thuộc production contract. |

`customerId` không xuất hiện trong browser path/body. Backend derive Customer từ session. `addressId` không thuộc Customer hiện tại phải trả giống địa chỉ không tồn tại để không lộ ownership.

## 4. Capability đề xuất

| ID | Method và path | Success |
| --- | --- | --- |
| `PROFILE-C01` | `GET /customers/me/profile` | `200 CustomerProfile` + `ETag` |
| `PROFILE-C02` | `PATCH /customers/me/profile` | `200 CustomerProfile` + ETag mới |
| `ADDRESS-C01` | `GET /customers/me/addresses` | `200 AddressList` |
| `ADDRESS-C02` | `POST /customers/me/addresses` | `201 CustomerAddress` + `Location/ETag` |
| `ADDRESS-C03` | `PATCH /customers/me/addresses/{addressId}` | `200 CustomerAddress` + ETag mới |
| `ADDRESS-C04` | `DELETE /customers/me/addresses/{addressId}` | `200 AddressList` đã đồng bộ default |
| `ADDRESS-C05` | `PUT /customers/me/addresses/{addressId}/default` | `200 AddressList` với đúng một default |

## 5. Quyết định phản ánh trong candidate

- Email là dữ liệu chỉ đọc; đổi email/re-verification không được lách qua Profile API.
- `fullName` bắt buộc; phone, birthday và gender có thể `null`.
- Profile/address dùng `ETag` và `If-Match` để tránh lost update.
- Phone gửi sau khi loại separator; Backend vẫn chịu trách nhiệm normalize và validate lại.
- Address có code và display-name snapshot cho đơn vị hành chính.
- Địa chỉ đầu tiên tự trở thành default; tối đa một default.
- Đổi default là idempotent và nguyên tử.
- Xóa default trả lại danh sách đã có default thay thế nếu vẫn còn địa chỉ.
- Order/Checkout phải sao chép snapshot; sửa/xóa Address Book không thay đổi đơn đã tạo.
- `HOME/OFFICE/GIFT/OTHER` chỉ là nhãn UI, không quyết định serviceability hay phí ship.

## 6. Phát hiện từ implementation thực tế

### Field UI đang dùng

- Profile đọc `id/fullName/email/emailVerified/phone/dateOfBirth/gender/version/updatedAt`.
- Profile update chỉ gửi bốn field editable; không gửi email hoặc customer ID.
- Address form gửi toàn bộ recipient, area, address line, note và default intent.
- UI dùng `version` để tạo `If-Match`; map lỗi theo `errors[].field`.
- Mutation Address Book cập nhật client state ngay sau response vì Mockoon là stateless.

### Gap cần xử lý sau review

| Gap | Hiện trạng | Hướng xử lý |
| --- | --- | --- |
| Auth guard | BFF nhận cookie nhưng Mockoon chưa xác minh session thật. | Gateway/backend enforce `cookieAuth`; E2E 401/ownership. |
| Administrative areas | UI chỉ có sample Huế/Đà Nẵng để chứng minh form. | Chốt catalog/provider API chung với EPIC 06/11. |
| Address limit | Có error code nhưng chưa có số giới hạn. | Product/Backend chốt trước Approved. |
| Email/phone verification | Email read-only; phone chưa có verified state. | Thiết kế flow riêng với Auth/Notification nếu cần. |
| Optimistic concurrency | BFF forward `If-Match`; Mock không thực sự so ETag. | Backend implement compare-and-update và contract test 409. |
| Delete default selection | Candidate chọn address còn lại cập nhật gần nhất. | Backend/Product xác nhận thứ tự chính xác. |

## 7. Sai khác có chủ đích so với Mockoon

- OpenAPI không chứa `/health` và `X-Mock-Scenario`.
- Mockoon trả fixture cố định sau create/update; UI merge input để mô phỏng state. Production phải echo canonical persisted representation.
- Mockoon không kiểm tra cookie, ownership, ETag hoặc persistence; đó là nghĩa vụ Backend/Gateway.
- Candidate yêu cầu `If-Match` cho update/delete; mock chỉ nhận header và không đánh giá.
- OpenAPI cho `district=null` để không khóa contract vào duy nhất một mô hình địa giới/provider.

## 8. Câu hỏi cần Backend/Architecture/Security chốt

| ID | Câu hỏi | Candidate hiện tại |
| --- | --- | --- |
| `PROFILE-CR-01` | Profile data thuộc Auth Service hay Customer/Profile Service? | Gateway public contract không lộ service ownership nội bộ. |
| `PROFILE-CR-02` | Có giữ `/customers/me` hay Gateway map sang subject khác? | Giữ `/customers/me` để browser không truyền customer ID. |
| `PROFILE-CR-03` | `ETag/If-Match` hay version trong body? | ETag/If-Match; `version` vẫn trả để debug/UI state. |
| `PROFILE-CR-04` | Phone canonical format và quốc gia mặc định? | Nhận `+` quốc tế hoặc local VN; Backend canonicalize. |
| `PROFILE-CR-05` | Nguồn code/tên địa giới là master-data nội bộ hay 3PL? | Chưa chốt; contract giữ code + name snapshot. |
| `PROFILE-CR-06` | Giới hạn địa chỉ mỗi Customer? | Có `ADDRESS_LIMIT_REACHED`, chưa chốt con số. |
| `PROFILE-CR-07` | Thuật toán chọn default sau khi xóa default? | Địa chỉ còn lại cập nhật gần nhất. |
| `PROFILE-CR-08` | PATCH có nhận partial fields hay toàn bộ editable representation? | Candidate yêu cầu đủ bốn field profile và toàn bộ Address write fields. |
| `PROFILE-CR-09` | Có audit mọi thay đổi PII/address hay chỉ field nhạy cảm? | Audit policy cần EPIC 23 xác nhận. |
| `PROFILE-CR-10` | PII encryption, log redaction, retention/erasure? | Bắt buộc Architecture/Security/Data review. |
| `PROFILE-CR-11` | CSRF policy ngoài same-origin Origin check? | Chưa chốt; dùng chung quyết định Auth contract. |
| `PROFILE-CR-12` | Checkout đọc Address bằng API hay nhận snapshot trong order command? | Address Book cung cấp dữ liệu; EPIC 06 chịu trách nhiệm snapshot. |

## 9. Tiêu chí đổi sang Approved

- Backend xác nhận ownership, persistence, route/schema/status và default invariant.
- Architecture xác nhận service boundary, concurrency và Address → Checkout snapshot.
- Security/Data xác nhận authorization, CSRF, audit, PII, retention/erasure.
- Checkout/Shipping xác nhận administrative-area codes và provider compatibility.
- Product chốt address limit, phone verification và delete-default policy.
- Frontend types, Mockoon và generated client được đồng bộ theo contract đã review.
- Contract tests chứng minh 401, cross-customer isolation, 409 concurrency và CRUD/default invariant trên Gateway thật.

Cho đến khi hoàn tất, version phải giữ hậu tố `-candidate` và trạng thái **PROPOSED / NOT APPROVED**.

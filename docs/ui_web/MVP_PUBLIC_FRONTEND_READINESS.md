# MVP Public Frontend — Readiness handoff

Tài liệu này là điểm bàn giao runtime cho Web Customer sau khi đồng bộ public contract. Backend và frontend dùng nó cùng contract được pin; không dùng các Mockoon JSON cũ làm nguồn chuẩn.

## 1. Baseline và kết quả

| Mục | Giá trị |
| --- | --- |
| Release | `mvp-2026-10-26` |
| Contract commit | `9416090bb476406b4928f376c594cc1748671c49` |
| Public OpenAPI | `docs/03_api_specs/mvp-customer.openapi.yaml` tại commit trên |
| Trạng thái contract | `integration-target` |
| Tổng operation | 30 |
| Frontend `ALIGNED` | 26 |
| Frontend `GATED` | 4 |
| Runtime acceptance | `PENDING` |

Chi tiết ánh xạ từng operation nằm trong [`MVP_PUBLIC_CONTRACT_MIGRATION_MATRIX.md`](./MVP_PUBLIC_CONTRACT_MIGRATION_MATRIX.md).

## 2. Luồng MVP được frontend hỗ trợ

Luồng chuẩn là:

`Catalog/Product → Cart → Checkout quote → Checkout accept → Operation polling → Owned Order → VietQR Payment`

Các nguyên tắc runtime:

- Browser gọi Next.js BFF cùng origin; BFF gọi public API gateway bằng credential đã xác minh.
- DTO wire `snake_case` dừng tại contract boundary; component chỉ dùng view model.
- Quote backend là nguồn giá cuối cùng. Frontend không tự tính phí vận chuyển, discount hoặc trạng thái tài chính.
- Mutation không tự retry. Một intent chưa rõ kết quả phải giữ nguyên canonical payload và idempotency key.
- Order, Payment, Inventory, Cancellation và Compensation là các projection độc lập; UI không suy một trạng thái từ trạng thái khác.

## 3. Capability gate

| Capability | Biến môi trường | Mặc định | Điều kiện được bật |
| --- | --- | --- | --- |
| COD | `NEXT_PUBLIC_MVP_COD_ENABLED` | `false` | Provider, policy và runtime acceptance của COD đã sẵn sàng. |
| Voucher | `NEXT_PUBLIC_MVP_VOUCHER_ENABLED` | `false` | Quote provider chấp nhận voucher và có acceptance evidence. |
| Guest Order | `NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED` | `false` | Guest session/payment grant public đã được triển khai và nghiệm thu. |

Gate chỉ kiểm soát việc phát hành capability. Authorization thực tế vẫn phải do backend kiểm tra. Với COD/Voucher, BFF Checkout fail closed ngay cả khi client gọi trực tiếp.

Gate không thay thế Identity. Một capability được bật vẫn yêu cầu principal hợp lệ, mapping Customer và capability/permission tương ứng. Chi tiết tại [`MVP_PUBLIC_IDENTITY_AUTHORIZATION.md`](./MVP_PUBLIC_IDENTITY_AUTHORIZATION.md).

Bốn public operation còn `GATED` là paid cancellation request, create guest session, issue guest payment access và revoke guest payment access.

## 4. Dependency runtime

| Dependency | Frontend cần | Trạng thái bàn giao |
| --- | --- | --- |
| Public API gateway | `CUSTOMER_CORE_UPSTREAM_URL`; cung cấp đúng public contract được pin. | Chờ backend runtime acceptance. |
| Identity/session | BFF giữ token trong cookie HttpOnly, xác minh principal qua backend/gateway và không tin identity header từ browser. | Frontend boundary đã hoàn tất; chờ E2E với runtime thật. |
| Customer capability projection | `CUSTOMER_EXTENSIONS_UPSTREAM_URL`; BFF hiện dùng để xác nhận capability Customer. | Cần duy trì cho đến khi capability được hợp nhất vào Identity/gateway. |
| Catalog/Profile/Order/Payment providers | DTO, ETag/version, idempotency và error mapping đúng contract. | Registry owning-service acceptance còn `PENDING`. |
| Locations | Danh sách tỉnh/thành và phường/xã phục vụ form địa chỉ. | Support API ngoài 30-operation target; backend phải công bố contract/runtime riêng. |

Không cấu hình trực tiếp URL của từng mock feature trong Web production. Mọi public operation phải đi qua gateway chuẩn.

## 5. Ma trận trạng thái bắt buộc

| Trạng thái | Contract/runtime signal | Hành vi frontend |
| --- | --- | --- |
| Loading | Request đang chờ hoặc operation chưa terminal. | Hiển thị loading/skeleton, khóa submit trùng; Checkout poll theo `Retry-After`. |
| Empty | List `items=[]`, Cart không có item hoặc dữ liệu optional vắng mặt. | Empty state có lối phục hồi; không biến dữ liệu thiếu thành `0`. |
| Error | `PublicError`, timeout/network hoặc response sai schema. | Hiển thị thông báo an toàn và retry chỉ cho read; mutation không tự retry. |
| Success | Response đúng schema và invariant. | Map sang view model; chỉ điều hướng khi operation đã `SUCCEEDED`. |
| Unauthorized | HTTP `401`. | Yêu cầu đăng nhập lại và bảo toàn `returnUrl` nội bộ hợp lệ. |
| Forbidden | HTTP `403`. | Không hiển thị hành động bị cấm; deep link/direct request vẫn bị BFF/backend từ chối. |
| Conflict/stale | HTTP `409` hoặc `412`. | Đọc lại snapshot/ETag/version canonical rồi yêu cầu người dùng xác nhận lại. |
| Accepted/async | HTTP `202`, `Location`, operation ID, `Retry-After`. | Hiển thị processing; poll operation; không coi `202` là Order đã hoàn tất. |
| Manual review | Checkout operation `MANUAL_REVIEW`. | Dừng poll tự động, giữ reference và hướng dẫn liên hệ/hành động tiếp theo. |
| Rate limited | HTTP `429`, có thể có `Retry-After`. | Thông báo chờ; không loop retry hoặc phát sinh mutation mới. |

Backend cần trả `PublicError` có `code`, `message`, `requestId`, `retryable` và `details` khi phù hợp để frontend phân loại các trạng thái trên. Không dùng tên scenario/mock trong response production.

## 6. Contract gaps không được tự suy diễn

- Order list có `next_cursor` trong response nhưng public operation chưa khai báo cursor request parameter; frontend chưa tự thêm query phân trang.
- Tracking response hiện chỉ là `object`; frontend chưa dựng carrier/checkpoint/timeline field không có schema.
- Locations chưa thuộc public 30-operation target.
- Guest public credential, paid cancellation, COD provider và voucher policy chưa có runtime acceptance.
- Contract CI PASS không chứng minh provider/client adapter hoặc live-service E2E đã PASS.

Các gap phải được sửa ở contract/registry hoặc được feature-gate; không vá bằng DTO riêng chỉ tồn tại ở frontend.

## 7. Mock và bằng chứng nghiệm thu

Các script `dev:cart`, `dev:checkout`, `dev:orders`, `dev:payment` và Mockoon theo feature được giữ để tham khảo/kiểm tra lát cắt cũ. Chúng chưa triển khai đồng nhất target 30-operation nên không phải acceptance environment.

Backend có thể cung cấp gateway thật hoặc một mock mới sinh từ public target. Dù dùng phương án nào, nghiệm thu cần chạy cùng một base URL và cùng schema/error semantics.

## 8. Checklist E2E khi backend sẵn sàng

- Đăng nhập, hết hạn session và principal thiếu quyền trả đúng `401`/`403`.
- Employee chỉ dùng Customer flow khi backend trả mapping `customer_id`; Employee không có mapping phải nhận `403` ở BFF và access-denied ở UI.
- Duyệt Catalog/Product, chọn SKU, thêm/sửa/xóa Cart và xử lý revision conflict.
- Tạo quote, nhận thay đổi giá/tồn, accept bằng cùng idempotency key và xử lý cả `201` lẫn `202`.
- Poll operation tới `SUCCEEDED`, `FAILED` và `MANUAL_REVIEW`; không tạo Order trùng.
- Xem list/detail/tracking Order thuộc owner; tài khoản khác không truy cập được.
- Cancel Order chưa thanh toán với `If-Match`; kiểm tra conflict và trạng thái compensation độc lập.
- Đọc VietQR instructions/status; kiểm tra expiry, polling interval, paid/reconciled/refunded projection.
- Xác nhận COD, Voucher, Guest Order và paid cancellation đều fail closed khi gate/runtime capability chưa được bật.
- Kiểm tra loading, empty, error, success, unauthorized, forbidden, conflict, async và rate-limit bằng response contract, không bằng tên scenario mock.

Chỉ sau checklist này và registry runtime acceptance, target mới được coi là sẵn sàng phát hành end-to-end.

## 9. Frontend verification

Chạy tại `apps/web-user`:

```text
npm run verify:mvp
```

Kết quả local ngày 2026-10-30: typecheck PASS, ESLint PASS, 26 test files/87 tests PASS. Chi tiết và giới hạn bằng chứng tại [`MVP_PUBLIC_RELEASE_VERIFICATION.md`](./MVP_PUBLIC_RELEASE_VERIFICATION.md).

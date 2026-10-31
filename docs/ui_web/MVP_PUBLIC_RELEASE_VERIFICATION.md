# MVP Public Frontend — Release verification

Tài liệu này ghi bằng chứng Giai đoạn 10 cho source Web User. Nó xác nhận regression phía frontend, không xác nhận backend/provider runtime.

## 1. Cổng kiểm tra

Lệnh chuẩn tại `apps/web-user`:

```text
npm run verify:mvp
```

Lệnh chạy tuần tự:

1. `node scripts/typecheck.mjs` — chạy `next typegen` và `tsc --noEmit`, sau đó khôi phục `next-env.d.ts` generated để working tree không bị đổi theo dev/typegen mode.
2. `eslint .`
3. `vitest run`

Vitest dùng cùng alias `@` với TypeScript. Test files chạy tuần tự trong một worker dùng chung để tránh lỗi transform file tạm trên Windows sandbox.

## 2. Kết quả local

| Kiểm tra | Kết quả ngày 2026-10-31 |
| --- | --- |
| Next route type generation | PASS |
| TypeScript `--noEmit` | PASS |
| ESLint toàn bộ Web User | PASS |
| Vitest | PASS — 27 files, 89 tests |
| Next.js production build | PASS — 18 static pages generated; dynamic BFF/routes compiled |
| `git diff --check` | PASS |

## 3. Regression quan trọng

| Phạm vi | Bằng chứng |
| --- | --- |
| Token trust boundary | Browser `Authorization` bị bỏ qua; chỉ cookie `oma_access_token` tạo Bearer upstream. |
| Unauthorized | Không có trusted token trả `401 AUTH_REQUIRED`. |
| Employee mapping | Employee không có `customer_id` trả `403 CUSTOMER_MAPPING_REQUIRED`; có mapping mới nhận Customer access. |
| Permission/capability | Principal hợp lệ nhưng thiếu capability trả `403 CAPABILITY_FORBIDDEN`. |
| Principal projection | Customer/Employee, role, permission và Customer mapping được parse tách biệt; token không lộ trong session response. |
| Profile contract | Email read-only không bị validate/gửi lại trong Profile mutation. |
| Catalog contract | Availability dùng `AVAILABLE`, `UNAVAILABLE`, `UNKNOWN`; test cũ không còn dựa vào boolean tự suy diễn. |
| Full-cart Checkout | Preparation giữ toàn bộ dòng Cart tại đúng revision; quote bắt buộc đúng một dạng địa chỉ. |
| Order pagination | BFF chỉ nhận `limit/status/cursor`, từ chối query trùng/không hỗ trợ và giữ cursor opaque. |
| Minimal tracking | DTO chỉ nhận `shipment_id` nullable và `order_status`; từ chối timeline/checkpoint tự thêm. |

## 4. Sửa hạ tầng kiểm tra

- Thêm `vitest.config.mts` để resolve `@` về `src` giống `tsconfig.json`.
- Tắt file parallelism/isolation và dùng một worker, loại bỏ lỗi transform `ENOENT` không ổn định trong Windows sandbox.
- Thêm script `verify:mvp` để developer và CI chạy cùng một chuỗi kiểm tra.

## 5. Giới hạn bằng chứng

Các mục sau vẫn cần môi trường backend/gateway thật:

- OIDC Authorization Code + PKCE, refresh, logout/revocation và token expiry.
- Gateway xác minh issuer/audience/signature/JWKS.
- Employee → Customer mapping với tài khoản runtime thật.
- 30-operation public gateway, provider adapter và ownership isolation.
- Checkout `201/202`, operation polling, Payment projection và idempotency end-to-end.
- Capability được bật/tắt theo registry runtime acceptance.

Frontend regression xanh không được dùng để đánh dấu owning-service acceptance hoặc production readiness khi các mục trên còn pending.

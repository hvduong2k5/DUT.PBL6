# Ô Mạ Web User

Next.js App Router application cho Web D2C. Hiện có Authentication, Customer Profile & Address Book, Product Discovery, Product Detail, Shopping Cart, Checkout, Payment và Order Management MVP.

Product Discovery dùng Mockoon `oma-product-discovery-mvp.json` tại port `4012`. Sau khi chạy mock và web app, mở `http://localhost:3000/products`. Có thể kiểm tra state lỗi/loading bằng `?mockScenario=catalog-error` hoặc `?mockScenario=catalog-slow`.

## Local development

### Return & Refund development

Từ `apps/web-user`, khởi động Return & Refund Mockoon (`4018`) và Next.js. Script chỉ bật mock cần cho feature này:

```powershell
npm run dev:returns
```

Các route kiểm tra chính: `/orders/order-mock-20260926-001/after-sales/request` và `/after-sales/return-case-mock-001`.

### Order Management development

Từ `apps/web-user`, khởi động Order Management Mockoon (`4017`) và Next.js. Script chỉ bật mock cần cho feature này để giảm tài nguyên; dùng script của feature trước nếu cần kiểm tra xuyên suốt luồng mua hàng:

```powershell
npm run dev:orders
```

Các route kiểm tra chính: `/orders/order-mock-20260926-001/confirmation`, `/account/orders`, `/orders/order-mock-20260926-001` và `/track-order`. Guest fixture local dùng Order `OMA-260926-001`, contact `0914288668`, OTP `789214`.

### Payment development

Từ `apps/web-user`, dùng một lệnh để khởi động toàn bộ luồng Product Discovery (`4012`) → Product Detail (`4013`) → Shopping Cart (`4014`) → Checkout (`4015`) → Payment (`4016`) và Next.js:

```powershell
npm run dev:payment
```

Mở `/cart?mockScenario=cart-prefilled`, chọn sản phẩm, sang Checkout và xác nhận Order. Ứng dụng sẽ chuyển đến `/payment/{orderId}`. Payment chỉ dùng tài khoản/QR demo và không được dùng để chuyển tiền thật.

Các trạng thái có thể kiểm tra bằng cách thêm query vào URL Payment: `payment-confirmed`, `payment-failed`, `payment-expired`, `payment-mismatch`, `payment-error`, `payment-slow` hoặc `payment-not-found`.

### Shopping Cart development

Từ `apps/web-user`, dùng một lệnh để khởi động Product Discovery (`4012`), Product Detail (`4013`), Shopping Cart (`4014`) và Next.js:

```powershell
npm run dev:cart
```

Script sẽ tái sử dụng Mockoon/Next.js đã chạy và tự khởi động port còn thiếu. Dừng toàn bộ tiến trình do script tạo bằng `Ctrl+C`.

Chạy riêng `npm run dev` **không** khởi động Mockoon. Khi đó `/api/cart` sẽ trả `503 CART_UPSTREAM_UNAVAILABLE` nếu port `4014` chưa chạy.

### Manual development

1. Khởi động các Mockoon environment cần dùng từ repository root (mỗi lệnh ở một terminal):

   ```powershell
   npx @mockoon/cli start --data .\mocks\mockoon\oma-auth-mvp.json
   npx @mockoon/cli start --data .\mocks\mockoon\oma-customer-profile-mvp.json
   npx @mockoon/cli start --data .\mocks\mockoon\oma-product-discovery-mvp.json
   npx @mockoon/cli start --data .\mocks\mockoon\oma-product-detail-mvp.json
   npx @mockoon/cli start --data .\mocks\mockoon\oma-shopping-cart-mvp.json
   npx @mockoon/cli start --data .\mocks\mockoon\oma-checkout-mvp.json
   npx @mockoon/cli start --data .\mocks\mockoon\oma-payment-mvp.json
   npx @mockoon/cli start --data .\mocks\mockoon\oma-order-management-mvp.json
   ```

2. Cài dependency và chạy web app:

   ```powershell
   cd apps\web-user
   Copy-Item .env.example .env.local
   npm install
   npm run dev
   ```

3. Mở `http://localhost:3000/products`, `/products/banh-ngu-sac-cung-dinh`, `/cart` hoặc đi qua Checkout để nhận quyền truy cập `/payment/{orderId}`.

Next Route Handler tại `/api/auth/*`, `/api/customer/*`, `/api/catalog/*`, `/api/cart/*`, `/api/checkout/*` và `/api/payments/*` làm BFF; component không gọi trực tiếp URL Mockoon. Header `X-Mock-Scenario` chỉ được forward ngoài production và có thể thử qua query `?mockScenario=<scenario-name>`.

Magic link local dùng URL mẫu:

- Registration: `/register?verificationToken=mock-token`
- Recovery: `/account-recovery?recoveryToken=mock-token`

## Quality checks

```powershell
npm test
npm run typecheck
npm run lint
npm run build
npm audit
```

Ứng dụng dùng Next.js 16 thay cho Next.js 14 ghi trong tài liệu kiến trúc ban đầu vì dòng 14 không còn bản vá cho các advisory hiện tại. App Router và toàn bộ ranh giới BFF vẫn giữ nguyên.

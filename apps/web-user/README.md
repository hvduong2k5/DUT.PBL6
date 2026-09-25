# Ô Mạ Web User

Next.js App Router application cho Web D2C. Hiện có Authentication, Customer Profile & Address Book, Product Discovery, Product Detail và Shopping Cart MVP.

Product Discovery dùng Mockoon `oma-product-discovery-mvp.json` tại port `4012`. Sau khi chạy mock và web app, mở `http://localhost:3000/products`. Có thể kiểm tra state lỗi/loading bằng `?mockScenario=catalog-error` hoặc `?mockScenario=catalog-slow`.

## Local development

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
   ```

2. Cài dependency và chạy web app:

   ```powershell
   cd apps\web-user
   Copy-Item .env.example .env.local
   npm install
   npm run dev
   ```

3. Mở `http://localhost:3000/products`, `/products/banh-ngu-sac-cung-dinh` hoặc `/cart`.

Next Route Handler tại `/api/auth/*`, `/api/customer/*`, `/api/catalog/*` và `/api/cart/*` làm BFF; component không gọi trực tiếp URL Mockoon. Header `X-Mock-Scenario` chỉ được forward ngoài production và có thể thử qua query `?mockScenario=<scenario-name>`.

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

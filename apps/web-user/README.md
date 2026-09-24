# Ô Mạ Web User

Next.js App Router application cho Web D2C. Hiện có Authentication MVP cùng Customer Profile & Address Book MVP.

## Local development

1. Khởi động hai Mockoon environment từ repository root (mỗi lệnh ở một terminal):

   ```powershell
   npx @mockoon/cli start --data .\mocks\mockoon\oma-auth-mvp.json
   npx @mockoon/cli start --data .\mocks\mockoon\oma-customer-profile-mvp.json
   ```

2. Cài dependency và chạy web app:

   ```powershell
   cd apps\web-user
   Copy-Item .env.example .env.local
   npm install
   npm run dev
   ```

3. Mở `http://localhost:3000/login`, `/account/profile` hoặc `/account/addresses`.

Next Route Handler tại `/api/auth/*` và `/api/customer/*` làm BFF proxy tới `AUTH_UPSTREAM_URL` và `CUSTOMER_UPSTREAM_URL`, do đó component không gọi trực tiếp URL Mockoon. Header `X-Mock-Scenario` chỉ được forward ngoài production và có thể thử qua query `?mockScenario=<scenario-name>`.

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

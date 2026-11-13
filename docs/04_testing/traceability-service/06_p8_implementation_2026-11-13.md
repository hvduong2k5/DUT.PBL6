# Triển khai và kiểm thử Traceability P8

**Ngày ghi nhận:** 13/11/2026 (UTC+7)
**Nhánh:** `feat/traceability-service` · **Mốc thiết kế:** `dev@239c8db`

## Thay đổi đã triển khai

1. Catalog đọc binding chính xác qua `GET /api/v1/internal/traceability/products/{product_id}/skus/{sku_code}`. Phiên bản nguồn kết hợp ID variant và revision của product/variant. Inventory cung cấp `omamx.traceability.v1.BatchSource/GetTraceabilityBatch` theo UUID và SKU, kể cả lô hết tồn hoặc bị hạn chế. Traceability đối chiếu cả hai nguồn khi submit và publish. Nếu snapshot đổi sau khi submit, publish trả conflict và hồ sơ cần qua duyệt lại.
2. Traceability integration xác minh token RS256 của Keycloak với audience `omama-api`, đối chiếu nhân viên còn hoạt động qua Identity `/api/v1/users/me` và chỉ dùng quyền GLOBAL có trong cả token lẫn quyền hiện hành. Migration Identity V12 đăng ký chín permission. Gán quyền cho vai trò soạn và duyệt riêng. Workflow giữ kiểm tra người đóng góp không được tự duyệt.
3. Kong chuyển public `/api/v2/trace/{qr_code}` đến Traceability `/api/v1/trace/{qr_code}` và chuyển admin `/api/v1/admin/trace/records` cùng các thao tác con. Web dùng upstream riêng khi cấu hình, giữ mã HTTP và hiển thị riêng QR đã ngừng công bố, dữ liệu thiếu và hồ sơ xác minh thủ công.

## Cấu hình môi trường tích hợp

1. Chạy Catalog với `CATALOG_TRACEABILITY_TOKEN` riêng, tối thiểu 32 byte. Inventory cần các migration terminal và batch version, profile `INVENTORY_TERMINAL_ENABLED=true`, TLS hợp lệ và `INVENTORY_TRACEABILITY_TOKEN` riêng. Hai API nguồn chỉ dùng trong mạng nội bộ.
2. Áp dụng Identity V12. Cấp các quyền trong tài liệu 09 cho hai tài khoản nhân viên riêng với scope GLOBAL và đồng bộ token Keycloak.
3. Tạo database Traceability riêng. Chạy `go run ./cmd/migrate -runtime-role <role>` với `APP_MODE=integration` và DSN của migrator. Khi khởi động API, chuyển sang DSN của runtime role. Điền các biến `TRACEABILITY_*` trong `.env.example`, dùng URL Web HTTPS và chứng thư CA phù hợp với tên máy chủ Inventory. Bind hồ sơ với `source_namespace=inventory` và UUID của bảng `batches`.
4. Từ thư mục Traceability, chạy `docker compose -f compose.integration.yaml up -d --build`. Đặt `TRACEABILITY_NETWORK` cùng mạng riêng với Kong. Đặt `TRACEABILITY_UPSTREAM_URL=http://kong-gateway:8000/api/v2/trace` nếu DNS của Gateway trên mạng đó là `kong-gateway`.

Route đã được thử với Kong DB-less đọc `config/kong.yml`. Kong production dùng PostgreSQL cần được provision hai service/route tương ứng trong cấu hình quản lý của môi trường đó.

## Kết quả kiểm thử

| Hạng mục | Kết quả |
| --- | --- |
| Traceability với PostgreSQL và Redpanda | PASS. `scripts/dev.ps1 -Action Test`, gồm test, vet và build. Coverage 77,1%. |
| Adapter nguồn và quyền nhân viên | PASS. Test Catalog HTTP/gRPC giả lập cho binding sai, nguồn lỗi, ngày hết hạn. Test JWT ký RS256 và Identity HTTP giả lập cho quyền GLOBAL, quyền bị thu hồi, tài khoản bị khóa và sai audience. |
| Catalog | PASS Docker build và HTTP smoke với PostgreSQL. Thiếu credential trả `401`, binding đúng với SKU đã lưu trữ trả `200`, SKU sai trả `404`. Gradle chạy trực tiếp trên Windows gặp lỗi loopback nên dùng build Linux trong Docker. |
| Inventory | PASS. `go test ./...` và test riêng RPC kiểm tra credential, binding và lô hạn chế/hết tồn. |
| Kong đến API Traceability sandbox | PASS. Kong 3.9.3 trả `404` cho mã lạ, `410` cho QR đã đình chỉ và `401` cho admin thiếu token. |
| Web | PASS TypeScript `tsc --noEmit` và HTTP Web BFF → Kong → sandbox. BFF giữ `404`, `410` và trả `Cache-Control: no-store`. |
| Contract MVP | PASS kiểm tra tĩnh 43 OpenAPI, 14 proto và Go/Python codegen. |
| Nghiệm thu liên service | PASS ngày 13/11/2026. Catalog và Inventory đọc PostgreSQL với dữ liệu seed. Inventory dùng gRPC TLS. Hai tài khoản test nhận token từ Keycloak 24.0.5 và được Identity đối chiếu quyền GLOBAL. Hồ sơ qua tạo → gửi duyệt → phê duyệt → publish → đọc QR qua Kong. |
| Ca lỗi liên service | PASS. Admin thiếu token trả `401`. Người soạn có quyền duyệt vẫn không tự duyệt được (`403`). Quyền bị thu hồi bị chặn dù token cũ còn hiệu lực (`403`). Lô không tồn tại, lô cách ly và thay đổi revision trước publish đều trả `409`. Hồ sơ publish thất bại vẫn trả `404` khi tra QR. |
| Trình duyệt | PASS. Edge chạy headless qua Web → BFF → Kong → Traceability integration. QR đã publish trả `200`, hiển thị xác minh thủ công và trường thiếu. Sau khi đình chỉ, QR trả `410` và Web hiển thị trạng thái ngừng công bố. Mã lạ trả `404`. |

## Lỗi phát hiện khi chạy thật

- Entry point của Traceability còn từ chối `APP_MODE=integration`. Đã bỏ điều kiện cũ để cấu hình integration đi qua wiring và kiểm tra database/adapter hiện hành.
- Dockerfile Inventory mặc định build binary ARM64. Đã dùng `TARGETARCH` do Docker BuildKit xác định và xác nhận container chạy trên amd64.

Môi trường E2E dùng mạng và database Docker tạm, lô seed và hai tài khoản riêng trong realm test. Kong chạy DB-less. Đã dọn các container và mạng test sau khi chạy. Kết quả áp dụng cho stack local này.

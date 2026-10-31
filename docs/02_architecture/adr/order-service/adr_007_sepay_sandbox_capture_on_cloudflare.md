# ADR007 SePay sandbox capture trên Cloudflare

Ngày24/10/2026. Accepted cho sandbox onboarding, không thay thế kiến trúc Payment09 hoặc bật Live.

## Bối cảnh

User yêu cầu cấu hình URL thật để nhập webhook SePay Test mode. Cloudflare inspection thấy tunnel c2c down/0connections, hostname c2c.npanhbao.dev thuộc kết nối riêng; local Docker đang tắt. Source chỉ có legacy VietQR simulator callback; route SePay mới vẫn là candidate. Đổi payload/auth của callback legacy hoặc giả ACK từ một route chưa lưu dữ liệu sẽ sai.

## Quyết định

Deploy Worker `order-sepay-sandbox` + D1 riêng `order-sepay-sandbox`, custom hostname `sepay-test.npanhbao.dev`. Giữ path `/api/v1/payments/sepay/webhook` để onboarding. Đây là capture tool nằm trong tools, không đăng ký như transport của Core hoặc tài chính. Không thay c2c tunnel/Kong/legacy callback, không cần máy local luôn bật.

HMAC-SHA256 theo SePay, raw bytes và timestamp±300s, constant-time qua WebCrypto verify, body tối đa64KiB và JSON. Chỉ ACK200 `success:true` sau D1 write thành công; failure503. POST only, không có read API công khai. Body/signature/timestamp được lưu AES256GCM với random96-bitnonce và purpose/receipt-bound AAD. Secrets trong Worker secret bindings và local user-only ACL backup ngoài Git, không in ra chat/log. Source/manifest không chứa secrets.

Dedup là observation identity `sha256(numericEventID:rawHash)`: exact retries tăng deliveries; cùng numeric ID khác raw giữ hai observations. Không dùng receipt hoặc numeric ID này làm financial UUID. Cho phép ID0 vì Dashboard send-test, không coi đó là verified bank receipt. Chưa receiver binding/API verify/import vào Core; không claim P1 hoàn chỉnh hoặc PAID.

## Hệ quả và gap còn mở

- URL luôn có thể nhận test trên Cloudflare, độc lập trạng thái máy local. Cloudflare deployment environment tên production chỉ là tên kỹ thuật, không có nghĩa payment Live.
- Hai observation payload khác nhau có thể cùng là một giao dịch; đây không phải financial double credit vì không có money effect.
- Không có tự động replay vào Order Service, rewrap/rotation hoặc automatic retention. Trước khi sử dụng production phải thay capture bằng runtime đã nghiệm thu và áp đủ gates Payment09. Evidence test được quản lý/xóa riêng sau đợt test; không tái sử dụng schema này làm ledger.
- Custom CPU limit không được plan Free hỗ trợ; bỏ tùy chọn này, không nâng plan.
- Kiểm thử local dùng fake D1 để kiểm crypto/ACK/error paths; actual remote probe + D1 query kiểm runtime/storage thật. Actual sender delivery cần user hoàn tất Dashboard và Gửi thử; probe tự ký không thay thế evidence SePay.

## Artifacts và evidence

[Source và vận hành](../../../../services/order-service/tools/sepay-sandbox/README.md), [tests](../../../../services/order-service/tools/sepay-sandbox/worker.test.mjs), [probe](../../../../services/order-service/tools/sepay-sandbox/probe.mjs).

Protocol: [SePay HMAC](https://developer.sepay.vn/vi/sepay-webhooks/xac-thuc), [bank payload và ACK](https://developer.sepay.vn/vi/sepay-webhooks/tich-hop-webhook). Deployment: Cloudflare Workers module upload, custom domain, D1 binding. Chưa thay production/Core contracts.

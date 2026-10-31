# SePay sandbox endpoint verification

Ngày24/10/2026. User yêu cầu URL thực tế để cấu hình webhook SePay Test mode. Phạm vi là sandbox capture, không nghiệm thu Payment toàn luồng.

## Deployment

- URL: `https://sepay-test.npanhbao.dev/api/v1/payments/sepay/webhook`.
- Worker: `order-sepay-sandbox`, compatibility2026-10-24, Free plan hiện tại.
- D1 riêng: `order-sepay-sandbox`, region APAC, read replication disabled.
- Custom domain được Cloudflare gắn thành công, TLS/HTTPS đã kiểm từ máy local qua public Internet. Không sửa tunnel c2c hoặc gateway sandbox.
- HMAC/key mã hóa được bind bằng secret_text, backup local ngoài Git với user-only ACL. Không ghi giá trị trong báo cáo.

## Kết quả

`node --test services/order-service/tools/sepay-sandbox/worker.test.mjs`: PASS4 suites. Unit tests dùng fake D1, WebCrypto thật; kiểm raw HMAC, expired/future/tampered/missing signatures, malformed payload, body limit, encryption/decryption, retry/conflicting observation retention, storage failure503 và không có read route.

`node services/order-service/tools/sepay-sandbox/probe.mjs`: PASS qua hostname HTTPS thật, bao gồm:

| Check | Actual result |
| --- | --- |
| GET healthz |200, ready=true, payment_processing=false |
| Authenticated synthetic POST |200, exact JSON success=true |
| Same raw event sent twice | Cùng receipt ID |
| Sai signature / timestamp quá hạn |401 |
| GET webhook |405 |
| Public receipts read |404 |
| Query actual D1 | Một receipt, deliveries=2, encrypted envelope có mặt |
| Decrypt actual D1 receipt bằng local key | PASS AESGCM/AAD, raw SHA256 khớp và đúng payload synthetic |

Remote synthetic receipt ID: `974d913df15bf8472f169844e5263ef41dc08f74d195e2c4ad2e1277937326de`. Event ID0 dùng để đánh dấu probe/send-test, không có tiền/Order effect.

## Evidence boundary

Probe được tự ký bằng cùng secret trên máy local, không phải delivery từ SePay. Chưa claim Dashboard webhook đã tạo hoặc actual sender test đã pass. User cần nhập URL, chọn HMAC-SHA256, copy local webhook-secret vào Dashboard Test mode, rồi Gửi thử hoặc tạo giao dịch mô phỏng.

Chưa có financial API verification, receiver binding hoặc Core payment workflow. ACK200 chứng minh capture đã commit vào D1, không phải xác nhận thanh toán. Full Go regression không chạy lại vì không thay Go/domain/SQL migrations trong lượt này.

[Source/runbook](../../../services/order-service/tools/sepay-sandbox/README.md), [ADR007](../../02_architecture/adr/order-service/adr_007_sepay_sandbox_capture_on_cloudflare.md).

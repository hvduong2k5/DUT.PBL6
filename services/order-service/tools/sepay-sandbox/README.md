# SePay sandbox webhook receiver

URL để nhập vào Dashboard **Test mode**:

```text
https://sepay-test.npanhbao.dev/api/v1/payments/sepay/webhook
```

URL này nhận HTTP POST thông báo giao dịch từ SePay. Chạy trên Cloudflare Worker + D1 riêng, không cần local PC/Docker/tunnel đang bật. Hostname `c2c.npanhbao.dev` và tunnel cũ giữ nguyên. Cloudflare gọi environment triển khai là `production`; đây vẫn là ứng dụng capture sandbox, không phải luồng thanh toán Live.

## Cấu hình trên SePay

- Tên: `Order Service - SePay Test`.
- Tiền vào, JSON, retry bật. Chọn đúng tài khoản ngân hàng test.
- Không lọc prefix, tắt bỏ qua giao dịch không có mã; mã không nhận diện vẫn cần evidence.
- Bảo mật: **HMAC-SHA256**. Copy nội dung file `%LOCALAPPDATA%\PBL\sepay-test\webhook-secret.txt` vào Secret Key. File chỉ chứa secret, không dấu ngoặc hoặc tên biến. Không gửi nội dung vào chat hoặc commit.
- Lưu rồi dùng **Gửi thử** hoặc tạo giao dịch mô phỏng. GET bằng trình duyệt vào webhook trả405 là đúng vì route chỉ nhậnPOST; health ở `https://sepay-test.npanhbao.dev/healthz`.

HMAC dùng chính bytes gốc, `timestamp.raw_body`, headers X-SePay-Timestamp/X-SePay-Signature, window±300s. D1 lưu receipt mã hóa AES256GCM trước ACK200 với body `{"success":true}`. Receipt đồng nhất theo numeric event ID + raw hash; replay chỉ tăng deliveries. Cùng ID nhưng payload khác được giữ riêng để xem xét, không tự ghi đè hoặc quyết định nghiệp vụ. Mã0 được phép vì SePay Dashboard dùng dữ liệu gửi thử.

Key mã hóa ở Worker secret CAPTURE_KEY; backup local `%LOCALAPPDATA%\PBL\sepay-test\capture-key.txt`. Không nhập key này vào SePay. Thư mục secret được giới hạn ACL theo Windows user. Giữ backup key để đọc lại evidence; không tự đổi key khi còn dữ liệu cũ. Không có API công khai để liệt kê hoặc đọc receipts. Worker không log body, secret hoặc thông tin tài khoản.

## Giới hạn

Đây là **capture và xác thực đường truyền sandbox**, không phải Core inbox/P1 hoàn chỉnh. Chưa có provider API verification, account/connection binding, financial dedup, ledger, PAID transition hoặc background importer về Order Service. ACK chỉ xác nhận lưu notification, không xác nhận tiền/đơn. Không dùng cho Live. Khi implement Payment runtime, giữ nguyên hoặc chuyển hostname có kiểm soát sang handler thật, ghi ADR và kiểm lại auth/ACK/retry trước chuyển.

Receipts là dữ liệu test, lưu riêng trong D1. Chưa có automatic retention; xóa evidence bằng quy trình được review sau đợt test, không áp dụng cleanup này lên financial records. HMAC hợp lệ chứng minh người giữ secret đã ký, không đủ chứng minh payload là tiền verified. Chưa tạo webhook thay user trong tài khoản SePay.

## Kiểm tra và triển khai lại

```text
node --test services/order-service/tools/sepay-sandbox/worker.test.mjs
node services/order-service/tools/sepay-sandbox/probe.mjs
node services/order-service/tools/sepay-sandbox/api-check.mjs
```

Probe gọi URL thật bằng dữ liệu synthetic, không gửi tiền. Nó ghi một receipt và gửi lại cùng raw payload một lần; kiểm D1 thấy deliveries=2. Đừng gọi đây là giao dịch do SePay gửi.

`api-check.mjs` đọc API Access Test từ `%LOCALAPPDATA%\PBL\sepay-test\api-token.txt`, chỉ GET host sandbox. Kiểm account/list/detail/filter/pagination/401/404; không log token, số tài khoản hoặc nội dung giao dịch. Private API snapshots chỉ lưu trong cùng thư mục local có ACL, không vào Git. Có thể truyền đường dẫn output summary làm argument thứ nhất. Đây là provider contract probe, không phải adapter đã tích hợp vào Core.

Để đối chiếu actual sender deliveries, export tối đa50 rows metadata+ciphertext từ D1 receipts bằng Cloudflare connector vào `%LOCALAPPDATA%\PBL\sepay-test\audit-receipts-private.json`, rồi chạy `node services/order-service/tools/sepay-sandbox/cross-check.mjs`. Script dùng local capture key để decrypt, kiểm hash và HMAC tại thời điểm nhận, correlate với API bằng reference/receiver/direction/amount/content/time. Không so numeric webhook ID với API UUID. Nó bỏ qua probe ID0, không import hoặc đổi trạng thái đơn; yêu cầu private account snapshot `bank-accounts-private.json` và transaction snapshot do api-check tạo. Khi test lại phải refresh các snapshots cùng đợt, không dùng export cũ để suy trạng thái mới.

Triển khai qua Cloudflare connector: áp dụng schema.sql vào đúng database ID trong deployment.json, upload worker.mjs dạng module và bindings DB/WEBHOOK_SECRET/CAPTURE_KEY, compatibility_date đã pin. Khi redeploy phải giữ hai secrets hiện có (inherit bindings hoặc đọc local backup), không drop chúng. Gắn Workers custom domain `sepay-test.npanhbao.dev`; không cần route qua Kong cho endpoint capture này. Không chỉnh DNS wildcard hoặc c2c tunnel. Không nâng gói Cloudflare; dùng cấu hình Free hiện có, không đặt custom CPU limit.

Xem [ADR007](../../../../docs/02_architecture/adr/order-service/adr_007_sepay_sandbox_capture_on_cloudflare.md).

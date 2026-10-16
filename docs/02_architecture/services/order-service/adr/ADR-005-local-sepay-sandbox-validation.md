# ADR-005 — Kiểm chứng Gateway sandbox từ máy phát triển

- Ngày: 2026-10-16.
- Trạng thái: Accepted cho môi trường phát triển; không cho phép production.

## Bối cảnh

Các kiểm thử trước dùng phản hồi SePay giả lập. Lần tích hợp merchant sandbox thật cho thấy HTTP 200 ở `/checkout/init` vẫn có thể là trang báo “Yêu cầu không hợp lệ”. Không được dùng riêng HTTP status làm bằng chứng đã tạo QR.

## Quyết định

1. Form từ Order luôn có `order_description` không rỗng, đứng trước các return URL và nằm trong chuỗi ký HMAC. Tham chiếu SDK ghi trường này là tùy chọn, nhưng thử trực tiếp ngày trên cho thấy form thiếu mô tả bị từ chối; thêm mô tả cho cùng invoice được chuyển tiếp sang trang checkout. Đây là lựa chọn tương thích đã quan sát, không phải khẳng định mọi merchant đều có quy tắc giống nhau.
2. Dùng database `order_mvp_sandbox` và `order_sepay_simulator` riêng qua `docker-compose.sepay-local.yml`. Không đổi cấu hình merchant cho các đơn đang chạy của profile cũ.
3. Merchant key chỉ ở `.env.local` được Git và Docker build context bỏ qua. Không ghi key, callback thô hay token khách hàng vào bằng chứng được commit.
4. Đường HTTPS tạm chỉ chuyển tiếp POST `/api/v1/payments/sepay/ipn`; kiểm tra `X-Secret-Key` trước khi gọi backend. Thêm GET `/healthz` và GET `/payment-result` trả nội dung cố định, không đọc trạng thái đơn hay tin query parameter. Không công khai endpoint cấp token, API quản trị hoặc simulator. Khi chưa cấu hình hay backend lỗi, trả lỗi; không trả 200 giả.
5. Trang khởi tạo demo chỉ mở ở loopback. Ưu tiên return URL HTTPS công khai qua `/payment-result`; lần thử có return URL loopback đã bị Cloudflare phía SePay chặn sau khi checkout init nhận form. Đây là quan sát, chưa kết luận quy tắc WAF cụ thể. Return URL không được dùng để xác nhận đã thanh toán; vẫn phải dựa trên IPN hoặc query đã xác thực. Tài khoản khách hàng, danh mục, kho và vận chuyển của demo vẫn là dữ liệu mô phỏng.

## Đánh đổi

- Có thêm một trường ký nhưng không thay đổi số tiền, invoice hay quy tắc phân bổ tiền.
- Tunnel miễn phí thuận tiện để thử; URL không bền vững và máy phải chạy. Khi restart phải kiểm tra và cập nhật cấu hình IPN trên SePay.
- Không thể xem callback tự gửi bằng key hợp lệ là bằng chứng SePay đã gửi IPN. Báo cáo phải phân biệt callback tự kiểm tra, query provider và IPN thật.
- Mỗi đơn demo có hạn thanh toán riêng. Hết hạn phải tạo đơn mới; không kéo dài hạn trong database chỉ để làm bài test vượt qua.

## Cách chạy

Từ thư mục gốc repository, sau khi chuẩn bị hai database ở trên:

```powershell
docker compose --env-file services/order-service/.env.local -f services/order-service/docker-compose.test.yml -f services/order-service/docker-compose.mvp.yml -f services/order-service/docker-compose.sepay-local.yml up -d api simulator
python services/order-service/scripts/sepay-ipn-proxy.py
# Ở terminal khác:
cloudflared tunnel --url http://127.0.0.1:18005 --no-autoupdate
# Cập nhật URL tunnel + /api/v1/payments/sepay/ipn trên SePay, chọn SECRET_KEY.
# Ở terminal khác:
python services/order-service/scripts/sepay-sandbox-demo.py
python -m http.server 18006 --bind 127.0.0.1 --directory .system_generated/order-sepay/public
```

Mở `http://127.0.0.1:18006/`, dùng thao tác giả lập trên trang SePay sandbox. Không chuyển tiền thật. Muốn tạo đơn demo mới, chạy lại script `sepay-sandbox-demo.py` rồi tải lại trang local.

## Tham chiếu

- [SDK Node.js chính thức](https://developer.sepay.vn/vi/cong-thanh-toan/sdk/nodejs).
- [Sandbox](https://developer.sepay.vn/vi/cong-thanh-toan/sandbox).
- [IPN và SECRET_KEY](https://developer.sepay.vn/vi/cong-thanh-toan/IPN).
- [Báo cáo implementation](../14_mvp_implementation_and_adr_report.md).

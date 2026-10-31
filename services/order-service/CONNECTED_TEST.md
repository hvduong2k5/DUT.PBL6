# Chạy luồng Order Service đã nối trong môi trường TEST

Ngày 24/10/2026. [ADR010](../../docs/02_architecture/adr/order-service/adr_010_connected_checkout_lifecycle_and_test_bridge.md) ghi các quyết định và gate; [báo cáo09](../../docs/04_testing/order-service/09_connected_order_service.md) ghi evidence. Bật bằng `ORDER_CORE_ENABLED=true` cùng `SEPAY_ENABLED=true`, chỉ `APP_MODE=sandbox`.

## Những gì đang được nối

Public Cart → Quote → durable CheckoutOperation → revalidate → Inventory reserve → atomic Order/Snapshot/Intent → SePay webhook observation → canonical API receipt → exact allocation → Inventory finalize proof → PAID → READY → packing/shipping source evidence → DELIVERED → Care completion → COMPLETED. COD không có Intent và cần stock COMMITTED trước PLACE. Có timeout/self-cancel/release, cleanup có điều kiện, retry/recovery, immutable outbox → Kafka và inbox/replay.

Catalog/Profile/Shipping/Care/Inventory trong stack này là owning-service **TEST facades** dùng PostgreSQL, HTTP/gRPC và command proofs; chưa thay thế triển khai các service thật. Chỉ SePay account scanner của runtime liên tục gọi API v2 Sandbox thật. Không dùng hosted gateway IPN hoặc API chuyển tiền.

## Khởi động / khởi động lại

Chạy từ repository root. Docker Desktop cần đang chạy. Dùng lại project/volume và cấu hình riêng; không đổi UUID/bootstrap/keys để làm sạch lỗi hoặc reset dedup.

Private directory hiện tại: `%LOCALAPPDATA%\PBL\sepay-test`. Có token/HMAC, canonical account response và các file runtime đã chuẩn bị. **Không đưa directory này vào Git hoặc gửi nguyên nội dung runtime.env/token cho người khác.**

```powershell
$privateDir = "$env:LOCALAPPDATA\PBL\sepay-test"
python services/order-service/scripts/prepare-connected-runtime.py --directory $privateDir
docker compose --env-file "$privateDir\runtime.env" -p codex-order-connected-20261024 -f services/order-service/docker-compose.test.yml -f services/order-service/docker-compose.connected.yml --profile tunnel up -d --build
```

Script import file vào named volume `pbl-order-connected-secrets` với owner10001/permission0400; tránh phụ thuộc bind mount Windows private directory. Script giữ các identity/keys đã tạo; mất khóa phải restore bản gốc. `cryptography` cần cho lần tạo Guest P-256 key đầu tiên. `runtime-settings.json`, `runtime-receiver.json`, `runtime-guest-key.pem`, `runtime.env` và `runtime-tunnel-token.txt` đều ở private directory.

API: `http://127.0.0.1:18004`, `/readyz`; Order gRPC19004. Simulator18104/Inventory gRPC19104 chỉ localhost. Project còn giữ PostgreSQL15434/Redis16384/Kafka19094. Dừng bằng cùng command Compose với `stop`; không `down -v` hoặc xóa secret volume khi muốn giữ dữ liệu. Nếu dev machine/Docker ngừng thì webhook bridge trả503, account sync ngừng; đây chưa phải hosting24/7.

## URL SePay có ý nghĩa gì

Giữ URL Dashboard:

`https://sepay-test.npanhbao.dev/api/v1/payments/sepay/webhook`

Tên miền `sepay-test.npanhbao.dev` đưa request tới Cloudflare Worker. Phần `/api/v1/payments/sepay/webhook` chọn endpoint nhận thông báo. Worker xác thực HMAC và lưu capture mã hóa vào D1, sau đó forward nguyên body/chữ ký tới `sepay-origin-test.npanhbao.dev` → tunnel mới → API8004 trong Docker. Backend lưu observation/job trước ACK. Bật retry ở Dashboard để SePay gửi lại khi backend không nhận được.

Origin chỉ mở **đúng webhook path**; `/internal/events`, `/readyz`, `/api/v1/cart` và simulator/token/admin không được route qua tunnel. `/healthz` trên hostname Worker chỉ cho biết capture/forwarding configuration, không chứng minh backend đang khỏe hoặc đơn đã thanh toán.

Tunnel `pbl-order-sepay-test-20261024`, ID `99dfc3ac-444a-4542-b5c5-49fe8e2ee5ce`. Không chỉnh tunnel `c2c-*`. Worker `FORWARD_ENABLED=true` đang bật; ID0 của Dashboard send-test vẫn capture-only. ACK `X-Payment-Processing: runtime-observation` nghĩa là backend đã nhận durable, **không phải PAID**. Webhook không được dùng làm financial authority.

Rollback bridge: deploy lại Worker gốc từ private backup `worker-before-bridge-20261024.json`, giữ nguyên D1 và secret bindings, hoặc tắt `FORWARD_ENABLED`. Không xóa D1/ledger, không đổi HMAC hoặc connection UUID để rollback. Nếu đã ACK capture trước đây, dùng scanner đối chiếu API; không sửa timestamp của chữ ký lịch sử để replay.

## Tạo đơn và nghiệm thu thanh toán từ Dashboard

Script sau tạo một Customer TEST, gọi các public API để tạo đơn/Intent thật của runtime, rồi lưu token và payment instructions vào private file. Không tạo giao dịch ngân hàng và không tự đánh dấu PAID:

```powershell
node services/order-service/scripts/smoke-connected-runtime.mjs "$env:LOCALAPPDATA\PBL\sepay-test"
```

`runtime-smoke-private.json` chứa `instructions.payment_code`, `amount`, `expires_at`, `transfer_content`, `qr_url`, `order_id` và token đọc API. Mỗi lượt tạo một đơn TEST mới; sử dụng mã của đơn còn hạn.

1. Trong SePay **Test mode**, tạo giao dịch tiền vào đúng account đã liên kết.
2. Nhập đúng `amount` và `transfer_content` có mã OMP từ instructions. Không dùng “Test QR chuyển khoản” hoặc mã OMP tự nghĩ.
3. Kiểm tra webhook delivery thành công và header runtime-observation. Chờ account scanner lấy canonical API transaction UUID.
4. Gọi `GET /api/v1/orders/{order_id}/payment` bằng token trong private file. Chỉ nghiệm thu khi allocated_amount bằng expected_amount, Order qua PAID và stock COMMITTED; READY còn phụ thuộc blockers.
5. Giao thiếu/thừa, trùng khoản, đến sau timeout hoặc sai mã đi reconciliation. Không sửa amount/expiry/snapshot hay mark-paid SQL để đạt kết quả.

Sau khi tạo giao dịch Test mode, dùng verifier chỉ đọc để chờ PAID/READY và đối chiếu canonical UUID, allocation, milestone, linked webhook và published outbox:

```powershell
node services/order-service/scripts/verify-sepay-sandbox-payment.mjs "$env:LOCALAPPDATA\PBL\sepay-test" --report-file "$env:TEMP\sepay-paid-acceptance.json" --wait-seconds 180
```

Verifier không tự tạo giao dịch hoặc sửa financial rows. SePay API date-filter query dùng local Vietnam datetime theo ADR011; checkpoint nội bộ vẫn UTC. Evidence cũ trước fix date filters không tự chứng minh bank coverage.


`--probe` gửi thêm một webhook ký HMAC giả lập qua URL public, rồi khẳng định Order vẫn PENDING_PAYMENT/allocated0. Nó dùng cho kiểm tra đường truyền và chống false-PAID; **không thay bước tạo giao dịch canonical trên Dashboard**. Script không in token/account hoặc raw webhook.

## Kiểm thử hồi quy và contract

Sau khi hạ tầng healthy, chạy container test bằng `--no-deps` để không recreate PostgreSQL/Kafka khi bộ test đang chạy:

```powershell
docker compose --env-file "$env:LOCALAPPDATA\PBL\sepay-test\runtime.env" -p codex-order-connected-20261024 -f services/order-service/docker-compose.test.yml -f services/order-service/docker-compose.connected.yml run --rm -T --no-deps tests sh -c 'go vet ./... && go test -race -json -count=1 ./...' > "$env:TEMP\order-connected-test.jsonl"
python services/order-service/scripts/verify-payment-evidence.py --test-log "$env:TEMP\order-connected-test.jsonl"
python services/order-service/scripts/verify-connected-evidence.py --test-log "$env:TEMP\order-connected-test.jsonl"
```

Test tạo/drop schema UUID riêng, RedisDB1 và source Kafka topic riêng; runtime dùng public schema/RedisDB0. Test có actual Kafka ingress/egress, HTTP/gRPC, race detector; không dùng cùng provider credential với runtime thật. Full flow tới COMPLETED dùng API SePay stand-in và rút ngắn riêng nonfinancial completion window; bank expiry/delivered fact không bị sửa.

## Còn giữ gate

Ngày25/10/2026 đã nghiệm thu canonical OMP payment từ Dashboard SePay Test mode → PAID/READY, xem [báo cáo10](../../docs/04_testing/order-service/10_sepay_sandbox_paid_acceptance.md) và [ADR011](../../docs/02_architecture/adr/order-service/adr_011_sepay_local_date_filters_and_sandbox_paid_acceptance.md). Chưa live. Chưa đủ real owning services/Identity/Finance consumer, full paid-cancel remote-stop/committed-stock reversal, advanced candidate control/query APIs, production TLS/key rotation/backup-restore drill. Sau task acceptance, admission-only hold bị từ chối409 thay vì giả vờ đã dừng fulfillment. Các gate này không làm mất bằng chứng luồng chính TEST đã nối nhưng cần hoàn thiện trước production.

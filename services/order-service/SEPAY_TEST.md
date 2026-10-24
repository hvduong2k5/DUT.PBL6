# Payment SePay/VietQR TEST runtime

Code theo [Payment09](../../docs/02_architecture/services/order-service/09_payment_sepay_vietqr_design.md), [resolution10](../../docs/02_architecture/services/order-service/10_payment_sepay_review_resolution.md), [ADR009](../../docs/02_architecture/adr/order-service/adr_009_sepay_payment_runtime_and_capability_gates.md). Mặc định tắt; startup chỉ chấp nhận `APP_MODE=sandbox`.

## Cấu hình

Các tên biến ở [.env.sepay.example](.env.sepay.example) là tài liệu; phải truyền vào process/container. Migration001–006 tự chạy trước khi EnableSePayFromEnv. Không chạy chung nhiều binary writer khác phiên bản migration trên cùng DB.

- `SEPAY_API_TOKEN_FILE`: token API v2 Sandbox64 ký tự; đọc file, không in token trong log/command line.
- `SEPAY_WEBHOOK_SECRET_FILE`: HMAC secret của Bank Webhooks Test mode, tối thiểu32 bytes; không phải API token. Nội dung file phải đúng secret, không thêm newline.
- `SEPAY_CONNECTION_ID`, `SEPAY_RECEIVER_ID`: UUID ổn định được cấp một lần cho merchant/environment/profile; giữ qua restart. `SEPAY_MERCHANT_IDENTITY` phải đúng connection đã lưu; không dùng connection mới để bỏ dedup.
- `SEPAY_EGRESS_SCOPE`: cùng một giá trị cho mọi replica/account dùng cùng API egress quota. `SEPAY_BOOTSTRAP_AT`: RFC3339, mốc bắt đầu account coverage, giữ ổn định qua restart.
- `SEPAY_RECEIVER_FILE`: JSON trusted config, mã hóa khi lưu SQL. Ví dụ toàn bộ là dữ liệu giả:

```json
{
  "bank_account_uuid": "11111111-1111-4111-8111-111111111111",
  "bank_code": "VCB",
  "bank_name": "Vietcombank",
  "account_number": "0000000000",
  "account_holder": "SYNTHETIC ONLY",
  "mode": "BANK_ACCOUNT",
  "currency": "VND"
}
```

Chỉ BANK_ACCOUNT/VND đang được hỗ trợ. Profile đã dùng không sửa bank/account/config tại chỗ. VA/SEVQR hoặc receiver rotation cần capability riêng.

`ORDER_ENCRYPTION_KEY` và audit/credential key hiện đi qua `CALLBACK_SECRET` phải ổn định để đọc evidence/dedup/Guest session. Không thay key để reset data; production cần keyring/migration riêng. File token/secret/profile/key lưu ngoài repository, chỉ user/service account được đọc.

`SEPAY_INVENTORY_GRPC` phải là Inventory3.2 có command proof. Bản này chỉ cho TEST internal network với internal token; không xem insecure gRPC sandbox là transport production. Nếu chưa cấu hình, receipt vẫn được đối chiếu nhưng Order chưa thể qua FINALIZING thành PAID.

Care/Finance URLs là base URL của owning authority theo ADR009, không nhận URL tùy ý từ public body. Evidence hosts là danh sách hostname HTTPS được phép, phân cách bằng dấu phẩy. Chưa có các authority này thì manual commands fail closed. Guest grants chỉ mint khi có PKCS8 PEM P-256 key và pinned key ID; sandbox Guest sessions tối đa24h, grants10m.

## Endpoint và cách hiểu trạng thái

`POST /api/v1/payments/sepay/webhook`: public ingress, provider-authenticated raw HMAC. `200 {"success":true}` chỉ xác nhận observation/work đã durable. Nó không chứng minh Order PAID. Wrong signature401, JSON lỗi400, quá lớn413, sai media415, lưu DB/job thất bại503. Dashboard ID0 không là bank transaction; dùng giao dịch test thực để nghiệm thu.

Owner reads: `GET /api/v1/orders/{id}/payment`, `GET /api/v1/orders/{id}/payment-instructions`. Generic Guest bearer không đọc được; mint/revoke grant qua `POST/DELETE /api/v1/orders/{id}/guest-payment-access` và logout qua `DELETE /api/v1/guest-sessions/current`. Reads `Cache-Control: no-store`. QR chỉ trả khi FAST coverage gần đây HEALTHY và intent còn đủ hạn; webhook observation chuyển UI VERIFYING, quá60s DELAYED mà không reset timer.

Staff routes nằm dưới `/api/v1/admin/payments`; bắt buộc permission, merchant/receiver scope và MFA cho thao tác nhạy cảm. `/internal/payments/refunds` chỉ nhận `approval_id` với internal authentication. Kong không expose `/internal`.

Luồng tài chính: Order+Intent atomic → webhook observation hoặc account scan → canonical API UUID receipt → exact allocation → FINALIZING → Inventory COMMITTED proof → PAID + milestone + outbox. Lệch tiền/extra/late/unmatched đi reconciliation; không yêu cầu khách chuyển bổ sung, không tự reopen đơn.

## Kiểm thử

```powershell
docker compose -p codex-order-payment-review -f docker-compose.test.yml run --rm -T tests
```

Chạy từ thư mục service, dùng project riêng và ports không bị chiếm. Command trong test container chạy `go vet ./...` và `go test -race -count=1 ./...` với PostgreSQL/Redis/Kafka thật. Các external HTTP/gRPC stand-ins không gửi tiền hoặc gọi bank mutation. Để đối chiếu actual JSON với candidate schemas, lưu output `go test -json` rồi chạy:

```powershell
python scripts/verify-payment-evidence.py --test-log <absolute-path-to-go-test.jsonl>
```

Nếu virtual clock lùi, domain trả deadline guard và không booking/paid. Kiểm tra đồng hồ DB/VM; không gỡ guard hoặc backdate quyết định để test pass. Ghi lại cả lần fail lẫn rerun. Chỉ dừng project test do mình tạo; không dọn dự án Docker khác.

## Connected runtime và URL TEST hiện tại

Ngày24/10/2026, ADR010 nối public target checkout/Inventory3.2 TEST/timeout/admission hold/publisher/lifecycle. Bật thêm `ORDER_CORE_ENABLED=true` và merchant UUID để dùng luồng này; không bật cờ thì legacy VIETQR vẫn bị `TARGET_CHECKOUT_NOT_READY` khi SePay enabled.

URL `https://sepay-test.npanhbao.dev/api/v1/payments/sepay/webhook` đã chuyển từ capture-only sang capture + forward signed bytes tới runtime TEST qua tunnel riêng. ID0 Dashboard probe vẫn capture-only. Backend ACK chỉ là observation durable, chưa là receipt/PAID. API Sandbox scanner thật đã có HEALTHY cho bốn stream, bao gồm response rỗng `last_page=0`.

Đọc [CONNECTED_TEST.md](CONNECTED_TEST.md) để khởi động lại stack, tạo Intent bằng public checkout và nghiệm thu giao dịch Dashboard chứa đúng OMP/amount trong hạn. Full-stack với stand-ins đã kiểm tới COMPLETED; canonical OMP transaction từ SePay Dashboard chưa được nghiệm thu. Owning authority, full cancellation/reversal và live gates vẫn cần hoàn thiện.

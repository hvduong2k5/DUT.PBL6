# ADR011 — Bộ lọc ngày SePay dùng giờ Việt Nam và nghiệm thu Sandbox PAID

Ngày25/10/2026. Accepted cho SePay TEST adapter. Bổ sung ADR009/010; không mở Live hoặc đổi financial deadline policy.

## Gap phát hiện trên provider thật

Tạo Order/Intent bằng public checkout, rồi dùng form **Mô phỏng giao dịch** trong SePay Dashboard Test mode để gửi tiền vào đúng account, amount250000 và nội dung OMP của Intent. SePay nhận diện đúng code, webhook qua Worker/tunnel đến backend; Order lên VERIFYING nhưng account scanner chưa nhận canonical transaction dù báo HEALTHY.

Kiểm GET API v2 Sandbox cho cùng một cửa sổ thời gian và cùng bank account:

| Query dates | Số giao dịch phù hợp |
| --- | --- |
| UTC RFC3339: `2026-10-25T08:28:00Z` → `08:32:00Z` |0 |
| Local Vietnam datetime: `2026-10-25 15:28:00` → `15:32:00` |1 |
| RFC3339 Vietnam offset: `2026-10-25T15:28:00+07:00` → `15:32:00+07:00` |1 |

Canonical transaction có `transaction_date=2026-10-25T15:29:11+07:00`. Đây là observed Sandbox behavior: UTC `Z` filters không chọn cùng instant như hai format local trên. Không suy hành vi này từ việc endpoint trả200 hoặc empty-page pagination hợp lệ.

## Quyết định

1. **Giữ time instants nội bộ và checkpoint ở UTC.** Chỉ adapter chuyển `Window.From/To` sang UTC+7 rồi format `2006-01-02 15:04:05` cho query `transaction_date_from/to`, khớp datetime format trong [tài liệu API](https://developer.sepay.vn/vi/sepay-api/v2/giao-dich/danh-sach). VND bank time Việt Nam không có DST trong runtime này.
2. **Giữ response explicit timezone.** Vẫn dùng `timestamp_format=iso8601`; transaction parser chuyển bank time về UTC. Không dùng chuỗi wall time làm timestamp nội bộ, không backdate allocation và không kéo dài Intent/reservation expiry để nhận tiền.
3. **Giữ identity và reconciliation.** Không reset connection/receiver, ledger hoặc checkpoint để làm test pass. Sau redeploy, scanner theo lịch hiện hữu đã tìm thấy receipt và finalize trong hạn gốc. Các coverage scan chạy với adapter cũ không đủ chứng minh bank coverage; khi áp dụng cho account khác cần authoritative backfill/overlap với stable identities, không suy bank_reconciled_through từ last_scanned_end.
4. **Evidence là giao dịch SePay Sandbox thực.** Giao dịch được tạo trên Dashboard Test mode, không chèn vào PostgreSQL receipts và không tự ký callback thay provider. API v2 trả canonical UUID; backend tự allocation, Inventory TEST gRPC finalize và PAID/FIRST_PAID/outbox. Inventory và owning services còn là TEST facades.
5. **Verifier chỉ đọc.** `verify-sepay-sandbox-payment.mjs` đọc Intent của public checkout, poll owner APIs, đối chiếu canonical API và SQL trace. Nó không gọi mutation ngân hàng, không mark-paid, không sửa financial row; report không chứa bearer/API token, account number/holder hoặc QR URL. Query SQL chỉ dùng order UUID đã kiểm format.
6. **Regression cần kiểm timezone boundary.** Thêm test HTTP client assert UTC instant phải trở thành local provider query, kể cả đổi ngày và input có timezone khác; response vẫn ISO8601. Full-stack stand-in không tự chứng minh filter semantics của provider thật.

## Acceptance

Xem [báo cáo10](../../../04_testing/order-service/10_sepay_sandbox_paid_acceptance.md). Gate canonical SePay Sandbox OMP → PAID đã có evidence. Gate Live, owning Inventory/Catalog/Shipping/Care/Identity/Finance thật, full paid cancellation, ops/restore và regression Kafka issue33 vẫn độc lập; không tự đóng theo thành công Payment này.
